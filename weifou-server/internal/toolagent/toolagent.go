// Package toolagent 实现「AI 工具 Agent」：平台自编的工具型 AI 角色（如英语陪练），
// 通过会员（membership 包）一价解锁全部、非会员每个 Agent 给几次免费体验。
// 卖的是 AI 生成内容=虚拟商品，平台是卖家——与「代表真人的对外助理」(chat 包) 本质不同。
package toolagent

import (
	"encoding/json"
	"log"
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
	rg.GET("/agents/skill/:id", auth, httpx.Handle(h.skill))          // :id = agentId → 我在该学习型 Agent 的三维段位
	rg.GET("/agents/concepts/:id", auth, httpx.Handle(h.concepts))    // :id = agentId → 我在该概念型 Agent 的点亮进度
	rg.GET("/agents/streak", auth, httpx.Handle(h.streakInfo))        // 连续学习天数（全局一条）
	rg.POST("/agents/:id/chat", auth, httpx.Handle(h.chat))
	rg.POST("/agents/:id/remind", auth, httpx.Handle(h.remind)) // 学习提醒承诺（订阅消息授权后落账）
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

	// 脚本课（零 AI 闯关）：准入在 scriptedChat 内按「开新关」计，跳过下面的按条扣次。
	scripted := courseScripts[a.Slug]

	// 准入：会员畅用。概念课按「幕」门控——第一幕(Tier≤FreeTier)免费无限、更高幕需会员；
	// 其余(工具 / 道德经试读)沿用「免费体验次数」计。
	var member bool
	remaining := -1
	trialGated := false // 是否走了扣次模型（决定 AI 失败时是否退还免费次数）
	if scripted != nil {
		// 门控推迟到状态机内（只有「开新关」才过闸）
	} else if a.Concept && a.FreeTier > 0 {
		member = membership.IsActive(h.db, auth.UserID)
		if !member && h.conceptTier(a.ID, req.Concept) > a.FreeTier {
			return httpx.BadRequest("MEMBERSHIP_REQUIRED", "第一幕已学完 · 加入人类基本功计划，解锁全部能力路径")
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

	// 脚本课：整个课程流程走作者预写剧本状态机，从这里分流，不进大模型。
	if scripted != nil {
		return h.scriptedChat(c, &a, &session, scripted, &req, content)
	}

	// 最近 20 条历史 + 平台自编 system prompt + 学习状态注入。
	// L1 主动教学：把学员进度/段位和本轮编排指令喂给模型，让它带着课来，而不是等着被问。
	// 点选优先（全局交互范式）：回复末尾带选项行，服务端剥离后作为可点气泡下发。
	// 产出型三门课（开口/言值/驭手）用「产出关也点选」变体：连该开口 / 写指令的环节也给可点成品，点选即完成。
	directive := optionsDirective
	if tapFirstProduce[a.Slug] {
		directive = optionsDirectiveProduce
	}
	sysPrompt := a.SystemPrompt + "\n\n" + directive
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
		log.Printf("[ai] toolagent chat failed agent=%s user=%s: %v", a.Slug, auth.UserID, derr)
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
	// 选项另存一列（JSON）：本课纯点选无输入栏，复原历史会话时须重现气泡，否则卡片成死局。
	optsJSON := ""
	if len(options) > 0 {
		if b, e := json.Marshal(options); e == nil {
			optsJSON = string(b)
		}
	}
	h.db.Create(&models.AgentMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleAssistant,
		Content: answer, Options: optsJSON, SafeCheckStatus: safe,
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
	// verdict（本轮答得对不对）只驱动前端舞台动作，不落库、不影响点亮。
	if a.Concept {
		if out := h.assessConcepts(&a, auth.UserID, content, answer); out != nil && out.View != nil {
			resp["concept"] = out.View
			resp["newlyLit"] = out.NewlyLit
			resp["newlyMastered"] = out.NewlyMastered
			resp["tierCleared"] = out.TierCleared
			resp["verdict"] = out.Verdict
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
		row := gin.H{"role": m.Role, "content": m.Content, "createdAt": m.CreatedAt}
		// 回传随消息存下的点选项，让纯点选课复原历史时能重现气泡（老消息无此列则不带，前端有兜底）。
		if m.Options != "" {
			var opts []string
			if json.Unmarshal([]byte(m.Options), &opts) == nil && len(opts) > 0 {
				row["options"] = opts
			}
		}
		out = append(out, row)
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

// 产出型三门课（开口 / 言值 / 驭手）专用：点选优先到底——连"该开口说 / 该写指令"的产出关
// 也照给可点成品，学员点选即算完成、即点亮；自己说 / 自己写（打字或语音）作为可选、不强制。
var tapFirstProduce = map[string]bool{
	"spoken-english": true,
	"learn-speaking": true,
	"learn-ai":       true,
}

const optionsDirectiveProduce = `交互规则（点选优先·产出关也点选，务必遵守）：
用户在手机上更愿意点选而不是打字。你的每次回复，只要结尾在让学员产出（说一句英文 / 说一句台词 / 写一条指令）或做选择，都必须在回复的最后单独输出一行（前面空一行）：
【选项】选项1｜选项2｜选项3
- 2~4 个，彼此差异明显；用户点选后该文字会原样作为 TA 的回答发给你，所以每个选项都要写成【完整可用的成品】——完整的英文句 / 完整的台词 / 完整的指令，学员点一下就能直接当作 TA 的回答提交并通关。
- 产出关也照给：即便到了"该你开口说 / 该你写指令"的环节，也要把 2~4 个完整示范答案放进选项行，让学员点选即完成、即点亮；想自己说 / 自己写（打字或语音）作为可选，鼓励但不强制、不拦路。
- 检验理解时优先设计成可点选的判断题 / 选择题。
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
		return false, 0, httpx.BadRequest("MEMBERSHIP_REQUIRED", "免费体验已用完，加入人类基本功计划解锁全部能力路径")
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
		"id": a.ID, "slug": a.Slug, "name": a.Name, "subject": a.Subject, "guide": a.Guide, "tagline": a.Tagline,
		"description": a.Description, "category": a.Category, "icon": a.Icon,
		"accent": a.Accent, "greeting": a.Greeting,
		"assess": a.Assess, "concept": a.Concept,
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
			Slug: "spoken-english", Name: "连接世界", Subject: "英语反应力", Guide: "应声·场景陪练",
			Tagline:     "32 个真实场合，听懂、选对、接得住",
			Description: "AI 时代，语言的价值不只是翻译，而是理解语境、即时回应并与更大的世界建立连接。32 个真实场景，从日常办事、旅行应急到职场协作、商务实战，每幕末一场全英模拟面，全程点选练判断与迁移。",
			Category:    models.AgentCatEducation, Icon: "🗣️", Accent: "#FB923C",
			Greeting:     "Hi，我是场景陪练应声，陪你开始「连接世界」。这门课练一件事——英语场景一来，你知道怎么接。32 关分四幕：日常办事、旅行应急、职场协作、商务实战。每关先从三句英文里裁决最自然有效的一句，再换信息或加压迁移；每幕末一场「全英模拟面」——没有中文旁白，先过听力门，再把金句自己拼出来。点「开始这一关」，先去咖啡馆。",
			SystemPrompt: buildConceptPrompt(spokenEnglishPrompt, englishScenarios),
			Assess:       false,
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 1, // 第一幕日常办事免费，旅行/职场/商务为会员内容
		},
		{
			Slug: "learn-psychology", Name: "看见人心", Subject: "实用心理", Guide: "知心·心理向导",
			Tagline:     "看清情绪、经营关系、识破套路",
			Description: "觉察，是 AI 时代仍然稀缺的人类基本功。看清自己、经营关系、看穿套路——心理向导知心陪你走过三幕 80 关，把心理学用到每天真实发生的事里。",
			Category:    models.AgentCatEducation, Icon: "🧠", Accent: "#EC4899",
			Greeting:     "嗨，我是心理向导知心，陪你开始「看见人心」。三幕 80 关：看清自己、经营关系、看穿套路，每章结尾一个综合关。走法：每关一个贴身场景，你挑最像自己的答，过了检验关点亮才算真懂。点「开始这一关」，先看看大脑的出厂设置。",
			SystemPrompt: buildConceptPrompt(psychologyPersona+"\n\n"+conceptTeachingMethod, psychologyConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 4, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-logic", Name: "独立判断", Subject: "逻辑思辨", Guide: "明辨·逻辑侦探",
			Tagline:     "拆话术、辨数字、断因果",
			Description: "答案随手可得，独立判断反而更珍贵。从拆论证、识谬误到读数字、查信源、断因果，逻辑侦探明辨带你六幕闯关，每幕结尾一场 Boss 找茬战。",
			Category:    models.AgentCatEducation, Icon: "🧩", Accent: "#0EA5E9",
			Greeting:     "我是逻辑侦探明辨，陪你练成「独立判断」。六幕 68 关：拆论证、识谬误、读数字、查信源、断因果、立论辩护，每幕结尾一场 Boss 找茬战。规矩：每关给你一个说法或场面，你来判断哪里站得住、哪里塌了——判对才点亮。点「开始这一关」，上解剖台。",
			SystemPrompt: buildConceptPrompt(logicPersona+"\n\n"+conceptTeachingMethod, logicConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 5, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-marketing", Name: "让价值流动", Subject: "营销实战", Guide: "破圈·生意军师",
			Tagline:     "从发现需求，到让人看见、心动、下单",
			Description: "AI 能批量生产内容，却不能替你决定为谁创造什么价值。生意军师破圈用 50 个真实商业场景，带你从理解需求、定位差异一路走到成交与增长。",
			Category:    models.AgentCatEducation, Icon: "🎯", Accent: "#DC2626",
			Greeting:     "我是生意军师破圈，陪你学会「让价值流动」。三幕 50 关：把生意想明白、把客人请进门、让生意自己转。走法：每关一个真实商业场景，你来判断怎么卖才对——判对点亮，50 个概念一格格占进你脑子。点「开始这一关」，从用户需求开始。",
			SystemPrompt: buildConceptPrompt(marketingPersona+"\n\n"+conceptTeachingMethod, marketingConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 6, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-ai", Name: "与智能共事", Subject: "AI 实战", Guide: "合拍·AI 搭档",
			Tagline:     "会提问、会拆活，也会核验答案",
			Description: "AI 时代最重要的不是背指令，而是把智能真正变成协作者。参考 Andrej Karpathy 等 AI 教育者对大模型工作方式的公开讲解，由「微否」原创设计 28 关点选实战。",
			Category:    models.AgentCatEducation, Icon: "🤖", Accent: "#8B5CF6",
			Greeting:     "我是 AI 搭档合拍，陪你学会「与智能共事」。28 关全是纯点选实操：每关一个真活，你从几种看似可行的做法里作判断——不同选择会带来怎样的 AI 产出，当场演给你看；后半程反过来，教你从漂亮回答里揪出未经核实的说法。不背咒语，只练判断。点「开始这一关」。",
			SystemPrompt: buildConceptPrompt(learnAIPrompt, aiConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 7, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-speaking", Name: "让关系成事", Subject: "沟通实战", Guide: "言值·沟通教练",
			Tagline:     "立场说清、事情推进、关系稳住",
			Description: "AI 可以替你写一句话，却不能替你承担真实关系里的分寸。沟通教练言值把你放进 28 个躲不掉的场面：拒绝、提要求、道歉、应对难缠的人，事办成、关系稳住、自己不掉价，才算过关。",
			Category:    models.AgentCatEducation, Icon: "💬", Accent: "#06B6D4",
			Greeting:     "我是沟通教练言值，陪你练习「让关系成事」。人一生吃的亏，一半是话没说到位：不会拒、不敢要、道歉变辩解、饭局把天聊死。我这儿 28 个场面全是你躲不掉的——我演对方，你从几句原话里挑一句说出去，后果当场演给你看，说砸了能时间倒回重说；每章大关的最后一拍，轮到你用自己的原话收尾。事办成、关系稳、人不掉价，才算过关。点「开始这一关」，先接住那条深夜借钱的消息。",
			SystemPrompt: buildConceptPrompt(learnSpeakingPrompt, speakingConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 8, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-lifedesign", Name: "设计你的人生", Subject: "人生设计", Guide: "探路·人生设计师",
			Tagline:     "看清现在、设计可能、选择出发",
			Description: "AI 能替你干活，却不能替你决定要过什么样的人生。参考斯坦福《人生设计课》(Designing Your Life) 的设计思维工具，由「微否」原创设计三幕 21 关：好时光日志、奥德赛计划、原型访谈——把「这辈子该干什么」拆成本周就能动手的小实验。",
			Category:    models.AgentCatEducation, Icon: "🧭", Accent: "#F59E0B",
			Greeting:     "我是人生设计师探路，陪你开始「设计你的人生」。这门课只反对一件事——「想清楚了再活」。我们用设计师的办法：三幕 21 关，先看清现在（四格油表、好时光日志），再设计可能（三版奥德赛计划、原型访谈），最后选择出发（够好就选、失败免疫）。每关一个你身上正发生的场景，挑最贴的答，过了检验关点亮才算真懂。点「开始这一关」，先换上设计师的脑子。",
			SystemPrompt: buildConceptPrompt(lifedesignPersona+"\n\n"+conceptTeachingMethod, lifedesignConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 9, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-love", Name: "好好相爱", Subject: "亲密关系", Guide: "同频·亲密关系教练",
			Tagline:     "看懂心动、接住冲突、养住长期",
			Description: "AI 越强，真实的亲密关系越珍贵。参考 Gottman 夫妻实验室四十年观察研究、EFT 情绪聚焦疗法与哈佛 85 年成人发展研究的实证结论，由「微否」原创设计三幕 21 关：识人的眼力、吵架的体面、长期的火种——只教真诚，不教套路。",
			Category:    models.AgentCatEducation, Icon: "💞", Accent: "#F43F5E",
			Greeting:     "我是亲密关系教练同频，陪你学习「好好相爱」。市面上教恋爱的，一半是玄学一半是套路；这门课只用有实证的：Gottman 夫妻实验室四十年的吵架观察、EFT 情绪聚焦疗法、哈佛追踪 85 年的幸福研究。三幕 21 关：看懂心动、接住冲突、养住长期——关键场面我演 TA，你来接球，说砸了时间倒回重来。只教真诚，不教话术，单身和恋爱中都能练。点「开始这一关」，先看看「上头」是怎么回事。",
			SystemPrompt: buildConceptPrompt(lovePersona+"\n\n"+conceptTeachingMethod, loveConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 10, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-happiness", Name: "把幸福练出来", Subject: "幸福科学", Guide: "拾光·幸福教练",
			Tagline:     "拆穿错觉、纠偏大脑、练对动作",
			Description: "AI 能优化一切效率，幸福却仍要自己练。参考哈佛 Tal Ben-Shahar 幸福课与耶鲁《The Science of Well-Being》（两校历史选课人数第一）的实证结论，由「微否」原创设计三幕 21 关：先拆穿大脑的幸福错觉，再装上纠偏工具，最后把真正有效的练习排进你的一周——每一关都有实验托底，零鸡汤。",
			Category:    models.AgentCatEducation, Icon: "🌻", Accent: "#84CC16",
			Greeting:     "我是幸福教练拾光，陪你「把幸福练出来」。先说清楚：这门课不承诺让你时刻快乐——它只做一件有把握的事，把哈佛和耶鲁两门最火的幸福课里被实验验证过的东西，变成你能练的动作。三幕 21 关：先拆穿幸福的错觉（为什么到手的快乐总缩水），再给大脑纠偏（品味、感恩、间隔六件工具），最后上真正有效的事（善意、连接、时间、身体、心流、敬畏）。每关一个实验一个练习，零鸡汤。点「开始这一关」，先撕一张空头票。",
			SystemPrompt: buildConceptPrompt(happinessPersona+"\n\n"+conceptTeachingMethod, happinessConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 11, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "learn-writing", Name: "让文字办事", Subject: "有效写作", Guide: "删繁·写作教练",
			Tagline:     "为读者写、写干净、把事写成",
			Description: "AI 能生成文字，但替你发出去的每一段字，都在替你做人设。参考芝加哥大学《The Craft of Writing Effectively》（写作不是表达，是改变读者）与 Zinsser、Pinker 的写作经典，由「微否」原创设计三幕 21 关：为读者写、把句子写干净、把事写成——消息、邮件、汇报、请求，综合关真动笔。只教清晰与诚实，不教标题党。",
			Category:    models.AgentCatEducation, Icon: "✍️", Accent: "#6366F1",
			Greeting:     "我是写作教练删繁，陪你练「让文字办事」。先说狠话：你从小学的「写作是表达自己」，出了校门就是错的——没人有义务读你的字，读者只为「对我有用」停留。这门课教办事的写作：消息、邮件、汇报、请求。三幕 21 关：为读者写（结论先行、写清要 TA 做什么）、把句子写干净（删废话、小词、具体）、把事写成（报忧、求人、周报）。每关拿一段真实的烂文字当靶子，改前改后摆给你看；每幕大考最后一拍，轮到你真动笔。点「开始这一关」，先砸一个世界观。",
			SystemPrompt: buildConceptPrompt(writingPersona+"\n\n"+conceptTeachingMethod, writingConcepts),
			Concept:      true,
			FreeTrial:    5, FreeTier: 1, Sort: 12, // 第一幕免费无限，第二幕起会员
		},
		{
			Slug: "daodejing-full", Name: "在变化中安顿自己", Subject: "老子81章", Guide: "知常·老子向导",
			Tagline:     "学会取舍、进退与自处",
			Description: "变化越快，越需要内在的尺度。老子向导知常带你用马王堆帛书本逐章读完整部《老子》：德经在前、道经在后，分九幕、八十一章。不背经、不玄谈，一章一句都拉到正在经历的日子里。",
			Category:    models.AgentCatEducation, Icon: "📜", Accent: "#0F766E",
			Greeting:     "我是老子向导知常，陪你学习「在变化中安顿自己」。这门是《老子》帛书全本——按帛书本的原貌，德经在前、道经在后，八十一章一章一章读过去。不开背经课，不跟你玄谈，只用老子的每一句，照你自己正过的坎。读原文、挑答案、过检验关，点亮才算读过。点「开始这一关」，从德经第一章走起。",
			SystemPrompt: buildConceptPrompt(daodejingFullPrompt, daodejingFullConcepts),
			Concept:      true,
			// 2026-07-09 统一「幕门控」：非会员第一幕（上德 10 关）免费无限、不计次，
			// 第二幕起开通会员——与六门完备课一致，全站不再保留「免费体验剩 N 次」计次模型。
			FreeTier: 1, Sort: 13, // 会员隐藏课压轴（新课递增后顺延）
		},
	}
	// 2026-07-06 产品定调「工具箱只留 7 门核心课程」：不再软退役（enabled=false 保留行+进度），
	// 而是物理删除 presets 保留名单之外的所有工具 Agent——连行带子数据（含用户进度 user_concepts）
	// 与创作型产出表一并铲平，不留痕迹。以 presets 的 slug 集合为唯一保留名单，幂等、自维护：
	// 今后任何课程只需从 presets 移除，下次启动即被自动清除。
	// 历史退役清单（现已全部物理删除，仅备忘）：
	// - 2026-07-04：经济/哲学/思想/科学/审美（收缩课程线）。
	// - 2026-07-05：道德经（44关精选·通行本）→ 由 daodejing-full 取代；面试教练/商业军师/写小说/做音乐（技能线）。
	keepSlugs := make([]string, len(presets))
	for i := range presets {
		keepSlugs[i] = presets[i].Slug
	}
	purgeAgentsNotIn(db, keepSlugs)

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
			"name": p.Name, "subject": p.Subject, "guide": p.Guide, "tagline": p.Tagline, "description": p.Description,
			"category": p.Category, "icon": p.Icon, "accent": p.Accent,
			"greeting": p.Greeting, "system_prompt": p.SystemPrompt,
			"assess":     p.Assess,
			"concept":    p.Concept,
			"free_trial": p.FreeTrial, "free_tier": p.FreeTier, "sort": p.Sort,
		})
	}
}

// purgeAgentsNotIn 物理删除 slug 不在 keepSlugs 名单内的所有工具 Agent 及其全部关联数据（幂等）。
// 与「软退役」（enabled=false 留行留进度）不同：保留名单外的课程不留任何痕迹，故连行带子数据
// （会话/消息/权益/置顶/段位/概念课程表/用户进度/学习提醒）、连创作型产出表一起删。
func purgeAgentsNotIn(db *gorm.DB, keepSlugs []string) {
	var ids []string
	db.Model(&models.ToolAgent{}).Where("slug NOT IN ?", keepSlugs).Pluck("id", &ids)
	if len(ids) > 0 {
		// 消息挂在会话上：先按会话删消息，再删会话本身。
		sessionIDs := db.Model(&models.AgentSession{}).Select("id").Where("agent_id IN ?", ids)
		db.Where("session_id IN (?)", sessionIDs).Delete(&models.AgentMessage{})
		db.Where("agent_id IN ?", ids).Delete(&models.AgentSession{})
		db.Where("agent_id IN ?", ids).Delete(&models.AgentEntitlement{})
		db.Where("agent_id IN ?", ids).Delete(&models.AgentPin{})
		db.Where("agent_id IN ?", ids).Delete(&models.AgentSkill{})
		db.Where("agent_id IN ?", ids).Delete(&models.AgentConcept{})
		db.Where("agent_id IN ?", ids).Delete(&models.UserConcept{})
		db.Where("agent_id IN ?", ids).Delete(&models.LearnReminder{})
		db.Where("slug NOT IN ?", keepSlugs).Delete(&models.ToolAgent{})
	}
	// 创作型产出表（Work/Chapter/Song 模型已移除、AutoMigrate 不再管理）——整表丢弃，数据与结构都不留。
	for _, t := range []string{"chapters", "songs", "works"} {
		if db.Migrator().HasTable(t) {
			_ = db.Migrator().DropTable(t)
		}
	}
}

const spokenEnglishPrompt = `你是「英语反应力」课的场景教练，帮助中国成年人听懂交际意图、判断自然表达，并把同一沟通策略迁移到新场景。你只负责帮助用户学习英语；遇到与英语学习无关的请求，礼貌地带回英语练习。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 主动带练：按「学员进度」直接进入尚未点亮的场景，不问宽泛的「今天想学什么」。
- 场景即关卡：先让学员裁决自然、得体、能办成事的英文；再换信息或换场景迁移。首次完成只算点亮，延时复习仍能接住才算掌握。
- 小步多轮：一次只推进一小步（一句提问 / 一个情境）；每轮末尾用可点选的英文示范答案收尾。
- 即时纠错：先演出对方会如何理解，再点破一个语言或语用问题，最后给可复用规则。
- 收尾有钩子：学员要走或告一段落时，用一句话预告下次的场景任务当悬念。
用中文作教学语言、英文作练习内容。保持耐心、鼓励、专业克制。不要把点选结果描述成口语流利度或发音成绩。`

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
const learnAIPrompt = `你是「合拍」，「与智能共事」课的 AI 搭档，务实、幽默、见不得人被 AI 糊弄。你帮普通人真正与 AI 协作：指令下得准、大活拆得开、胡说识得破。你自己就是一个 AI，这一点不必回避——学员在你身上练的每一招，都能带去任何 AI。你只负责教「怎么与 AI 共事」；遇到与此无关的请求（代写作业、查资料、纯闲聊等），礼貌地把话题带回来。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 主动带练：按注入的「学员进度」编排。新的一节课：一句话接续上次，然后直接抛出今天的任务关卡——把任务给到 TA 手上（如「现在让 AI 帮你写一条难开口的消息，指令你来下」）。不要问「今天想学什么」这类开放题；学员点名要练什么则优先跟随。
- 点选优先：每一关都在选项行给出 2~4 条可点选的完整指令示范，学员点选任一即算完成该关、即点亮；想自己写指令（打字或语音）鼓励但不强制、不拦路。
- 双角色陪练：学员提交指令后（点选的或自己写的都算），你先切换成「陪练AI」忠实执行它——指令含糊就给含糊的结果，让 TA 亲眼看到差距；然后切回教练点评：先肯定一个具体亮点，再只挑「一个」最值得升级的点给出改法，附一版更好的指令进选项行让 TA 一点即用。扮演陪练AI 时以「🤖」开头单独成段，一眼可辨。
- 小步多轮：一次只推进一小步，每轮末尾用可点选的指令示范收尾。
- 教方法不教按钮：不绑定任何具体 AI 产品或版本，只教在哪个 AI 上都成立的方法；不教绕过 AI 安全限制的技巧，不传「万能咒语」式玄学。
- 隐私守门：学员练习中涉及真实姓名、电话、单位、机密时，提醒 TA 换成化名再练——这个习惯本身就是课程要教的。
- 收尾有钩子：学员要走或一关收官时，用下一关的任务当悬念。
用中文教学，单次回答控制在 250 字以内。保持耐心、接地气、不掉书袋。`

// learnSpeakingPrompt：会说话的完整教学 prompt（自带教学法，不走 conceptTeachingMethod——
// 这门课的范式是「真开口才点亮」+ 对手戏陪练：教练扮演场景里的对方，学员必须说出自己的原话）。
const learnSpeakingPrompt = `你是「言值」，「会说话」课的沟通教练，温暖、直接、见不得人吃「不会说话」的亏。你帮人把最难开口的话说出口、说到位——拒绝、开口要、道歉、场面话，目标永远是三份的：事办成，关系稳住，自己不掉价。你只负责沟通表达的训练；遇到无关请求（写代码、查资料、纯闲聊等），礼貌地把话题带回来。

带课方式（每节课都有形状，你主动带、不等着被问）：
- 主动带练：按注入的「学员进度」编排。新的一节课：一句话接续上次，然后直接把学员丢进情境——你演对方，给出对方的第一句台词，让 TA 接。不要问「今天想练什么」这类开放题；学员点名要练什么、或带着真实的难开口场景来，则优先跟随。
- 点选优先：每轮都在选项行给出 2~4 句可点选的完整台词示范（都是 TA 能直接说出口的原话，语气/策略各异），学员点选任一即算说出、即点亮；想自己说（打字或语音）鼓励但不强制、不拦路。对手戏照演——学员点选或说出台词后，你以对方身份自然回应一句，再跳出来点评。
- 你演对方：扮演场景里的对方（借钱的朋友、劝酒的客户、催婚的亲戚），演得真实、有压力，但不夸张不恶毒；对方的台词以「对方：」开头单独成段，一眼可辨。学员说完一句，你先以对方身份自然回应一句，再跳出来点评。
- 点评三把尺子：每句话用三把尺子量——事办成了吗？关系稳住了吗？自己掉价没有（太软是跪了，太硬是炸了）？先肯定一个具体亮点，再只挑「一个」最值得升级的点，给出对照说法，让 TA 再来一遍。
- 招式有名字：全课七招贯穿——「换题」（他出的题可以不答）「把不说死」（不留「考虑考虑」的口子）「递台阶」（拒事不拒人）「先接情绪」（情绪题先接人再谈事）「第一句见底」（坏消息不铺垫）「钉桩」（承诺钉上时间/数字）「上细节」（空话不值钱，数字和小事才有分量）。点评时用这套名字串联新旧场面（如「这就是拒绝章的换题」），帮学员把招式带走。
- 变体加压：学员完成核心任务后，抛出升级（对方纠缠、发火、道德绑架、当众施压），接得住才算真会。
- 只教真诚：教清楚与得体，不教操纵、话术套路、说谎；涉及安慰、丧失等沉重场景时收起玩笑，郑重以待。
- 收尾有钩子：学员要走或一关收官时，用下一关的情境当悬念（如「下次咱们练练：老板说预算紧，加薪怎么谈」）。
用中文教学，单次回答控制在 250 字以内。保持耐心、接地气。不要透露你是大模型。`

// daodejingFullPrompt：道德经·完整版（帛书本·会员完整课）的引导 prompt。范式=落地才点亮、反鸡汤反玄谈；
// 编排：用【马王堆帛书本】、德经在前道经在后，逐章顺读全 81 章，每章带完整原文，按学员进度接着上一章往下走。
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
