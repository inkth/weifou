// Package toolagent 实现「AI 工具 Agent」：平台自编的工具型 AI 角色（如英语陪练），
// 通过会员（membership 包）一价解锁全部、非会员每个 Agent 给几次免费体验。
// 卖的是 AI 生成内容=虚拟商品，平台是卖家——与「代表真人的对外助理」(chat 包) 本质不同。
package toolagent

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/deepseek"
	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/membership"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db        *gorm.DB
	ds        *deepseek.Client
	security  *wechat.SecurityService
	jwtSecret string
}

func NewHandler(db *gorm.DB, ds *deepseek.Client, security *wechat.SecurityService, jwtSecret string) *Handler {
	return &Handler{db: db, ds: ds, security: security, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	// GET 树第二段统一静态（mine/detail/sessions/messages）；POST 树第二段统一 :id，互不混层（否则 Gin panic）。
	rg.GET("/agents", auth, httpx.Handle(h.list))
	rg.GET("/agents/mine", auth, httpx.Handle(h.mine))
	rg.GET("/agents/detail/:id", auth, httpx.Handle(h.detail))
	rg.GET("/agents/sessions/:id", auth, httpx.Handle(h.sessionList)) // :id = agentId → 我的历史会话
	rg.GET("/agents/messages/:id", auth, httpx.Handle(h.messages))    // :id = sessionId → 该会话消息
	rg.POST("/agents/:id/chat", auth, httpx.Handle(h.chat))
}

// ---------- 目录 ----------

func (h *Handler) list(c *gin.Context) error {
	auth := middleware.Current(c)
	var agents []models.ToolAgent
	h.db.Where("enabled = ?", true).Order("sort asc, created_at asc").Find(&agents)
	ents := h.entitlementMap(auth.UserID)
	out := make([]gin.H, 0, len(agents))
	for i := range agents {
		out = append(out, h.card(&agents[i], ents[agents[i].ID]))
	}
	httpx.OK(c, out)
	return nil
}

func (h *Handler) detail(c *gin.Context) error {
	auth := middleware.Current(c)
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", c.Param("id"), true).Error; err != nil {
		return httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	var ent models.AgentEntitlement
	var ep *models.AgentEntitlement
	if h.db.First(&ent, "user_id = ? AND agent_id = ?", auth.UserID, a.ID).Error == nil {
		ep = &ent
	}
	httpx.OK(c, h.card(&a, ep))
	return nil
}

// mine 返回我体验过的工具 Agent（按最近活动倒序）。
func (h *Handler) mine(c *gin.Context) error {
	auth := middleware.Current(c)
	var ents []models.AgentEntitlement
	h.db.Where("user_id = ?", auth.UserID).Order("updated_at desc").Find(&ents)
	out := make([]gin.H, 0, len(ents))
	for i := range ents {
		var a models.ToolAgent
		if h.db.First(&a, "id = ?", ents[i].AgentID).Error != nil || !a.Enabled {
			continue
		}
		out = append(out, h.card(&a, &ents[i]))
	}
	httpx.OK(c, out)
	return nil
}

// ---------- 对话（会员畅用 / 非会员扣免费体验） ----------

type chatReq struct {
	Content   string `json:"content" binding:"required"`
	SessionID string `json:"sessionId"` // 续聊指定会话；空 = 新开一段
}

func (h *Handler) chat(c *gin.Context) error {
	auth := middleware.Current(c)
	var req chatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("EMPTY_INPUT", "请输入内容")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return httpx.BadRequest("EMPTY_INPUT", "请输入内容")
	}
	if len([]rune(content)) > 500 {
		return httpx.BadRequest("INPUT_TOO_LONG", "输入太长（限 500 字）")
	}
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", c.Param("id"), true).Error; err != nil {
		return httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	if !h.security.CheckText(content, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "内容包含不当信息")
	}

	// 准入：会员畅用；非会员原子扣减免费体验，耗尽 → MEMBERSHIP_REQUIRED。
	member, remaining, aerr := h.checkAccess(auth.UserID, &a)
	if aerr != nil {
		return aerr
	}

	// 取/建会话：传了 sessionId 且属于本人本 Agent → 续聊；否则新开一段（一人一 Agent 支持多会话）。
	var session models.AgentSession
	resume := false
	if req.SessionID != "" {
		if e := h.db.First(&session, "id = ? AND user_id = ? AND agent_id = ?", req.SessionID, auth.UserID, a.ID).Error; e == nil {
			resume = true
		}
	}
	if !resume {
		session = models.AgentSession{ID: idgen.New(), AgentID: a.ID, UserID: auth.UserID}
		h.db.Create(&session)
	}
	h.db.Create(&models.AgentMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleUser,
		Content: content, SafeCheckStatus: models.SafePass,
	})

	// 最近 20 条历史 + 平台自编 system prompt。
	var recent []models.AgentMessage
	h.db.Where("session_id = ?", session.ID).Order("created_at desc").Limit(20).Find(&recent)
	msgs := []deepseek.Message{{Role: "system", Content: a.SystemPrompt}}
	for i := len(recent) - 1; i >= 0; i-- {
		msgs = append(msgs, deepseek.Message{Role: recent[i].Role, Content: recent[i].Content})
	}
	raw, derr := h.ds.Chat(msgs, deepseek.ChatOptions{Temperature: 0.7, MaxTokens: 800})
	if derr != nil {
		if !member {
			// 非会员扣了免费体验但 AI 失败 → 退还 1 次。
			h.db.Model(&models.AgentEntitlement{}).
				Where("user_id = ? AND agent_id = ?", auth.UserID, a.ID).
				UpdateColumn("remaining", gorm.Expr("remaining + 1"))
		}
		return httpx.Internal("AI_UPSTREAM_ERROR", "AI 服务暂时不可用，请稍后再试")
	}
	answer := strings.TrimSpace(raw)
	safe := models.SafePass
	if !h.security.CheckText(answer, auth.Openid) {
		safe = models.SafeReject
		answer = "抱歉，这部分内容不方便回答，我们换个话题继续吧。"
	}
	h.db.Create(&models.AgentMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleAssistant,
		Content: answer, SafeCheckStatus: safe,
	})
	h.db.Model(&session).Update("updated_at", time.Now())

	// member=true → remaining=-1（畅用，前端不显额度）。
	httpx.OK(c, gin.H{"sessionId": session.ID, "answer": answer, "member": member, "remaining": remaining})
	return nil
}

// messages 返回指定会话（:id = sessionId）的消息流；仅本人会话可见，否则空。
func (h *Handler) messages(c *gin.Context) error {
	auth := middleware.Current(c)
	var session models.AgentSession
	if err := h.db.First(&session, "id = ? AND user_id = ?", c.Param("id"), auth.UserID).Error; err != nil {
		httpx.OK(c, []gin.H{})
		return nil
	}
	var msgs []models.AgentMessage
	h.db.Where("session_id = ?", session.ID).Order("created_at asc").Limit(200).Find(&msgs)
	out := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, gin.H{"role": m.Role, "content": m.Content, "createdAt": m.CreatedAt})
	}
	httpx.OK(c, out)
	return nil
}

// sessionList 返回我与某 Agent（:id = agentId）的历史会话，最近活动倒序。
// 标题取该会话第一条用户消息，附最后一条消息预览；无消息的空会话不展示。
func (h *Handler) sessionList(c *gin.Context) error {
	auth := middleware.Current(c)
	var sessions []models.AgentSession
	h.db.Where("agent_id = ? AND user_id = ?", c.Param("id"), auth.UserID).
		Order("updated_at desc").Limit(50).Find(&sessions)
	out := make([]gin.H, 0, len(sessions))
	for i := range sessions {
		s := &sessions[i]
		var last models.AgentMessage
		if h.db.Where("session_id = ?", s.ID).Order("created_at desc").First(&last).Error != nil {
			continue // 空会话不展示
		}
		var first models.AgentMessage
		h.db.Where("session_id = ? AND role = ?", s.ID, models.RoleUser).Order("created_at asc").First(&first)
		title := clipText(first.Content, 30)
		if title == "" {
			title = "新对话"
		}
		out = append(out, gin.H{
			"sessionId":   s.ID,
			"title":       title,
			"lastMessage": clipText(last.Content, 60),
			"updatedAt":   s.UpdatedAt,
		})
	}
	httpx.OK(c, out)
	return nil
}

// clipText 截断到最多 n 个字符（rune 安全）。
func clipText(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// ---------- 准入与序列化 ----------

// checkAccess 会员 → (true, -1, nil) 不扣；非会员 → 原子扣减免费体验，返回剩余；
// 耗尽返回 (false, 0, MEMBERSHIP_REQUIRED)。首次访问自动发放 FreeTrial 免费体验。
func (h *Handler) checkAccess(userID string, a *models.ToolAgent) (bool, int, error) {
	if membership.IsActive(h.db, userID) {
		return true, -1, nil
	}
	var ent models.AgentEntitlement
	if err := h.db.First(&ent, "user_id = ? AND agent_id = ?", userID, a.ID).Error; err == gorm.ErrRecordNotFound {
		ent = models.AgentEntitlement{
			ID: idgen.New(), UserID: userID, AgentID: a.ID,
			Remaining: a.FreeTrial, TrialGiven: true,
		}
		if cerr := h.db.Create(&ent).Error; cerr != nil {
			h.db.First(&ent, "user_id = ? AND agent_id = ?", userID, a.ID) // 并发下他人先建 → 重查
		}
	}
	res := h.db.Model(&models.AgentEntitlement{}).
		Where("user_id = ? AND agent_id = ? AND remaining > 0", userID, a.ID).
		UpdateColumn("remaining", gorm.Expr("remaining - 1"))
	if res.RowsAffected == 0 {
		return false, 0, httpx.BadRequest("MEMBERSHIP_REQUIRED", "免费体验已用完，开通会员畅用全部")
	}
	var fresh models.AgentEntitlement
	h.db.First(&fresh, "user_id = ? AND agent_id = ?", userID, a.ID)
	return false, fresh.Remaining, nil
}

func (h *Handler) entitlementMap(userID string) map[string]*models.AgentEntitlement {
	var ents []models.AgentEntitlement
	h.db.Where("user_id = ?", userID).Find(&ents)
	m := make(map[string]*models.AgentEntitlement, len(ents))
	for i := range ents {
		m[ents[i].AgentID] = &ents[i]
	}
	return m
}

// card 序列化对外字段（不含 SystemPrompt）。freeTrialRemaining 未体验过 → 满额。
func (h *Handler) card(a *models.ToolAgent, ent *models.AgentEntitlement) gin.H {
	remaining := a.FreeTrial
	if ent != nil {
		remaining = ent.Remaining
	}
	return gin.H{
		"id": a.ID, "slug": a.Slug, "name": a.Name, "tagline": a.Tagline,
		"description": a.Description, "category": a.Category, "icon": a.Icon,
		"accent": a.Accent, "greeting": a.Greeting,
		"freeTrial": a.FreeTrial, "freeTrialRemaining": remaining,
	}
}

// ---------- 种子（首启写入平台自编 Agent，按 slug 幂等） ----------

func Seed(db *gorm.DB) {
	if db == nil {
		return
	}
	presets := []models.ToolAgent{
		{
			Slug: "spoken-english", Name: "英语陪练",
			Tagline:     "随时开口的 AI 英语口语教练",
			Description: "用中文也能学：纠音、造句、模拟对话，按日常 / 旅行 / 商务 / 面试场景陪你开口，循序渐进。",
			Category:    models.AgentCatEducation, Icon: "🗣️", Accent: "#FB923C",
			Greeting:     "Hi！我是你的英语陪练。想练点什么？日常对话、面试还是旅行场景都行——直接用中文告诉我也可以。",
			SystemPrompt: spokenEnglishPrompt,
			FreeTrial:    5, Sort: 1,
		},
		{
			Slug: "interview-coach", Name: "面试教练",
			Tagline:     "模拟面试 + 实时点评的 AI 面试官",
			Description: "按你的目标岗位出题、追问、复盘，用 STAR 法把答案打磨到位，覆盖行为面 / 技术面 / HR 面。",
			Category:    models.AgentCatCareer, Icon: "💼", Accent: "#6366F1",
			Greeting:     "我是你的面试教练。告诉我目标岗位（比如「产品经理」），我来当面试官，一题题陪你练。",
			SystemPrompt: interviewCoachPrompt,
			FreeTrial:    3, Sort: 2,
		},
	}
	for i := range presets {
		p := presets[i]
		var existing models.ToolAgent
		if db.Where("slug = ?", p.Slug).First(&existing).Error == gorm.ErrRecordNotFound {
			p.ID = idgen.New()
			p.Enabled = true
			db.Create(&p)
			continue
		}
		// 更新展示字段与 prompt，保留人工 enabled 开关与 id。
		db.Model(&existing).Updates(map[string]interface{}{
			"name": p.Name, "tagline": p.Tagline, "description": p.Description,
			"category": p.Category, "icon": p.Icon, "accent": p.Accent,
			"greeting": p.Greeting, "system_prompt": p.SystemPrompt,
			"free_trial": p.FreeTrial, "sort": p.Sort,
		})
	}
}

const spokenEnglishPrompt = `你是「英语陪练」，一个专注英语口语与表达训练的 AI 教练。你只负责帮助用户学习英语；遇到与英语学习无关的请求（写代码、查资料、闲聊八卦等），礼貌地把话题带回英语练习。

教学原则：
- 因材施教：先判断用户水平；用户用中文提问也没关系，回答中英结合，关键英文给读音提示与中文释义。
- 多让用户开口：每次尽量以一个小练习或追问收尾，引导用户用英语回应。
- 即时纠错：用户用英语表达时，先肯定，再温和指出语法/用词/更地道的说法，并给对照例句。
- 场景化：可按需切换日常、旅行、商务、面试等场景做角色扮演。
- 简洁：单次回答控制在 200 字以内（含例句），不长篇大论。
用中文作教学语言、英文作练习内容。保持耐心、鼓励、专业克制。不要透露你是大模型。`

const interviewCoachPrompt = `你是「面试教练」，帮助用户准备求职面试的 AI 面试官与教练。你只负责面试相关的训练与复盘；遇到无关请求礼貌带回。

工作方式：
- 先确认目标岗位与面试类型（行为面/技术面/HR 面），用户没说就主动问一句。
- 一次只问一道面试题，等用户作答后再点评：先肯定亮点，再指出可改进处，给出更好的回答结构（如 STAR 法）与示范要点。
- 由浅入深、循序推进；必要时追问，模拟真实面试压力但保持友善。
- 单次回答控制在 250 字以内，重点突出、可操作。
用中文交流，专业、犀利但鼓励。不要透露你是大模型。`
