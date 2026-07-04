package toolagent

import "testing"

func TestSplitOptions(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantBody string
		wantOpts []string
	}{
		{
			name:     "末尾选项行",
			in:       "锚定效应讲完了。你觉得哪个是锚定？\n\n【选项】超市原价划线｜朋友迟到｜天下雨",
			wantBody: "锚定效应讲完了。你觉得哪个是锚定？",
			wantOpts: []string{"超市原价划线", "朋友迟到", "天下雨"},
		},
		{
			name:     "无选项行原样返回",
			in:       "Now say it in English: I'd like a latte, please.",
			wantBody: "Now say it in English: I'd like a latte, please.",
			wantOpts: nil,
		},
		{
			name:     "半角分隔符",
			in:       "继续吗？\n【选项】继续|换个场景",
			wantBody: "继续吗？",
			wantOpts: []string{"继续", "换个场景"},
		},
		{
			name:     "超过四个截断",
			in:       "选一个\n【选项】一｜二｜三｜四｜五",
			wantBody: "选一个",
			wantOpts: []string{"一", "二", "三", "四"},
		},
		{
			name:     "选项行为空退化为无选项",
			in:       "好的\n【选项】",
			wantBody: "好的\n【选项】",
			wantOpts: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, opts := splitOptions(c.in)
			if body != c.wantBody {
				t.Fatalf("body = %q, want %q", body, c.wantBody)
			}
			if len(opts) != len(c.wantOpts) {
				t.Fatalf("opts = %v, want %v", opts, c.wantOpts)
			}
			for i := range opts {
				if opts[i] != c.wantOpts[i] {
					t.Fatalf("opts[%d] = %q, want %q", i, opts[i], c.wantOpts[i])
				}
			}
		})
	}
}
