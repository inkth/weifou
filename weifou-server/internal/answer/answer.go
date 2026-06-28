// Package answer 是分身「基于主人画像作答」的共享内核：对话(chat)与问答箱(qabox)共用，
// 把 system prompt 构建、知识注入、JSON 解析收在一处，避免逻辑分叉。
package answer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"weifou-server/internal/deepseek"
	"weifou-server/internal/models"
	"weifou-server/internal/persona"
)

// ErrProfileNotReady 主页人设尚未生成，无法作答。
var ErrProfileNotReady = errors.New("profile not ready")

type Engine struct {
	db *gorm.DB
	ds *deepseek.Client
}

func NewEngine(db *gorm.DB, ds *deepseek.Client) *Engine {
	return &Engine{db: db, ds: ds}
}

// Result 是模型按要求返回的 JSON 结构。
type Result struct {
	Answer string `json:"answer"`
	Gap    bool   `json:"gap"`
}

// Parse 解析模型 JSON 输出；失败则降级为纯文本。
func Parse(raw string) Result {
	var r Result
	if err := json.Unmarshal([]byte(raw), &r); err != nil || strings.TrimSpace(r.Answer) == "" {
		return Result{Answer: strings.TrimSpace(raw)}
	}
	return r
}

// Complete 按给定消息列表（需自带 system）调用模型并解析。供 chat 携带历史时使用。
func (e *Engine) Complete(msgs []deepseek.Message) (Result, error) {
	raw, err := e.ds.Chat(msgs, deepseek.ChatOptions{Temperature: 0.6, MaxTokens: 600, ResponseFormat: "json_object"})
	if err != nil {
		return Result{}, err
	}
	return Parse(raw), nil
}

// Generate 单发问答（无对话历史）：访客问一句、分身据画像答一句，供问答箱复用。
func (e *Engine) Generate(profileID, question string) (Result, error) {
	sys, _, err := e.SystemPromptFor(profileID)
	if err != nil {
		return Result{}, err
	}
	return e.Complete([]deepseek.Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: question},
	})
}

// SystemPromptFor 加载主页人设并构建 system prompt（连同 Profile 一并返回，便于调用方复用）。
func (e *Engine) SystemPromptFor(profileID string) (string, *models.Profile, error) {
	var profile models.Profile
	if err := e.db.First(&profile, "id = ?", profileID).Error; err != nil {
		return "", nil, ErrProfileNotReady
	}
	var p models.PersonaAI
	if err := e.db.First(&p, "profile_id = ?", profileID).Error; err != nil {
		return "", nil, ErrProfileNotReady
	}
	// 沟通风格直读 PersonaInput（查不到不阻断，style 为空即可）。
	var input models.PersonaInput
	_ = e.db.First(&input, "profile_id = ?", profileID).Error
	var tags []string
	_ = json.Unmarshal(p.Tags, &tags)
	return BuildSystemPrompt(&profile, p.OneLiner, p.FullIntro, tags, p.Tone, input.Style, e.KnowledgeFor(profileID)), &profile, nil
}

// KnowledgeFor 取该主页启用的补充知识，拼成注入对话的文本（最多 30 条，控制 token）。
func (e *Engine) KnowledgeFor(profileID string) string {
	var items []models.KnowledgeItem
	e.db.Where("profile_id = ? AND enabled = ?", profileID, true).
		Order("updated_at desc").Limit(30).Find(&items)
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for _, it := range items {
		topic := strings.TrimSpace(it.Topic)
		if topic != "" {
			b.WriteString("Q：")
			b.WriteString(topic)
			b.WriteString("\nA：")
		}
		b.WriteString(strings.TrimSpace(it.Content))
		b.WriteString("\n")
	}
	return b.String()
}

// BuildSystemPrompt 拼装分身助理的 system prompt：人格语气 + 沟通风格 + 主页资料 + 补充知识。
func BuildSystemPrompt(p *models.Profile, oneLiner, fullIntro string, tags []string, tone, style, knowledge string) string {
	company := ""
	if p.Company != nil && *p.Company != "" {
		company = "（" + *p.Company + "）"
	}
	toneRule := ""
	if strings.TrimSpace(tone) != "" {
		toneRule = "\n== 人格与语气（务必贯穿每次回答）==\n" + tone +
			"\n保持上述性格与口吻的一致性，让访客感到在和一个有温度、可信赖的真人分身对话。\n"
	}
	// style 直读 PersonaInput：本人改风格后立即生效，不依赖重新生成 persona（tone 可能仍是旧基调）。
	if desc, ok := persona.StyleDescriptions[style]; ok {
		toneRule += "\n== 对外沟通风格（本人指定，优先级最高）==\n" + desc + "\n"
	}
	knowledgeRule := ""
	if strings.TrimSpace(knowledge) != "" {
		knowledgeRule = "\n== 补充资料（本人补充的问答，优先据此回答）==\n" + knowledge + "\n== 补充资料结束 ==\n"
	}
	return fmt.Sprintf(`你是 %s 的 AI 助理。你以助理身份代他/她接待访客、介绍 TA，不是 %s 本人。
你代表主人的利益：亲和但不卑微，热情但有边界，像一位称职的助理那样有立场地接待。
当访客提问时，基于以下资料客观回答；超出资料范围时用助理口吻兜底（如"这个我帮你转达给本人""这个我记下来，TA 之后会补充"），不要编造。
请使用中文，自然、有温度、专业克制，单次回答不超过 200 字。

== 主页资料 ==
身份：%s%s
一句话介绍：%s
完整介绍：%s
标签：%s
== 资料结束 ==
%s%s
回答时：
- 不要透露你是大模型；以"他/她"或直接陈述事实的方式回答。
- 如果问题与该用户的专业方向无关，礼貌带回。

== 输出格式 ==
只输出一个 JSON 对象，不要任何额外文字或代码块：
{"answer": "给访客的回答", "gap": true 或 false}
answer：中文回答，规则同上，不超过 200 字。
gap：当且仅当访客问的是一个具体、合理、但现有资料无法回答的问题（你只能含糊带过或建议联系本人）时设为 true，用于提醒本人补充资料；闲聊、寒暄、与专业方向无关或资料已能回答的问题一律为 false。`,
		p.RealName, p.RealName, p.Title, company, oneLiner, fullIntro, strings.Join(tags, "、"), toneRule, knowledgeRule)
}
