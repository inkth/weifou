package wechat

import (
	"strings"
	"testing"
	"time"
)

func TestYuan(t *testing.T) {
	cases := map[int]string{4900: "49.00元", 100: "1.00元", 0: "0.00元", 1999: "19.99元"}
	for fen, want := range cases {
		if got := yuan(fen); got != want {
			t.Errorf("yuan(%d) = %q, want %q", fen, got, want)
		}
	}
}

func TestClip(t *testing.T) {
	if got := clip("短文本", 20); got != "短文本" {
		t.Errorf("clip should keep short text, got %q", got)
	}
	long := "一二三四五六七八九十一二三四五六七八九十一二三" // 23 runes
	got := clip(long, 20)
	if r := []rune(got); len(r) != 20 {
		t.Errorf("clip(_,20) returned %d runes: %q", len(r), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("clip truncation should end with ellipsis: %q", got)
	}
}

// 关键安全属性：未配置模板 ID / 拿不到凭证时，订阅消息全链路 no-op，
// 不 panic、不阻断业务（开发环境与未申请模板时都靠这个降级）。
func TestSubscribeNoopWhenUnconfigured(t *testing.T) {
	login := NewLoginClient("", "") // 无凭证，AccessToken 必失败
	now := time.Now()

	// 模板 ID 全空：send 在取 token 之前就 return。
	s := NewSubscribeService(login, "", "", "", "", "", "")
	s.NotifyNewQuestion("openid_x", "你好这是一个测试问题", 4900, now, "pages/inbox/index")
	s.NotifyAnswered("openid_x", "张三", "这是回答内容", now, "p")
	s.NotifyRefunded("openid_x", "问题内容", 4900, "超时未答", "p")
	s.NotifyNewLead("openid_x", "想约个时间聊聊", "访客小明", now, "p")
	s.NotifyLearnRemind("openid_x", "学心理", "下一个待点亮：『锚定效应』", now, "p")
	if s.LearnRemindReady() {
		t.Error("LearnRemindReady should be false when tmpl empty")
	}

	// 配了模板但拿不到 token：仍应静默降级，不 panic。
	s2 := NewSubscribeService(login, "tmpl_new", "tmpl_ans", "tmpl_rfd", "tmpl_lead", "tmpl_learn", "")
	if !s2.LearnRemindReady() {
		t.Error("LearnRemindReady should be true when tmpl set")
	}
	if s2.miniState != "formal" {
		t.Errorf("miniState should default to formal, got %q", s2.miniState)
	}
	s2.NotifyAnswered("openid_y", "李四", "答", now, "p")
}
