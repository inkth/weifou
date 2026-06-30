package visit

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

type Handler struct {
	db        *gorm.DB
	jwtSecret string
}

func NewHandler(db *gorm.DB, jwtSecret string) *Handler {
	return &Handler{db: db, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/visit/:profileId", middleware.OptionalJWT(h.jwtSecret), httpx.Handle(h.record))
	rg.GET("/visit/stats/mine", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.stats))
	rg.GET("/visit/visitors", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.visitors)) // 谁看过我（访客列表）
	rg.POST("/visit/event/track", middleware.OptionalJWT(h.jwtSecret), httpx.Handle(h.trackEvent))
}

// 埋点事件类型白名单（与 weifou-miniapp/utils/track.js 同步维护）
var allowedEventTypes = map[string]bool{
	"share_tap":          true, // 分享菜单触发（chat/profile）
	"own_hook_show":      true, // 裂变钩子曝光（meta: light/strong）
	"own_hook_click":     true, // 裂变钩子点击
	"quick_create_enter": true, // 进入快速创建
}

type eventReq struct {
	Type      string `json:"type" binding:"required"`
	ProfileID string `json:"profileId"`
	Meta      string `json:"meta"`
}

// trackEvent 漏斗埋点：fire-and-forget，非法类型静默丢弃（不给客户端报错面）。
func (h *Handler) trackEvent(c *gin.Context) error {
	var req eventReq
	if err := c.ShouldBindJSON(&req); err != nil || !allowedEventTypes[req.Type] {
		httpx.OK(c, gin.H{"ok": true})
		return nil
	}
	if len(req.ProfileID) > 32 {
		req.ProfileID = req.ProfileID[:32]
	}
	if len(req.Meta) > 256 {
		req.Meta = req.Meta[:256]
	}
	e := models.Event{ID: idgen.New(), Type: req.Type, ProfileID: req.ProfileID, Meta: req.Meta}
	if auth := middleware.Current(c); auth != nil && auth.Openid != "" {
		e.Openid = &auth.Openid
	}
	h.db.Create(&e)
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

func (h *Handler) record(c *gin.Context) error {
	profileID := c.Param("profileId")
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", profileID).Error; err != nil {
		httpx.OK(c, gin.H{"ok": true})
		return nil
	}

	auth := middleware.Current(c)
	v := models.Visit{ID: idgen.New(), ProfileID: profileID}
	if auth != nil && auth.Openid != "" {
		v.VisitorOpenid = &auth.Openid
	}
	ip := c.ClientIP()
	if fwd := c.GetHeader("X-Forwarded-For"); fwd != "" {
		ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if ip != "" {
		sum := sha256.Sum256([]byte(ip))
		hash := hex.EncodeToString(sum[:])[:32]
		v.VisitorIPHash = &hash
	}
	if ua := c.GetHeader("User-Agent"); ua != "" {
		if len(ua) > 200 {
			ua = ua[:200]
		}
		v.UserAgent = &ua
	}
	h.db.Create(&v)
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

func (h *Handler) stats(c *gin.Context) error {
	auth := middleware.Current(c)
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建主页")
	}
	var pv int64
	h.db.Model(&models.Visit{}).Where("profile_id = ?", profile.ID).Count(&pv)

	var uv int64
	h.db.Model(&models.Visit{}).
		Where("profile_id = ? AND visitor_openid IS NOT NULL", profile.ID).
		Distinct("visitor_openid").Count(&uv)

	var askCount int64
	h.db.Model(&models.ChatMessage{}).
		Joins("JOIN chat_sessions ON chat_sessions.id = chat_messages.session_id").
		Where("chat_sessions.profile_id = ? AND chat_messages.role = ?", profile.ID, models.RoleUser).
		Count(&askCount)

	// 主人收入可见：累计成交（已支付的打赏订单）+ 本月成交。
	// 通话/付费咨询已下线、分账已移除，C2C 收费仅剩打赏（自愿赠予）。incomeNet 保留为 0 兼容前端。
	payTypes := []string{models.OrderTip}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var incomeGross, incomeMonth int64
	h.db.Model(&models.Order{}).
		Where("payee_user_id = ? AND status = ? AND type IN ?", auth.UserID, models.OrderPaid, payTypes).
		Select("COALESCE(SUM(amount),0)").Scan(&incomeGross)
	h.db.Model(&models.Order{}).
		Where("payee_user_id = ? AND status = ? AND type IN ? AND created_at >= ?", auth.UserID, models.OrderPaid, payTypes, monthStart).
		Select("COALESCE(SUM(amount),0)").Scan(&incomeMonth)

	httpx.OK(c, gin.H{
		"profileId": profile.ID, "pv": pv, "uv": uv, "askCount": askCount,
		"incomeGross": incomeGross, "incomeMonth": incomeMonth, "incomeNet": 0,
	})
	return nil
}

// visitors 列出看过「我的名片」的访客（按访客去重、最近优先）。
// 只露登录访客（有 openid→user）；全实名口径下展示身份；能关联到名片的给 profileId（可点进看 TA 的名片）。
func (h *Handler) visitors(c *gin.Context) error {
	auth := middleware.Current(c)
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建主页")
	}

	type row struct {
		VisitorOpenid string
		LastAt        time.Time
	}
	var rows []row
	h.db.Model(&models.Visit{}).
		Select("visitor_openid, MAX(created_at) as last_at").
		Where("profile_id = ? AND visitor_openid IS NOT NULL AND visitor_openid <> ?", profile.ID, auth.Openid).
		Group("visitor_openid").
		Order("last_at desc").
		Limit(100).
		Scan(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		// openid → user（复用跨端 openid 链路）
		var u models.User
		if h.db.Where("openid = ? OR wx_mp_openid = ? OR wx_app_openid = ?",
			r.VisitorOpenid, r.VisitorOpenid, r.VisitorOpenid).First(&u).Error != nil {
			continue // 非登录用户/查不到 → 不露
		}
		name, avatar := "", ""
		if u.Nickname != nil {
			name = *u.Nickname
		}
		if u.AvatarURL != nil {
			avatar = *u.AvatarURL
		}
		item := gin.H{"name": name, "avatar": avatar, "lastVisitAt": r.LastAt, "profileId": ""}
		// 访客也建了名片 → 给 profileId + realName 兜底名字
		var vp models.Profile
		if h.db.Where("user_id = ?", u.ID).First(&vp).Error == nil {
			item["profileId"] = vp.ID
			if name == "" && vp.RealName != "" {
				item["name"] = vp.RealName
			}
		}
		if item["name"] == "" {
			item["name"] = "微信用户"
		}
		out = append(out, item)
	}
	httpx.OK(c, out)
	return nil
}
