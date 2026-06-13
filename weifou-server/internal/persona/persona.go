package persona

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"weifou-server/internal/deepseek"
	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

const systemPrompt = `你是一位资深的个人品牌顾问，帮助用户基于他们提供的信息生成一个"AI 助理"人设。
这个 AI 助理会以"本人的助理"身份代替本人接待访客、与访客对话，所以既要专业可信，也要有鲜明、自然、好聊的人格。
你的输出必须是合法 JSON（不要包含任何额外文字、代码块标记），结构如下：
{
  "oneLiner": "一句话介绍（不超过 35 字，专业、具体、有辨识度，不要使用"我是"开头）",
  "fullIntro": "完整介绍（120-220 字，第三人称，结构：身份与方向、近期在做的事、可被合作的方式）",
  "tags": ["3-5 个简短的人格/方向标签，每个不超过 6 字"],
  "starters": ["3-4 个'访客最该问 TA 的问题'，每个不超过 14 字，第二人称口吻（如'你在做什么项目？'），用于引导对话"],
  "greeting": "开场白（40-70 字，以'我是XX的AI助理'的助理身份第一人称开口，口语、温度感，作为访客进入对话看到的第一句话，体现代主人接待的姿态，自然引出'你可以问我什么'，不要客套套话）",
  "tone": "语气与性格描述（30-60 字，第三人称，描述 TA 该用怎样的口吻、节奏、性格特征与访客对话，用于约束后续 AI 回复的人格一致性；若用户提供了'对外沟通风格'，tone 必须以该风格为基调展开）",
  "voiceStyle": "建议音色，从这几个里选一个最贴合的：温暖男声 / 沉稳男声 / 清朗男声 / 温暖女声 / 知性女声 / 活泼女声"
}
要求：
- 内容务必真实、不夸大，禁止编造未提供的事实。
- oneLiner/fullIntro 语气专业克制；greeting 可更口语、有亲和力，但不浮夸。
- 不要出现"AI"、"人工智能"等空泛词，除非用户原文本就强调。
- starters 要具体、能勾起兴趣、贴合 TA 的方向，避免空泛（不要"你是谁"这种）。`

// Service 负责调用 DeepSeek 生成主页文案并写库。
type Service struct {
	db       *gorm.DB
	ds       *deepseek.Client
	security *wechat.SecurityService
}

func NewService(db *gorm.DB, ds *deepseek.Client, security *wechat.SecurityService) *Service {
	return &Service{db: db, ds: ds, security: security}
}

// CheckText 暴露内容安全校验，供本人手动编辑人设时复用。
func (s *Service) CheckText(text, openid string) bool {
	return s.security.CheckText(text, openid)
}

type Result struct {
	OneLiner   string   `json:"oneLiner"`
	FullIntro  string   `json:"fullIntro"`
	Tags       []string `json:"tags"`
	Starters   []string `json:"starters"`
	Greeting   string   `json:"greeting"`
	Tone       string   `json:"tone"`
	VoiceStyle string   `json:"voiceStyle"`
}

// StyleDescriptions 对外沟通风格：枚举 id → 注入 LLM 的描述。
// 是 PersonaInput.Style 的唯一白名单；前端展示文案在 weifou-miniapp/pages/create/index.js 的 STYLE_OPTIONS，新增/调整需两处同步。
// 每条末尾统一拼专业边界句，守住「经纪人」定位不滑向娱乐化。
var StyleDescriptions = map[string]string{
	"steady":   "沉稳专业：措辞克制、逻辑清晰、少用语气词，始终保持专业可信，不油腻不娱乐化",
	"warm":     "亲和健谈：口语化、爱用比喻、会主动追问对方，始终保持专业可信，不油腻不娱乐化",
	"sharp":    "犀利直接：观点鲜明、先给结论、不绕弯子，始终保持专业可信，不油腻不娱乐化",
	"humorous": "幽默轻松：适度自嘲与玩笑，谈正事时切回认真，始终保持专业可信，不油腻不娱乐化",
}

// NormalizeStyle 白名单校验：非法值落空串（空=未选，由 AI 自行判断）。
func NormalizeStyle(style string) string {
	if _, ok := StyleDescriptions[style]; ok {
		return style
	}
	return ""
}

// 允许的音色，模型自由发挥时归一到默认值。
var allowedVoices = map[string]bool{
	"温暖男声": true, "沉稳男声": true, "清朗男声": true,
	"温暖女声": true, "知性女声": true, "活泼女声": true,
}

func buildUserPrompt(p *models.Profile, in *models.PersonaInput) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("姓名：%s\n", p.RealName))
	b.WriteString(fmt.Sprintf("职业：%s\n", p.Title))
	if p.Company != nil && *p.Company != "" {
		b.WriteString(fmt.Sprintf("公司：%s\n", *p.Company))
	}
	if p.City != nil && *p.City != "" {
		b.WriteString(fmt.Sprintf("城市：%s\n", *p.City))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Q1 最擅长什么：%s", in.Strengths))
	// 快速创建只有 Q1；空项不进 prompt，避免模型对着空字段编造
	if strings.TrimSpace(in.RecentWork) != "" {
		b.WriteString(fmt.Sprintf("\nQ2 最近在做什么：%s", in.RecentWork))
	}
	if strings.TrimSpace(in.HowToKnow) != "" {
		b.WriteString(fmt.Sprintf("\nQ3 希望别人如何认识你：%s", in.HowToKnow))
	}
	if desc, ok := StyleDescriptions[in.Style]; ok {
		b.WriteString(fmt.Sprintf("\n\n本人选择的对外沟通风格：%s", desc))
	}
	return b.String()
}

var fenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

func parseResult(raw string) (*Result, error) {
	text := strings.TrimSpace(raw)
	if m := fenceRe.FindStringSubmatch(text); m != nil {
		text = strings.TrimSpace(m[1])
	}
	var r Result
	if err := json.Unmarshal([]byte(text), &r); err != nil {
		return nil, err
	}
	r.OneLiner = strings.TrimSpace(r.OneLiner)
	r.FullIntro = strings.TrimSpace(r.FullIntro)
	cleaned := make([]string, 0, len(r.Tags))
	for _, t := range r.Tags {
		t = strings.TrimSpace(t)
		if t != "" {
			cleaned = append(cleaned, t)
		}
	}
	if len(cleaned) > 5 {
		cleaned = cleaned[:5]
	}
	r.Tags = cleaned

	starters := make([]string, 0, len(r.Starters))
	for _, t := range r.Starters {
		t = strings.TrimSpace(t)
		if t != "" {
			starters = append(starters, t)
		}
	}
	if len(starters) > 4 {
		starters = starters[:4]
	}
	r.Starters = starters

	r.Greeting = strings.TrimSpace(r.Greeting)
	r.Tone = strings.TrimSpace(r.Tone)
	r.VoiceStyle = strings.TrimSpace(r.VoiceStyle)
	if !allowedVoices[r.VoiceStyle] {
		r.VoiceStyle = "温暖女声" // 归一默认音色
	}
	if r.OneLiner == "" || r.FullIntro == "" || len(r.Tags) == 0 {
		return nil, fmt.Errorf("missing fields")
	}
	return &r, nil
}

const ingestSystemPrompt = `你是一位个人资料整理助手。用户会粘贴一段关于他/她自己的原始文本（如简历、朋友圈、公众号文章、自我介绍），
你的任务是把其中**对访客有价值的事实**拆成若干条独立的「问答式知识」，供其 AI 主页在与访客对话时引用。
你的输出必须是合法 JSON（不要任何额外文字或代码块标记），结构如下：
{
  "items": [
    {"topic": "这条知识对应访客可能会问的问题（不超过 20 字，如'你的报价是多少'、'做过哪些项目'）",
     "content": "对应的客观回答（不超过 120 字，第三人称或直接陈述，只用原文中的事实）"}
  ]
}
要求：
- 只提取原文中**明确出现**的事实，严禁编造、推断或补充未提供的信息。
- 每条聚焦一个点；合并重复，剔除空泛口号与纯情绪表达。
- 最多 20 条；若原文没有可提取的有效事实，返回 {"items": []}。
- topic 用访客视角的疑问句或短名词；content 简洁、不夸大。`

type ingestItem struct {
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

type ingestResult struct {
	Items []ingestItem `json:"items"`
}

// ExtractKnowledge 把一段原始文本拆成多条结构化知识并入库（source=ingest），返回新建条数。
// 这是"轻量灌入"：用户一次粘贴 = 几十条事实，比逐条填表轻一个量级。
func (s *Service) ExtractKnowledge(profileID, openid, rawText string) (int, error) {
	text := strings.TrimSpace(rawText)
	if text == "" {
		return 0, httpx.BadRequest("EMPTY_INPUT", "请粘贴要整理的内容")
	}
	if len([]rune(text)) > 4000 {
		return 0, httpx.BadRequest("INPUT_TOO_LONG", "内容太长（限 4000 字），请分段整理")
	}
	if !s.security.CheckText(text, openid) {
		return 0, httpx.BadRequest("CONTENT_UNSAFE", "内容包含敏感信息，请修改后重试")
	}

	raw, err := s.ds.Chat([]deepseek.Message{
		{Role: "system", Content: ingestSystemPrompt},
		{Role: "user", Content: text},
	}, deepseek.ChatOptions{Temperature: 0.3, MaxTokens: 1500, ResponseFormat: "json_object"})
	if err != nil {
		return 0, httpx.Internal("AI_UPSTREAM_ERROR", "AI 服务暂时不可用，请稍后再试")
	}

	cleaned := strings.TrimSpace(raw)
	if m := fenceRe.FindStringSubmatch(cleaned); m != nil {
		cleaned = strings.TrimSpace(m[1])
	}
	var parsed ingestResult
	if jerr := json.Unmarshal([]byte(cleaned), &parsed); jerr != nil {
		return 0, httpx.BadRequest("AI_PARSE_ERROR", "整理失败，请重试")
	}

	count := 0
	for _, it := range parsed.Items {
		topic := strings.TrimSpace(it.Topic)
		content := strings.TrimSpace(it.Content)
		if content == "" {
			continue
		}
		if len([]rune(topic)) > 128 {
			topic = string([]rune(topic)[:128])
		}
		if len([]rune(content)) > 1000 {
			content = string([]rune(content)[:1000])
		}
		// 逐条过内容安全，跳过不合规的条目而非整批失败。
		if !s.security.CheckText(topic+"\n"+content, openid) {
			continue
		}
		s.db.Create(&models.KnowledgeItem{
			ID: idgen.New(), ProfileID: profileID, Topic: topic,
			Content: content, Source: models.KnowledgeSourceIngest, Enabled: true,
		})
		count++
		if count >= 20 {
			break
		}
	}
	return count, nil
}

const extractSystemPrompt = `你是一位个人主页建立助手。下面是「AI 助理」与用户之间为创建主页而进行的对话。
请仅依据**用户**的回答，抽取建立主页所需的字段，并据用户说话的口吻判断其对外沟通风格。
你的输出必须是合法 JSON（不要任何额外文字或代码块标记），结构如下：
{
  "realName": "用户的称呼/姓名，没说则空",
  "title": "用户的职业/身份，一句话，没说则空",
  "strengths": "用户最擅长、最能帮别人解决的问题，没说则空",
  "recentWork": "用户最近在做的项目或方向，没说则空",
  "howToKnow": "用户最希望别人记住他的点/想立的标签，没说则空",
  "style": "据用户口吻判断，只能是 steady|warm|sharp|humorous 之一；判断不了则空",
  "followup": "若 realName/title/strengths 三项中有缺失，给出下一句要问用户的话（口语、亲切、一次只问一个缺失项，不超过20字）；三项都齐则返回空串"
}
要求：
- 只用对话中**用户明确表达**的信息，严禁编造或推断事实（仅 style 可据语气判断）。
- strengths/recentWork/howToKnow 各不超过 120 字，简洁、第一人称素材即可。
- style 判断参考：措辞克制专业=steady；热情口语爱追问=warm；犀利先抛结论=sharp；爱开玩笑=humorous。`

// ExtractedProfile 是对话式创建从用户自由回答中抽取的字段。
type ExtractedProfile struct {
	RealName   string `json:"realName"`
	Title      string `json:"title"`
	Strengths  string `json:"strengths"`
	RecentWork string `json:"recentWork"`
	HowToKnow  string `json:"howToKnow"`
	Style      string `json:"style"`    // 已过白名单，非法则空
	Followup   string `json:"followup"` // 必填未齐时的下一问；齐则空
	Complete   bool   `json:"complete"` // realName+title+strengths 是否齐（服务端计算）
}

func clampRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > n {
		return string([]rune(s)[:n])
	}
	return s
}

// ExtractProfileFields 从对话文本抽取建主页字段 + 判定沟通风格。无状态：每次传完整对话，返回合并后的最佳结果。
func (s *Service) ExtractProfileFields(openid, dialogue string) (*ExtractedProfile, error) {
	text := strings.TrimSpace(dialogue)
	if text == "" {
		return nil, httpx.BadRequest("EMPTY_INPUT", "还没收到你的介绍")
	}
	if len([]rune(text)) > 4000 {
		text = string([]rune(text)[:4000])
	}
	if !s.security.CheckText(text, openid) {
		return nil, httpx.BadRequest("CONTENT_UNSAFE", "内容包含敏感信息，请修改后重试")
	}

	raw, err := s.ds.Chat([]deepseek.Message{
		{Role: "system", Content: extractSystemPrompt},
		{Role: "user", Content: text},
	}, deepseek.ChatOptions{Temperature: 0.2, MaxTokens: 600, ResponseFormat: "json_object"})
	if err != nil {
		return nil, httpx.Internal("AI_UPSTREAM_ERROR", "AI 服务暂时不可用，请稍后再试")
	}

	cleaned := strings.TrimSpace(raw)
	if m := fenceRe.FindStringSubmatch(cleaned); m != nil {
		cleaned = strings.TrimSpace(m[1])
	}
	var r ExtractedProfile
	if jerr := json.Unmarshal([]byte(cleaned), &r); jerr != nil {
		return nil, httpx.BadRequest("AI_PARSE_ERROR", "没太听懂，换个说法再试试")
	}
	r.RealName = clampRunes(r.RealName, 40)
	r.Title = clampRunes(r.Title, 40)
	r.Strengths = clampRunes(r.Strengths, 200)
	r.RecentWork = clampRunes(r.RecentWork, 200)
	r.HowToKnow = clampRunes(r.HowToKnow, 200)
	r.Style = NormalizeStyle(strings.TrimSpace(r.Style))
	r.Followup = clampRunes(r.Followup, 40)
	r.Complete = r.RealName != "" && r.Title != "" && r.Strengths != ""
	if r.Complete {
		r.Followup = ""
	}
	return &r, nil
}

// GenerateForProfile 生成并 upsert PersonaAI。
func (s *Service) GenerateForProfile(profileID, openid string) (*Result, error) {
	var profile models.Profile
	if err := s.db.First(&profile, "id = ?", profileID).Error; err != nil {
		return nil, httpx.BadRequest("PROFILE_INCOMPLETE", "请先填写主页信息")
	}
	var input models.PersonaInput
	if err := s.db.First(&input, "profile_id = ?", profileID).Error; err != nil {
		return nil, httpx.BadRequest("PROFILE_INCOMPLETE", "请先填写主页信息")
	}

	if !s.security.CheckText(input.Strengths+"\n"+input.RecentWork+"\n"+input.HowToKnow, openid) {
		return nil, httpx.BadRequest("CONTENT_UNSAFE", "你填写的内容包含敏感信息，请修改后重试")
	}

	raw, err := s.ds.Chat([]deepseek.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: buildUserPrompt(&profile, &input)},
	}, deepseek.ChatOptions{Temperature: 0.7, MaxTokens: 800, ResponseFormat: "json_object"})
	if err != nil {
		return nil, httpx.Internal("AI_UPSTREAM_ERROR", "AI 服务暂时不可用，请稍后再试")
	}

	result, err := parseResult(raw)
	if err != nil {
		return nil, httpx.BadRequest("AI_PARSE_ERROR", "AI 生成失败，请重试")
	}

	if !s.security.CheckText(result.OneLiner+"\n"+result.FullIntro+"\n"+result.Greeting, openid) {
		return nil, httpx.BadRequest("CONTENT_UNSAFE", "AI 生成内容未通过审核，请稍后重试")
	}

	tagsJSON, _ := json.Marshal(result.Tags)
	startersJSON, _ := json.Marshal(result.Starters)
	var existing models.PersonaAI
	err = s.db.First(&existing, "profile_id = ?", profileID).Error
	if err == gorm.ErrRecordNotFound {
		s.db.Create(&models.PersonaAI{
			ID:           idgen.New(),
			ProfileID:    profileID,
			OneLiner:     result.OneLiner,
			FullIntro:    result.FullIntro,
			Tags:         datatypes.JSON(tagsJSON),
			Starters:     datatypes.JSON(startersJSON),
			Greeting:     result.Greeting,
			Tone:         result.Tone,
			VoiceStyle:   result.VoiceStyle,
			ModelVersion: s.ds.ModelVersion(),
			GeneratedAt:  time.Now(),
		})
	} else {
		s.db.Model(&existing).Updates(map[string]interface{}{
			"one_liner":     result.OneLiner,
			"full_intro":    result.FullIntro,
			"tags":          datatypes.JSON(tagsJSON),
			"starters":      datatypes.JSON(startersJSON),
			"greeting":      result.Greeting,
			"tone":          result.Tone,
			"voice_style":   result.VoiceStyle,
			"model_version": s.ds.ModelVersion(),
			"generated_at":  time.Now(),
		})
	}
	return result, nil
}
