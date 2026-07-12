package toolagent

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 脚本课护城河检查：有剧本的课程，其课程表每一关都必须有完备剧本——
// 开场点选 2-4 项、检验选项 2-4 项且 Correct 下标合法、每个选项 Label/Reply 非空、
// Label ≤20 字（点选气泡容量）、Clear/Note 非空、Note ≤18 字（UserConcept.Note 战报）。
// 同时反向守护：剧本条目不得指向课程表外的 slug（防手误）。
func TestCourseScriptsComplete(t *testing.T) {
	for agentSlug, script := range courseScripts {
		list, ok := curricula[agentSlug]
		if !ok {
			t.Fatalf("%s 有剧本但没有课程表", agentSlug)
		}
		content := curatedContent[agentSlug]
		seen := make(map[string]bool, len(list))
		for _, c := range list {
			seen[c.Slug] = true
			lv, ok := script[c.Slug]
			if !ok {
				t.Errorf("%s/%s 缺少剧本", agentSlug, c.Slug)
				continue
			}
			// 剧本依赖精编 Hook/Check 做开场与题面，缺一不可。
			if hc := content[c.Slug]; hc.Hook == "" || hc.Check == "" {
				t.Errorf("%s/%s 剧本依赖的精编 Hook/Check 不完整", agentSlug, c.Slug)
			}
			if len(lv.Nodes) > 0 {
				checkNodes(t, agentSlug, c.Slug, lv.Nodes)
			} else {
				checkOptions(t, agentSlug, c.Slug, "Taps", lv.Taps)
			}
			checkOptions(t, agentSlug, c.Slug, "CheckOpts", lv.CheckOpts)
			if lv.Correct < 0 || lv.Correct >= len(lv.CheckOpts) {
				t.Errorf("%s/%s Correct=%d 越界（CheckOpts 共 %d 项）", agentSlug, c.Slug, lv.Correct, len(lv.CheckOpts))
			}
			for vi, v := range lv.Variants {
				if strings.TrimSpace(v.Ask) == "" {
					t.Errorf("%s/%s Variants[%d] 缺少题面 Ask", agentSlug, c.Slug, vi)
				}
				checkOptions(t, agentSlug, c.Slug, "Variants", v.Opts)
				if v.Correct < 0 || v.Correct >= len(v.Opts) {
					t.Errorf("%s/%s Variants[%d] Correct=%d 越界（共 %d 项）", agentSlug, c.Slug, vi, v.Correct, len(v.Opts))
				}
			}
			if strings.TrimSpace(lv.Clear) == "" {
				t.Errorf("%s/%s 缺少点亮语 Clear", agentSlug, c.Slug)
			}
			if strings.TrimSpace(lv.Note) == "" {
				t.Errorf("%s/%s 缺少战报 Note", agentSlug, c.Slug)
			} else if n := utf8.RuneCountInString(lv.Note); n > 18 {
				t.Errorf("%s/%s Note 超长（%d>18 字）：%s", agentSlug, c.Slug, n, lv.Note)
			}
		}
		for slug := range script {
			if !seen[slug] {
				t.Errorf("%s 剧本条目 %s 不在课程表里（slug 手误？）", agentSlug, slug)
			}
		}
	}
}

func checkOptions(t *testing.T, agentSlug, slug, field string, opts []tapOption) {
	t.Helper()
	if len(opts) < 2 || len(opts) > 4 {
		t.Errorf("%s/%s %s 应为 2-4 项，实际 %d", agentSlug, slug, field, len(opts))
	}
	labelSeen := make(map[string]bool, len(opts))
	for i, o := range opts {
		label := strings.TrimSpace(o.Label)
		if label == "" || strings.TrimSpace(o.Reply) == "" {
			t.Errorf("%s/%s %s[%d] Label/Reply 不得为空", agentSlug, slug, field, i)
			continue
		}
		if n := utf8.RuneCountInString(label); n > 20 {
			t.Errorf("%s/%s %s[%d] Label 超长（%d>20 字）：%s", agentSlug, slug, field, i, n, label)
		}
		// Label 是点选匹配键：同组重复会让 matchIndex 永远命中前者。
		if labelSeen[label] {
			t.Errorf("%s/%s %s Label 重复：%s", agentSlug, slug, field, label)
		}
		labelSeen[label] = true
		// 导航保留词不得用作关内选项，否则收尾阶段无法与之区分。
		switch label {
		case optNext, optMap, optReviewMore, optBackCourse:
			t.Errorf("%s/%s %s[%d] Label 撞导航保留词：%s", agentSlug, slug, field, i, label)
		}
	}
}

// Correct 下标应在关与关之间变化（不恒定同一位置），防「永远选 B」被玩家摸出规律。
func TestCourseScriptsCorrectVaries(t *testing.T) {
	for agentSlug, script := range courseScripts {
		dist := map[int]int{}
		for _, lv := range script {
			dist[lv.Correct]++
		}
		if len(script) >= 10 && len(dist) < 2 {
			t.Errorf("%s 全部 %d 关的 Correct 都在同一下标，答案位置要有变化", agentSlug, len(script))
		}
	}
}

// checkNodes 校验节点图关卡：每个节点要么是点选节点（选项 2-4、Label ≤20 字、Reply 非空、
// Next 合法），要么是跟读节点（Say/SayOK/SayFail 非空、SayNext 合法），要么是产出节点
// （Free 非空、FreeNext 合法、不与 Options/Say 混用）；全图必须存在通关出口。
func checkNodes(t *testing.T, agentSlug, slug string, nodes []scriptNode) {
	t.Helper()
	validNext := func(n int) bool { return n == NodeClear || (n >= 0 && n < len(nodes)) }
	hasClear := false
	for ni, nd := range nodes {
		if nd.Free != "" {
			if len(nd.Options) > 0 || nd.Say != "" {
				t.Errorf("%s/%s Nodes[%d] 产出节点不得再带 Options/Say", agentSlug, slug, ni)
			}
			if !validNext(nd.FreeNext) {
				t.Errorf("%s/%s Nodes[%d] FreeNext=%d 非法", agentSlug, slug, ni, nd.FreeNext)
			}
			if freeJudgeRubrics[agentSlug] == "" {
				t.Errorf("%s/%s 有产出节点但课程未在 freeJudgeRubrics 登记判定口径", agentSlug, slug)
			}
			if nd.FreeNext == NodeClear {
				hasClear = true
			}
			continue
		}
		if nd.Say != "" {
			if len(nd.Options) > 0 {
				t.Errorf("%s/%s Nodes[%d] 跟读节点不得再带 Options", agentSlug, slug, ni)
			}
			if strings.TrimSpace(nd.SayOK) == "" || strings.TrimSpace(nd.SayFail) == "" {
				t.Errorf("%s/%s Nodes[%d] 跟读节点缺 SayOK/SayFail", agentSlug, slug, ni)
			}
			if !validNext(nd.SayNext) {
				t.Errorf("%s/%s Nodes[%d] SayNext=%d 非法", agentSlug, slug, ni, nd.SayNext)
			}
			if nd.SayNext == NodeClear {
				hasClear = true
			}
			continue
		}
		if len(nd.Options) < 2 || len(nd.Options) > 4 {
			t.Errorf("%s/%s Nodes[%d] 选项应为 2-4 项，实际 %d", agentSlug, slug, ni, len(nd.Options))
		}
		labelSeen := make(map[string]bool, len(nd.Options))
		for oi, o := range nd.Options {
			label := strings.TrimSpace(o.Label)
			if label == "" || strings.TrimSpace(o.Reply) == "" {
				t.Errorf("%s/%s Nodes[%d].Options[%d] Label/Reply 不得为空", agentSlug, slug, ni, oi)
				continue
			}
			// 节点选项是学员的完整台词（原话候选），比概念课的短标签宽；
			// 英文整句（开口课）按字符计更长，上限放到 60。
			if n := utf8.RuneCountInString(label); n > 60 {
				t.Errorf("%s/%s Nodes[%d].Options[%d] Label 超长（%d>60 字）：%s", agentSlug, slug, ni, oi, n, label)
			}
			if labelSeen[label] {
				t.Errorf("%s/%s Nodes[%d] Label 重复：%s", agentSlug, slug, ni, label)
			}
			labelSeen[label] = true
			switch label {
			case optNext, optMap, optReviewMore, optBackCourse:
				t.Errorf("%s/%s Nodes[%d].Options[%d] Label 撞导航保留词：%s", agentSlug, slug, ni, oi, label)
			}
			if o.Next != NodeRetry && !validNext(o.Next) {
				t.Errorf("%s/%s Nodes[%d].Options[%d] Next=%d 非法", agentSlug, slug, ni, oi, o.Next)
			}
			if o.Next == NodeClear {
				hasClear = true
			}
		}
	}
	if !hasClear {
		t.Errorf("%s/%s 节点图没有任何通关出口（NodeClear）", agentSlug, slug)
	}
}

// 跟读匹配：ASR 转写有错字/丢标点也应命中，完全不相干的话不得命中。
func TestMatchSay(t *testing.T) {
	target := "Hi, I'd like an oat latte with less sugar, please."
	for _, ok := range []string{
		"hi i'd like an oat latte with less sugar please",
		"Hi, I like an oat latte with less sugar please", // ASR 丢了 'd
		"hi id like an oat latte with less sugar",        // 尾词丢失（互含）
	} {
		if !matchSay(ok, target) {
			t.Errorf("应命中却未命中：%s", ok)
		}
	}
	for _, bad := range []string{
		"", "今天天气不错", "I want coffee",
		// 目标句的碎片不算「真开口」：反向包含漏洞（只说一个词就通关）的回归守护。
		"sorry", "please", "oat latte",
	} {
		if matchSay(bad, target) {
			t.Errorf("不应命中却命中：%s", bad)
		}
	}
	// 中文跟读（言值可选用）
	if !matchSay("这周实在排不开，下次一定到", "这周实在排不开，下次一定到。") {
		t.Error("中文去标点应命中")
	}
}

// 产出节点判定结果解析：正常 JSON、带空白、坏 JSON 三种路径。
func TestParseFreeVerdict(t *testing.T) {
	pass, note, err := parseFreeVerdict(` {"pass":true,"note":"事和关系都稳住了"} `)
	if err != nil || !pass || note != "事和关系都稳住了" {
		t.Errorf("正常 JSON 解析失败：pass=%v note=%q err=%v", pass, note, err)
	}
	pass, note, err = parseFreeVerdict(`{"pass":false,"note":"太软，最后一句松口了"}`)
	if err != nil || pass || note == "" {
		t.Errorf("不通过 JSON 解析失败：pass=%v note=%q err=%v", pass, note, err)
	}
	if _, _, err = parseFreeVerdict("对不起，我没法输出 JSON"); err == nil {
		t.Error("坏 JSON 应返回错误（调用方按判定失败放行）")
	}
}

// 状态机纯函数部分的单测（不依赖 DB）：选项匹配与标签提取。
func TestMatchOption(t *testing.T) {
	opts := []tapOption{{Label: "见过，朋友圈摆拍那种", Reply: "a"}, {Label: "我自己也有点爱表现", Reply: "b"}}
	if i := matchIndex(opts, " 我自己也有点爱表现 "); i != 1 {
		t.Errorf("带空白应命中 1，得 %d", i)
	}
	if i := matchIndex(opts, "别的话"); i != -1 {
		t.Errorf("未命中应为 -1，得 %d", i)
	}
	if got := labels(opts); len(got) != 2 || got[0] != opts[0].Label {
		t.Errorf("labels 提取错误：%v", got)
	}
}
