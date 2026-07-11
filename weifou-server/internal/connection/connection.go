// Package connection 实现「交换名片」：两个都建了分身的用户互相进对方的名片夹。
// 合规取向：连接只留「关系 + 可问对方 AI + 异步举手」，不做真人实时私信（那是社交 IM，
// 需交友类目资质 + 内容审核）；真人实时沟通仍走各自公开的联系方式（出微信）。
package connection

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db        *gorm.DB
	security  *wechat.SecurityService
	subscribe *wechat.SubscribeService
	jwtSecret string
}

func NewHandler(db *gorm.DB, security *wechat.SecurityService, subscribe *wechat.SubscribeService, jwtSecret string) *Handler {
	return &Handler{db: db, security: security, subscribe: subscribe, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/connect/:profileId", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.connect))
	rg.GET("/connections", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.list))
}

type connectReq struct {
	Note string `json:"note"` // 交换时可选「捎句话」（点选意向），有则同时落一条 Lead 供主人离线跟进
}

// connect 当前用户与 profileId 的主人交换名片。要求当前用户自己已建分身，
// 否则回 NO_PROFILE 让前端引导去创建（天然裂变）。幂等：任一方向已连接则直接成功。
// 可带 note：交换=建立关系，note=携带具体诉求，二者一次点选完成（前端把意向点选折进交换弹层）。
func (h *Handler) connect(c *gin.Context) error {
	auth := middleware.Current(c)
	targetProfileID := c.Param("profileId")

	var req connectReq
	_ = c.ShouldBindJSON(&req) // note 可选，绑定失败（如空 body）不阻断
	note := strings.TrimSpace(req.Note)
	if len([]rune(note)) > 300 {
		return httpx.BadRequest("INPUT_TOO_LONG", "留言太长（限 300 字）")
	}
	// note 会进主人线索 + 订阅消息（微信会审），有内容就先过安全检测。
	if note != "" && h.security != nil && !h.security.CheckText(note, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "内容包含不当信息")
	}

	var target models.Profile
	if err := h.db.First(&target, "id = ?", targetProfileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "AI 分身不存在")
	}
	if target.UserID == auth.UserID {
		return httpx.BadRequest("CANNOT_CONNECT_SELF", "不能和自己交换名片")
	}

	// 必须先有自己的分身才能交换（前端据此把无分身访客导去创建）。
	var mine models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&mine).Error; err != nil {
		return httpx.BadRequest("NO_PROFILE", "先建一个你的 AI 分身才能交换名片")
	}

	// 幂等：任一方向已有边就当已连接（连接是互相的，不重复建）；但带了 note 仍要落线索/通知。
	var existing models.Connection
	already := h.db.Where(
		"(from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?)",
		auth.UserID, target.UserID, target.UserID, auth.UserID,
	).First(&existing).Error == nil

	if !already {
		h.db.Create(&models.Connection{
			ID:            idgen.New(),
			FromUserID:    auth.UserID,
			FromProfileID: mine.ID,
			ToUserID:      target.UserID,
			ToProfileID:   target.ID,
		})
	}

	// 带了「捎句话」→ 落一条线索，主人可离线跟进这条具体诉求。
	if note != "" {
		h.db.Create(&models.Lead{
			ID: idgen.New(), ProfileID: target.ID, VisitorOpenid: auth.Openid,
			Note: note, Status: models.LeadNew,
		})
	}

	// 通知对方本人（复用线索模板，避免新增模板 ID）。已连接且无 note 时不重复打扰。
	if !already || note != "" {
		go h.notifyConnected(target.UserID, mine.RealName, note)
	}

	httpx.OK(c, gin.H{"ok": true, "already": already})
	return nil
}

type connItem struct {
	ProfileID   string    `json:"profileId"`
	RealName    string    `json:"realName"`
	Title       string    `json:"title"`
	Company     *string   `json:"company,omitempty"`
	City        *string   `json:"city,omitempty"`
	AvatarStyle string    `json:"avatarStyle"`
	ConnectedAt time.Time `json:"connectedAt"`
}

// list 返回我的名片夹：与我相连的对方名片（我是发起或被连接方都算），按最近连接倒序。
func (h *Handler) list(c *gin.Context) error {
	auth := middleware.Current(c)

	var edges []models.Connection
	h.db.Where("from_user_id = ? OR to_user_id = ?", auth.UserID, auth.UserID).
		Order("created_at desc").Find(&edges)
	if len(edges) == 0 {
		httpx.OK(c, gin.H{"connections": []connItem{}})
		return nil
	}

	// 取每条边中「对方」的 profileId
	otherIDs := make([]string, 0, len(edges))
	connectedAt := make(map[string]time.Time, len(edges))
	for _, e := range edges {
		other := e.ToProfileID
		if e.ToUserID == auth.UserID {
			other = e.FromProfileID
		}
		if _, ok := connectedAt[other]; !ok {
			otherIDs = append(otherIDs, other)
			connectedAt[other] = e.CreatedAt
		}
	}

	var profiles []models.Profile
	h.db.Where("id IN ?", otherIDs).Find(&profiles)
	byID := make(map[string]models.Profile, len(profiles))
	for _, p := range profiles {
		byID[p.ID] = p
	}

	// 按 otherIDs（已是倒序）拼装，跳过已删除/查不到的
	items := make([]connItem, 0, len(otherIDs))
	for _, id := range otherIDs {
		p, ok := byID[id]
		if !ok {
			continue
		}
		items = append(items, connItem{
			ProfileID:   p.ID,
			RealName:    p.RealName,
			Title:       p.Title,
			Company:     p.Company,
			City:        p.City,
			AvatarStyle: p.AvatarStyle,
			ConnectedAt: connectedAt[id],
		})
	}
	httpx.OK(c, gin.H{"connections": items})
	return nil
}

func (h *Handler) notifyConnected(hostUserID, connectorName, note string) {
	if h.subscribe == nil {
		return
	}
	var host models.User
	if h.db.First(&host, "id = ?", hostUserID).Error != nil {
		return
	}
	openid := host.Openid
	if host.WxMpOpenid != nil && *host.WxMpOpenid != "" {
		openid = *host.WxMpOpenid
	}
	name := strings.TrimSpace(connectorName)
	if name == "" {
		name = "有人"
	}
	msg := name + " 和你交换了名片"
	page := "pages/connections/index"
	if note != "" {
		msg += "，还说：" + note
		page = "pages/inbox/index" // 带话的落收件箱，主人直接看到诉求
	}
	h.subscribe.NotifyNewLead(openid, msg, name, time.Now(), page)
}
