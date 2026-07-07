// 脚本闯关引擎（零 AI）：把「像游戏轻于游戏」落到底——课程流程全程走作者预写剧本，
// 不调用大模型。一关的形状：Hook 开场（含原文，来自 curatedContent）→ 点选应答（每个
// 选项配预写回应，误读当场点破）→ 检验关（一个「懂了」项 + 常见误读项，错了温柔纠偏、
// 重选不罚）→ 答对即点亮（确定性，不经判定器）→ 顺路下一关 / 回地图。
// 这是 curatedContent「人工精编=护城河」的补完：Hook/Check 之外，把选项与回应也纳入策展。
// 收益：进关秒开、点选零延迟、永不弹「AI 服务不可用」、内容可测试守护（品质恒定）。
package toolagent

import (
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/membership"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

// tapOption 一个可点选项：Label 是点选气泡文字（≤20 字），Reply 是选中后的预写回应。
// 用在检验关时：正确项的 Reply=肯定语（为什么对），错误项的 Reply=温柔纠偏（不直给答案）。
type tapOption struct{ Label, Reply string }

// levelScript 一关的作者化剧本。Hook 与检验题面复用 curatedContent（不重复维护）。
// 两种形态：
//   - 两段式（概念课）：Taps 开场点选 → CheckOpts 检验关，答对点亮；
//   - 节点图（产出课的对手戏，Nodes 非空时生效）：Hook 开场后进 Nodes[0]，
//     每个选项自带对方的预写反应（后果演给学员看）与去向；走到 NodeClear 即点亮。
//     此时 Taps 不用；CheckOpts 仍必填——复习挑战（检索练习）用它快问快答升掌握。
type levelScript struct {
	Taps      []tapOption  // 开场点选（2-3 项，接住 Hook 结尾的提问；节点图关卡置空）
	Nodes     []scriptNode // 多轮分支节点图（对手戏）；空 = 两段式
	CheckOpts []tapOption  // 检验关选项（两段式的主流程门；节点图关卡只用于复习挑战）
	Correct   int          // CheckOpts 中正确项下标
	Clear     string       // 点亮语：一句收官 + 下一关悬念钩子
	Note      string       // 战报（≤20 字，落 UserConcept.Note，复习/卡片流回显）
}

// scriptNode 对手戏的一个节点。两种节点：
//   - 点选节点（Options 非空）：Prompt 是场面推进/对方加压的话（节点 0 可空——Hook 已含开场），
//     学员从 Options 挑一句「自己的原话」，Reply 里对方的反应把后果演出来；
//   - 跟读节点（Say 非空，开口课用）：学员按住麦克风读出 Say 这句，ASR 转写与之模糊匹配，
//     命中回 SayOK 并走 SayNext，未中回 SayFail 再试（不罚）。
type scriptNode struct {
	Prompt  string
	Options []nodeOption
	Say     string // 跟读目标句（非空 = 跟读节点，Options 置空）
	SayOK   string // 跟读命中的回应
	SayFail string // 未命中的鼓励语（引擎会自动补「再读一遍」提示）
	SayNext int    // 跟读命中后的去向（NodeClear 或节点下标）
}

// nodeOption 对手戏选项：Label = 学员的原话候选（≤20 字），Reply = 对方的预写反应 + 教练点破。
// Next 去向：NodeRetry 时间倒回本节点重选（后果看完不罚）、NodeClear 通关点亮、>=0 进入该节点。
type nodeOption struct {
	Label string
	Reply string
	Next  int
}

const (
	NodeRetry = -1 // 留在本节点重选（先看完 Reply 里的后果）
	NodeClear = -2 // 通关：点亮本关
)

// courseScripts：agent slug → concept slug → 剧本。有剧本的课程走脚本引擎，其余照旧走 LLM。
// 迁移判据（2026-07-06 定调）：概念课（检验=判断/找茬）适合脚本化；
// 产出课（开口/言值/驭手，点亮判据=学员真实产出）保留 LLM——纯点选会杀死课魂。
var courseScripts = map[string]map[string]levelScript{
	"daodejing-full":   daodejingFullScript,
	"learn-logic":      learnLogicScript,
	"learn-psychology": learnPsychologyScript,
	"learn-marketing":  learnMarketingScript,
	"learn-speaking":   learnSpeakingScript, // 节点图对手戏（产出课点选化第一门）
	"learn-ai":         learnAIScript,       // 节点图指令对比+找茬（产出课点选化第二门）
	"spoken-english":   learnEnglishScript,  // 节点图剧本对话+ASR跟读（真开口零LLM，收官门）
}

// 脚本课阶段（存 AgentSession.ScriptStage）。
const (
	stageTap    = "tap"    // 已发 Hook，等开场点选
	stageCheck  = "check"  // 已进检验关，等作答
	stageNode   = "node"   // 节点图关卡进行中（当前节点存 ScriptNode）
	stageDone   = "done"   // 本关已收尾，等「顺路下一关/回地图/再来一题」
	stageReview = "review" // 复习挑战：等 Check 作答（答对升「已掌握」）
)

// 收尾/复习导航固定选项（前端点选即原文发回，按文字匹配）。
const (
	optNext       = "顺路下一关"
	optMap        = "回地图挑一关"
	optReviewMore = "再来一题"
	optBackCourse = "回到课程"
)

// scriptedChat 脚本课的一轮。session 的用户消息已入库；这里推进状态机、存回复、组装响应
// （响应形状与 LLM 路径完全一致：answer/options/concept/newlyLit/…，前端零改动）。
func (h *Handler) scriptedChat(c *gin.Context, a *models.ToolAgent, session *models.AgentSession, script map[string]levelScript, req *chatReq, content string) error {
	auth := middleware.Current(c)
	member := membership.IsActive(h.db, auth.UserID)

	st := &scriptTurn{h: h, a: a, script: script, userID: auth.UserID, member: member, remaining: -1}
	if !member {
		// 非会员每轮都回真实试读余额（前端显示「剩 N 次」；-1 只留给会员）。
		// 本轮若开新关，startLevel 里扣减后会覆盖成新值。
		st.remaining = h.trialRemaining(auth.UserID, a)
	}

	// 地图指定关：学员点选了某关 → 无条件切到该关（地图是最高指挥权）。
	// 其余按当前阶段推进；阶段为空（新会话/首次进课）→ 从下一未点亮关开课。
	var err error
	switch {
	case req.Mode == "review" || (session.ScriptStage == stageReview && content != optBackCourse):
		err = st.review(session, content, req.Mode == "review")
	case req.Concept != "" && hasLevel(script, req.Concept):
		err = st.startLevel(session, req.Concept)
	case session.ScriptStage == stageTap:
		st.onTap(session, content)
	case session.ScriptStage == stageCheck:
		st.onCheck(session, content)
	case session.ScriptStage == stageNode:
		st.onNode(session, content)
	case session.ScriptStage == stageDone:
		err = st.onDone(session, content)
	default:
		err = st.startLevel(session, st.nextSlug(""))
	}
	if err != nil {
		return err
	}

	// 剧本文本是平台作者静态内容（上线前整体审），不逐条过 msg_sec_check。
	h.db.Create(&models.AgentMessage{
		ID: idgen.New(), SessionID: session.ID, Role: models.RoleAssistant,
		Content: st.answer, SafeCheckStatus: models.SafePass,
	})
	h.db.Model(session).Updates(map[string]interface{}{
		"script_concept": session.ScriptConcept, "script_stage": session.ScriptStage,
		"script_node": session.ScriptNode,
	})

	resp := gin.H{"sessionId": session.ID, "answer": st.answer, "options": st.options, "member": member, "remaining": st.remaining}
	days, newDay, usedFreeze := bumpStreak(h.db, auth.UserID)
	resp["streak"] = gin.H{"days": days, "newDay": newDay, "freeze": usedFreeze}
	// 技能型脚本课（如开口）：通关即一轮有效开口，三维段位确定性爬升（零 AI）。
	if a.Assess {
		sk := h.loadSkill(auth.UserID, a.ID)
		if st.cleared {
			resp["levelUp"] = h.bumpSkillScripted(sk, st.clearedNote)
		}
		resp["skill"] = skillView(sk)
	}
	if st.progress != nil {
		resp["concept"] = st.progress
		resp["newlyLit"] = st.newlyLit
		resp["newlyMastered"] = st.newlyMastered
		resp["tierCleared"] = st.tierCleared
	}
	httpx.OK(c, resp)
	return nil
}

// scriptTurn 单轮推进的工作区：状态机方法往里写 answer/options/进度增量。
type scriptTurn struct {
	h      *Handler
	a      *models.ToolAgent
	script map[string]levelScript
	userID string
	member bool

	answer      string
	options     []string
	remaining   int    // 非会员本轮后剩余试读关数；-1 = 会员/未扣
	cleared     bool   // 本轮是否通关点亮（Assess 课据此做确定性段位爬升）
	clearedNote string // 通关关卡的战报（作段位 LastNote）

	progress                gin.H
	newlyLit, newlyMastered []string
	tierCleared             []string
}

func hasLevel(script map[string]levelScript, slug string) bool {
	_, ok := script[slug]
	return ok
}

// hookOf/checkOf 从 curatedContent 取该关的精编 Hook/检验题面（剧本不重复存一份）。
func (st *scriptTurn) hookOf(slug string) string  { return curatedContent[st.a.Slug][slug].Hook }
func (st *scriptTurn) checkOf(slug string) string { return curatedContent[st.a.Slug][slug].Check }

// startLevel 开一关：试读闸（非会员开「未点亮」的新关才扣 1 次；重玩已点亮关免费）→
// 发 Hook + 开场点选。slug 为空（全部点亮/越界）→ 引导回地图。
func (st *scriptTurn) startLevel(session *models.AgentSession, slug string) error {
	if slug == "" {
		st.answer = "这一程的关卡你都走过了。点上方「看概念地图」回味任何一章——或者去复习挑战，把点亮的章节升成真正握在手里的。"
		st.options = []string{optReviewMore}
		session.ScriptStage = stageDone
		return nil
	}
	// 准入二选一（与 LLM 路径同构）：FreeTier>0 走「幕门控」（第一幕免费无限、不计次）；
	// 否则走「试读闸」按「开新关」计次（不按消息计——点选流一关几十次点击，按条扣会三点即锁）。
	if st.a.FreeTier > 0 {
		if !st.member {
			if c := st.concept(slug); c != nil && c.Tier > st.a.FreeTier {
				return httpx.BadRequest("MEMBERSHIP_REQUIRED", "第一幕已学完 · 第二幕起开通会员畅用全部")
			}
		}
	} else if !st.member && st.a.FreeTrial > 0 && !st.isLit(slug) {
		_, remaining, err := st.h.checkAccess(st.userID, st.a)
		if err != nil {
			return err
		}
		st.remaining = remaining
	}
	lv := st.script[slug]
	session.ScriptConcept = slug
	session.ScriptNode = 0
	// 节点图关卡（对手戏）：Hook 开场后直接进 Nodes[0]。
	if len(lv.Nodes) > 0 {
		st.answer = st.hookOf(slug)
		st.enterNode(session, lv, "", 0)
		return nil
	}
	st.answer = st.hookOf(slug)
	st.options = labels(lv.Taps)
	session.ScriptStage = stageTap
	return nil
}

// enterNode 进入节点 next：在既有 answer（如 Hook）之后，依次拼上一步回应 reply、
// 本节点场面推进、跟读指令，摆好选项。
func (st *scriptTurn) enterNode(session *models.AgentSession, lv levelScript, reply string, next int) {
	nd := lv.Nodes[next]
	parts := make([]string, 0, 4)
	if st.answer != "" {
		parts = append(parts, st.answer)
	}
	if reply != "" {
		parts = append(parts, reply)
	}
	if nd.Prompt != "" {
		parts = append(parts, nd.Prompt)
	}
	if nd.Say != "" {
		parts = append(parts, "🎙️ 按住麦克风，读出这句：\n"+nd.Say)
	}
	st.answer = strings.Join(parts, "\n\n")
	st.options = nodeLabels(nd.Options)
	session.ScriptStage = stageNode
	session.ScriptNode = next
}

// onNode 对手戏推进：点选节点按选项走剧本（后果演给学员看），跟读节点做 ASR 模糊匹配。
func (st *scriptTurn) onNode(session *models.AgentSession, content string) {
	slug := session.ScriptConcept
	lv := st.script[slug]
	if len(lv.Nodes) == 0 || session.ScriptNode < 0 || session.ScriptNode >= len(lv.Nodes) {
		// 状态漂移（剧本改版等）：温和重进本关，不扣次。
		st.enterNode(session, lv, "（我们把这个场面从头再来一遍）\n\n"+st.hookOf(slug), 0)
		return
	}
	nd := lv.Nodes[session.ScriptNode]
	if nd.Say != "" {
		if matchSay(content, nd.Say) {
			st.leaveNode(session, lv, slug, nd.SayOK, nd.SayNext)
		} else {
			st.answer = nd.SayFail + "\n\n🎙️ 再按住麦克风读一遍：\n" + nd.Say
		}
		return
	}
	idx := matchNodeIndex(nd.Options, content)
	if idx < 0 {
		st.answer = "这一关是对手戏——挑一句你的原话："
		if nd.Prompt != "" {
			st.answer = nd.Prompt + "\n\n" + st.answer
		}
		st.options = nodeLabels(nd.Options)
		return
	}
	opt := nd.Options[idx]
	if opt.Next == NodeRetry {
		st.answer = opt.Reply + "\n\n⏪ 时间倒回，这句重说："
		st.options = nodeLabels(nd.Options)
		return
	}
	st.leaveNode(session, lv, slug, opt.Reply, opt.Next)
}

// leaveNode 按去向离开当前节点：通关点亮或进入下一节点（越界按通关处理，防剧本手误卡死）。
func (st *scriptTurn) leaveNode(session *models.AgentSession, lv levelScript, slug, reply string, next int) {
	if next == NodeClear || next < 0 || next >= len(lv.Nodes) {
		st.clearLevel(session, slug, reply)
		return
	}
	st.enterNode(session, lv, reply, next)
}

// onTap 开场点选：命中 → 预写回应 + 进检验关；未命中（自由输入兜底）→ 温和收回点选。
func (st *scriptTurn) onTap(session *models.AgentSession, content string) {
	slug := session.ScriptConcept
	lv := st.script[slug]
	if opt := matchOption(lv.Taps, content); opt != nil {
		st.answer = opt.Reply + "\n\n🗝️ 检验关：" + st.checkOf(slug)
		st.options = labels(lv.CheckOpts)
		session.ScriptStage = stageCheck
		return
	}
	st.answer = "这句我记下了。这一关咱们用选的走——挑一个最贴你的："
	st.options = labels(lv.Taps)
}

// onCheck 检验关：答对 → 点亮（确定性，只升不降）+ 收尾导航；答错 → 该项的纠偏语 + 重选（不罚）。
func (st *scriptTurn) onCheck(session *models.AgentSession, content string) {
	slug := session.ScriptConcept
	lv := st.script[slug]
	idx := matchIndex(lv.CheckOpts, content)
	if idx < 0 {
		st.answer = "再读一眼题目，选出最贴的那个："
		st.options = labels(lv.CheckOpts)
		return
	}
	if idx != lv.Correct {
		st.answer = lv.CheckOpts[idx].Reply
		st.options = labels(lv.CheckOpts)
		return
	}
	st.clearLevel(session, slug, lv.CheckOpts[idx].Reply)
}

// clearLevel 通关点亮 + 收尾导航（两段式检验关与节点图共用）。
func (st *scriptTurn) clearLevel(session *models.AgentSession, slug, reply string) {
	lv := st.script[slug]
	c := st.concept(slug)
	name := slug
	if c != nil {
		name = c.Name
		st.light(c, 1, lv.Note)
	}
	st.answer = reply + "\n\n✨ 「" + name + "」点亮。" + lv.Clear
	if st.nextSlug(slug) != "" {
		st.options = []string{optNext, optMap}
	} else {
		st.options = []string{optMap, optReviewMore} // 全课终关：只剩回味与复习
	}
	st.cleared = true
	st.clearedNote = lv.Note
	session.ScriptStage = stageDone
}

// onDone 关卡收尾导航。未命中导航词 → 重发导航（不猜意图）。
func (st *scriptTurn) onDone(session *models.AgentSession, content string) error {
	switch content {
	case optNext:
		return st.startLevel(session, st.nextSlug(session.ScriptConcept))
	case optMap:
		st.answer = "好。点上方「看概念地图」，挑你想读的一章，点开即开课。想顺着走也随时回来。"
		st.options = []string{optNext}
		return nil
	case optReviewMore:
		return st.review(session, content, true)
	case optBackCourse:
		return st.startLevel(session, st.nextSlug(session.ScriptConcept))
	}
	st.answer = "接着怎么走？"
	st.options = []string{optNext, optMap}
	return nil
}

// review 复习挑战（检索练习）：抽最该复习的已点亮章节，用检验关快问快答；
// 答对升「已掌握」。fresh=true 表示新抽一题（进入复习 / 再来一题）。
func (st *scriptTurn) review(session *models.AgentSession, content string, fresh bool) error {
	if !fresh && session.ScriptStage == stageReview {
		slug := session.ScriptConcept
		lv := st.script[slug]
		idx := matchIndex(lv.CheckOpts, content)
		switch {
		case idx < 0:
			st.answer = "选出最贴的那个："
			st.options = labels(lv.CheckOpts)
			return nil
		case idx != lv.Correct:
			st.answer = lv.CheckOpts[idx].Reply
			st.options = labels(lv.CheckOpts)
			return nil
		}
		c := st.concept(slug)
		name := slug
		if c != nil {
			name = c.Name
			st.light(c, 2, "") // 复习答对 → 已掌握；空 note 不覆盖旧战报
		}
		st.answer = lv.CheckOpts[idx].Reply + "\n\n✅ 复习通过，「" + name + "」升到已掌握。"
		st.options = []string{optReviewMore, optBackCourse}
		session.ScriptStage = stageDone
		return nil
	}
	// 抽题：到期优先，无到期抽最生疏；一个点亮的都没有 → 引导先闯关。
	picks := reviewPick(st.h.db, st.userID, st.a.ID, 1, true)
	if len(picks) == 0 {
		picks = reviewPick(st.h.db, st.userID, st.a.ID, 1, false)
	}
	if len(picks) == 0 {
		st.answer = "还没有点亮的章节可复习——先闯一关再来。"
		st.options = []string{optNext}
		session.ScriptStage = stageDone
		return nil
	}
	c := picks[0]
	if !hasLevel(st.script, c.Slug) { // 课程表与剧本理论上同集；防御孤儿
		st.answer = "先闯一关再来。"
		st.options = []string{optNext}
		session.ScriptStage = stageDone
		return nil
	}
	st.answer = "🔁 复习挑战 · 「" + c.Name + "」\n\n" + st.checkOf(c.Slug)
	st.options = labels(st.script[c.Slug].CheckOpts)
	session.ScriptConcept = c.Slug
	session.ScriptStage = stageReview
	return nil
}

// ---------- 小工具 ----------

// trialRemaining 只读非会员的试读余额；尚未发放（无权益行）按满额算（与 card() 口径一致）。
func (h *Handler) trialRemaining(userID string, a *models.ToolAgent) int {
	var ent models.AgentEntitlement
	if h.db.First(&ent, "user_id = ? AND agent_id = ?", userID, a.ID).Error != nil {
		return a.FreeTrial
	}
	return ent.Remaining
}

func (st *scriptTurn) concept(slug string) *models.AgentConcept {
	var c models.AgentConcept
	if st.h.db.First(&c, "agent_id = ? AND slug = ?", st.a.ID, slug).Error != nil {
		return nil
	}
	return &c
}

func (st *scriptTurn) isLit(slug string) bool {
	c := st.concept(slug)
	if c == nil {
		return false
	}
	var uc models.UserConcept
	if st.h.db.First(&uc, "user_id = ? AND concept_id = ?", st.userID, c.ID).Error == gorm.ErrRecordNotFound {
		return false
	}
	return uc.Level >= 1
}

// nextSlug 课程表顺序里 after 之后的第一个未点亮关（after 空 = 从头找）。
// 顺读课天然 = 下一章；全点亮返回 ""。
func (st *scriptTurn) nextSlug(after string) string {
	concepts := st.h.conceptList(st.a.ID)
	levels, _ := st.h.userConceptLevels(st.userID, st.a.ID)
	from := 0
	if after != "" {
		for i := range concepts {
			if concepts[i].Slug == after {
				from = i + 1
				break
			}
		}
	}
	for i := from; i < len(concepts); i++ {
		if levels[concepts[i].ID] < 1 {
			return concepts[i].Slug
		}
	}
	return ""
}

// light 确定性点亮/升档（复用判定器的落库与视图管线，但不经 LLM）：
// bumpConcept 只升不降；随后重算进度视图与「新点亮/新掌握/幕打通」增量。
func (st *scriptTurn) light(c *models.AgentConcept, target int, note string) {
	concepts := st.h.conceptList(st.a.ID)
	levels, notes := st.h.userConceptLevels(st.userID, st.a.ID)
	preCleared := tierClearedSet(concepts, levels)

	old := levels[c.ID]
	if target >= 2 && old < 2 {
		st.newlyMastered = append(st.newlyMastered, c.Name)
	} else if target >= 1 && old < 1 {
		st.newlyLit = append(st.newlyLit, c.Name)
	}
	st.h.bumpConcept(st.userID, st.a.ID, c.ID, target, note)
	if target > old {
		levels[c.ID] = target
	}
	if note != "" {
		notes[c.ID] = note
	}

	tLabels, tOrder := tiersFor(st.a.Slug)
	postCleared := tierClearedSet(concepts, levels)
	for _, t := range tOrder {
		if postCleared[t] && !preCleared[t] {
			st.tierCleared = append(st.tierCleared, tLabels[t])
		}
	}
	st.progress = conceptProgressView(st.a.Slug, concepts, levels, notes)
}

func labels(opts []tapOption) []string {
	out := make([]string, len(opts))
	for i := range opts {
		out[i] = opts[i].Label
	}
	return out
}

func matchOption(opts []tapOption, content string) *tapOption {
	if i := matchIndex(opts, content); i >= 0 {
		return &opts[i]
	}
	return nil
}

func matchIndex(opts []tapOption, content string) int {
	c := strings.TrimSpace(content)
	for i := range opts {
		if strings.TrimSpace(opts[i].Label) == c {
			return i
		}
	}
	return -1
}

func nodeLabels(opts []nodeOption) []string {
	out := make([]string, len(opts))
	for i := range opts {
		out[i] = opts[i].Label
	}
	return out
}

func matchNodeIndex(opts []nodeOption, content string) int {
	c := strings.TrimSpace(content)
	for i := range opts {
		if strings.TrimSpace(opts[i].Label) == c {
			return i
		}
	}
	return -1
}

// ---------- 跟读匹配（开口课 ASR）----------

// matchSay 判定学员的语音转写是否读出了目标句：两边归一化（小写、只留字母数字与汉字）后，
// 互为包含即命中；否则按编辑距离折算相似度 ≥ 0.72 算命中——ASR 难免错字，卡太严会劝退开口。
func matchSay(content, target string) bool {
	a, b := normSay(content), normSay(target)
	if a == "" || b == "" {
		return false
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}
	ra, rb := []rune(a), []rune(b)
	m := len(ra)
	if len(rb) > m {
		m = len(rb)
	}
	return float64(m-levenshtein(ra, rb))/float64(m) >= 0.72
}

// normSay 归一化转写文本：小写化，丢掉标点、空白与语气符号，只留字母/数字/汉字。
func normSay(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// levenshtein 标准编辑距离（rune 级，滚动数组）。
func levenshtein(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
