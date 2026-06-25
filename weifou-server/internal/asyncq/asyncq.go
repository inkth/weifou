// Package asyncq 实现「付费异步咨询」：访客付费向主人提问，主人在 SLA 时限内异步作答。
// 付费购买的是「主人本人作答」（真人服务），AI 不参与付费内容生产——这是变现合规的承重墙。
// 支付/退款/分账复用 payment 包；超时未答自动退款由 tasks 定时任务驱动。
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
	"weifou-server/internal/payment"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db        *gorm.DB
	pay       *payment.Handler
	security  *wechat.SecurityService
	subscribe *wechat.SubscribeService
	jwtSecret string
}

func NewHandler(db *gorm.DB, pay *payment.Handler, security *wechat.SecurityService, subscribe *wechat.SubscribeService, jwtSecret string) *Handler {
	return &Handler{db: db, pay: pay, security: security, subscribe: subscribe, jwtSecret: jwtSecret}
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

func normSource(s string) string {
	if s == "chat_card" {
		return "chat_card"
	}
	return "profile"
}

// ---------- 访客：付费提问下单 ----------

type createReq struct {
	ProfileID string `json:"profileId" binding:"required"`
	Question  string `json:"question" binding:"required"`
	Source    string `json:"source"`
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
	var setting models.ConsultSetting
	if err := h.db.First(&setting, "user_id = ?", profile.UserID).Error; err != nil || !setting.AsyncEnabled {
		return httpx.Forbidden("ASYNC_DISABLED", "对方未开放付费提问")
	}
	if setting.AsyncPrice < 100 {
		return httpx.BadRequest("PRICE_INVALID", "提问价格未正确设置")
	}
	if !h.security.CheckText(q, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "问题包含不当内容")
	}

	order := models.Order{
		ID: idgen.New(), OutTradeNo: idgen.WithPrefix("ASK"), Type: models.OrderAsyncQuestion,
		Amount: setting.AsyncPrice, ProfileID: profile.ID, PayerOpenid: auth.Openid,
		PayerUserID: &auth.UserID, PayeeUserID: profile.UserID,
		Source: normSource(req.Source),
	}
	if err := h.db.Create(&order).Error; err != nil {
		return httpx.Internal("ORDER_CREATE_FAILED", "下单失败")
	}
	h.db.Create(&models.AsyncQuestion{
		ID: idgen.New(), OrderID: order.ID, ProfileID: profile.ID,
		HostUserID: profile.UserID, AskerOpenid: auth.Openid, AskerUserID: &auth.UserID,
		Question: q, Price: setting.AsyncPrice, Status: models.AsyncPendingPayment,
	})

	// 复用支付下单（profitSharing=true：开启分账时冻结资金，作答提交时再分账）。
	return h.pay.PrepayOrder(c, &order, "付费提问 · "+profile.RealName, true)
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
	if qst.Status != models.AsyncPaid {
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
	// 原子作答：仅当仍为 paid 时置 answered，防止与超时自动退款竞态。
	res := h.db.Model(&models.AsyncQuestion{}).
		Where("id = ? AND status = ?", qst.ID, models.AsyncPaid).
		Updates(map[string]interface{}{
			"answer": ans, "voice_url": voiceURL, "voice_duration": dur,
			"answered_at": now, "status": models.AsyncAnswered,
		})
	if res.RowsAffected == 0 {
		return httpx.BadRequest("NOT_ANSWERABLE", "该提问当前不可回答（可能已超时退款）")
	}

	// 服务已交付，触发分账。
	go h.pay.Settle(qst.OrderID)

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
	} else {
		q = q.Where("status <> ?", models.AsyncPendingPayment) // 默认排除未支付
	}
	var items []models.AsyncQuestion
	q.Order("created_at desc").Limit(100).Find(&items)
	httpx.OK(c, h.decorate(items, "host"))
	return nil
}

func (h *Handler) myList(c *gin.Context) error {
	auth := middleware.Current(c)
	var items []models.AsyncQuestion
	h.db.Where("(asker_user_id = ? OR asker_openid = ?) AND status <> ?",
		auth.UserID, auth.Openid, models.AsyncPendingPayment).
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
		"question": q.Question, "price": q.Price, "status": q.Status,
		"answer": q.Answer, "voiceUrl": q.VoiceURL, "voiceDuration": q.VoiceDuration,
		"answeredAt": q.AnsweredAt, "answerDeadline": q.AnswerDeadline,
		"paidAt": q.PaidAt, "createdAt": q.CreatedAt, "role": role,
	}
}
