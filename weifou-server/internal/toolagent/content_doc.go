// Package toolagent — content_doc.go：课程内容外置的文档 schema。
//
// 一门课一个 JSON 文件（content/courses/<slug>.json），装下该课的全部作者内容：
// 课程表（概念行，含精编 Hook/Check 与章末知识卡）、幕名、剧本、产出判分口径。
// 结构体与引擎运行时结构一一对应，加载即用、不做二次转换（见 content_load.go）。
package toolagent

// courseDoc 一门课的完整外置内容。
type courseDoc struct {
	Slug string `json:"slug"` // agent slug，与文件名一致（加载时校验）

	// 幕名与展示顺序（tiersFor 的数据源）。导出时总是写全，作者一目了然。
	TierLabels map[int]string `json:"tierLabels"`
	TierOrder  []int          `json:"tierOrder"`

	// 产出节点（Free）的 LLM 判分口径；无产出节点的课程为空。
	FreeJudgeRubric string `json:"freeJudgeRubric,omitempty"`

	// 课程表：顺序即 Sort（SeedConcepts 按下标写库），一行一关。
	Concepts []conceptDoc `json:"concepts"`

	// 剧本：concept slug → 关卡剧本（变式题库已并入 variants）。
	Levels map[string]levelScript `json:"levels"`
}

// conceptDoc 课程表一行 = seedConcept + 精编 Hook/Check + 章末知识卡（仅 Boss 关非空）。
// 三张表在 Go 时代按 slug 分开维护，外置后合并为一行——作者改一关只看一处。
type conceptDoc struct {
	Slug     string `json:"slug"`
	Theme    string `json:"theme"`
	Name     string `json:"name"`
	Blurb    string `json:"blurb"`
	Tier     int    `json:"tier"`
	Hook     string `json:"hook,omitempty"`
	Check    string `json:"check,omitempty"`
	Takeaway string `json:"takeaway,omitempty"` // 章末知识卡结论（仅 Boss 关）
	Source   string `json:"source,omitempty"`   // 知识卡引用来源（仅 Boss 关）
}
