package toolagent

import "testing"

// 四门完备课的护城河检查：每个概念都必须有精编 Hook+Check，且精编条目不得指向不存在的概念（防 slug 手误）。
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
