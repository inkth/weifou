// Package asyncq 实现「异步提问」：访客免费向主人提问，主人异步作答。
// 这是「AI 默认对外作答」之外的可选动作——让主人本人也能回一句，一问一答闭环（不是私信）。
// 不涉及任何支付/分账/退款；所有分身默认开放提问，无开关。
package asyncq

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
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.POST("/async-question", auth, httpx.Handle(h.create))
	rg.POST("/async-question/:id/answer", auth, httpx.Handle(h.answer))
	rg.GET("/async-question/host", auth, httpx.Handle(h.hostList))
	rg.GET("/async-question/mine", auth, httpx.Handle(h.myList))
	// detail 走 /detail/:id，避免与 host/mine 静态段在同层与 :id 通配冲突（Gin 会 panic）。
	rg.GET("/async-question/detail/:id", auth, httpx.Handle(h.detail))
}

// ---------- 访客：免费提问 ----------

type createReq struct {
	ProfileID string `json:"profileId" binding:"required"`
	Question  string `json:"question" binding:"required"`
}

func (h *Handler) create(c *gin.Context) error {
	auth := middleware.Current(c)
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	q := strings.TrimSpace(req.Question)
	if n := len([]rune(q)); n < 5 {
		return httpx.BadRequest("QUESTION_TOO_SHORT", "问题太短，请描述清楚一点")
	} else if n > 500 {
		return httpx.BadRequest("QUESTION_TOO_LONG", "问题太长了，请精简到 500 字内")
	}
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", req.ProfileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "主页不存在")
	}
	if profile.UserID == auth.UserID {
		return httpx.BadRequest("CANNOT_ASK_SELF", "不能向自己提问")
	}
	if !h.security.CheckText(q, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "问题包含不当内容")
	}

	question := models.AsyncQuestion{
		ID: idgen.New(), ProfileID: profile.ID,
		HostUserID: profile.UserID, AskerOpenid: auth.Openid, AskerUserID: &auth.UserID,
		Question: q, Status: models.AsyncPending,
	}
	if err := h.db.Create(&question).Error; err != nil {
		return httpx.Internal("QUESTION_CREATE_FAILED", "提交失败，请稍后重试")
	}

	go h.notifyHostNewQuestion(&question)

	httpx.OK(c, gin.H{"id": question.ID, "status": question.Status})
	return nil
}

// notifyHostNewQuestion 给主人下发「有新的提问」订阅消息（按小程序 openid，纯 App 主人会静默失败）。
func (h *Handler) notifyHostNewQuestion(q *models.AsyncQuestion) {
	if h.subscribe == nil {
		return
	}
	var host models.User
	if h.db.First(&host, "id = ?", q.HostUserID).Error != nil {
		return
	}
	openid := host.Openid
	if host.WxMpOpenid != nil && *host.WxMpOpenid != "" {
		openid = *host.WxMpOpenid
	}
	h.subscribe.NotifyNewQuestion(openid, q.Question, 0, time.Now(), "pages/inbox/index")
}

// ---------- 主人：作答 ----------

type answerReq struct {
	Answer        string `json:"answer"`
	VoiceURL      string `json:"voiceUrl"`
	VoiceDuration int    `json:"voiceDuration"`
}

func (h *Handler) answer(c *gin.Context) error {
	auth := middleware.Current(c)
	var req answerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	ans := strings.TrimSpace(req.Answer)
	voiceURL := strings.TrimSpace(req.VoiceURL)
	if ans == "" && voiceURL == "" {
		return httpx.BadRequest("ANSWER_EMPTY", "回答不能为空")
	}
	var qst models.AsyncQuestion
	if err := h.db.First(&qst, "id = ?", c.Param("id")).Error; err != nil {
		return httpx.NotFound("QUESTION_NOT_FOUND", "提问不存在")
	}
	if qst.HostUserID != auth.UserID {
		return httpx.Forbidden("NOT_HOST", "只有本人可以回答")
	}
	if qst.Status != models.AsyncPending {
		return httpx.BadRequest("NOT_ANSWERABLE", "该提问当前不可回答")
	}
	// 文字部分过安全审核；语音内容审核暂缺（MVP：仅本人可发，风险可控），后续可接音频审核。
	if ans != "" && !h.security.CheckText(ans, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "回答包含不当内容")
	}
	dur := req.VoiceDuration
	if dur < 0 {
		dur = 0
	}
	now := time.Now()
	// 原子作答：仅当仍为 pending 时置 answered。
	res := h.db.Model(&models.AsyncQuestion{}).
		Where("id = ? AND status = ?", qst.ID, models.AsyncPending).
		Updates(map[string]interface{}{
			"answer": ans, "voice_url": voiceURL, "voice_duration": dur,
			"answered_at": now, "status": models.AsyncAnswered,
		})
	if res.RowsAffected == 0 {
		return httpx.BadRequest("NOT_ANSWERABLE", "该提问当前不可回答")
	}

	// 通知访客「已回答」。
	if h.subscribe != nil && qst.AskerOpenid != "" {
		hostName := "本人"
		var hp models.Profile
		if h.db.Select("real_name").First(&hp, "user_id = ?", qst.HostUserID).Error == nil && hp.RealName != "" {
			hostName = hp.RealName
		}
		notifyText := ans
		if notifyText == "" {
			notifyText = "[语音回答]"
		}
		go h.subscribe.NotifyAnswered(qst.AskerOpenid, hostName, notifyText, now, "pages/my-questions/index")
	}

	httpx.OK(c, gin.H{"id": qst.ID, "status": models.AsyncAnswered, "answeredAt": now})
	return nil
}

// ---------- 列表 / 详情 ----------

func (h *Handler) hostList(c *gin.Context) error {
	auth := middleware.Current(c)
	q := h.db.Where("host_user_id = ?", auth.UserID)
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	var items []models.AsyncQuestion
	q.Order("created_at desc").Limit(100).Find(&items)
	httpx.OK(c, h.decorate(items, "host"))
	return nil
}

func (h *Handler) myList(c *gin.Context) error {
	auth := middleware.Current(c)
	var items []models.AsyncQuestion
	h.db.Where("asker_user_id = ? OR asker_openid = ?", auth.UserID, auth.Openid).
		Order("created_at desc").Limit(100).Find(&items)
	httpx.OK(c, h.decorate(items, "asker"))
	return nil
}

func (h *Handler) detail(c *gin.Context) error {
	auth := middleware.Current(c)
	var qst models.AsyncQuestion
	if err := h.db.First(&qst, "id = ?", c.Param("id")).Error; err != nil {
		return httpx.NotFound("QUESTION_NOT_FOUND", "提问不存在")
	}
	isHost := qst.HostUserID == auth.UserID
	isAsker := (qst.AskerUserID != nil && *qst.AskerUserID == auth.UserID) || qst.AskerOpenid == auth.Openid
	if !isHost && !isAsker {
		return httpx.Forbidden("NOT_PARTICIPANT", "无权查看")
	}
	role := "asker"
	if isHost {
		role = "host"
	}
	httpx.OK(c, h.row(&qst, role))
	return nil
}

func (h *Handler) decorate(items []models.AsyncQuestion, role string) []gin.H {
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, h.row(&items[i], role))
	}
	return out
}

func (h *Handler) row(q *models.AsyncQuestion, role string) gin.H {
	var profile models.Profile
	h.db.Select("real_name").First(&profile, "id = ?", q.ProfileID)
	return gin.H{
		"id": q.ID, "profileId": q.ProfileID, "realName": profile.RealName,
		"question": q.Question, "status": q.Status,
		"answer": q.Answer, "voiceUrl": q.VoiceURL, "voiceDuration": q.VoiceDuration,
		"answeredAt": q.AnsweredAt, "createdAt": q.CreatedAt, "role": role,
	}
}
