// Package mirror 实现「镜子分身」：分身据主人画像，反过来对主人本人说「我眼中的你」。
//
// 复用 answer 引擎的知识注入；纯实时生成、不持久化（内容随资料更新而变，每次现拍）。
// 它既是建完分身后的即时情感回报（onboarding 的「哇」时刻），也是驱动主人继续喂养知识的钩子。
// 给主人本人看（auth + 仅取本人 profile），不涉及访客、不涉及支付。
package mirror

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/answer"
	"weifou-server/internal/deepseek"
	"weifou-server/internal/httpx"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

type Handler struct {
	db        *gorm.DB
	ds        *deepseek.Client
	engine    *answer.Engine
	jwtSecret string
}

func NewHandler(db *gorm.DB, ds *deepseek.Client, engine *answer.Engine, jwtSecret string) *Handler {
	return &Handler{db: db, ds: ds, engine: engine, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/mirror", auth, httpx.Handle(h.mine)) // 我的镜子：分身眼中的我
}

// mirrorResult 是模型按要求返回、也直接回前端的结构。
type mirrorResult struct {
	Tags      []string `json:"tags"`      // 3 个最贴合的特质词
	Strength  string   `json:"strength"`  // 最欣赏 TA 的一点
	Blindspot string   `json:"blindspot"` // 可留意 / 容易忽略的一点
	Quip      string   `json:"quip"`      // 一句有温度的小锐评
}

func (h *Handler) mine(c *gin.Context) error {
	auth := middleware.Current(c)

	var profile models.Profile
	if err := h.db.First(&profile, "user_id = ?", auth.UserID).Error; err != nil {
		return httpx.BadRequest("NO_PROFILE", "先建一个你的分身，我才看得懂你")
	}
	var p models.PersonaAI
	if err := h.db.First(&p, "profile_id = ?", profile.ID).Error; err != nil {
		return httpx.BadRequest("PROFILE_NOT_READY", "你的分身还在准备中，稍后再来")
	}
	var tags []string
	_ = json.Unmarshal(p.Tags, &tags)

	res, err := h.generate(&profile, p.OneLiner, p.FullIntro, tags, h.engine.KnowledgeFor(profile.ID))
	if err != nil {
		return httpx.Internal("MIRROR_GEN_FAIL", "分身一时语塞，待会儿再看？")
	}
	httpx.OK(c, gin.H{
		"realName":  profile.RealName,
		"tags":      res.Tags,
		"strength":  res.Strength,
		"blindspot": res.Blindspot,
		"quip":      res.Quip,
	})
	return nil
}

func (h *Handler) generate(p *models.Profile, oneLiner, fullIntro string, tags []string, knowledge string) (mirrorResult, error) {
	company := ""
	if p.Company != nil && *p.Company != "" {
		company = "（" + *p.Company + "）"
	}
	knowledgeBlock := ""
	if strings.TrimSpace(knowledge) != "" {
		knowledgeBlock = "\nTA 补充过的问答：\n" + knowledge
	}
	sys := fmt.Sprintf(`你非常了解 %s，以下是关于 TA 的资料。请以一个「很懂 TA、亦师亦友」的口吻，
直接对 %s 本人说几句你眼中的 TA——温暖、真诚、有洞察。可以有一点俏皮，但不冒犯、不谄媚、不喊口号。
只能基于资料合理推断，不要编造具体事实；全程用第二人称「你」。

== 资料 ==
身份：%s%s
一句话：%s
完整介绍：%s
标签：%s%s
== 资料结束 ==

只输出一个 JSON 对象，不要任何额外文字或代码块：
{"tags":["3 个最贴合你的特质词，每个 2-6 字"],"strength":"我最欣赏你的一点，第二人称，40-60 字，具体不空泛","blindspot":"我觉得你可以留意、容易忽略的一点，温和且有建设性，第二人称，40-60 字","quip":"一句有温度的总结或小锐评，20 字以内"}`,
		p.RealName, p.RealName, p.Title, company, oneLiner, fullIntro, strings.Join(tags, "、"), knowledgeBlock)

	out, err := h.ds.Chat([]deepseek.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: "说说你眼中的我吧。"},
	}, deepseek.ChatOptions{Temperature: 0.85, MaxTokens: 700, ResponseFormat: "json_object"})
	if err != nil {
		return mirrorResult{}, err
	}
	var r mirrorResult
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return mirrorResult{}, err
	}
	if strings.TrimSpace(r.Strength) == "" && strings.TrimSpace(r.Quip) == "" {
		return mirrorResult{}, fmt.Errorf("empty mirror result")
	}
	if len(r.Tags) > 3 {
		r.Tags = r.Tags[:3] // 兜底：标签最多 3 个
	}
	return r, nil
}
