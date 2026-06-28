// Package dating 实现「找对象」择偶测试（B 形态：我 × 平台预设原型）。
//
// 设计：AI 动态出题 + LLM 直接打分。匹配对象是平台预设的 6 个「关系原型」（非真人），
// 因此叙事是「自测 + 与原型对比」，不连接真人、避开婚恋交友红线。
// 闭环：start 出题 → submit 提交答案 → LLM 生成择偶画像 + 对各原型打分 →
// 画像回写为 KnowledgeItem 喂养主人的分身（成为对话记忆）。
//
// 已知权衡（产品主动选择）：题目每次动态生成 → 不同人题面不可比；LLM 直接打分 → 分数会漂、
// 理论上可被诱导。护栏：题目与答案存库可复盘、打分提示词固定维度与口径、每日限流防刷量。
package dating

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"weifou-server/internal/deepseek"
	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

// 每人每日最多出题次数（保护 LLM 成本 + 防刷）。
const dailyQuizLimit = 10

// 题目数量（动态生成，固定题量保证体验一致）。
const quizSize = 8

// 喂回分身的知识项主题（固定，便于重测时 upsert 覆盖而非堆叠）。
const knowledgeTopic = "我的择偶偏好"

// archetype 平台预设的关系原型（匹配对象）。固定集合 → 保证不同用户的匹配目标可比。
type archetype struct {
	Key  string
	Name string
	Desc string
}

var archetypes = []archetype{
	{"steady", "稳健务实型", "重视稳定与安全感，居家、踏实、把日子过好优先，节奏平稳不爱折腾。"},
	{"driven", "事业进取型", "有野心、目标导向，忙碌且自驱，欣赏同样上进、能并肩成长的伴侣。"},
	{"romantic", "自由浪漫型", "重视情绪体验与仪式感，自发、爱表达，渴望心动与精神共鸣。"},
	{"gentle", "温柔体贴型", "高共情、照顾型，以和谐为先，擅长经营关系里的情绪与温度。"},
	{"rational", "理性独立型", "重边界与个人空间，就事论事、独立自洽，反感黏腻与情绪绑架。"},
	{"lively", "活力社交型", "外向热闹、朋友多，喜欢一起探索世界，关系里需要新鲜感与共同活动。"},
}

type Handler struct {
	db        *gorm.DB
	ds        *deepseek.Client
	security  *wechat.SecurityService
	jwtSecret string
}

func NewHandler(db *gorm.DB, ds *deepseek.Client, security *wechat.SecurityService, jwtSecret string) *Handler {
	return &Handler{db: db, ds: ds, security: security, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.POST("/dating/start", auth, httpx.Handle(h.start))   // 出题
	rg.POST("/dating/submit", auth, httpx.Handle(h.submit)) // 提交答案 → 画像 + 匹配度
	rg.GET("/dating/result", auth, httpx.Handle(h.latest))  // 最近一次结果（结果页刷新/回看）
}

// ---------- 数据结构（前后端契约 / LLM JSON 形状） ----------

type quizOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type quizQuestion struct {
	ID      string       `json:"id"`
	Text    string       `json:"text"`
	Options []quizOption `json:"options"`
}

type answer struct {
	QuestionID string `json:"questionId"`
	Key        string `json:"key"`
}

type matchItem struct {
	Archetype string `json:"archetype"` // 原型显示名
	Score     int    `json:"score"`     // 0-100
	Reason    string `json:"reason"`    // 一句话匹配点评
}

// dimension 固定 3 维量纲（价值观/性格/生活方式）。固定轴 = LLM 打分护栏：
// 每次按同一套维度评分，分数更稳、跨用户可比，也对冲「LLM 直接打分会漂」的权衡。
type dimension struct {
	Name  string `json:"name"`  // 维度名（价值观/性格/生活方式）
	Score int    `json:"score"` // 0-100
	Type  string `json:"type"`  // 契合方式：相似 / 互补
	Note  string `json:"note"`  // 一句话说明
}

// headline 头条：最契合原型的总分 + 3 维拆解 + 一句话总结（结果页主视觉）。
type headline struct {
	Archetype  string      `json:"archetype"`
	Total      int         `json:"total"` // 0-100
	Dimensions []dimension `json:"dimensions"`
	Summary    string      `json:"summary"`
}

// 固定 3 维（也用于校验/兜底）。
var fixedDimensions = []string{"价值观", "性格", "生活方式"}

// ---------- 出题 ----------

func (h *Handler) start(c *gin.Context) error {
	auth := middleware.Current(c)

	// 日频限流。
	var todayCount int64
	since := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&models.DatingQuiz{}).
		Where("user_id = ? AND created_at >= ?", auth.UserID, since).
		Count(&todayCount)
	if todayCount >= dailyQuizLimit {
		return httpx.TooManyRequests("DATING_RATE_LIMIT", "今天测得有点多啦，明天再来吧")
	}

	questions, err := h.genQuestions()
	if err != nil {
		return httpx.Internal("DATING_GEN_FAIL", "出题失败，请稍后再试")
	}

	qjson, _ := json.Marshal(questions)
	quiz := models.DatingQuiz{
		ID:        idgen.New(),
		UserID:    auth.UserID,
		Questions: datatypes.JSON(qjson),
		Status:    models.DatingQuizOpen,
		Model:     h.ds.ModelVersion(),
	}
	h.db.Create(&quiz)

	httpx.OK(c, gin.H{"quizId": quiz.ID, "questions": questions})
	return nil
}

func (h *Handler) genQuestions() ([]quizQuestion, error) {
	sys := fmt.Sprintf(`你是一位温暖、专业的情感与关系测评设计师。请为一份「择偶偏好测试」出 %d 道单选题。
要求：
- 每题聚焦一个择偶/相处维度（如：相处节奏、对安全感的需求、冲突处理、生活方式、情绪表达、对未来的规划、独立 vs 黏腻、社交需求等），维度不要重复。
- 每题 4 个选项，覆盖不同性格/价值取向，没有对错，选项之间区分度明显。
- 语气轻松口语化，像朋友间的小测试，不要说教、不涉及任何敏感/露骨内容。
- 只输出 JSON，形如：{"questions":[{"id":"q1","text":"题面","options":[{"key":"A","label":"选项"},{"key":"B","label":"..."},{"key":"C","label":"..."},{"key":"D","label":"..."}]}]}
- id 用 q1..q%d，key 用 A/B/C/D。`, quizSize, quizSize)

	out, err := h.ds.Chat([]deepseek.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: "请出题。"},
	}, deepseek.ChatOptions{Temperature: 1.0, MaxTokens: 2000, ResponseFormat: "json_object"})
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Questions []quizQuestion `json:"questions"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Questions) == 0 {
		return nil, fmt.Errorf("empty questions")
	}
	return parsed.Questions, nil
}

// ---------- 提交 → 画像 + 匹配度 ----------

type submitReq struct {
	QuizID  string   `json:"quizId" binding:"required"`
	Answers []answer `json:"answers" binding:"required"`
}

func (h *Handler) submit(c *gin.Context) error {
	auth := middleware.Current(c)

	var req submitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("BAD_INPUT", "提交数据有误")
	}

	var quiz models.DatingQuiz
	if err := h.db.First(&quiz, "id = ? AND user_id = ?", req.QuizID, auth.UserID).Error; err != nil {
		return httpx.NotFound("QUIZ_NOT_FOUND", "测试不存在或已过期")
	}

	var questions []quizQuestion
	_ = json.Unmarshal(quiz.Questions, &questions)
	qa := buildQA(questions, req.Answers)
	if qa == "" {
		return httpx.BadRequest("NO_ANSWERS", "请先完成答题")
	}

	profileText, head, matches, err := h.analyze(qa)
	if err != nil {
		return httpx.Internal("DATING_ANALYZE_FAIL", "分析失败，请稍后再试")
	}

	ajson, _ := json.Marshal(req.Answers)
	mjson, _ := json.Marshal(matches)
	hjson, _ := json.Marshal(head)
	res := models.DatingResult{
		ID:       idgen.New(),
		UserID:   auth.UserID,
		QuizID:   quiz.ID,
		Answers:  datatypes.JSON(ajson),
		Profile:  profileText,
		Headline: datatypes.JSON(hjson),
		Matches:  datatypes.JSON(mjson),
		Model:    h.ds.ModelVersion(),
	}
	h.db.Create(&res)
	h.db.Model(&quiz).Update("status", models.DatingQuizDone)

	// 把择偶画像喂回主人的分身（若已创建分身）。
	h.feedToClone(auth.UserID, profileText)

	httpx.OK(c, gin.H{"profile": profileText, "headline": head, "matches": matches})
	return nil
}

// buildQA 按 quiz 还原「问题 + 用户所选项」文本，喂给打分模型（不信任前端传题面，防篡改）。
func buildQA(questions []quizQuestion, answers []answer) string {
	qmap := make(map[string]quizQuestion, len(questions))
	for _, q := range questions {
		qmap[q.ID] = q
	}
	var b strings.Builder
	for _, a := range answers {
		q, ok := qmap[a.QuestionID]
		if !ok {
			continue
		}
		label := a.Key
		for _, o := range q.Options {
			if o.Key == a.Key {
				label = o.Label
				break
			}
		}
		b.WriteString(fmt.Sprintf("- 问：%s\n  答：%s\n", q.Text, label))
	}
	return b.String()
}

func (h *Handler) analyze(qa string) (string, headline, []matchItem, error) {
	var typeList strings.Builder
	for _, a := range archetypes {
		typeList.WriteString(fmt.Sprintf("- %s：%s\n", a.Name, a.Desc))
	}

	sys := fmt.Sprintf(`你是一位资深的情感关系分析师。下面是用户在一份择偶测试中的作答。请完成三件事：

1) profile：一段「我的择偶画像」。用第一人称、温暖具体地描述这个人在亲密关系里看重什么、适合怎样的相处方式、理想伴侣的画像。120-180 字，落地不空泛，可直接作为对外资料。

2) matches：评估与以下 6 个「关系原型」的契合度，逐个打 0-100 分：
%s
打分口径：反映「和这类伴侣在一起的长期契合度」，要有区分度（别都打高分，最高与最低拉开差距），每个给一句话理由。

3) headline：针对 matches 中分数最高的那个原型，做一份「3 维契合度拆解」，固定且只用这 3 个维度（顺序不变）：
- 价值观（三观、对未来/金钱/家庭的看法）——一般越相似越契合
- 性格（情绪表达、冲突处理、独立 vs 黏腻）——相似或互补都可能契合
- 生活方式（节奏、兴趣、社交需求）——需要能共处
每个维度给：score（0-100）、type（"相似" 或 "互补"，说明这一维靠哪种方式契合）、note（一句话）。
再给 total（综合总分，应与该最高原型的 matches 分数一致或接近）和 summary（一句话总结，点出相似点与需磨合处）。

只输出 JSON：
{"profile":"...","matches":[{"archetype":"原型名","score":85,"reason":"..."}],"headline":{"archetype":"最高原型名","total":87,"dimensions":[{"name":"价值观","score":92,"type":"相似","note":"..."},{"name":"性格","score":68,"type":"互补","note":"..."},{"name":"生活方式","score":80,"type":"相似","note":"..."}],"summary":"..."}}
matches 覆盖全部 6 个原型；archetype 用上面给出的原型名称原文；headline.dimensions 必须正好是上述 3 个维度且 name 用原文。`, typeList.String())

	out, err := h.ds.Chat([]deepseek.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: "用户的作答如下：\n" + qa},
	}, deepseek.ChatOptions{Temperature: 0.7, MaxTokens: 2000, ResponseFormat: "json_object"})
	if err != nil {
		return "", headline{}, nil, err
	}

	var parsed struct {
		Profile  string      `json:"profile"`
		Matches  []matchItem `json:"matches"`
		Headline headline    `json:"headline"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return "", headline{}, nil, err
	}
	if strings.TrimSpace(parsed.Profile) == "" || len(parsed.Matches) == 0 {
		return "", headline{}, nil, fmt.Errorf("empty analyze result")
	}

	// 夹紧分数并按降序排，前端直接展示。
	for i := range parsed.Matches {
		parsed.Matches[i].Score = clamp(parsed.Matches[i].Score)
	}
	sort.SliceStable(parsed.Matches, func(i, j int) bool {
		return parsed.Matches[i].Score > parsed.Matches[j].Score
	})

	head := normalizeHeadline(parsed.Headline, parsed.Matches)
	return parsed.Profile, head, parsed.Matches, nil
}

func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

// normalizeHeadline 校正头条：夹紧分数、保证头条原型/总分与 matches 第一名对齐、
// 维度只保留固定 3 维（缺失则补占位），避免 LLM 偶发漂移把前端画乱。
func normalizeHeadline(h headline, matches []matchItem) headline {
	top := matchItem{}
	if len(matches) > 0 {
		top = matches[0]
	}
	// 头条原型与总分以 matches 第一名为准（matches 已排序，最可信）。
	h.Archetype = top.Archetype
	if h.Total <= 0 {
		h.Total = top.Score
	}
	h.Total = clamp(h.Total)

	// 按固定 3 维归位（防止漏维或多维）。
	byName := make(map[string]dimension, len(h.Dimensions))
	for _, d := range h.Dimensions {
		byName[d.Name] = d
	}
	dims := make([]dimension, 0, len(fixedDimensions))
	for _, name := range fixedDimensions {
		d, ok := byName[name]
		if !ok {
			d = dimension{Name: name, Score: h.Total, Type: "相似", Note: ""}
		}
		d.Name = name
		d.Score = clamp(d.Score)
		if d.Type != "互补" {
			d.Type = "相似"
		}
		dims = append(dims, d)
	}
	h.Dimensions = dims
	return h
}

// feedToClone 把择偶画像写回主人分身的知识库（按固定主题 upsert，重测覆盖不堆叠）。
// 无分身则跳过，不报错（找对象本身可独立使用）。
func (h *Handler) feedToClone(userID, profileText string) {
	var profile models.Profile
	if err := h.db.First(&profile, "user_id = ?", userID).Error; err != nil {
		return
	}
	var ki models.KnowledgeItem
	err := h.db.First(&ki, "profile_id = ? AND topic = ?", profile.ID, knowledgeTopic).Error
	if err == nil {
		h.db.Model(&ki).Updates(map[string]interface{}{"content": profileText, "enabled": true})
		return
	}
	h.db.Create(&models.KnowledgeItem{
		ID:        idgen.New(),
		ProfileID: profile.ID,
		Topic:     knowledgeTopic,
		Content:   profileText,
		Source:    models.KnowledgeSourceManual,
		Enabled:   true,
	})
}

// ---------- 最近结果 ----------

func (h *Handler) latest(c *gin.Context) error {
	auth := middleware.Current(c)
	var res models.DatingResult
	if err := h.db.Order("created_at desc").First(&res, "user_id = ?", auth.UserID).Error; err != nil {
		httpx.OK(c, nil) // 无结果不算错误
		return nil
	}
	var matches []matchItem
	_ = json.Unmarshal(res.Matches, &matches)
	var head headline
	_ = json.Unmarshal(res.Headline, &head)
	httpx.OK(c, gin.H{"profile": res.Profile, "headline": head, "matches": matches, "createdAt": res.CreatedAt})
	return nil
}
