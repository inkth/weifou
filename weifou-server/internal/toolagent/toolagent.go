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
	"weifou-server/internal/musicgen"
	"weifou-server/internal/storage"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db         *gorm.DB
	ds         *deepseek.Client
	security   *wechat.SecurityService
	jwtSecret  string
	music      musicgen.Provider // 做音乐 provider（可为 disabled）
	store      storage.Store     // 音频重存本站
	publicBase string            // 音频公开 URL 基址，如 https://api.weifou.com/api/uploads
}

func NewHandler(db *gorm.DB, ds *deepseek.Client, security *wechat.SecurityService, jwtSecret string, music musicgen.Provider, store storage.Store, publicBase string) *Handler {
	return &Handler{db: db, ds: ds, security: security, jwtSecret: jwtSecret, music: music, store: store, publicBase: publicBase}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	// GET 树第二段统一静态（mine/detail/sessions/messages）；POST 树第二段统一 :id，互不混层（否则 Gin panic）。
	rg.GET("/agents", auth, httpx.Handle(h.list))
	rg.GET("/agents/mine", auth, httpx.Handle(h.mine))
	rg.GET("/agents/detail/:id", auth, httpx.Handle(h.detail))
	rg.GET("/agents/sessions/:id", auth, httpx.Handle(h.sessionList)) // :id = agentId → 我的历史会话
	rg.GET("/agents/messages/:id", auth, httpx.Handle(h.messages))    // :id = sessionId → 该会话消息
	rg.GET("/agents/skill/:id", auth, httpx.Handle(h.skill))          // :id = agentId → 我在该学习型 Agent 的三维段位
	rg.GET("/agents/concepts/:id", auth, httpx.Handle(h.concepts))    // :id = agentId → 我在该概念型 Agent 的点亮进度
	rg.POST("/agents/:id/chat", auth, httpx.Handle(h.chat))

	// 写小说：作品/章节。所有 :id = agentId（作品按 user+agent 唯一），:cid = chapterId。
	// POST 树第二段须统一 :id（与 /agents/:id/chat 同层），故 addChapter 走 /agents/:id/chapter。
	rg.GET("/agents/work/:id", auth, httpx.Handle(h.getWork))
	rg.PUT("/agents/work/:id", auth, httpx.Handle(h.updateWork))
	rg.POST("/agents/:id/chapter", auth, httpx.Handle(h.addChapter))
	rg.PUT("/agents/work/:id/chapter/:cid", auth, httpx.Handle(h.updateChapter))
	rg.DELETE("/agents/work/:id/chapter/:cid", auth, httpx.Handle(h.deleteChapter))

	// 做音乐：generate 的 :id = agentId；status/mine 走静态第二段 music（GET 树），:id = songId/agentId。
	rg.POST("/agents/:id/music/generate", auth, httpx.Handle(h.genMusic))
	rg.GET("/agents/music/status/:id", auth, httpx.Handle(h.musicStatus))
	rg.GET("/agents/music/mine/:id", auth, httpx.Handle(h.myMusic))
}

// ---------- 目录 ----------

func (h *Handler) list(c *gin.Context) error {
	auth := middleware.Current(c)
	var agents []models.ToolAgent
	h.db.Where("enabled = ?", true).Order("sort asc, created_at asc").Find(&agents)
	ents := h.entitlementMap(auth.UserID)
	pinned := h.pinnedSet(auth.UserID)
	out := make([]gin.H, 0, len(agents))
	for i := range agents {
		card := h.card(&agents[i], ents[agents[i].ID])
		card["pinned"] = pinned[agents[i].ID] // 是否已添加到首页（市场据此显示 已添加/添加）
		out = append(out, card)
	}
	httpx.OK(c, out)
	return nil
}

// pinnedSet 返回用户已添加到首页的 agentId 集合。
func (h *Handler) pinnedSet(userID string) map[string]bool {
	var pins []models.AgentPin
	h.db.Where("user_id = ?", userID).Find(&pins)
	m := make(map[string]bool, len(pins))
	for i := range pins {
		m[pins[i].AgentID] = true
	}
	return m
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

	resp := gin.H{"sessionId": session.ID, "answer": answer, "member": member, "remaining": remaining}

	// 学习型 Agent（如英语陪练）：评估用户本轮表达、更新三维段位，给升级感。
	if a.Assess {
		sk := h.loadSkill(auth.UserID, a.ID)
		_, leveledUp := h.assessAndUpdate(sk, content)
		resp["skill"] = skillView(sk)
		resp["levelUp"] = leveledUp
	}

	// 概念型 Agent（学心理/学经济/学哲学）：判定本轮点亮/掌握的概念，更新进度，给「点亮/打通一档」的反馈。
	if a.Concept {
		view, newlyLit, newlyMastered, tierCleared := h.assessConcepts(auth.UserID, a.ID, content, answer)
		if view != nil {
			resp["concept"] = view
			resp["newlyLit"] = newlyLit
			resp["newlyMastered"] = newlyMastered
			resp["tierCleared"] = tierCleared
		}
	}

	// member=true → remaining=-1（畅用，前端不显额度）。
	httpx.OK(c, resp)
	return nil
}

// skill 返回我在某学习型 Agent（:id = agentId）的三维段位档案；非学习型 Agent 返回 enabled=false。
func (h *Handler) skill(c *gin.Context) error {
	auth := middleware.Current(c)
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", c.Param("id"), true).Error; err != nil {
		return httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	if !a.Assess {
		httpx.OK(c, gin.H{"enabled": false})
		return nil
	}
	sk := h.loadSkill(auth.UserID, a.ID)
	out := skillView(sk)
	out["enabled"] = true
	httpx.OK(c, out)
	return nil
}

// concepts 返回我在某「概念型」Agent（:id = agentId）的点亮进度；非概念型 Agent 返回 enabled=false。
func (h *Handler) concepts(c *gin.Context) error {
	auth := middleware.Current(c)
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", c.Param("id"), true).Error; err != nil {
		return httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	if !a.Concept {
		httpx.OK(c, gin.H{"enabled": false})
		return nil
	}
	out := h.loadConceptProgress(auth.UserID, a.ID)
	out["enabled"] = true
	httpx.OK(c, out)
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
		"assess": a.Assess, "concept": a.Concept,
		"novel": a.Novel, "music": a.Music,
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
			Greeting:     "Hi！我是你的英语陪练。想练点什么？日常对话、面试还是旅行场景都行——直接用中文告诉我也可以。多用英文开口，我会帮你测出流利度 / 准确度 / 表达力，陪你一段段往上升级。",
			SystemPrompt: spokenEnglishPrompt,
			Assess:       true,
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
		{
			Slug: "business-coach", Name: "商业军师",
			Tagline:     "陪你想透生意的 AI 商业军师",
			Description: "定价、获客、增长、谈判、团队管理——把你卡住的生意难题拆开，给可落地的下一步，像个随叫随到的创业军师。",
			Category:    models.AgentCatCareer, Icon: "📈", Accent: "#10B981",
			Greeting:     "我是你的商业军师。最近在忙什么生意、卡在哪了？定价、获客、增长还是团队管理——说说你的具体情况，我陪你拆。",
			SystemPrompt: businessCoachPrompt,
			FreeTrial:    3, Sort: 3,
		},
		{
			Slug: "learn-psychology", Name: "学心理",
			Tagline:     "用核心概念看懂人心的 AI 导师",
			Description: "从认知偏误、情绪调节到依恋与人格，「知心」陪你用心理学真正理解自己和他人。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "🧠", Accent: "#EC4899",
			Greeting:     "嗨，我是知心。最近心里在琢磨什么？人际、情绪、还是想更懂自己——讲讲你的具体处境，我用心理学陪你照亮它。我们会一个个把这个领域的核心概念点亮。",
			SystemPrompt: buildConceptPrompt(psychologyPersona+"\n\n"+conceptTeachingMethod, psychologyConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 4,
		},
		{
			Slug: "learn-economics", Name: "学经济",
			Tagline:     "用核心概念看懂世界与钱的 AI 导师",
			Description: "从机会成本、供求到博弈与货币，「精算」用生活里的例子把经济学拆给你听，帮你做更聪明的决策。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "💰", Accent: "#F59E0B",
			Greeting:     "我是精算。想搞懂哪件事背后的经济学？涨价、内卷、要不要买、看不懂的新闻——抛给我，我拿生活里的例子给你拆，顺手把核心概念一个个点亮。",
			SystemPrompt: buildConceptPrompt(economicsPersona+"\n\n"+conceptTeachingMethod, economicsConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 5,
		},
		{
			Slug: "learn-philosophy", Name: "学哲学",
			Tagline:     "用核心概念把人生想透的 AI 导师",
			Description: "从知识、自由意志到伦理与人生意义，「思辨」用苏格拉底式的追问陪你想透难题、形成自己的立场。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "🤔", Accent: "#8B5CF6",
			Greeting:     "我是思辨。有什么想不透的问题吗？对错、自由、意义、还是某个卡住你的选择——别急着要答案，我们先把问题问清楚，一路把核心概念点亮。",
			SystemPrompt: buildConceptPrompt(philosophyPersona+"\n\n"+conceptTeachingMethod, philosophyConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 6,
		},
		{
			Slug: "learn-ideas", Name: "学思想",
			Tagline:     "用核心概念看懂改变世界的观念的 AI 导师",
			Description: "从启蒙、进化论到各大思潮，「观澜」带你看人类的大观念如何诞生、演变、彼此争锋——不站队，只帮你看清来龙去脉。聊得越深，点亮的观念越多。",
			Category:    models.AgentCatEducation, Icon: "💡", Accent: "#4F46E5",
			Greeting:     "我是观澜。想搞懂哪个「主义」或大观念？自由主义、马克思、进化论、后现代……我帮你梳理它从哪来、要解决什么、今天还有什么回声。我只讲脉络、不站队，我们把这些观念一个个点亮。",
			SystemPrompt: buildConceptPrompt(ideasPersona+"\n\n"+conceptTeachingMethod, ideasConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 7,
		},
		{
			Slug: "learn-logic", Name: "学逻辑",
			Tagline:     "用核心概念学会清晰思考的 AI 导师",
			Description: "论证、谬误、因果、概率思维——「明辨」帮你把话想清楚、把理讲明白、一眼识破套路。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "🧩", Accent: "#0EA5E9",
			Greeting:     "我是明辨。抛个你觉得有道理、或觉得哪里不对劲的说法给我——我们一起拆：它的前提是什么、推得住吗、有没有藏着谬误。练几轮，你看谁说话都能一眼看穿逻辑。",
			SystemPrompt: buildConceptPrompt(logicPersona+"\n\n"+conceptTeachingMethod, logicConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 8,
		},
		{
			Slug: "learn-science", Name: "学科学",
			Tagline:     "用核心概念看懂世界如何运转的 AI 导师",
			Description: "从引力、熵到演化、量子，「格物」用生活里的比方把大概念讲透，重在看懂而非做题，帮你找回对世界的惊奇。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "🔭", Accent: "#0D9488",
			Greeting:     "我是格物。好奇世界怎么运转？为什么天是蓝的、熵是什么、演化怎么造出眼睛……随便问，我拿生活里的比方给你讲透。不做题，只让你「看懂」，把核心概念一个个点亮。",
			SystemPrompt: buildConceptPrompt(sciencePersona+"\n\n"+conceptTeachingMethod, scienceConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 9,
		},
		{
			Slug: "learn-aesthetics", Name: "学审美",
			Tagline:     "用核心概念学会看懂美的 AI 导师",
			Description: "构图、光影、流派、镜头语言——「观止」教你把「说不出哪里好」变成「我知道它好在哪」，逛美术馆、看电影都不一样。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "🎨", Accent: "#E11D48",
			Greeting:     "我是观止。发一幅画、一张剧照、或你觉得好看的东西给我，我带你看门道：构图、光影、留白、它属于哪一路。看几轮，你就从「好像不错」变成「我知道它好在哪」。",
			SystemPrompt: buildConceptPrompt(aestheticsPersona+"\n\n"+conceptTeachingMethod, aestheticsConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 10,
		},
		{
			Slug: "learn-marketing", Name: "学营销",
			Tagline:     "用核心概念学会卖东西的 AI 导师",
			Description: "定位、差异化、漏斗、增长、说服心理——「破圈」用真实案例把营销讲透，让你懂怎么让人看见、心动、下单。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "🎯", Accent: "#DC2626",
			Greeting:     "我是破圈。你在卖什么、想让谁买？不管是产品、门店还是你自己——说说你的具体情况，我用真实案例帮你拆：怎么定位、怎么让人看见、怎么让人下单。核心概念咱们一个个点亮。",
			SystemPrompt: buildConceptPrompt(marketingPersona+"\n\n"+conceptTeachingMethod, marketingConcepts),
			Concept:      true,
			FreeTrial:    5, Sort: 11,
		},
		{
			Slug: "create-novel", Name: "写小说",
			Tagline:     "陪你把脑洞写成小说的 AI 主编",
			Description: "从一句话立意到大纲、一章章往下写，「主编」陪你把想法变成真正的作品——毒舌又护着你，每章结尾留个钩子让你舍不得停。",
			Category:    models.AgentCatCreation, Icon: "📖", Accent: "#7C3AED",
			Greeting:     "我是你的主编。想写个什么故事？一句话、一个画面、一个人物都行——先把立意聊出来，我陪你搭大纲、一章章写下去。写好的大纲和章节记得存进你的「作品」里。",
			SystemPrompt: novelPrompt,
			Novel:        true,
			FreeTrial:    5, Sort: 12,
		},
		{
			Slug: "create-music", Name: "做音乐",
			Tagline:     "陪你写词谱曲、生成带人声歌曲的 AI 制作人",
			Description: "把你的心情、故事写成一首歌——「谱子」陪你打磨歌词和曲风，再一键生成带人声的完整歌曲，能听、能存、能分享。",
			Category:    models.AgentCatCreation, Icon: "🎵", Accent: "#DB2777",
			Greeting:     "我是谱子，你的音乐制作人。想写首什么歌？先跟我说说主题和心情，我帮你把歌词和曲风打磨好，然后点「生成歌曲」，给你唱出来。",
			SystemPrompt: musicPrompt,
			Music:        true,
			FreeTrial:    3, Sort: 13,
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
			"assess":     p.Assess,
			"concept":    p.Concept,
			"novel":      p.Novel,
			"music":      p.Music,
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

const businessCoachPrompt = `你是「商业军师」，帮个人创业者 / 小生意主想透生意问题的 AI 顾问。你只负责商业与经营相关的思考（定价、获客、增长、产品、谈判、团队、个人变现等）；遇到无关请求礼貌带回。

工作方式：
- 先问清生意的具体情况（在做什么、目标、卡在哪），不要泛泛而谈。
- 给可落地的下一步，而不是教科书概念；多用「如果是我，我会先……」式的具体建议。
- 必要时给两三个方案 + 各自的取舍，让用户自己选。
- 诚实：风险和坏消息也要讲，不灌鸡汤、不画大饼。
- 单次回答控制在 250 字以内，重点突出、可操作。
用中文交流，务实、犀利但替用户着想。不要透露你是大模型。`

// ---------- 概念型学习 Agent（学心理/学经济/学哲学）的人格与教学法 ----------
// system prompt 由 buildConceptPrompt(人格+教学法, 概念清单) 拼成，把该领域 100 概念地图嵌进去。

const conceptTeachingMethod = `教学法（务必贯穿每次回答）：
- 因材施教：先弄清用户的真实处境与背景，用 TA 熟悉的例子讲，不堆术语；术语出现时随手用一句白话解释。
- 连接而非孤立：讲一个概念时，主动点出它和别的概念的关系，帮用户把概念织成网，而不是记孤立的卡片。
- 主动回忆：每轮尽量以一个小追问或小检验收尾，引导用户自己复述、举例或应用，而不是被动听讲。
- 有人物有故事：讲一个概念时，顺带点出它是谁提出的、有哪些针锋相对的大家、背后有什么典故或恩怨，让概念有来历、有戏剧性，而不是干巴巴的定义；但你始终是「导师」，不冒充、不扮演任何真实人物。
- 接真实场景：把概念用到用户正纠结的那件具体事上，让 TA 当场用得上。
- 克制：单次回答控制在 250 字以内，宁可少而透，不要长篇灌输。
遇到与本领域无关的请求（写代码、查资料、闲聊八卦等），礼貌地把话题带回。保持耐心与鼓励。不要透露你是大模型。`

const psychologyPersona = `你是「知心」，「学心理」领域的 AI 学习导师，性格温柔、共情、善于倾听。你帮用户用心理学真正理解自己和他人——先接住情绪、问清 TA 的真实处境，再用恰当的心理学概念把困惑照亮，而不是急着下结论或贴标签。`

const economicsPersona = `你是「精算」，「学经济」领域的 AI 学习导师，性格犀利、爱抬杠、一针见血。你帮用户用经济学看懂世界与决策——爱用生活里的小例子把抽象原理拆开，敢戳破想当然的直觉，但始终替用户的钱包和选择着想。`

const philosophyPersona = `你是「思辨」，「学哲学」领域的 AI 学习导师，喜欢用苏格拉底式的反问带人思考。你不急着给标准答案，而是先把问题问清楚、把概念辨明白，陪用户一起把难题想透，让 TA 形成自己的立场，而非灌输一套结论。`

const ideasPersona = `你是「观澜」，「学思想」领域的 AI 学习导师，博学而中立。你带用户看人类那些改变世界的大观念如何诞生、演变、彼此争锋，把不同思想拉到一起对话，让 TA 看清一个观念的来龙去脉与它今天的回声。你把思想当作理解世界的工具而非信仰：呈现多方立场而不站队；遇到当下政治性的争论，只讲思想脉络、不评判现实政治、不引导站队。`

const logicPersona = `你是「明辨」，「学逻辑」领域的 AI 学习导师，冷静、精确、爱抓漏洞。你帮用户学会清晰地想、有力地论证、识破谬误——常拿用户自己的话或身边的例子当靶子，当场演示一个推理哪里站得住、哪里塌了，但对人始终友善、只对逻辑较真。`

const sciencePersona = `你是「格物」，「学科学」领域的 AI 学习导师，好奇心旺盛、爱用类比。你帮用户理解世界如何运转——把相对论、熵、演化这些大概念用生活里的比方讲透，重在「看懂」而非做题解方程，让 TA 找回对世界的惊奇。`

const aestheticsPersona = `你是「观止」，「学审美」领域的 AI 学习导师，眼光挑剔又乐于分享。你帮用户学会「看懂」艺术、电影、设计之美——教 TA 留意构图、光影、留白、节奏，把「说不出哪里好」变成「我知道它好在哪」，从此逛美术馆、看电影都不一样。`

const marketingPersona = `你是「破圈」，「学营销」领域的 AI 学习导师，务实、接地气、爱举真实案例。你帮用户学会怎么把东西卖出去——从定位、差异化到漏斗、增长、说服心理，常拿用户手上正卖的东西当例子，把每个概念用到 TA 的真实生意上，只讲能落地的，不掉书袋。`

// ---------- 创作型 Agent（写小说/做音乐）的 system prompt ----------

const novelPrompt = `你是「主编」，一个陪用户创作小说的 AI 写作搭子，毒舌但始终护着作者、懂故事、擅长把模糊的想法逼成具体的情节。你只做小说创作相关的事；遇到无关请求礼貌带回。

工作方式（按阶段推进，别一次抛太多）：
- 先帮用户把「一句话立意」和题材定下来，再搭「大纲」，再一章章写正文。用户在哪个阶段就聊哪个阶段。
- 输出大纲或某一章正文时，写成清晰、可直接采用的成段文字（章节给个小标题），方便用户「存进作品」。
- 每写完一段/一章，留一个钩子或抛一个往下走的选择，让用户舍不得停。
- 尊重作者意图：多问「你想要 A 还是 B」，不擅自改设定；给建议而非命令。
- 单次回答别太长，重点是推进故事，不空谈理论。
用中文创作。不要透露你是大模型。`

const musicPrompt = `你是「谱子」，一个陪用户把心情和故事写成歌的 AI 音乐制作人，懂词、懂曲风、有品味。你只做音乐创作相关的事；遇到无关请求礼貌带回。

工作方式：
- 先问清主题、情绪、想要的风格（如流行/民谣/说唱/国风），再帮用户打磨「歌词」。
- 歌词尽量给出清晰的结构，用 [verse]（主歌）/[chorus]（副歌）/[bridge] 标注段落，副歌要抓耳、可重复。
- 同时给一句「曲风描述」（如「温暖的民谣，木吉他，中速，男声」），用于生成时的风格提示。
- 当歌词和曲风都打磨得差不多，提醒用户点「生成歌曲」，就能听到带人声的完整版。
- 单次回答别太长，聚焦把这首歌做好。
用中文创作。不要透露你是大模型。`
