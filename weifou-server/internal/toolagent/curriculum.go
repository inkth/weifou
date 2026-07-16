// Package toolagent — curriculum.go：概念型学习 Agent 的「核心概念」课程表与点亮引擎。
//
// 六门完备课（英语/心理/营销/逻辑/会用AI/会说话）+ 一门会员隐藏课（道德经）各装载一份 curated 的核心概念/关卡课程表，
// 且全部人工精编 Hook+Check（人工策展骨架，不让模型现编 = 这条线的护城河，由 TestCuratedContentComplete 守护）。
// 分档给成就感（完成一档就庆祝），避免一条 X/100 的大分母进度条把人劝退：
// 英语/会用AI/会说话 分「入门 / 进阶」两档；逻辑课 2026-07-04 重构为六幕能力阶梯，2026-07-12 四幕补图表与信源扩至 68 关（档位标签按课程覆盖，
// 见 tiersFor；每幕结尾一个 Boss 综合找茬关）；心理课同日重构为三幕旅程（入门=看清自己 /
// 进阶=经营自己与关系 / 大师=看穿世界），章=Theme（按人生问题命名，不按教科书分类），
// 每章末一关是「综合关」Boss——用本章多个概念对真实情境做一次完整分析（对应英语课的全英模拟面）；
// 营销课同日重构为三档生意人旅程（入门=把生意想明白 / 进阶=把客人请进门 / 大师=让生意自己转），
// 2026-07-12 补「客户冷热」「报价设计」两个统帅关成 50 关。
// 用户在对话中「点亮」概念：每轮一问一答交给 DeepSeek 判定本轮真实涉及了哪些概念、是否已展现真正理解，
// 把概念档位往上推（0 未触及 / 1 已点亮 / 2 已掌握，只升不降）。整体照搬 skill.go 的范式。
package toolagent

import (
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/idgen"
	"weifou-server/internal/models"
)

// seedConcept 是课程表的一行（seed 用）。Slug 在单个 Agent 内稳定唯一。Tier：1 入门 / 2 进阶 / 3 大师。
// JSON 标签供课程内容外置（content/courses/*.json）序列化。
type seedConcept struct {
	Slug  string `json:"slug"`
	Theme string `json:"theme"`
	Name  string `json:"name"`
	Blurb string `json:"blurb"`
	Tier  int    `json:"tier"`
}

// hookCheck 人工精编内容（开课钩子 + 检验题）。与 seedConcept 分开存，SeedConcepts 时按 slug 合并写入。
type hookCheck struct {
	Hook  string `json:"hook"`
	Check string `json:"check"`
}

// bossCard 章末知识卡片：一句策展结论 + 引用来源。只给「综合关/Boss关」配（章的收尾），
// 按 concept slug 直接键入（不分课程），SeedConcepts 里零值安全地按 slug 查表写入。
// learn-marketing 50 关平铺无章节边界，本课不参与，通关沿用通用庆祝浮层。
type bossCard struct {
	Takeaway string `json:"takeaway"`
	Source   string `json:"source"`
}

// SeedConcepts 把各份课程表幂等写入 agent_concepts（按 agent_id+slug），并清掉已不在清单里的旧概念
// （让完成分母始终等于当前课程表；slug 不变的概念原地更新、用户进度无损）。须在 Seed(写入 ToolAgent) 之后调用。
func SeedConcepts(db *gorm.DB) {
	if db == nil {
		return
	}
	for agentSlug, list := range curricula {
		var agent models.ToolAgent
		if db.Where("slug = ?", agentSlug).First(&agent).Error != nil {
			continue // Agent 尚未 seed（理论上 Seed 先跑）
		}
		content := curatedContent[agentSlug] // 可能为 nil：未精编课程返回零值 hookCheck
		slugs := make([]string, 0, len(list))
		for i := range list {
			sc := list[i]
			hc := content[sc.Slug]
			bc := bossCardContent[sc.Slug] // 非 Boss 关零值安全，Takeaway/Source 为空
			slugs = append(slugs, sc.Slug)
			var existing models.AgentConcept
			if db.Where("agent_id = ? AND slug = ?", agent.ID, sc.Slug).First(&existing).Error == gorm.ErrRecordNotFound {
				db.Create(&models.AgentConcept{
					ID: idgen.New(), AgentID: agent.ID, Slug: sc.Slug,
					Theme: sc.Theme, Tier: sc.Tier, Name: sc.Name, Blurb: sc.Blurb,
					Hook: hc.Hook, Check: hc.Check, Takeaway: bc.Takeaway, Source: bc.Source, Sort: i,
				})
				continue
			}
			db.Model(&existing).Updates(map[string]interface{}{
				"theme": sc.Theme, "tier": sc.Tier, "name": sc.Name, "blurb": sc.Blurb,
				"hook": hc.Hook, "check": hc.Check, "takeaway": bc.Takeaway, "source": bc.Source, "sort": i,
			})
		}
		// 清除不在当前清单的旧概念（user_concepts 里的孤儿记录无害，进度视图只遍历现有概念）。
		db.Where("agent_id = ? AND slug NOT IN ?", agent.ID, slugs).Delete(&models.AgentConcept{})
	}
}

// ============================ 复习引擎（检索练习 + 间隔重复） ============================
// 点亮≠记住：隔期抽查（retrieval practice）才是把概念钉进长期记忆的机制。
// 2026-07-16 升级为扩展式间隔（Anki 精神、比 SM-2 简化）：间隔档 3→7→14→30→60 天，
// 复习答对升档、答错回到次日（ReviewCount 清零；Level 依旧只升不降——日程可以退，成就不退）。
// 课内重玩/再命中同样算一次成功检索：按当前档重新起表（bumpConcept 内）。
// 旧数据（ReviewDue 零值）按旧规则兜底：已点亮 3 天、已掌握 7 天未碰 → 到期。

// reviewIntervals 复习间隔档（天）。ReviewCount 索引，越答越稀。
var reviewIntervals = []int{3, 7, 14, 30, 60}

const (
	reviewDueLitDays      = 3 // 旧数据兜底窗口
	reviewDueMasteredDays = 7
	reviewFailDelayHours  = 24 // 复习答错：次日重来
)

// reviewIntervalDays 当前档的间隔天数。英语课首档 1 天（新场景次日先复习）。
func reviewIntervalDays(slug string, count int) int {
	if count <= 0 {
		if slug == "spoken-english" {
			return 1
		}
		return reviewIntervals[0]
	}
	if count >= len(reviewIntervals) {
		return reviewIntervals[len(reviewIntervals)-1]
	}
	return reviewIntervals[count]
}

// dueBy 该档位是否到期：新数据看 ReviewDue，旧数据（零值）按 3/7 天兜底。
func dueBy(uc *models.UserConcept, now time.Time, litDays, masteredDays int) bool {
	if !uc.ReviewDue.IsZero() {
		return !now.Before(uc.ReviewDue)
	}
	days := litDays
	if uc.Level >= 2 {
		days = masteredDays
	}
	return now.Sub(uc.UpdatedAt) >= time.Duration(days)*24*time.Hour
}

// reviewPick 挑该用户在该 Agent 下待复习的概念，最久未碰的在前。
// onlyDue=true 只要到期的；false 则不设窗口（用户主动要复习时，抽最生疏的也行）。
// limit<=0 表示不限量（用于数徽章）。
func reviewPick(db *gorm.DB, userID, agentID string, limit int, onlyDue bool) []models.AgentConcept {
	var ucs []models.UserConcept
	db.Where("user_id = ? AND agent_id = ? AND level >= 1", userID, agentID).
		Order("updated_at asc").Find(&ucs)
	if len(ucs) == 0 {
		return nil
	}
	now := time.Now()
	litDays, masteredDays := reviewDueLitDays, reviewDueMasteredDays
	var agent models.ToolAgent
	if db.Select("slug").First(&agent, "id = ?", agentID).Error == nil && agent.Slug == "spoken-english" {
		litDays = 1 // 兜底规则的英语特例（新数据由 reviewIntervalDays 管）。
	}
	var ids []string
	for i := range ucs {
		if onlyDue && !dueBy(&ucs[i], now, litDays, masteredDays) {
			continue
		}
		ids = append(ids, ucs[i].ConceptID)
		if limit > 0 && len(ids) >= limit {
			break
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var cs []models.AgentConcept
	db.Where("id IN ?", ids).Find(&cs)
	byID := make(map[string]models.AgentConcept, len(cs))
	for i := range cs {
		byID[cs[i].ID] = cs[i]
	}
	out := make([]models.AgentConcept, 0, len(ids))
	for _, id := range ids {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		}
	}
	return out
}

// dueCount 到期待复习的概念数（首页催课条 / 对话页复习徽章共用）。
func dueCount(db *gorm.DB, userID, agentID string) int {
	return len(reviewPick(db, userID, agentID, 0, true))
}

// ============================ 点亮引擎 ============================

// conceptList 取该 Agent 的课程表（按 sort）。
func (h *Handler) conceptList(agentID string) []models.AgentConcept {
	var cs []models.AgentConcept
	h.db.Where("agent_id = ?", agentID).Order("sort asc").Find(&cs)
	return cs
}

// userConceptLevels 取用户在该 Agent 各概念上的档位 map[conceptID]level 与战报 map[conceptID]note（只装非空）。
func (h *Handler) userConceptLevels(userID, agentID string) (map[string]int, map[string]string) {
	var ucs []models.UserConcept
	h.db.Where("user_id = ? AND agent_id = ?", userID, agentID).Find(&ucs)
	m := make(map[string]int, len(ucs))
	notes := make(map[string]string, len(ucs))
	for i := range ucs {
		m[ucs[i].ConceptID] = ucs[i].Level
		if ucs[i].Note != "" {
			notes[ucs[i].ConceptID] = ucs[i].Note
		}
	}
	return m, notes
}

// conceptProgressView 序列化进度给前端：总 total/lit/mastered + 按档（默认三档或逻辑六幕）分组的概念与各档进度。
// notes 为 conceptID→本课战报（nil 安全），随 item 下发给卡片流。
func conceptProgressView(agentSlug string, concepts []models.AgentConcept, levels map[string]int, notes map[string]string) gin.H {
	labels, order := tiersFor(agentSlug)
	lit, mastered := 0, 0
	type agg struct {
		total, lit, mastered int
		items                []gin.H
	}
	byTier := make(map[int]*agg, len(order))
	for i := range concepts {
		c := concepts[i]
		lv := levels[c.ID]
		if lv >= 1 {
			lit++
		}
		if lv >= 2 {
			mastered++
		}
		a := byTier[c.Tier]
		if a == nil {
			a = &agg{}
			byTier[c.Tier] = a
		}
		a.total++
		if lv >= 1 {
			a.lit++
		}
		if lv >= 2 {
			a.mastered++
		}
		a.items = append(a.items, gin.H{"slug": c.Slug, "name": c.Name, "blurb": c.Blurb, "level": lv, "theme": c.Theme, "hook": c.Hook, "note": notes[c.ID], "takeaway": c.Takeaway, "source": c.Source})
	}
	tiers := make([]gin.H, 0, len(order))
	for _, t := range order {
		a := byTier[t]
		if a == nil {
			continue
		}
		tiers = append(tiers, gin.H{
			"tier": labels[t], "tierNum": t, "total": a.total, "lit": a.lit, "mastered": a.mastered,
			"concepts": a.items,
		})
	}
	return gin.H{"total": len(concepts), "lit": lit, "mastered": mastered, "tiers": tiers}
}

// loadConceptProgress 给 GET /agents/concepts/:id 用：当前进度视图。
func (h *Handler) loadConceptProgress(userID string, a *models.ToolAgent) gin.H {
	levels, notes := h.userConceptLevels(userID, a.ID)
	return conceptProgressView(a.Slug, h.conceptList(a.ID), levels, notes)
}

// loadKnowledgeCards 聚合该用户跨全部课程的章末知识卡片（GET /agents/knowledge-cards：我的→我的卡片）。
// 一次查询取全部 Boss 关（Takeaway 非空，跨 agent），按课程分组；未解锁（level<2）的卡片隐去
// Takeaway/Source 只留剪影——不通关不能白嫖结论，否则削弱「通关才揭晓」的游戏动机。
func (h *Handler) loadKnowledgeCards(userID string) gin.H {
	var concepts []models.AgentConcept
	h.db.Where("takeaway <> ''").Order("agent_id, sort").Find(&concepts)
	if len(concepts) == 0 {
		return gin.H{"totalCards": 0, "unlockedCount": 0, "courses": []gin.H{}}
	}

	agentIDs := make([]string, 0, 16)
	seenAgent := make(map[string]bool, 16)
	conceptIDs := make([]string, 0, len(concepts))
	for i := range concepts {
		conceptIDs = append(conceptIDs, concepts[i].ID)
		if !seenAgent[concepts[i].AgentID] {
			seenAgent[concepts[i].AgentID] = true
			agentIDs = append(agentIDs, concepts[i].AgentID)
		}
	}

	var agents []models.ToolAgent
	h.db.Where("id IN ?", agentIDs).Find(&agents)
	agentByID := make(map[string]models.ToolAgent, len(agents))
	for i := range agents {
		agentByID[agents[i].ID] = agents[i]
	}

	var ucs []models.UserConcept
	h.db.Where("user_id = ? AND concept_id IN ?", userID, conceptIDs).Find(&ucs)
	levels := make(map[string]int, len(ucs))
	for i := range ucs {
		levels[ucs[i].ConceptID] = ucs[i].Level
	}

	type courseAgg struct {
		agent           models.ToolAgent
		cards           []gin.H
		unlocked, total int
	}
	byAgent := make(map[string]*courseAgg, len(agents))
	order := make([]string, 0, len(agents))
	total, unlockedCount := 0, 0
	for i := range concepts {
		cpt := concepts[i]
		a, ok := agentByID[cpt.AgentID]
		if !ok {
			continue // Agent 已下架，跳过（不展示已下架课程的卡）
		}
		agg := byAgent[cpt.AgentID]
		if agg == nil {
			agg = &courseAgg{agent: a}
			byAgent[cpt.AgentID] = agg
			order = append(order, cpt.AgentID)
		}
		labels, _ := tiersFor(a.Slug)
		unlocked := levels[cpt.ID] >= 2
		total++
		agg.total++
		card := gin.H{"slug": cpt.Slug, "name": cpt.Name, "tier": cpt.Tier, "tierLabel": labels[cpt.Tier], "unlocked": unlocked, "takeaway": "", "source": ""}
		if unlocked {
			unlockedCount++
			agg.unlocked++
			card["takeaway"] = cpt.Takeaway
			card["source"] = cpt.Source
		}
		agg.cards = append(agg.cards, card)
	}
	sort.Slice(order, func(i, j int) bool { return byAgent[order[i]].agent.Sort < byAgent[order[j]].agent.Sort })

	courses := make([]gin.H, 0, len(order))
	for _, id := range order {
		agg := byAgent[id]
		courses = append(courses, gin.H{
			"agentSlug": agg.agent.Slug, "agentId": agg.agent.ID, "name": agg.agent.Name,
			"icon": agg.agent.Icon, "accent": agg.agent.Accent,
			"unlocked": agg.unlocked, "total": agg.total, "cards": agg.cards,
		})
	}
	return gin.H{"totalCards": total, "unlockedCount": unlockedCount, "courses": courses}
}

// tierClearedSet 返回「已点满（lit==total）」的档位集合，用于检测本轮新打通了哪档。
func tierClearedSet(concepts []models.AgentConcept, levels map[string]int) map[int]bool {
	tot := map[int]int{}
	lit := map[int]int{}
	for i := range concepts {
		c := concepts[i]
		tot[c.Tier]++
		if levels[c.ID] >= 1 {
			lit[c.Tier]++
		}
	}
	res := map[int]bool{}
	for t, n := range tot {
		if n > 0 && lit[t] == n {
			res[t] = true
		}
	}
	return res
}

// bumpConcept 命中某概念：touches+1，档位 level 只升不降（upsert）。
// note 非空时写入战报（latest-wins）；空 note 不覆盖旧战报。
func (h *Handler) bumpConcept(userID, agentID, slug, conceptID string, level int, note string) {
	now := time.Now()
	var uc models.UserConcept
	if err := h.db.First(&uc, "user_id = ? AND concept_id = ?", userID, conceptID).Error; err == gorm.ErrRecordNotFound {
		uc = models.UserConcept{
			ID: idgen.New(), UserID: userID, AgentID: agentID, ConceptID: conceptID,
			Level: level, Touches: 1, Note: note,
			ReviewDue: now.Add(time.Duration(reviewIntervalDays(slug, 0)) * 24 * time.Hour),
		}
		if cerr := h.db.Create(&uc).Error; cerr == nil {
			return
		}
		h.db.First(&uc, "user_id = ? AND concept_id = ?", userID, conceptID) // 并发：他人先建 → 重查走更新
	}
	updates := map[string]interface{}{"touches": gorm.Expr("touches + 1")}
	if level > uc.Level {
		updates["level"] = level
	}
	if note != "" {
		updates["note"] = note
	}
	// 课内再命中（重玩/顺路聊到）= 一次成功检索：按当前间隔档重新起表，不升档。
	updates["review_due"] = now.Add(time.Duration(reviewIntervalDays(slug, uc.ReviewCount)) * 24 * time.Hour)
	h.db.Model(&uc).Updates(updates)
}

// reviewMark 复习挑战的调度回写：答对升档（间隔翻倍扩展），答错清档、次日重来。
// 只动日程（ReviewCount/ReviewDue），不动 Level——「只升不降」的成就纪律不被复习打破。
func (h *Handler) reviewMark(userID, conceptID, slug string, success bool) {
	var uc models.UserConcept
	if h.db.First(&uc, "user_id = ? AND concept_id = ?", userID, conceptID).Error != nil {
		return
	}
	now := time.Now()
	count := 0
	if success {
		count = uc.ReviewCount + 1
	}
	due := now.Add(time.Duration(reviewIntervalDays(slug, count)) * 24 * time.Hour)
	if !success {
		due = now.Add(reviewFailDelayHours * time.Hour)
	}
	h.db.Model(&uc).Updates(map[string]interface{}{"review_count": count, "review_due": due})
}

// ============================ 首页催课条（L2 再入口主动） ============================

// NudgeLine 给「钉在首页」的学习 Agent 一句动态状态，替代静态 tagline——
// 用户还没点进来，主动性已经发生。返回 ("", false) 表示无状态可催（保持 tagline）。
// 措辞纪律：不做「今天」类日期判断，句子本身永远可行动；没开始学的人不催（tagline 即邀请）。
func NudgeLine(db *gorm.DB, a *models.ToolAgent, userID string) (string, bool) {
	if db == nil || a == nil || userID == "" {
		return "", false
	}
	// streak 火焰前缀：昨天学了、今天还没学 → 「别断」是最强的一句催课
	prefix := ""
	if days, risk := streakAtRisk(db, userID); risk {
		prefix = "🔥 连学 " + strconv.Itoa(days) + " 天，别断 · "
	}
	if a.Concept {
		var concepts []models.AgentConcept
		db.Where("agent_id = ?", a.ID).Order("sort asc").Find(&concepts)
		if len(concepts) == 0 {
			return "", false
		}
		var ucs []models.UserConcept
		db.Where("user_id = ? AND agent_id = ?", userID, a.ID).Find(&ucs)
		levels := make(map[string]int, len(ucs))
		for i := range ucs {
			levels[ucs[i].ConceptID] = ucs[i].Level
		}
		lit, mastered := 0, 0
		litByTier, totByTier := map[int]int{}, map[int]int{}
		var next *models.AgentConcept
		for i := range concepts {
			c := &concepts[i]
			totByTier[c.Tier]++
			if levels[c.ID] >= 1 {
				lit++
				litByTier[c.Tier]++
			} else if next == nil {
				next = c
			}
			if levels[c.ID] >= 2 {
				mastered++
			}
		}
		if lit == 0 {
			return "", false // 还没开始学：tagline 本身就是邀请，不催
		}
		if n := dueCount(db, userID, a.ID); n > 0 {
			return "🔁 " + strconv.Itoa(n) + " 个概念好几天没碰了，快问快答保住它们", true
		}
		if next != nil {
			nLabels, _ := tiersFor(a.Slug)
			return prefix + "下一个待点亮：『" + next.Name + "』 · " + nLabels[next.Tier] + " " +
				strconv.Itoa(litByTier[next.Tier]) + "/" + strconv.Itoa(totByTier[next.Tier]), true
		}
		if mastered < len(concepts) {
			return prefix + "整张地图已点亮 · 还剩 " + strconv.Itoa(len(concepts)-mastered) + " 个概念待「掌握」", true
		}
		return "整张地图已通关 🎉 随时来聊聊新的困惑", true
	}
	return "", false
}
