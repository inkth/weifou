// Package toolagent 实现「AI 工具 Agent」：平台自编的工具型 AI 角色（如英语陪练），
// 通过会员（membership 包）一价解锁全部、非会员每个 Agent 给几次免费体验。
// 卖的是 AI 生成内容=虚拟商品，平台是卖家——与「代表真人的对外助理」(chat 包) 本质不同。
package toolagent

import (
	"regexp"
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
	rg.GET("/agents/streak", auth, httpx.Handle(h.streakInfo))        // 连续学习天数（全局一条）
	rg.POST("/agents/:id/chat", auth, httpx.Handle(h.chat))
	rg.POST("/agents/:id/remind", auth, httpx.Handle(h.remind)) // 学习提醒承诺（订阅消息授权后落账）

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
	Mode      string `json:"mode"`      // "review" = 复习挑战（概念型专用：只快问快答已点亮概念，不开新课）
	Concept   string `json:"concept"`   // 概念/关卡 slug：学员从闯关地图点选指定关进来（概念型专用）
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

	// 准入：会员畅用。概念课按「幕」门控——第一幕(Tier≤FreeTier)免费无限、更高幕需会员；
	// 其余(工具 / 道德经试读)沿用「免费体验次数」计。
	var member bool
	remaining := -1
	trialGated := false // 是否走了扣次模型（决定 AI 失败时是否退还免费次数）
	if a.Concept && a.FreeTier > 0 {
		member = membership.IsActive(h.db, auth.UserID)
		if !member && h.conceptTier(a.ID, req.Concept) > a.FreeTier {
			return httpx.BadRequest("MEMBERSHIP_REQUIRED", "第一幕已学完 · 第二幕起开通会员畅用全部")
		}
	} else {
		var aerr error
		member, remaining, aerr = h.checkAccess(auth.UserID, &a)
		if aerr != nil {
			return aerr
		}
		trialGated = !member
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

	// 最近 20 条历史 + 平台自编 system prompt + 学习状态注入。
	// L1 主动教学：把学员进度/段位和本轮编排指令喂给模型，让它带着课来，而不是等着被问。
	// 点选优先（全局交互范式）：回复末尾带选项行，服务端剥离后作为可点气泡下发。
	sysPrompt := a.SystemPrompt + "\n\n" + optionsDirective
	var sk *models.AgentSkill
	if a.Assess {
		sk = h.loadSkill(auth.UserID, a.ID)
		sysPrompt += "\n\n" + skillStateBrief(sk, !resume)
	}
	if a.Concept {
		if brief := h.conceptStateBrief(auth.UserID, &a, !resume); brief != "" {
			sysPrompt += "\n\n" + brief
		}
		// 复习挑战：追加快问快答编排（放进度简报之后，指令内已声明优先于开场编排）。
		if req.Mode == "review" {
			if d := h.reviewDirective(auth.UserID, a.ID); d != "" {
				sysPrompt += "\n\n" + d
			}
		}
		// 指定关卡：学员从闯关地图点选了某关，追加定向开课指令（slug 无效则静默忽略）。
		if req.Concept != "" {
			if d := h.conceptDirective(a.ID, req.Concept); d != "" {
				sysPrompt += "\n\n" + d
			}
		}
	}
	var recent []models.AgentMessage
	h.db.Where("session_id = ?", session.ID).Order("created_at desc").Limit(20).Find(&recent)
	msgs := []deepseek.Message{{Role: "system", Content: sysPrompt}}
	for i := len(recent) - 1; i >= 0; i-- {
		msgs = append(msgs, deepseek.Message{Role: recent[i].Role, Content: recent[i].Content})
	}
	raw, derr := h.ds.Chat(msgs, deepseek.ChatOptions{Temperature: 0.7, MaxTokens: 800})
	if derr != nil {
		if trialGated {
			// 非会员扣了免费体验但 AI 失败 → 退还 1 次（幕门控的免费幕不计次，无需退）。
			h.db.Model(&models.AgentEntitlement{}).
				Where("user_id = ? AND agent_id = ?", auth.UserID, a.ID).
				UpdateColumn("remaining", gorm.Expr("remaining + 1"))
		}
		return httpx.Internal("AI_UPSTREAM_ERROR", "AI 服务暂时不可用，请稍后再试")
	}
	answer, options := splitOptions(strings.TrimSpace(raw))
	safe := models.SafePass
	if !h.security.CheckText(answer+"\n"+strings.Join(options, "\n"), auth.Openid) {
		safe = models.SafeReject
		answer = "抱歉，这部分内容不方便回答，我们换个话题继续吧。"
		options = nil
	}
	// 入库存剥离选项后的正文：历史回放干净，模型看到的上下文也不带标记行。
	h.db.Create(&models.AgentMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleAssistant,
		Content: answer, SafeCheckStatus: safe,
	})
	h.db.Model(&session).Update("updated_at", time.Now())

	resp := gin.H{"sessionId": session.ID, "answer": answer, "options": options, "member": member, "remaining": remaining}

	// 连续学习天数：学习型（技能/概念）对话记一天；newDay 时前端做里程碑庆祝/提醒承诺邀请。
	if a.Assess || a.Concept {
		days, newDay, usedFreeze := bumpStreak(h.db, auth.UserID)
		resp["streak"] = gin.H{"days": days, "newDay": newDay, "freeze": usedFreeze}
	}

	// 学习型 Agent（如英语陪练）：评估用户本轮表达、更新三维段位，给升级感。
	// sk 已在组装 system prompt 时加载（状态注入），此处直接复用。
	if a.Assess && sk != nil {
		_, leveledUp := h.assessAndUpdate(sk, content)
		resp["skill"] = skillView(sk)
		resp["levelUp"] = leveledUp
	}

	// 概念型 Agent（学心理/学经济/学哲学）：判定本轮点亮/掌握的概念，更新进度，给「点亮/打通一档」的反馈。
	if a.Concept {
		view, newlyLit, newlyMastered, tierCleared := h.assessConcepts(&a, auth.UserID, content, answer)
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
	out := h.loadConceptProgress(auth.UserID, &a)
	out["enabled"] = true
	out["due"] = dueCount(h.db, auth.UserID, a.ID) // 到期待复习数（对话页复习徽章）
	out["freeTier"] = a.FreeTier                   // 免费幕阈值（learn-map 据此对 Tier>FreeTier 的关标会员锁）
	out["isMember"] = membership.IsActive(h.db, auth.UserID)
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

// ---------- 点选优先：选项行协议 ----------
// 全局交互范式「点选为主、输入兜底」：Agent 每次回复末尾按 optionsDirective 附一行候选，
// 服务端剥离该行、拆成 options 数组随回复下发，前端渲染成可点气泡；模型没给就没有，输入框兜底。

const optionsDirective = `交互规则（点选优先，务必遵守）：
用户在手机上更愿意点选而不是打字。你的每次回复，若结尾的提问 / 推进选择存在可枚举的典型回答，必须在回复的最后单独输出一行（前面空一行）：
【选项】选项1｜选项2｜选项3
- 2~4 个，每个不超过 16 字；用户点选后该文字会原样作为 TA 的回答发给你，所以选项要以用户口吻写、可直接当回答用，彼此差异明显。
- 检验理解时优先设计成可点选的判断题 / 选择题：把正确项混在似是而非的干扰项里，让点选本身就是思考。
- 例外：当你要求对方自由表达才算数时（如用英语开口说出一句话、自己举一个例子、用自己的话复述），不要输出选项行，让 TA 真正开口。
- 选项行必须是整个回复的最后一行，除此之外不要出现「【选项】」字样。`

// optsLineRe 匹配回复中的选项行（容忍模型偶发把它放在中间）。
var optsLineRe = regexp.MustCompile(`(?m)^\s*【选项】\s*(.+?)\s*$`)

// splitOptions 从模型回复中剥离选项行：返回干净正文 + 选项数组（无选项行则原样返回）。
func splitOptions(answer string) (string, []string) {
	m := optsLineRe.FindStringSubmatch(answer)
	if m == nil {
		return answer, nil
	}
	body := strings.TrimSpace(optsLineRe.ReplaceAllString(answer, ""))
	opts := make([]string, 0, 4)
	for _, p := range strings.FieldsFunc(m[1], func(r rune) bool { return r == '｜' || r == '|' }) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		opts = append(opts, clipText(p, 30))
		if len(opts) >= 4 {
			break
		}
	}
	if len(opts) == 0 {
		return body, nil
	}
	return body, opts
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
		"freeTier": a.FreeTier, // >0：概念课「第一幕免费」模型（前端据此换配额文案，不显示剩N次）
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
			Tagline:     "带你闯真实场景的 AI 英语口语教练",
			Description: "用中文也能学：从咖啡馆点单到全英面试，28 个真实场景一关关闯。开口完成任务才算通关，AI 陪你测流利度 / 准确度 / 表达力，一段段往上升级。",
			Category:    models.AgentCatEducation, Icon: "🗣️", Accent: "#FB923C",
			Greeting:     "Hi！我是你的英语陪练。咱们不背课文，直接闯真实场景——点咖啡、过海关、见客户、答面试。从地图挑一关，或者直接告诉我你最近哪个场合要用英语，我把你丢进情境里练。",
			SystemPrompt: buildConceptPrompt(spokenEnglishPrompt, englishScenarios),
			Assess:       true,
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 1, // 第一幕(生活+旅行)免费无限，职场/面试起会员
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
			Tagline:     "情绪和关系不再内耗的 AI 心理导师",
			Description: "看清自己、经营关系、看穿套路——「知心」带你走一趟三幕心理学之旅：情绪不内耗、关系不拧巴、决策不被套路。聊得越深，点亮的关卡越多。",
			Category:    models.AgentCatEducation, Icon: "🧠", Accent: "#EC4899",
			Greeting:     "嗨，我是知心。最近心里在琢磨什么？人际、情绪、还是想更懂自己——讲讲你的具体处境，我用心理学陪你照亮它。我们会从「看清自己」出发，一关关把整张地图点亮。",
			SystemPrompt: buildConceptPrompt(psychologyPersona+"\n\n"+conceptTeachingMethod, psychologyConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 4, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-logic", Name: "学逻辑",
			Tagline:     "用核心概念学会清晰思考的 AI 导师",
			Description: "从拆论证、识谬误到读数字、断因果——「明辨」带你六幕闯关学会清晰思考，每幕结尾一场 Boss 找茬战。聊得越深，点亮的关卡越多。",
			Category:    models.AgentCatEducation, Icon: "🧩", Accent: "#0EA5E9",
			Greeting:     "我是明辨。抛个你觉得有道理、或觉得哪里不对劲的说法给我——我们一起拆：它的前提是什么、推得住吗、有没有藏着谬误。练几轮，你看谁说话都能一眼看穿逻辑。",
			SystemPrompt: buildConceptPrompt(logicPersona+"\n\n"+conceptTeachingMethod, logicConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 5, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-marketing", Name: "学营销",
			Tagline:     "用核心概念学会卖东西的 AI 导师",
			Description: "定位、差异化、漏斗、增长、说服心理——「破圈」用真实案例把营销讲透，让你懂怎么让人看见、心动、下单。聊得越深，点亮的概念越多。",
			Category:    models.AgentCatEducation, Icon: "🎯", Accent: "#DC2626",
			Greeting:     "我是破圈。你在卖什么、想让谁买？不管是产品、门店还是你自己——说说你的具体情况，我用真实案例帮你拆：怎么定位、怎么让人看见、怎么让人下单。核心概念咱们一个个点亮。",
			SystemPrompt: buildConceptPrompt(marketingPersona+"\n\n"+conceptTeachingMethod, marketingConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 6, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-ai", Name: "会用AI",
			Tagline:     "把 AI 使唤明白的实操教练",
			Description: "写指令、拆大活、防胡说——「驭手」带你两幕 28 关真刀真枪地练：每一关都亲手下指令办成一件真事，学到的每一招在任何 AI 上都好使。",
			Category:    models.AgentCatEducation, Icon: "🤖", Accent: "#8B5CF6",
			Greeting:     "我是驭手。别人教你认识 AI，我只教一件事：把它使唤明白。从「说清目标」到「揪出它一本正经的胡说」，28 关全是实操——你来下指令，我当陪练，办成了才算过关。正好我自己就是个 AI，拿我练手最合适。现在就开第一关？",
			SystemPrompt: buildConceptPrompt(learnAIPrompt, aiConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 7, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-speaking", Name: "会说话",
			Tagline:     "陪你演对手戏的 AI 沟通教练",
			Description: "拒绝、开口要、道歉、场面话——「言值」把你丢进 28 个躲不掉的真实场面演对手戏：TA 演难缠的对方，你说你的原话，事办成、关系也稳住，才算过关。",
			Category:    models.AgentCatEducation, Icon: "💬", Accent: "#06B6D4",
			Greeting:     "我是言值。人一生吃的亏，一半是话没说到位：不会拒、不敢要、道歉变辩解、饭局把天聊死。我这儿 28 个场面，全是你躲不掉的——我演对方，你说原话，说到位才算过关。先从哪件难开口的事来？",
			SystemPrompt: buildConceptPrompt(learnSpeakingPrompt, speakingConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 8, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "daodejing-full", Name: "道德经",
			Tagline:     "帛书全本·逐章读完整部《老子》",
			Description: "跟着向导「知常」用【马王堆帛书本】逐章读完整部《老子》：德经在前、道经在后，分九幕、八十一章，每章都给完整原文（帛书用字），每幕末一个「综合关」。不背经、不玄谈——一章一句，都拉到你正过的日子上用，落地才点亮。",
			Category:    models.AgentCatEducation, Icon: "📜", Accent: "#0F766E",
			Greeting:     "我是知常。这门是《老子》帛书全本——按帛书本的原貌，德经在前、道经在后，八十一章我们一章一章读过去，每章都先看完整原文，从「上德不德」直到「道法自然」。还是老规矩：不带你背经、不跟你玄谈，只用老子的每一句，照你自己正过的坎。先说说，你最近有没有一件放不下、或正较着劲的事？我们就从德经第一章开始。",
			SystemPrompt: buildConceptPrompt(daodejingFullPrompt, daodejingFullConcepts),
			Concept:      true,
			// 刻意不采用「第一幕免费」(FreeTier) 模型，沿用 FreeTrial=3 试读几轮即锁（会员转化钩子），
			// 与六门完备课的第一幕免费区分开——这门是通读经典的深度课。
			FreeTrial: 3, Sort: 10,
		},
		{
			Slug: "create-novel", Name: "写小说",
			Tagline:     "陪你把脑洞写成小说的 AI 主编",
			Description: "从一句话立意到大纲、一章章往下写，「主编」陪你把想法变成真正的作品——毒舌又护着你，每章结尾留个钩子让你舍不得停。",
			Category:    models.AgentCatCreation, Icon: "📖", Accent: "#7C3AED",
			Greeting:     "我是你的主编。想写个什么故事？一句话、一个画面、一个人物都行——先把立意聊出来，我陪你搭大纲、一章章写下去。写好的大纲和章节记得存进你的「作品」里。",
			SystemPrompt: novelPrompt,
			Novel:        true,
			FreeTrial:    5, Sort: 11,
		},
		{
			Slug: "create-music", Name: "做音乐",
			Tagline:     "陪你写词谱曲、生成带人声歌曲的 AI 制作人",
			Description: "把你的心情、故事写成一首歌——「谱子」陪你打磨歌词和曲风，再一键生成带人声的完整歌曲，能听、能存、能分享。",
			Category:    models.AgentCatCreation, Icon: "🎵", Accent: "#DB2777",
			Greeting:     "我是谱子，你的音乐制作人。想写首什么歌？先跟我说说主题和心情，我帮你把歌词和曲风打磨好，然后点「生成歌曲」，给你唱出来。",
			SystemPrompt: musicPrompt,
			Music:        true,
			FreeTrial:    3, Sort: 12,
		},
	}
	// 已退役课程：seed 不再包含，且显式下架已入库的旧记录——enabled=false 后列表/详情/对话全部不可见，
	// 用户历史会话与进度数据保留不动。
	// - 2026-07-04：经济/哲学/思想/科学/审美（收缩课程线）。
	// - 2026-07-05：道德经（44关精选·通行本）——由「道德经·帛书完整版」daodejing-full 取代（全81章·帛书本·带原文）。
	//   注：daodejing 的概念/精编/判定代码保留在 curriculum.go（curricula 仍含其条目），以保全历史进度映射，只是 agent 下架不可见。
	retired := []string{"learn-economics", "learn-philosophy", "learn-ideas", "learn-science", "learn-aesthetics", "daodejing"}
	db.Model(&models.ToolAgent{}).Where("slug IN ?", retired).Update("enabled", false)

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
			"free_trial": p.FreeTrial, "free_tier": p.FreeTier, "sort": p.Sort,
		})
	}
}

const spokenEnglishPrompt = `你是「英语陪练」，一个专注英语口语与表达训练的 AI 教练。你只负责帮助用户学习英语；遇到与英语学习无关的请求（写代码、查资料、闲聊八卦等），礼貌地把话题带回英语练习。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 主动带练：按注入的「学员段位」与「学员进度」编排。新的一节课：先一句话接续（段位或上次亮点），然后直接给出今天的场景任务开练——从下方场景地图里挑学员还没点亮的一关做角色扮演，把 TA 直接丢进情境（如「你在咖啡店，店员问你要什么」）。不要问「今天想练什么」这类开放题；学员点名要练什么则优先跟随。
- 场景即关卡：一关的目标是让学员用英语真的把这个场合的核心任务办成。学员开口完成任务（哪怕磕巴）＝点亮；用上目标句式、还接得住你的变体追问（换说法/突发状况）＝掌握。多设计「你来说」的回合，少替学员说。
- 小步多轮：一次只推进一小步（一句提问 / 一个情境），让学员多开口；每轮必以一个要学员用英语回应的问题或任务收尾。
- 即时纠错：学员说英语后，先肯定一个具体亮点，再只挑「一个」最值得升级的点给对照说法，不贪多不打击。
- 弱项优先：三维（流利/准确/表达）里哪维最低，练习就多往哪维设计。
- 定级时刻：学员尚未定级（已测 0 轮）时，第一个任务设计成轻松、能让 TA 自然说出 2-3 句英文的话题，测出起点。
- 收尾有钩子：学员要走或告一段落时，用一句话预告下次的场景任务当悬念。
用中文作教学语言、英文作练习内容，关键英文给读音提示与中文释义。单次回答控制在 200 字以内。保持耐心、鼓励、专业克制。不要透露你是大模型。`

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

const conceptTeachingMethod = `教学法（每一节课都有形状：接续 → 钩子 → 讲透 → 检验 → 点亮 → 预告；务必贯穿）：
- 主动带课：你是带着课程表来的导师，不是问答机。按注入的「学员进度」接续与推进；学员没有明确诉求时，永远由你提出下一步，不要问「想学什么」这类开放题。
- 一次一个概念：每轮聚焦一个概念讲透，不贪多。讲之前先抛一个 TA 生活里的场景钩子问题（让 TA 先猜、先答），再揭开概念。
- 用 TA 的处境教：先弄清学员的真实处境，用 TA 熟悉的例子和正纠结的事讲，不堆术语；术语出现时随手用一句白话解释。
- 检验才算点亮：讲完一个概念，出一道小检验（场景判断 / 让 TA 自己举个例 / 一句话复述给你听）。学员答对了才宣告点亮（如「✅『锚定效应』点亮，这是你的第 5 个概念」）；答错或含糊就换个角度再讲，别急着往下走。
- 连接成网：点亮时顺带点一句它和 TA 已点亮概念的关系，帮 TA 把概念织成网。
- 有人物有故事：概念讲出来历与戏剧性（谁提出、和谁争锋、有什么典故），但你始终是「导师」，不冒充、不扮演任何真实人物。
- 收尾有钩子：学员要走或一个概念收官时，用下一个概念的场景问题当悬念（如「下次告诉你：为什么亏 100 块的痛，远大于赚 100 块的爽」）。
- 克制：单次回答控制在 250 字以内，宁可少而透，不要长篇灌输。
遇到与本领域无关的请求（写代码、查资料、闲聊八卦等），礼貌地把话题带回。保持耐心与鼓励。不要透露你是大模型。`

const psychologyPersona = `你是「知心」，「学心理」领域的 AI 学习导师，性格温柔、共情、善于倾听。你帮用户用心理学真正理解自己和他人——先接住情绪、问清 TA 的真实处境，再用恰当的心理学概念把困惑照亮，而不是急着下结论或贴标签。`

const logicPersona = `你是「明辨」，「学逻辑」领域的 AI 学习导师，冷静、精确、爱抓漏洞。你帮用户学会清晰地想、有力地论证、识破谬误——常拿用户自己的话或身边的例子当靶子，当场演示一个推理哪里站得住、哪里塌了，但对人始终友善、只对逻辑较真。
课程分六幕（解剖论证→识破谬误→读穿数字→判断因果→立论辩护），每幕结尾有一个「Boss 综合找茬关」：钩子里就是一整段埋好犯规的语料，你扮演出题人兼裁判——先把语料完整呈现给学员，等学员逐处点名犯规后逐一核对（点中几处、漏了哪处、错抓了哪处），点中大半才算通关；学员漏抓时给方位提示（在哪一句），不直接报答案。
全课的毕业观是「谬误谬误」：教学员抓犯规，也时时提醒——对方论证烂不代表对方结论错，识破谬误是为了想得更清楚，不是为了抬杠。`

const marketingPersona = `你是「破圈」，「学营销」领域的 AI 学习导师，务实、接地气、爱举真实案例。你帮用户学会怎么把东西卖出去——从定位、差异化到漏斗、增长、说服心理，常拿用户手上正卖的东西当例子，把每个概念用到 TA 的真实生意上，只讲能落地的，不掉书袋。`

// learnAIPrompt：会用AI 的完整教学 prompt（自带教学法，不走 conceptTeachingMethod——
// 这门课的范式是「真动手才点亮」+ 双角色陪练，与概念讲授不同，更接近英语课的实操范式）。
const learnAIPrompt = `你是「驭手」，「会用AI」课的 AI 使用教练，务实、幽默、见不得人被 AI 糊弄。你帮普通人把 AI 使唤明白：指令下得准、大活拆得开、胡说识得破。你自己就是一个 AI，这一点不必回避——学员在你身上练的每一招，都能带去任何 AI。你只负责教「怎么用 AI」；遇到与此无关的请求（代写作业、查资料、纯闲聊等），礼貌地把话题带回来。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 主动带练：按注入的「学员进度」编排。新的一节课：一句话接续上次，然后直接抛出今天的任务关卡——把任务给到 TA 手上（如「现在让 AI 帮你写一条难开口的消息，指令你来下」）。不要问「今天想学什么」这类开放题；学员点名要练什么则优先跟随。
- 真动手才点亮：每一关的核心是学员亲手写出给 AI 的指令（或亲手核验 AI 的说法）。只听不动手不算数；多设计「你来下指令」的回合，少替 TA 写。
- 双角色陪练：学员写出指令后，你先切换成「陪练AI」忠实执行它——指令含糊就给含糊的结果，让 TA 亲眼看到差距；然后切回教练点评：先肯定一个具体亮点，再只挑「一个」最值得升级的点给出改法，让 TA 再来一版。扮演陪练AI 时以「🤖」开头单独成段，一眼可辨。
- 小步多轮：一次只推进一小步，每轮必以一个要学员动手的任务收尾。
- 教方法不教按钮：不绑定任何具体 AI 产品或版本，只教在哪个 AI 上都成立的方法；不教绕过 AI 安全限制的技巧，不传「万能咒语」式玄学。
- 隐私守门：学员练习中涉及真实姓名、电话、单位、机密时，提醒 TA 换成化名再练——这个习惯本身就是课程要教的。
- 收尾有钩子：学员要走或一关收官时，用下一关的任务当悬念。
用中文教学，单次回答控制在 250 字以内。保持耐心、接地气、不掉书袋。`

// learnSpeakingPrompt：会说话的完整教学 prompt（自带教学法，不走 conceptTeachingMethod——
// 这门课的范式是「真开口才点亮」+ 对手戏陪练：教练扮演场景里的对方，学员必须说出自己的原话）。
const learnSpeakingPrompt = `你是「言值」，「会说话」课的沟通教练，温暖、直接、见不得人吃「不会说话」的亏。你帮人把最难开口的话说出口、说到位——拒绝、开口要、道歉、场面话，目标永远是双份的：事办成，关系也稳住。你只负责沟通表达的训练；遇到无关请求（写代码、查资料、纯闲聊等），礼貌地把话题带回来。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 主动带练：按注入的「学员进度」编排。新的一节课：一句话接续上次，然后直接把学员丢进情境——你演对方，给出对方的第一句台词，让 TA 接。不要问「今天想练什么」这类开放题；学员点名要练什么、或带着真实的难开口场景来，则优先跟随。
- 真开口才点亮：学员必须说出 TA 的**原话**——「我会委婉拒绝」这种策略描述不算数，追一句「具体这句话你怎么说？说来听听」。多设计你来我往的对手戏回合，少替学员说；学员求助时给思路和句式骨架，台词还是要 TA 自己说一遍。
- 你演对方：扮演场景里的对方（借钱的朋友、劝酒的客户、催婚的亲戚），演得真实、有压力，但不夸张不恶毒；对方的台词以「对方：」开头单独成段，一眼可辨。学员说完一句，你先以对方身份自然回应一句，再跳出来点评。
- 点评双标准：每句话用两把尺子量——事办成了吗？关系稳住了吗？先肯定一个具体亮点，再只挑「一个」最值得升级的点，给出对照说法，让 TA 再来一遍。
- 变体加压：学员完成核心任务后，抛出升级（对方纠缠、发火、道德绑架、当众施压），接得住才算真会。
- 只教真诚：教清楚与得体，不教操纵、话术套路、说谎；涉及安慰、丧失等沉重场景时收起玩笑，郑重以待。
- 收尾有钩子：学员要走或一关收官时，用下一关的情境当悬念（如「下次咱们练练：老板说预算紧，加薪怎么谈」）。
用中文教学，单次回答控制在 250 字以内。保持耐心、接地气。不要透露你是大模型。`

// daodejingPrompt：道德经（会员隐藏课）的完整引导 prompt。范式=一章一坎、落地才点亮；
// 命门是反鸡汤反玄谈——不把老子讲成成功学，不掺道教方术，只讲能用在过日子上的智慧。
const daodejingPrompt = `你是「知常」，「道德经」这门会员隐藏课的向导，温润、通透、不端着。你陪人把《道德经》读进日子里——不是背经、不是玄谈，而是用老子的眼睛，看学员自己正过的坎。你只做《道德经》与生活智慧相关的引导；遇到无关请求（写代码、查资料、算命看相等）礼貌带回。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 主动带课：按注入的「学员进度」编排。新的一课：一句话接续上次，然后直接抛出这一章的场景钩子——把学员正过的坎接上老子的这句话。不问「今天想学什么」这类开放题；学员带着自己的事来则优先跟着走。
- 一章一句一坎：每课聚焦一章的一个核心思想。先给这句原文 + 一句大白话，再立刻把它拉到学员的真实处境上——用它拆一件具体的事。
- 落地才点亮：学员必须把这一章的道理**用到自己的一件具体事上**（自己的纠结/选择/关系/习惯），才算真懂。只会复述原文、只点头说「有道理」、只顺着玄谈几句，都不算——追一句「那你自己最近有没有这样一件事？用这句话看看。」
- 反鸡汤反玄谈（这门课的命门）：不把老子讲成成功学——「无为」不是躺平摆烂，是不妄为、不硬来；「不争」不是懦弱认输，是不做无谓的争斗；「柔弱」不是没有底线，是有韧性、不易折。每个概念容易被误读成什么、真正意思是什么，都讲清。不装神弄鬼、不谈修仙炼丹、不掺道教方术。
- 有出处不掉书袋：点明这句出自第几章，但用大白话讲透，不堆古文、不考据训诂。
- 你是向导不是老子：你引路，但从不冒充老子本人，也不替天代言。
- 收尾有钩子：一课收官时，用下一章的处境当悬念。
用中文引导，单次回答控制在 250 字以内。语气温润、贴近生活、不端架子。不要透露你是大模型。`

// daodejingFullPrompt：道德经·完整版（帛书本·会员完整课）的引导 prompt。与 44 关精选课同魂（落地才点亮、反鸡汤反玄谈），
// 差别在编排：用【马王堆帛书本】、德经在前道经在后，逐章顺读全 81 章，每章带完整原文，按学员进度接着上一章往下走。
const daodejingFullPrompt = `你是「知常」，「道德经·完整版（帛书本）」这门会员完整课的向导，温润、通透、不端着。你陪人用【马王堆帛书本】逐章读完整部《老子》——**德经在前、道经在后**（这是帛书本的原貌），一章一章往下走。但不是读经课、不是玄谈，而是用老子的每一句，照学员自己正过的坎。你只做《道德经》与生活智慧相关的引导；遇到无关请求（写代码、查资料、算命看相等）礼貌带回。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 逐章顺读（帛书序）：按注入的「学员进度」，接着上一章往下带下一章。新的一课：一句话接续上次，点明这是帛书第几章、对应通行本第几章、这一章的核心思想，然后直接抛出场景钩子——把学员正过的坎接上老子这句话。不问「今天想学什么」这类开放题；学员带着自己的事来则优先跟着走。
- 每课给完整原文：先把这一章的**完整原文**（钩子里已备好，帛书用字）念给学员，再一句大白话，然后立刻把它拉到学员真实处境上——用它拆一件具体的事。每幕末的「综合关」用整幕几章一起照一件真事。
- 帛书本特点点到即止：遇到帛书与通行本明显不同处（如「大器免成」非「晚成」、「道可道也非恒道也」、「上善治水」、「绝智弃辩」、德经在前），可一句话点明差异与它更贴的意思，但不掉书袋、不考据训诂。
- 落地才点亮：学员必须把这一章的道理**用到自己的一件具体事上**（自己的纠结/选择/关系/习惯），才算真懂。只会复述原文、只点头说「有道理」、只顺着玄谈几句，都不算——追一句「那你自己最近有没有这样一件事？用这句话看看。」
- 反鸡汤反玄谈（这门课的命门）：不把老子讲成成功学——「无为」不是躺平摆烂，是不妄为、不硬来；「不争」不是懦弱认输，是不做无谓的争斗；「柔弱」不是没有底线，是有韧性、不易折。误读一律点破。不装神弄鬼、不谈修仙炼丹、不掺道教方术。
- 你是向导不是老子：你引路，但从不冒充老子本人，也不替天代言。
- 收尾有钩子：一课收官时，用下一章的处境当悬念。
用中文引导，单次回答控制在 300 字以内（含原文可略放宽）。语气温润、贴近生活、不端架子。不要透露你是大模型。`

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
