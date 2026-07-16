package toolagent

// 内容导出器：把当前注册表里的课程内容序列化为 content/courses/<slug>.json。
//
// 双重用途：
//  1. 一次性迁移（内容还在 Go 源码时）：DUMP_COURSE_CONTENT_DIR=<dir> go test -run TestDumpCourseContent
//     从编译好的数据结构原样导出——保真由机器保证，零手抄。
//  2. 等价性验证（切换到 JSON 加载后）：再跑一次导出到新目录，与迁移时的产物逐字节 diff；
//     一致即证明「导出→加载→再导出」无损，之后才允许删除 Go 源内容。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDumpCourseContent(t *testing.T) {
	dir := os.Getenv("DUMP_COURSE_CONTENT_DIR")
	if dir == "" {
		t.Skip("set DUMP_COURSE_CONTENT_DIR to dump course content")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for slug, list := range curricula {
		content := curatedContent[slug]
		labels, order := tiersFor(slug)
		doc := courseDoc{
			Slug:            slug,
			TierLabels:      labels,
			TierOrder:       order,
			FreeJudgeRubric: freeJudgeRubrics[slug],
			Levels:          courseScripts[slug],
			Concepts:        make([]conceptDoc, 0, len(list)),
		}
		for _, sc := range list {
			hc := content[sc.Slug]
			bc := bossCardContent[sc.Slug]
			doc.Concepts = append(doc.Concepts, conceptDoc{
				Slug: sc.Slug, Theme: sc.Theme, Name: sc.Name, Blurb: sc.Blurb, Tier: sc.Tier,
				Hook: hc.Hook, Check: hc.Check, Takeaway: bc.Takeaway, Source: bc.Source,
			})
		}
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if err := os.WriteFile(filepath.Join(dir, slug+".json"), append(b, '\n'), 0o644); err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
	}
}
