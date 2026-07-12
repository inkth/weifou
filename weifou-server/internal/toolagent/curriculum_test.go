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

// 人生设计课结构守护：三幕各 7 关（6 常规 + 1 综合关 Boss），综合关必须是每幕末关。
func TestLifedesignCourseHasThreeActsOfSeven(t *testing.T) {
	if len(lifedesignConcepts) != 21 {
		t.Fatalf("人生设计应为 21 关，实际 %d", len(lifedesignConcepts))
	}
	counts := map[int]int{}
	lastOfTier := map[int]seedConcept{}
	for _, c := range lifedesignConcepts {
		counts[c.Tier]++
		lastOfTier[c.Tier] = c
	}
	for tier := 1; tier <= 3; tier++ {
		if counts[tier] != 7 {
			t.Errorf("人生设计第 %d 幕应为 7 关，实际 %d", tier, counts[tier])
		}
		if !strings.Contains(lastOfTier[tier].Name, "综合关") {
			t.Errorf("人生设计第 %d 幕末关应为综合关，实际 %s", tier, lastOfTier[tier].Name)
		}
	}
}

// 英语课结构守护：四幕各 8 关（7 常规 + 1 全英模拟面 Boss），模拟面必须是每幕末关。
func TestEnglishCourseHasFourActsOfEight(t *testing.T) {
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
