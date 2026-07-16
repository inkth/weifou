package toolagent

import (
	"strings"
	"testing"
)

// 六门完备课的护城河检查：每个概念都必须有精编 Hook+Check，且精编条目不得指向不存在的概念（防 slug 手误）。
func TestCuratedContentComplete(t *testing.T) {
	for agentSlug, list := range curricula {
		content, ok := curatedContent[agentSlug]
		if !ok {
			t.Fatalf("%s 缺少精编内容表", agentSlug)
		}
		seen := make(map[string]bool, len(list))
		for _, c := range list {
			seen[c.Slug] = true
			hc, ok := content[c.Slug]
			if !ok {
				t.Errorf("%s/%s 缺少精编条目", agentSlug, c.Slug)
				continue
			}
			if hc.Hook == "" || hc.Check == "" {
				t.Errorf("%s/%s 的 Hook/Check 不完整", agentSlug, c.Slug)
			}
		}
		for slug := range content {
			if !seen[slug] {
				t.Errorf("%s 精编条目 %s 不在课程表里（slug 手误？）", agentSlug, slug)
			}
		}
	}
}

// isBossConcept 判定标准与前端 utils/learn-nodes.js 的 isBoss() 完全同构：
// slug 以 "boss-" 开头，或 Name 含 "综合关"/"Boss"（涵盖英语课的 slug 前缀写法、
// 逻辑课的 "Boss·xxx" 命名、其余课程的 "综合关·xxx" 命名、道德经无连字符的 "boss1".."boss9"）。
func isBossConcept(c seedConcept) bool {
	return strings.HasPrefix(c.Slug, "boss-") || strings.Contains(c.Name, "综合关") || strings.Contains(c.Name, "Boss")
}

// 章末知识卡片护城河：每个 Boss/综合关都必须配一句策展 Takeaway + Source（learn-marketing 50 关平铺无
// 章节边界，不参与，跳过）；bossCardContent 里也不能有指向不存在概念的多余条目（防 slug 手误）。
func TestBossCardsComplete(t *testing.T) {
	seen := make(map[string]bool, len(bossCardContent))
	total := 0
	for agentSlug, list := range curricula {
		if agentSlug == "learn-marketing" {
			continue
		}
		for _, c := range list {
			if !isBossConcept(c) {
				continue
			}
			total++
			seen[c.Slug] = true
			bc, ok := bossCardContent[c.Slug]
			if !ok {
				t.Errorf("%s/%s 是章末 Boss 关，但缺少知识卡片", agentSlug, c.Slug)
				continue
			}
			tw := []rune(bc.Takeaway)
			if len(tw) < 10 || len(tw) > 40 {
				t.Errorf("%s/%s 的 Takeaway 长度 %d 超出合理范围（10-40字）：%q", agentSlug, c.Slug, len(tw), bc.Takeaway)
			}
			if src := []rune(bc.Source); len(src) == 0 || len(src) > 20 {
				t.Errorf("%s/%s 的 Source 长度 %d 超出合理范围（1-20字）：%q", agentSlug, c.Slug, len(src), bc.Source)
			}
		}
	}
	for slug := range bossCardContent {
		if !seen[slug] {
			t.Errorf("bossCardContent 条目 %s 不对应任何课程的 Boss 关（slug 手误？）", slug)
		}
	}
	if total != len(bossCardContent) {
		t.Errorf("Boss 关总数 %d 与 bossCardContent 条目数 %d 不一致", total, len(bossCardContent))
	}
}

// 档位守护：每个概念的 Tier 必须在该课程的档位表里有标签（防 tier 手误 / 新增幕漏配幕名）。
func TestTiersLabeled(t *testing.T) {
	for agentSlug, list := range curricula {
		labels, _ := tiersFor(agentSlug)
		for _, c := range list {
			if labels[c.Tier] == "" {
				t.Errorf("%s/%s 的 Tier %d 没有档位标签", agentSlug, c.Slug, c.Tier)
			}
		}
	}
}

// 三幕课结构守护（人生设计/好好相爱）：三幕各 7 关（6 常规 + 1 综合关 Boss），综合关必须是每幕末关。
func TestThreeActCoursesStructure(t *testing.T) {
	for name, list := range map[string][]seedConcept{
		"人生设计":   curricula["learn-lifedesign"],
		"好好相爱":   curricula["learn-love"],
		"把幸福练出来": curricula["learn-happiness"],
		"让文字办事":  curricula["learn-writing"],
		"学什么都快":  curricula["learn-learning"],
		"争取更多":   curricula["learn-negotiation"],
		"习惯的复利":  curricula["learn-habits"],
		"看懂生意":   curricula["learn-business"],
		"把心练稳":   curricula["learn-meditation"],
	} {
		if len(list) != 21 {
			t.Fatalf("%s 应为 21 关，实际 %d", name, len(list))
		}
		counts := map[int]int{}
		lastOfTier := map[int]seedConcept{}
		for _, c := range list {
			counts[c.Tier]++
			lastOfTier[c.Tier] = c
		}
		for tier := 1; tier <= 3; tier++ {
			if counts[tier] != 7 {
				t.Errorf("%s 第 %d 幕应为 7 关，实际 %d", name, tier, counts[tier])
			}
			if !strings.Contains(lastOfTier[tier].Name, "综合关") {
				t.Errorf("%s 第 %d 幕末关应为综合关，实际 %s", name, tier, lastOfTier[tier].Name)
			}
		}
	}
}

// 约会课结构守护：四幕各 7 关（6 常规 + 1 综合关 Boss），综合关必须是每幕末关。
func TestDatingCourseFourActsOfSeven(t *testing.T) {
	datingConcepts := curricula["learn-dating"]
	if len(datingConcepts) != 28 {
		t.Fatalf("清醒去爱应为 28 关，实际 %d", len(datingConcepts))
	}
	counts := map[int]int{}
	lastOfTier := map[int]seedConcept{}
	for _, c := range datingConcepts {
		counts[c.Tier]++
		lastOfTier[c.Tier] = c
	}
	for tier := 1; tier <= 4; tier++ {
		if counts[tier] != 7 {
			t.Errorf("清醒去爱第 %d 幕应为 7 关，实际 %d", tier, counts[tier])
		}
		if !strings.Contains(lastOfTier[tier].Name, "综合关") {
			t.Errorf("清醒去爱第 %d 幕末关应为综合关，实际 %s", tier, lastOfTier[tier].Name)
		}
	}
}

// 英语课结构守护：四幕各 8 关（7 常规 + 1 全英模拟面 Boss），模拟面必须是每幕末关。
func TestEnglishCourseHasFourActsOfEight(t *testing.T) {
	englishScenarios := curricula["spoken-english"]
	if len(englishScenarios) != 32 {
		t.Fatalf("英语反应力应为 32 关，实际 %d", len(englishScenarios))
	}
	counts := map[int]int{}
	lastOfTier := map[int]seedConcept{}
	for _, c := range englishScenarios {
		counts[c.Tier]++
		lastOfTier[c.Tier] = c
	}
	for tier := 1; tier <= 4; tier++ {
		if counts[tier] != 8 {
			t.Errorf("英语第 %d 幕应为 8 关，实际 %d", tier, counts[tier])
		}
		if !strings.HasPrefix(lastOfTier[tier].Slug, "boss-") {
			t.Errorf("英语第 %d 幕末关应为全英模拟面（slug 前缀 boss-），实际 %s", tier, lastOfTier[tier].Slug)
		}
	}
}
