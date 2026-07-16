// Package toolagent — content_load.go：课程内容加载器（2026-07-16 内容外置）。
//
// 课程内容（课程表/精编 Hook+Check/剧本/幕名/判分口径/章末知识卡）不再硬编码在 Go 源码，
// 而是外置为 content/courses/<slug>.json：
//   - 默认走 go:embed 内嵌（内容随二进制走，部署零变化，测试守护照常）；
//   - 设置 COURSE_CONTENT_DIR=<目录> 可按文件覆盖：目录下的同名 JSON 优先生效，
//     没有的课回落内嵌版——运营改一道题 = 改服务器上的 JSON 重启，不用重新编译发版。
//     覆盖文件解析失败只告警并回落内嵌版，坏编辑不允许拖垮启动。
//
// 加载发生在包 init：注册表（curricula/curatedContent/courseScripts/freeJudgeRubrics/
// bossCardContent/courseTiers）全部由 JSON 构建，引擎与 Seed 逻辑无感知。
package toolagent

import (
	"embed"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed content/courses/*.json
var embeddedCourseFS embed.FS

// 注册表：由 loadCourseContent 从 JSON 构建（Go 时代的字面量已随内容外置删除）。
var (
	// curricula：agent slug → 课程表（顺序即 Sort）。SeedConcepts 据此幂等写入 agent_concepts。
	curricula = map[string][]seedConcept{}
	// curatedContent：agent slug → concept slug → 精编 Hook/Check（这条产品线的护城河）。
	curatedContent = map[string]map[string]hookCheck{}
	// courseScripts：agent slug → concept slug → 剧本。全部课程走脚本引擎（零 LLM 闯关）。
	courseScripts = map[string]map[string]levelScript{}
	// freeJudgeRubrics：agent slug → 产出节点（Free）的 LLM 判分口径。
	freeJudgeRubrics = map[string]string{}
	// bossCardContent：concept slug → 章末知识卡（由概念行的 takeaway/source 重建）。
	bossCardContent = map[string]bossCard{}
	// courseTiers：agent slug → 幕名与展示顺序（tiersFor 的数据源）。
	courseTiers = map[string]courseTierSet{}
)

type courseTierSet struct {
	labels map[int]string
	order  []int
}

// 默认三档（课程 JSON 未给幕名时的兜底；导出器总是写全，正常不会走到）。
var defaultTierLabels = map[int]string{1: "入门", 2: "进阶", 3: "大师"}
var defaultTierOrder = []int{1, 2, 3}

// tiersFor 取某课程的档位标签与顺序（数据源=课程 JSON 的 tierLabels/tierOrder）。
func tiersFor(agentSlug string) (map[int]string, []int) {
	if t, ok := courseTiers[agentSlug]; ok && len(t.labels) > 0 && len(t.order) > 0 {
		return t.labels, t.order
	}
	return defaultTierLabels, defaultTierOrder
}

func init() {
	loadCourseContent()
}

func loadCourseContent() {
	entries, err := embeddedCourseFS.ReadDir("content/courses")
	if err != nil {
		panic("toolagent: 内嵌课程内容缺失: " + err.Error())
	}
	overrideDir := strings.TrimSpace(os.Getenv("COURSE_CONTENT_DIR"))
	seen := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		data, rerr := embeddedCourseFS.ReadFile("content/courses/" + name)
		if rerr != nil {
			panic("toolagent: 读内嵌课程内容失败 " + name + ": " + rerr.Error())
		}
		// 目录覆盖：同名文件优先；解析失败告警回落内嵌版（坏编辑不拖垮启动）。
		if overrideDir != "" {
			if b, oerr := os.ReadFile(filepath.Join(overrideDir, name)); oerr == nil {
				if parseCourseDoc(name, b) == nil {
					log.Printf("[course-content] %s 覆盖文件解析失败，回落内嵌版", name)
				} else {
					data = b
				}
			}
		}
		doc := parseCourseDoc(name, data)
		if doc == nil {
			panic("toolagent: 内嵌课程内容非法 " + name)
		}
		registerCourseDoc(doc)
		seen[name] = true
	}
	// 覆盖目录里内嵌没有的新课文件也装载（配合 Seed preset 可小步上新内容）。
	if overrideDir != "" {
		if extra, derr := os.ReadDir(overrideDir); derr == nil {
			for _, e := range extra {
				name := e.Name()
				if seen[name] || !strings.HasSuffix(name, ".json") {
					continue
				}
				b, oerr := os.ReadFile(filepath.Join(overrideDir, name))
				if oerr != nil {
					continue
				}
				if doc := parseCourseDoc(name, b); doc != nil {
					registerCourseDoc(doc)
					log.Printf("[course-content] 装载覆盖目录新增课程 %s", doc.Slug)
				} else {
					log.Printf("[course-content] 覆盖目录新增文件 %s 解析失败，忽略", name)
				}
			}
		}
	}
}

// parseCourseDoc 解析并做最低限度校验：slug 与文件名一致、课程表非空。非法返回 nil。
func parseCourseDoc(filename string, data []byte) *courseDoc {
	var doc courseDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	if doc.Slug == "" || doc.Slug+".json" != filename || len(doc.Concepts) == 0 {
		return nil
	}
	return &doc
}

// registerCourseDoc 把一份课程文档拆回引擎的各注册表。
func registerCourseDoc(doc *courseDoc) {
	list := make([]seedConcept, 0, len(doc.Concepts))
	content := make(map[string]hookCheck, len(doc.Concepts))
	for _, c := range doc.Concepts {
		list = append(list, seedConcept{Slug: c.Slug, Theme: c.Theme, Name: c.Name, Blurb: c.Blurb, Tier: c.Tier})
		content[c.Slug] = hookCheck{Hook: c.Hook, Check: c.Check}
		if c.Takeaway != "" || c.Source != "" {
			bossCardContent[c.Slug] = bossCard{Takeaway: c.Takeaway, Source: c.Source}
		}
	}
	curricula[doc.Slug] = list
	curatedContent[doc.Slug] = content
	if doc.Levels != nil {
		courseScripts[doc.Slug] = doc.Levels
	}
	if doc.FreeJudgeRubric != "" {
		freeJudgeRubrics[doc.Slug] = doc.FreeJudgeRubric
	}
	courseTiers[doc.Slug] = courseTierSet{labels: doc.TierLabels, order: doc.TierOrder}
}
