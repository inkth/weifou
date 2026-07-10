package toolagent

import (
	"encoding/json"
	"strings"
	"testing"
)

// verdict 归一化：只认 right/wrong，其余（含模型臆造值、大小写、空白）一律中性。
// 中性＝舞台安静，绝不能让脏值驱动出一场战斗动画。
func TestNormalizeVerdict(t *testing.T) {
	cases := map[string]string{
		"right": "right", "wrong": "wrong",
		" Right ": "right", "WRONG": "wrong",
		"": "", "neutral": "", "对": "", "true": "", "right-ish": "",
	}
	for in, want := range cases {
		if got := normalizeVerdict(in); got != want {
			t.Errorf("normalizeVerdict(%q) = %q，期望 %q", in, got, want)
		}
	}
}

// 判定器返回体必须能解析出 verdict；旧格式（无该字段）解析后为空＝中性，不破坏向后兼容。
func TestConceptAssessResultParsesVerdict(t *testing.T) {
	var r conceptAssessResult
	if err := json.Unmarshal([]byte(`{"touched":["a"],"mastered":[],"note":"好","verdict":"wrong"}`), &r); err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if r.Verdict != "wrong" || len(r.Touched) != 1 {
		t.Errorf("解析结果不对：%+v", r)
	}
	var old conceptAssessResult
	if err := json.Unmarshal([]byte(`{"touched":[],"mastered":[],"note":""}`), &old); err != nil {
		t.Fatalf("旧格式解析失败：%v", err)
	}
	if old.Verdict != "" {
		t.Errorf("旧格式 verdict 应为空，得到 %q", old.Verdict)
	}
}

// verdict 规则的注入边界：判断型课（有对错的检验关）注入，英语/道德经不注入。
// 前端 DUEL_SLUGS 与此一一对应——两处漂移就会出现「舞台演战斗但服务端从不给 verdict」的哑火。
func TestVerdictRuleOnlyForJudgementCourses(t *testing.T) {
	withVerdict := map[string]string{
		"learn-psychology": conceptAssessPrompt, // 默认判定器（心理/逻辑/营销共用）
		"learn-ai":         aiAssessPrompt,
		"learn-speaking":   speakingAssessPrompt,
	}
	for slug, p := range withVerdict {
		if !strings.Contains(p, `verdict="wrong"`) {
			t.Errorf("%s 的判定 prompt 未注入 verdict 规则", slug)
		}
		if !strings.Contains(p, `"verdict":"right|wrong|"`) {
			t.Errorf("%s 的判定 prompt 输出格式里没有 verdict 字段", slug)
		}
	}
	for slug, p := range map[string]string{
		"spoken-english": englishAssessPrompt,
		"daodejing-full": daodejingAssessPrompt,
	} {
		if strings.Contains(p, "verdict") {
			t.Errorf("%s 无对错语义，不该注入 verdict 规则", slug)
		}
	}
	// 心理/逻辑/营销确实落在默认判定器上（没有被 conceptAssessPrompts 悄悄覆盖成无 verdict 的版本）
	for _, slug := range []string{"learn-psychology", "learn-logic", "learn-marketing"} {
		if p, ok := conceptAssessPrompts[slug]; ok && !strings.Contains(p, "verdict") {
			t.Errorf("%s 被覆盖为无 verdict 的判定 prompt", slug)
		}
	}
}
