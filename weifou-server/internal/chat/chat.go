package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"weifou-server/internal/deepseek"
	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/persona"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db        *gorm.DB
	rdb       *redis.Client
	ds        *deepseek.Client
	security  *wechat.SecurityService
	subscribe *wechat.SubscribeService
	jwtSecret string
	freeQuota int
}

func NewHandler(db *gorm.DB, rdb *redis.Client, ds *deepseek.Client, security *wechat.SecurityService, subscribe *wechat.SubscribeService, jwtSecret string, freeQuota int) *Handler {
	return &Handler{db: db, rdb: rdb, ds: ds, security: security, subscribe: subscribe, jwtSecret: jwtSecret, freeQuota: freeQuota}
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

func buildSystemPrompt(p *models.Profile, oneLiner, fullIntro string, tags []string, tone, style, knowledge string, canOffer bool) string {
	company := ""
	if p.Company != nil && *p.Company != "" {
		company = "（" + *p.Company + "）"
	}
	toneRule := ""
	if strings.TrimSpace(tone) != "" {
		toneRule = "\n== 人格与语气（务必贯穿每次回答）==\n" + tone +
			"\n保持上述性格与口吻的一致性，让访客感到在和一个有温度、可信赖的真人分身对话。\n"
	}
	// style 直读 PersonaInput：本人改风格后立即生效，不依赖重新生成 persona（tone 可能仍是旧基调）。
	if desc, ok := persona.StyleDescriptions[style]; ok {
		toneRule += "\n== 对外沟通风格（本人指定，优先级最高）==\n" + desc + "\n"
	}
	knowledgeRule := ""
	if strings.TrimSpace(knowledge) != "" {
		knowledgeRule = "\n== 补充资料（本人补充的问答，优先据此回答）==\n" + knowledge + "\n== 补充资料结束 ==\n"
	}
	offerRule := "本人当前未开放可预约档期，offerConsult 必须为 false。"
	if canOffer {
		offerRule = `当访客表达合作、咨询、深入沟通或约时间的意向时，将 offerConsult 设为 true，系统会在对话内展示真实的预约入口；answer 里可用助理口吻自然引导（如"我看一下 TA 的档期"），其余情况设为 false。严禁在 answer 里编造价格或具体时间——真实档期由系统展示。`
	}
	return fmt.Sprintf(`你是 %s 的 AI 助理。你以助理身份代他/她接待访客、介绍 TA，不是 %s 本人。
你代表主人的利益：亲和但不卑微，热情但有边界，像一位称职的助理那样有立场地接待。
当访客提问时，基于以下资料客观回答；超出资料范围时用助理口吻兜底（如"这个我帮你转达给本人""这个我记下来，TA 之后会补充"），不要编造。
请使用中文，自然、有温度、专业克制，单次回答不超过 200 字。

== 主页资料 ==
身份：%s%s
一句话介绍：%s
完整介绍：%s
标签：%s
== 资料结束 ==
%s%s
回答时：
- 不要透露你是大模型；以"他/她"或直接陈述事实的方式回答。
- 如果问题与该用户的专业方向无关，礼貌带回。

== 输出格式 ==
只输出一个 JSON 对象，不要任何额外文字或代码块：
{"answer": "给访客的回答", "offerConsult": true 或 false, "gap": true 或 false}
answer：中文回答，规则同上，不超过 200 字。
offerConsult：%s
gap：当且仅当访客问的是一个具体、合理、但现有资料无法回答的问题（你只能含糊带过或建议联系本人）时设为 true，用于提醒本人补充资料；闲聊、寒暄、与专业方向无关或资料已能回答的问题一律为 false。`,
		p.RealName, p.RealName, p.Title, company, oneLiner, fullIntro, strings.Join(tags, "、"), toneRule, knowledgeRule, offerRule)
}

// aiResult 是模型按要求返回的 JSON 结构。
type aiResult struct {
	Answer       string `json:"answer"`
	OfferConsult bool   `json:"offerConsult"`
	Gap          bool   `json:"gap"`
}

type slotBrief struct {
	ID          string    `json:"id"`
	StartAt     time.Time `json:"startAt"`
	DurationMin int       `json:"durationMin"`
}

type offerCard struct {
	Type    string      `json:"type"` // "consult_offer"
	Price30 int         `json:"price30"`
	Price60 int         `json:"price60"`
	Slots   []slotBrief `json:"slots"`
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

	// 是否可在对话内促成预约：本人已开放付费咨询 + 存在未来开放档期
	setting, openSlots := h.consultOffer(profile.UserID)
	canOffer := setting != nil && setting.Enabled && len(openSlots) > 0

	var tags []string
	_ = json.Unmarshal(p.Tags, &tags)
	sys := buildSystemPrompt(&profile, p.OneLiner, p.FullIntro, tags, p.Tone, input.Style, h.knowledgeFor(profileID), canOffer)

	// 最近 20 条历史（沉浸式对话需要更长记忆窗口）
	var recent []models.ChatMessage
	h.db.Where("session_id = ?", session.ID).Order("created_at desc").Limit(20).Find(&recent)
	msgs := []deepseek.Message{{Role: "system", Content: sys}}
	for i := len(recent) - 1; i >= 0; i-- {
		msgs = append(msgs, deepseek.Message{Role: recent[i].Role, Content: recent[i].Content})
	}

	raw, err := h.ds.Chat(msgs, deepseek.ChatOptions{Temperature: 0.6, MaxTokens: 600, ResponseFormat: "json_object"})
	if err != nil {
		return httpx.Internal("AI_UPSTREAM_ERROR", "AI 服务暂时不可用，请稍后再试")
	}

	// 解析 JSON 输出；失败则降级为纯文本，不展示卡片
	var result aiResult
	if jerr := json.Unmarshal([]byte(raw), &result); jerr != nil || strings.TrimSpace(result.Answer) == "" {
		result = aiResult{Answer: strings.TrimSpace(raw), OfferConsult: false}
	}

	safe := models.SafePass
	finalAnswer := result.Answer
	if !h.security.CheckText(finalAnswer, auth.Openid) {
		safe = models.SafeReject
		finalAnswer = "抱歉，这个问题不方便由 AI 直接回答，建议联系本人沟通。"
		result.OfferConsult = false
	}
	h.db.Create(&models.ChatMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleAssistant,
		Content: finalAnswer, SafeCheckStatus: safe,
	})

	// 答不上来 → 记录知识缺口，回流给主人补充（对话飞轮的喂养端）
	if result.Gap && safe == models.SafePass {
		h.recordGap(profileID, content)
	}

	resp := gin.H{"sessionId": session.ID, "answer": finalAnswer}
	if result.OfferConsult && canOffer {
		briefs := make([]slotBrief, 0, len(openSlots))
		for _, s := range openSlots {
			briefs = append(briefs, slotBrief{ID: s.ID, StartAt: s.StartAt, DurationMin: s.DurationMin})
		}
		resp["card"] = offerCard{
			Type: "consult_offer", Price30: setting.Price30, Price60: setting.Price60, Slots: briefs,
		}
	}
	httpx.OK(c, resp)
	return nil
}

// consultOffer 取本人的咨询设置与最多 3 个未来开放档期（用于对话内成交卡片）。
func (h *Handler) consultOffer(hostUserID string) (*models.ConsultSetting, []models.ConsultSlot) {
	var setting models.ConsultSetting
	if err := h.db.First(&setting, "user_id = ?", hostUserID).Error; err != nil || !setting.Enabled {
		return nil, nil
	}
	var slots []models.ConsultSlot
	h.db.Where("host_user_id = ? AND status = ? AND start_at >= ?",
		hostUserID, models.SlotOpen, time.Now()).
		Order("start_at asc").Limit(3).Find(&slots)
	return &setting, slots
}

// knowledgeFor 取该主页启用的补充知识，拼成注入对话的文本（最多 30 条，控制 token）。
func (h *Handler) knowledgeFor(profileID string) string {
	var items []models.KnowledgeItem
	h.db.Where("profile_id = ? AND enabled = ?", profileID, true).
		Order("updated_at desc").Limit(30).Find(&items)
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for _, it := range items {
		topic := strings.TrimSpace(it.Topic)
		if topic != "" {
			b.WriteString("Q：")
			b.WriteString(topic)
			b.WriteString("\nA：")
		}
		b.WriteString(strings.TrimSpace(it.Content))
		b.WriteString("\n")
	}
	return b.String()
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
