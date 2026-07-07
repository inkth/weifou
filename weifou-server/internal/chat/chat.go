package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"weifou-server/internal/answer"
	"weifou-server/internal/deepseek"
	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db        *gorm.DB
	rdb       *redis.Client
	engine    *answer.Engine
	security  *wechat.SecurityService
	subscribe *wechat.SubscribeService
	jwtSecret string
	freeQuota int
}

func NewHandler(db *gorm.DB, rdb *redis.Client, engine *answer.Engine, security *wechat.SecurityService, subscribe *wechat.SubscribeService, jwtSecret string, freeQuota int) *Handler {
	return &Handler{db: db, rdb: rdb, engine: engine, security: security, subscribe: subscribe, jwtSecret: jwtSecret, freeQuota: freeQuota}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/chat/:profileId/ask", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.ask))
	rg.POST("/chat/:profileId/lead", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.lead))
	rg.GET("/chat/sessions/mine", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.mySessions))
	rg.GET("/chat/sessions/host", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.hostSessions))
	rg.GET("/chat/sessions/:sessionId/messages", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.sessionMessages))
}

type sessionItem struct {
	SessionID   string    `json:"sessionId"`
	ProfileID   string    `json:"profileId"`
	RealName    string    `json:"realName"`
	AvatarURL   *string   `json:"avatarUrl"`
	LastMessage string    `json:"lastMessage"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// mySessions 返回"我作为访客"的最近对话列表（对话 Tab）。
func (h *Handler) mySessions(c *gin.Context) error {
	auth := middleware.Current(c)
	var sessions []models.ChatSession
	h.db.Where("visitor_openid = ?", auth.Openid).
		Order("created_at desc").Limit(50).Find(&sessions)

	items := make([]sessionItem, 0, len(sessions))
	for _, s := range sessions {
		var profile models.Profile
		if err := h.db.First(&profile, "id = ?", s.ProfileID).Error; err != nil {
			continue
		}
		var last models.ChatMessage
		h.db.Where("session_id = ?", s.ID).Order("created_at desc").First(&last)
		updated := s.UpdatedAt
		if !last.CreatedAt.IsZero() {
			updated = last.CreatedAt
		}
		var avatar *string
		var u models.User
		if err := h.db.First(&u, "id = ?", profile.UserID).Error; err == nil {
			avatar = u.AvatarURL
		}
		items = append(items, sessionItem{
			SessionID:   s.ID,
			ProfileID:   s.ProfileID,
			RealName:    profile.RealName,
			AvatarURL:   avatar,
			LastMessage: last.Content,
			UpdatedAt:   updated,
		})
	}
	// 按最近活动时间倒序。
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	httpx.OK(c, items)
	return nil
}

type hostSessionItem struct {
	SessionID     string    `json:"sessionId"`
	VisitorName   string    `json:"visitorName"`
	VisitorAvatar *string   `json:"visitorAvatar"`
	LastMessage   string    `json:"lastMessage"`
	MessageCount  int64     `json:"messageCount"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// hostSessions 会话回放（主人侧）：我的 AI 助理替我接待了哪些访客、聊了什么。
// 主人敢把链接发给重要的人的前提，是能看到助理替自己说过什么。
func (h *Handler) hostSessions(c *gin.Context) error {
	auth := middleware.Current(c)
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建主页")
	}
	var sessions []models.ChatSession
	h.db.Where("profile_id = ?", profile.ID).Order("updated_at desc").Limit(50).Find(&sessions)

	items := make([]hostSessionItem, 0, len(sessions))
	for _, s := range sessions {
		var count int64
		h.db.Model(&models.ChatMessage{}).Where("session_id = ?", s.ID).Count(&count)
		if count == 0 {
			continue // 进了对话但一句没说的会话不展示
		}
		var last models.ChatMessage
		h.db.Where("session_id = ?", s.ID).Order("created_at desc").First(&last)

		name := "匿名访客"
		var avatar *string
		var u models.User
		if err := h.db.First(&u, "openid = ?", s.VisitorOpenid).Error; err == nil {
			if u.Nickname != nil && *u.Nickname != "" {
				name = *u.Nickname
			}
			avatar = u.AvatarURL
		}
		if s.VisitorOpenid == auth.Openid {
			name = "我自己（试聊）"
		}
		updated := s.UpdatedAt
		if !last.CreatedAt.IsZero() {
			updated = last.CreatedAt
		}
		items = append(items, hostSessionItem{
			SessionID:     s.ID,
			VisitorName:   name,
			VisitorAvatar: avatar,
			LastMessage:   last.Content,
			MessageCount:  count,
			UpdatedAt:     updated,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	httpx.OK(c, items)
	return nil
}

// sessionMessages 返回单个会话的消息流；仅会话访客本人或被访主页的主人可看。
func (h *Handler) sessionMessages(c *gin.Context) error {
	auth := middleware.Current(c)
	sid := c.Param("sessionId")
	var s models.ChatSession
	if err := h.db.First(&s, "id = ?", sid).Error; err != nil {
		return httpx.NotFound("SESSION_NOT_FOUND", "会话不存在")
	}
	allowed := s.VisitorOpenid == auth.Openid
	if !allowed {
		var profile models.Profile
		if err := h.db.First(&profile, "id = ?", s.ProfileID).Error; err == nil && profile.UserID == auth.UserID {
			allowed = true
		}
	}
	if !allowed {
		return httpx.Forbidden("FORBIDDEN", "无权查看该会话")
	}
	var msgs []models.ChatMessage
	h.db.Where("session_id = ?", sid).Order("created_at asc").Limit(200).Find(&msgs)
	out := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, gin.H{"role": m.Role, "content": m.Content, "createdAt": m.CreatedAt})
	}
	httpx.OK(c, out)
	return nil
}

type askReq struct {
	Content string `json:"content" binding:"required"`
}

func (h *Handler) quotaKey(profileID, openid string) string {
	d := time.Now().UTC()
	return fmt.Sprintf("chat:quota:%d%02d%02d:%s:%s", d.Year(), d.Month(), d.Day(), profileID, openid)
}

func (h *Handler) checkQuota(ctx context.Context, profileID, openid string) error {
	key := h.quotaKey(profileID, openid)
	count, err := h.rdb.Incr(ctx, key).Result()
	if err != nil {
		// Redis 不可用时不阻断
		return nil
	}
	if count == 1 {
		h.rdb.Expire(ctx, key, 26*time.Hour)
	}
	if int(count) > h.freeQuota {
		return httpx.TooManyRequests("CHAT_QUOTA_EXCEEDED",
			fmt.Sprintf("今日提问已达上限（%d 条），明天再来", h.freeQuota))
	}
	return nil
}

func (h *Handler) ask(c *gin.Context) error {
	auth := middleware.Current(c)
	profileID := c.Param("profileId")
	var req askReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("EMPTY_INPUT", "请输入问题")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return httpx.BadRequest("EMPTY_INPUT", "请输入问题")
	}
	if len([]rune(content)) > 200 {
		return httpx.BadRequest("INPUT_TOO_LONG", "问题太长（限 200 字）")
	}

	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", profileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_READY", "主页未生成")
	}
	var p models.PersonaAI
	if err := h.db.First(&p, "profile_id = ?", profileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_READY", "主页未生成")
	}
	// 沟通风格直读 PersonaInput（查不到不阻断，style 为空即可）
	var input models.PersonaInput
	_ = h.db.First(&input, "profile_id = ?", profileID).Error

	if err := h.checkQuota(c.Request.Context(), profileID, auth.Openid); err != nil {
		return err
	}

	if !h.security.CheckText(content, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "问题包含不当内容")
	}

	// 取/建会话
	var session models.ChatSession
	err := h.db.Where("profile_id = ? AND visitor_openid = ?", profileID, auth.Openid).
		Order("created_at desc").First(&session).Error
	if err == gorm.ErrRecordNotFound {
		session = models.ChatSession{ID: idgen.New(), ProfileID: profileID, VisitorOpenid: auth.Openid}
		h.db.Create(&session)
	}

	h.db.Create(&models.ChatMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleUser,
		Content: content, SafeCheckStatus: models.SafePass,
	})

	var tags []string
	_ = json.Unmarshal(p.Tags, &tags)
	sys := answer.BuildSystemPrompt(&profile, p.OneLiner, p.FullIntro, tags, p.Tone, input.Style, h.engine.KnowledgeFor(profileID))

	// 最近 20 条历史（沉浸式对话需要更长记忆窗口）
	var recent []models.ChatMessage
	h.db.Where("session_id = ?", session.ID).Order("created_at desc").Limit(20).Find(&recent)
	msgs := []deepseek.Message{{Role: "system", Content: sys}}
	for i := len(recent) - 1; i >= 0; i-- {
		msgs = append(msgs, deepseek.Message{Role: recent[i].Role, Content: recent[i].Content})
	}

	result, err := h.engine.Complete(msgs)
	if err != nil {
		log.Printf("[ai] chat complete failed profile=%s openid=%s: %v", profileID, auth.Openid, err)
		return httpx.Internal("AI_UPSTREAM_ERROR", "AI 服务暂时不可用，请稍后再试")
	}

	safe := models.SafePass
	finalAnswer := result.Answer
	suggestions := result.Suggestions
	if !h.security.CheckText(finalAnswer+"\n"+strings.Join(suggestions, "\n"), auth.Openid) {
		safe = models.SafeReject
		finalAnswer = "抱歉，这个问题不方便由 AI 直接回答，建议联系本人沟通。"
		suggestions = nil
	}
	h.db.Create(&models.ChatMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleAssistant,
		Content: finalAnswer, SafeCheckStatus: safe,
	})

	// 答不上来 → 记录知识缺口，回流给主人补充（对话飞轮的喂养端）
	if result.Gap && safe == models.SafePass {
		h.recordGap(profileID, content)
	}

	// suggestions：分身生成的追问候选（点选优先——访客点一下即发出，代替打字）
	httpx.OK(c, gin.H{"sessionId": session.ID, "answer": finalAnswer, "suggestions": suggestions})
	return nil
}

// recordGap upsert 一条知识缺口：同一问题已存在则累加次数，否则新建。
func (h *Handler) recordGap(profileID, question string) {
	q := strings.TrimSpace(question)
	if q == "" {
		return
	}
	if len([]rune(q)) > 200 {
		q = string([]rune(q)[:200])
	}
	now := time.Now()
	var existing models.KnowledgeGap
	err := h.db.Where("profile_id = ? AND question = ? AND status = ?",
		profileID, q, models.GapOpen).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		h.db.Create(&models.KnowledgeGap{
			ID: idgen.New(), ProfileID: profileID, Question: q,
			AskedCount: 1, Status: models.GapOpen, LastAskedAt: now,
		})
		return
	}
	if err == nil {
		h.db.Model(&existing).Updates(map[string]interface{}{
			"asked_count":   existing.AskedCount + 1,
			"last_asked_at": now,
		})
	}
}

type leadReq struct {
	Note    string  `json:"note"`
	Contact *string `json:"contact"`
}

// lead 访客在对话内留言/留资 → 生成线索通知主人（零门槛、iOS 合规的轻成交）。
func (h *Handler) lead(c *gin.Context) error {
	auth := middleware.Current(c)
	profileID := c.Param("profileId")
	var req leadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return httpx.BadRequest("EMPTY_INPUT", "请填写留言内容")
	}
	if len([]rune(note)) > 300 {
		return httpx.BadRequest("INPUT_TOO_LONG", "留言太长（限 300 字）")
	}

	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", profileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "主页不存在")
	}

	contactCheck := note
	if req.Contact != nil {
		contactCheck += "\n" + *req.Contact
	}
	if !h.security.CheckText(contactCheck, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "内容包含不当信息")
	}

	// 关联访客与本主页最近的会话（便于主人查看上下文）
	var sessionID *string
	var session models.ChatSession
	if err := h.db.Where("profile_id = ? AND visitor_openid = ?", profileID, auth.Openid).
		Order("created_at desc").First(&session).Error; err == nil {
		sessionID = &session.ID
	}

	var contact *string
	if req.Contact != nil {
		if cc := strings.TrimSpace(*req.Contact); cc != "" {
			if len([]rune(cc)) > 100 {
				return httpx.BadRequest("INPUT_TOO_LONG", "联系方式太长")
			}
			contact = &cc
		}
	}

	h.db.Create(&models.Lead{
		ID: idgen.New(), ProfileID: profileID, VisitorOpenid: auth.Openid,
		SessionID: sessionID, Note: note, Contact: contact, Status: models.LeadNew,
	})
	// 主人召回（推）：免费线索是比付费提问更高频的"有人找你"信号，异步下发订阅消息。
	go h.notifyHostNewLead(profile.UserID, note, auth.Openid)
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

// notifyHostNewLead 给主人下发「有新访客线索」订阅消息（按小程序 openid，纯 App 主人静默失败）。
func (h *Handler) notifyHostNewLead(hostUserID, note, visitorOpenid string) {
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
	name := "访客"
	var v models.User
	if h.db.Where("openid = ? OR wx_mp_openid = ? OR wx_app_openid = ?",
		visitorOpenid, visitorOpenid, visitorOpenid).First(&v).Error == nil {
		if v.Nickname != nil && *v.Nickname != "" {
			name = *v.Nickname
		}
	}
	h.subscribe.NotifyNewLead(openid, note, name, time.Now(), "pages/inbox/index")
}
