package models

import (
	"time"

	"gorm.io/datatypes"
)

// 主键统一用 cuid 风格字符串（由应用生成）

type User struct {
	ID string `gorm:"primaryKey;size:32" json:"id"`
	// Openid：首次登录端的 openid，作为历史/规范锚点（保持不变，下游沿用）。
	Openid string `gorm:"uniqueIndex;size:64" json:"openid"`
	// 路线 A：分端存储 openid，用 Unionid 跨端打通同一真人。
	WxMpOpenid  *string `gorm:"size:64;index" json:"-"` // 小程序 openid
	WxAppOpenid *string `gorm:"size:64;index" json:"-"` // 移动应用 openid
	// Unionid：同一开放平台主体下跨端唯一，作为账号合并主键。
	Unionid   *string `gorm:"uniqueIndex;size:64" json:"unionid,omitempty"`
	Nickname  *string `gorm:"size:128" json:"nickname,omitempty"`
	AvatarURL *string `gorm:"size:512" json:"avatarUrl,omitempty"`
	// 最近一次小程序 session_key（虚拟支付发货确认 NotifyProvideGoods 的离线兜底；会过期）。
	WxSessionKey *string `gorm:"size:64" json:"-"`
	// 首页默认 Agent 是否已种过（只种一次，避免用户「移除全部」后默认又回来）。
	HomeSeeded bool      `gorm:"default:false" json:"-"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (User) TableName() string { return "users" }

const (
	ProfileDraft     = "draft"
	ProfilePublished = "published"
)

type Profile struct {
	ID             string  `gorm:"primaryKey;size:32" json:"id"`
	UserID         string  `gorm:"uniqueIndex;size:32" json:"userId"`
	RealName       string  `gorm:"size:64" json:"realName"`
	Title          string  `gorm:"size:64" json:"title"`
	Company        *string `gorm:"size:128" json:"company,omitempty"`
	City           *string `gorm:"size:64" json:"city,omitempty"`
	ContactWechat  *string `gorm:"size:64" json:"contactWechat,omitempty"`
	ContactPhone   *string `gorm:"size:64" json:"contactPhone,omitempty"`
	ContactVisible bool    `gorm:"default:false" json:"contactVisible"`
	// 是否公开到人物广场（opt-in，默认私密，仅链接分享）。
	Discoverable bool   `gorm:"default:false;index" json:"discoverable"`
	AvatarStyle  string `gorm:"size:32" json:"avatarStyle"`
	// 裂变归因：创建者是体验了谁的 Agent 后转化来的（仅创建时写入，不更新）。
	ReferrerProfileID *string   `gorm:"size:32;index" json:"referrerProfileId,omitempty"`
	Status            string    `gorm:"size:16;default:draft;index" json:"status"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (Profile) TableName() string { return "profiles" }

type PersonaInput struct {
	ID         string    `gorm:"primaryKey;size:32" json:"id"`
	ProfileID  string    `gorm:"uniqueIndex;size:32" json:"profileId"`
	Strengths  string    `gorm:"type:text" json:"strengths"`
	RecentWork string    `gorm:"type:text" json:"recentWork"`
	HowToKnow  string    `gorm:"type:text" json:"howToKnow"`
	Style      string    `gorm:"size:32" json:"style"` // 对外沟通风格枚举 id（白名单见 persona.StyleDescriptions），空=未选由 AI 自行判断
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (PersonaInput) TableName() string { return "persona_inputs" }

type PersonaAI struct {
	ID        string         `gorm:"primaryKey;size:32" json:"id"`
	ProfileID string         `gorm:"uniqueIndex;size:32" json:"profileId"`
	OneLiner  string         `gorm:"type:text" json:"oneLiner"`
	FullIntro string         `gorm:"type:text" json:"fullIntro"`
	Tags      datatypes.JSON `gorm:"type:jsonb" json:"tags"`
	Starters  datatypes.JSON `gorm:"type:jsonb" json:"starters"`
	// 沉浸式对话升级（星野/猫箱级）
	Greeting     string    `gorm:"type:text" json:"greeting"` // 开场白：进入对话首条 AI 消息
	Tone         string    `gorm:"type:text" json:"tone"`     // 语气/性格描述，注入 chat system prompt 保人格一致
	VoiceStyle   string    `gorm:"size:32" json:"voiceStyle"` // 音色标识（映射 TTS 音色，Phase B 用）
	AvatarURL    *string   `gorm:"size:512" json:"avatarUrl,omitempty"`
	ModelVersion string    `gorm:"size:64" json:"modelVersion"`
	GeneratedAt  time.Time `json:"generatedAt"`
}

func (PersonaAI) TableName() string { return "persona_ai" }

// ========== 知识库 / 缺口 / 线索（对话飞轮） ==========

const (
	KnowledgeSourceManual = "manual" // 主人手动添加
	KnowledgeSourceGap    = "gap"    // 由访客问倒的缺口回答而来
	KnowledgeSourceIngest = "ingest" // 批量灌入（预留）
)

// KnowledgeItem 是可增长的结构化知识，对话时按需注入 system prompt，
// 让 Agent 不再被"一次性 3 文本框"限死。
type KnowledgeItem struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	ProfileID string    `gorm:"size:32;index:idx_ki_profile" json:"profileId"`
	Topic     string    `gorm:"size:128" json:"topic"`    // 主题/问题，如"报价""合作方式"
	Content   string    `gorm:"type:text" json:"content"` // 答案正文
	Source    string    `gorm:"size:16;default:manual" json:"source"`
	Enabled   bool      `gorm:"default:true;index:idx_ki_profile" json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (KnowledgeItem) TableName() string { return "knowledge_items" }

const (
	GapOpen      = "open"      // 待主人回答
	GapAnswered  = "answered"  // 已回答（已转知识）
	GapDismissed = "dismissed" // 主人忽略
)

// KnowledgeGap 记录 Agent 答不上来的访客问题，回流给主人喂养，
// 把访客流量变成 Agent 成长的燃料。
type KnowledgeGap struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	ProfileID   string    `gorm:"size:32;index:idx_gap_profile_status" json:"profileId"`
	Question    string    `gorm:"size:512" json:"question"`
	AskedCount  int       `gorm:"default:1" json:"askedCount"`
	Status      string    `gorm:"size:16;default:open;index:idx_gap_profile_status" json:"status"`
	LastAskedAt time.Time `json:"lastAskedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (KnowledgeGap) TableName() string { return "knowledge_gaps" }

const (
	LeadNew     = "new"
	LeadHandled = "handled"
)

// Lead 是访客在对话内留下的线索（求联系/留言），主人侧 CRM 雏形。
type Lead struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	ProfileID     string    `gorm:"size:32;index:idx_lead_profile" json:"profileId"`
	VisitorOpenid string    `gorm:"size:64" json:"visitorOpenid"`
	SessionID     *string   `gorm:"size:32" json:"sessionId,omitempty"`
	Note          string    `gorm:"type:text" json:"note"`             // 访客留言
	Contact       *string   `gorm:"size:128" json:"contact,omitempty"` // 访客自填联系方式（可选）
	Status        string    `gorm:"size:16;default:new" json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (Lead) TableName() string { return "leads" }

// Connection「交换名片」：两个都建了分身的用户互相进对方名片夹的一条边（方向记发起方，
// 关系视为互相；查询「我的名片夹」时 from/to 任一是我都算）。合规上只表示「关系」，不含私信。
type Connection struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	FromUserID    string    `gorm:"size:32;index:idx_conn_from;uniqueIndex:idx_conn_pair" json:"fromUserId"`
	FromProfileID string    `gorm:"size:32" json:"fromProfileId"`
	ToUserID      string    `gorm:"size:32;index:idx_conn_to;uniqueIndex:idx_conn_pair" json:"toUserId"`
	ToProfileID   string    `gorm:"size:32" json:"toProfileId"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (Connection) TableName() string { return "connections" }

type Visit struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	ProfileID     string    `gorm:"size:32;index:idx_visit_profile_time" json:"profileId"`
	VisitorOpenid *string   `gorm:"size:64" json:"visitorOpenid,omitempty"`
	VisitorIPHash *string   `gorm:"size:64" json:"visitorIpHash,omitempty"`
	UserAgent     *string   `gorm:"size:256" json:"userAgent,omitempty"`
	CreatedAt     time.Time `gorm:"index:idx_visit_profile_time" json:"createdAt"`
}

func (Visit) TableName() string { return "visits" }

// Event 漏斗埋点事件（分享点击/裂变钩子曝光与点击/快速创建进入等）。
// 类型白名单在 internal/visit 校验；用于计算 分享→进对话→创建→再分享 漏斗与 K 值。
type Event struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	Type      string    `gorm:"size:32;index:idx_event_type_time" json:"type"`
	ProfileID string    `gorm:"size:32;index" json:"profileId"`
	Openid    *string   `gorm:"size:64" json:"openid,omitempty"`
	Meta      string    `gorm:"size:256" json:"meta"`
	CreatedAt time.Time `gorm:"index:idx_event_type_time" json:"createdAt"`
}

func (Event) TableName() string { return "events" }

type ChatSession struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	ProfileID     string    `gorm:"size:32;index:idx_session_profile_visitor" json:"profileId"`
	VisitorOpenid string    `gorm:"size:64;index:idx_session_profile_visitor" json:"visitorOpenid"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (ChatSession) TableName() string { return "chat_sessions" }

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"

	SafePending = "pending"
	SafePass    = "pass"
	SafeReject  = "reject"
)

type ChatMessage struct {
	ID              string    `gorm:"primaryKey;size:32" json:"id"`
	SessionID       string    `gorm:"size:32;index:idx_msg_session_time" json:"sessionId"`
	Role            string    `gorm:"size:16" json:"role"`
	Content         string    `gorm:"type:text" json:"content"`
	SafeCheckStatus string    `gorm:"size:16;default:pending" json:"safeCheckStatus"`
	CreatedAt       time.Time `gorm:"index:idx_msg_session_time" json:"createdAt"`
}

func (ChatMessage) TableName() string { return "chat_messages" }

// ========== 订单 ==========

const (
	OrderAgent      = "agent"      // 旧：单 Agent 次卡（已被会员制取代，保留兼容）
	OrderMembership = "membership" // 会员：一价解锁全部工具 Agent（虚拟商品，平台自营、不分账）

	OrderPending = "pending"
	OrderPaid    = "paid"
	OrderClosed  = "closed"
)

type Order struct {
	ID            string     `gorm:"primaryKey;size:32" json:"id"`
	OutTradeNo    string     `gorm:"uniqueIndex;size:64" json:"outTradeNo"`
	Type          string     `gorm:"size:16" json:"type"`
	Status        string     `gorm:"size:16;default:pending;index:idx_order_status_time" json:"status"`
	Amount        int        `json:"amount"`
	PayerOpenid   string     `gorm:"size:64;index:idx_order_payer" json:"payerOpenid"`
	PayerUserID   *string    `gorm:"size:32;index" json:"payerUserId,omitempty"`
	AgentID       *string    `gorm:"size:32" json:"agentId,omitempty"` // OrderAgent：购买的工具 Agent
	PlanID        *string    `gorm:"size:32" json:"planId,omitempty"`  // OrderMembership：购买的会员套餐
	Platform      string     `gorm:"size:16" json:"platform"`          // 下单端 ios/android/devtools（iOS 虚拟支付红线分流/兜底）
	PrepayID      *string    `gorm:"size:128" json:"prepayId,omitempty"`
	TransactionID *string    `gorm:"size:64" json:"transactionId,omitempty"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	CreatedAt     time.Time  `gorm:"index:idx_order_status_time" json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (Order) TableName() string { return "orders" }

// ========== 异步提问（访客免费向本人提问，本人异步作答） ==========

const (
	AsyncPending    = "pending"     // 待主人作答
	AsyncAnswered   = "answered"    // 主人已作答
	AsyncAIAnswered = "ai_answered" // 问答箱：分身已即时作答、主人尚未补充
)

// AsyncQuestion 来源：区分历史「异步问」与「问答箱(qabox)」。
const SourceQABox = "qabox"

// AsyncQuestion 访客向分身提问、一问一答闭环（不是私信）。「问 TA」统一走问答箱：
//   - 访客匿名问、分身据画像即时作答（AIAnswer 已填，status=ai_answered）；
//   - 分身答不好/访客点名时升温为「请本人亲自回答」（EscalatedAt 非空，主人补 Answer 后 status=answered）。
//
// 历史「异步问」(Source 空、纯 pending)接口仍保留作 App 兼容，但小程序已不再单独发起。
// 不涉及任何支付。NGL 匿名靠展示层保证（对外/对主人均不下发访客身份）。
type AsyncQuestion struct {
	ID            string     `gorm:"primaryKey;size:32" json:"id"`
	ProfileID     string     `gorm:"size:32;index" json:"profileId"`
	HostUserID    string     `gorm:"size:32;index:idx_aq_host_status" json:"hostUserId"`
	AskerOpenid   string     `gorm:"size:64;index:idx_aq_asker" json:"askerOpenid"`
	AskerUserID   *string    `gorm:"size:32" json:"askerUserId,omitempty"`
	Question      string     `gorm:"type:text" json:"question"`
	Source        string     `gorm:"size:16;index" json:"source"` // ""=异步问，qabox=问答箱
	Status        string     `gorm:"size:16;default:pending;index:idx_aq_host_status" json:"status"`
	AIAnswer      string     `gorm:"type:text" json:"aiAnswer"` // 分身即时作答（问答箱），与本人 Answer 并存
	Answer        string     `gorm:"type:text" json:"answer"`   // 主人文字回答（可与语音并存，也可为空）
	VoiceURL      string     `gorm:"type:text" json:"voiceUrl"` // 语音回答的公开 URL（空=无语音）
	VoiceDuration int        `json:"voiceDuration"`             // 语音时长（秒）
	AnsweredAt    *time.Time `json:"answeredAt,omitempty"`
	EscalatedAt   *time.Time `json:"escalatedAt,omitempty"` // 访客对 AI 已答的问题「点名请本人亲自回答」的时间（空=未点名）
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (AsyncQuestion) TableName() string { return "async_questions" }

// ========== AI 工具 Agent（平台预设、付费使用的虚拟商品；与"代表真人的对外助理"并存） ==========
//
// 与对外助理的本质差异：ToolAgent 不绑定任何真人 Profile，平台是卖家。
// 卖的是 AI 生成内容=虚拟商品，受 iOS 虚拟支付红线约束（仅 Android 端可购买，见 clientcfg）。

const (
	AgentCatEducation = "education"
	AgentCatLife      = "life"
)

// ToolAgent 平台自编的工具型 AI 角色（如英语陪练）。SystemPrompt 不下发前端。
type ToolAgent struct {
	ID           string    `gorm:"primaryKey;size:32" json:"id"`
	Slug         string    `gorm:"uniqueIndex;size:48" json:"slug"`
	Name         string    `gorm:"size:64" json:"name"`    // 能力课程主标题，如「独立判断」
	Subject      string    `gorm:"size:16" json:"subject"` // 实用学习路牌，如「逻辑思辨」
	Guide        string    `gorm:"size:48" json:"guide"`   // 课程内闯关搭档，如「明辨·逻辑侦探」
	Tagline      string    `gorm:"size:128" json:"tagline"`
	Description  string    `gorm:"type:text" json:"description"`
	Category     string    `gorm:"size:24;index" json:"category"`
	Icon         string    `gorm:"size:16" json:"icon"`   // emoji
	Accent       string    `gorm:"size:16" json:"accent"` // 主题色（前端氛围）
	Greeting     string    `gorm:"type:text" json:"greeting"`
	SystemPrompt string    `gorm:"type:text" json:"-"`
	Assess       bool      `gorm:"default:false" json:"assess"`  // 是否为「学习型」Agent：每轮评估用户能力、给三维段位升级感（如英语陪练）
	Concept      bool      `gorm:"default:false" json:"concept"` // 是否为「概念型」学习 Agent：对话中点亮该领域核心概念、可视化 X/100（如学心理/学经济/学哲学）
	Price        int       `json:"price"`                        // 一次购买价格（分）
	QuotaPerPack int       `json:"quotaPerPack"`                 // 每次购买发放的对话条数
	FreeTrial    int       `json:"freeTrial"`                    // 首次免费体验条数（非概念课/道德经试读用；概念课改用 FreeTier 幕门控）
	FreeTier     int       `json:"freeTier"`                     // 概念课免费幕阈值：非会员可免费畅用 Tier≤FreeTier 的关；更高幕需会员。0=不启用（走 FreeTrial 计次）
	Enabled      bool      `gorm:"default:true;index" json:"enabled"`
	Sort         int       `gorm:"default:0" json:"sort"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (ToolAgent) TableName() string { return "tool_agents" }

// AgentEntitlement 用户对某工具 Agent 的可用额度（次卡）。免费体验 + 已购累加，对话扣减。
type AgentEntitlement struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	UserID      string    `gorm:"size:32;uniqueIndex:idx_ent_user_agent" json:"userId"`
	AgentID     string    `gorm:"size:32;uniqueIndex:idx_ent_user_agent" json:"agentId"`
	Remaining   int       `json:"remaining"`   // 剩余可用条数
	TotalBought int       `json:"totalBought"` // 累计购买条数（>0 即已付费）
	TrialGiven  bool      `gorm:"default:false" json:"trialGiven"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (AgentEntitlement) TableName() string { return "agent_entitlements" }

// AgentPin 用户「添加到首页」的工具 Agent（首页精选 / 我的小队）。
// 与 AgentEntitlement 分离：后者管额度（会被对话扣减），这里只管首页组成与顺序。
type AgentPin struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	UserID    string    `gorm:"size:32;uniqueIndex:idx_pin_uniq;index:idx_pin_user_sort" json:"userId"`
	AgentID   string    `gorm:"size:32;uniqueIndex:idx_pin_uniq" json:"agentId"`
	Sort      int       `gorm:"index:idx_pin_user_sort" json:"sort"` // 首页展示顺序
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (AgentPin) TableName() string { return "agent_pins" }

// AgentSession 用户与某工具 Agent 的对话会话（一人一 Agent 一会话，持续累积）。
// ScriptConcept/ScriptStage：脚本课（零 AI 闯关）的状态机指针——当前关卡 slug 与所处阶段
// （tap 开场点选 / check 检验关 / review 复习挑战 / done 关卡收尾）；非脚本课恒为空。
type AgentSession struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	AgentID       string    `gorm:"size:32;index:idx_asess_agent_user" json:"agentId"`
	UserID        string    `gorm:"size:32;index:idx_asess_agent_user" json:"userId"`
	ScriptConcept string    `gorm:"size:64" json:"-"`
	ScriptStage   string    `gorm:"size:16" json:"-"`
	ScriptNode    int       `gorm:"default:0" json:"-"` // 多轮分支剧本（对手戏）当前节点下标
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (AgentSession) TableName() string { return "agent_sessions" }

type AgentMessage struct {
	ID              string    `gorm:"primaryKey;size:32" json:"id"`
	SessionID       string    `gorm:"size:32;index:idx_amsg_session_time" json:"sessionId"`
	Role            string    `gorm:"size:16" json:"role"`
	Content         string    `gorm:"type:text" json:"content"`
	Options         string    `gorm:"type:text" json:"-"` // 助手消息剥离下发的点选项（JSON []string）；供历史恢复重现气泡，否则纯点选课复原的卡片没气泡成死局。用户/无项消息为空。
	SafeCheckStatus string    `gorm:"size:16;default:pending" json:"safeCheckStatus"`
	CreatedAt       time.Time `gorm:"index:idx_amsg_session_time" json:"createdAt"`
}

func (AgentMessage) TableName() string { return "agent_messages" }

// AgentSkill 用户在某「学习型」工具 Agent（Assess=true，如英语陪练）上的能力档案。
// 三维（流利度/准确度/表达力）+ 由三维派生的总段位，给用户「升级感」。
// 设计纪律（见产品决策）：三维分数「只升不降」——状态差的一轮不掉分（消除惩罚感）；
// 首轮直接定级（baseline），其后每轮按曲线小幅向上爬（前段快后段慢），每次对话都能看到自己挪了一点。
type AgentSkill struct {
	ID         string    `gorm:"primaryKey;size:32" json:"id"`
	UserID     string    `gorm:"size:32;uniqueIndex:idx_skill_user_agent" json:"userId"`
	AgentID    string    `gorm:"size:32;uniqueIndex:idx_skill_user_agent" json:"agentId"`
	Fluency    int       `json:"fluency"`                  // 流利度（敢说·不卡）0-100
	Accuracy   int       `json:"accuracy"`                 // 准确度（说得对）0-100
	Expression int       `json:"expression"`               // 表达力（说得地道/高级）0-100
	Assessed   int       `json:"assessed"`                 // 已评估轮次（0 = 尚未定级）
	LastNote   string    `gorm:"size:255" json:"lastNote"` // 最近一次涨分归因（"因为你把 very like 升级成 really into"）
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (AgentSkill) TableName() string { return "agent_skills" }

// AgentConcept 某「概念型」学习 Agent（Concept=true）的课程表：该领域的核心概念清单（平台 seed，按 agent_id+slug 幂等）。
// 与 AgentSkill 的三维段位是两套互补的进度机制：技能型走三维分数，概念型走「点亮 X/100」。
type AgentConcept struct {
	ID       string `gorm:"primaryKey;size:32" json:"id"`
	AgentID  string `gorm:"size:32;uniqueIndex:idx_concept_agent_slug" json:"agentId"` // FK ToolAgent
	Slug     string `gorm:"size:64;uniqueIndex:idx_concept_agent_slug" json:"slug"`    // Agent 内稳定 id，如 "anchoring"
	Theme    string `gorm:"size:48;index" json:"theme"`                                // 主题分组，如 "认知偏误"
	Tier     int    `gorm:"default:1" json:"tier"`                                     // 分档：1 入门 / 2 进阶（成就感按档给，避免大分母劝退）
	Name     string `gorm:"size:64" json:"name"`                                       // 概念名，如 "锚定效应"
	Blurb    string `gorm:"size:255" json:"blurb"`                                     // 一句话点题（前端展示 + 给打点 LLM 锚定）
	Hook     string `gorm:"type:text" json:"-"`                                        // 人工精编：开课钩子（生活场景问题，导师用它开场）；空=未精编，模型自拟。text：帛书课含整章原文，远超 255
	Check    string `gorm:"type:text" json:"-"`                                        // 人工精编：检验题（应用/迁移型，讲透后用它检验；复习挑战也用它）
	Takeaway string `gorm:"type:text" json:"-"`                                        // 策展知识卡片一句话结论：仅章末 Boss 关非空，通关时弹卡展示
	Source   string `gorm:"size:64" json:"-"`                                          // 知识卡引用来源短标注，如 "Gottman 情绪实验室"
	Sort     int    `gorm:"default:0" json:"sort"`
}

func (AgentConcept) TableName() string { return "agent_concepts" }

// UserConcept 用户在某概念上的掌握档位。Level：0 未触及 / 1 已点亮 / 2 已掌握。
// 设计纪律同 AgentSkill：「只升不降」——状态差的一轮不掉档（消除惩罚感），每次对话都可能点亮新概念。
// 间隔重复（2026-07-16）：ReviewCount/ReviewDue 驱动扩展式复习调度——复习答对间隔翻档（3→7→14→30→60天），
// 答错回到次日重来（只缩间隔不降 Level，惩罚落在日程不落在成就上）。ReviewDue 零值=老数据，读取端按旧 3/7 天规则兜底。
type UserConcept struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	UserID      string    `gorm:"size:32;uniqueIndex:idx_uc_user_concept" json:"userId"`
	ConceptID   string    `gorm:"size:32;uniqueIndex:idx_uc_user_concept" json:"conceptId"`
	AgentID     string    `gorm:"size:32;index:idx_uc_user_agent" json:"agentId"` // 冗余，便于按 Agent 聚合 X/100
	Level       int       `json:"level"`                                          // 0/1/2
	Touches     int       `json:"touches"`                                        // 命中次数
	Note        string    `gorm:"size:120" json:"note"`                           // 本课战报：判定器一句话（≤20字），latest-wins；空轮不覆盖
	ReviewCount int       `json:"reviewCount"`                                    // 连续复习答对次数（决定当前间隔档；答错清零）
	ReviewDue   time.Time `json:"reviewDue"`                                      // 下次到期复习时间（零值=旧数据，按旧规则兜底）
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (UserConcept) TableName() string { return "user_concepts" }

// LearnStreak 用户的连续学习天数（跨全部学习型 Agent 全局一条；技能型/概念型对话即记一天）。
// 温和版纪律：每自然月可自动补签 1 天（断一天不清零），拒绝焦虑轰炸；日期按东八区。
type LearnStreak struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	UserID      string    `gorm:"size:32;uniqueIndex" json:"userId"`
	Current     int       `json:"current"`                   // 当前连续天数
	Best        int       `json:"best"`                      // 历史最佳
	LastDay     string    `gorm:"size:10" json:"lastDay"`    // 最近学习日 YYYY-MM-DD（东八区）
	FreezeMonth string    `gorm:"size:7" json:"freezeMonth"` // 最近一次自动补签的月份 YYYY-MM（每月限 1 次）
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (LearnStreak) TableName() string { return "learn_streaks" }

// LearnReminder 学习提醒承诺：用户在课后点「明天叫我」并授权一次性订阅消息后建一条，
// 到点由后台循环发一条订阅消息（一次授权=一次发送，微信硬约束）。发过即 Sent，不重试不轰炸。
type LearnReminder struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	UserID    string    `gorm:"size:32;index:idx_lr_user_agent" json:"userId"`
	AgentID   string    `gorm:"size:32;index:idx_lr_user_agent" json:"agentId"`
	SendAt    time.Time `gorm:"index" json:"sendAt"` // 计划发送时间（=承诺时刻 +24h，"明天这个点"）
	Sent      bool      `gorm:"index" json:"sent"`
	CreatedAt time.Time `json:"createdAt"`
}

func (LearnReminder) TableName() string { return "learn_reminders" }

// ========== 会员（一价解锁全部工具 Agent；虚拟商品，平台自营） ==========

// Membership 账号级会员状态（一人一条）。ExpiresAt 之前为有效会员，工具 Agent 畅用。
// 渠道无关：微信支付 / 将来的 H5 / 支付宝 都往这条状态叠加（续费顺延）。
type Membership struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	UserID    string    `gorm:"uniqueIndex;size:32" json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Membership) TableName() string { return "memberships" }

// MembershipPlan 会员套餐（周期通行证，非自动续费）。平台定价，首启种子。
type MembershipPlan struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	Slug      string    `gorm:"uniqueIndex;size:32" json:"slug"`
	Name      string    `gorm:"size:32" json:"name"`
	Days      int       `json:"days"`                     // 时长（天）
	Price     int       `json:"price"`                    // 现价（分）
	OrigPrice int       `json:"origPrice"`                // 划线原价（分）；0=不展示。锚定用（如年付现价 119、划线 199）
	ProductID string    `gorm:"size:64" json:"productId"` // 米大师商品 ID（虚拟支付商品直购；空则回退用 goodsPrice 现价）
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (MembershipPlan) TableName() string { return "membership_plans" }

// MembershipLead 留资意向（多为 iOS 用户：当下不能在小程序内开通，记录意向便于服务号/站外触达）。
type MembershipLead struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	UserID    string    `gorm:"size:32;index" json:"userId"`
	Openid    string    `gorm:"size:64" json:"openid"`
	Platform  string    `gorm:"size:16" json:"platform"`
	CreatedAt time.Time `json:"createdAt"`
}

func (MembershipLead) TableName() string { return "membership_leads" }

// ========== 好友邀请返奖（推荐好友开通会员，双边送会员时长） ==========
//
// 合规边界：奖励只挂「好友完成支付」不挂分享动作（微信利诱分享红线）；
// 只做一级（推荐人→好友）、只送时长不返现金；奖励过观察期后由定时任务发放。

// ReferralBinding 邀请绑定：被邀人首次通过邀请链接进入会员页时写入（每人只绑一次，先到先得）。
// 只在被邀人尚无已支付会员订单时允许绑定；奖励发放后保留作历史归因。
type ReferralBinding struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	InviteeUserID string    `gorm:"uniqueIndex;size:32" json:"inviteeUserId"`
	InviterUserID string    `gorm:"size:32;index" json:"inviterUserId"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (ReferralBinding) TableName() string { return "referral_bindings" }

const (
	ReferralRewardPending = "pending" // 推荐人奖励等待观察期结束
	ReferralRewardGranted = "granted" // 已发放
)

// ReferralReward 一笔成功邀请的奖励账目（每个被邀人一生只产生一条）。
// 被邀人加赠（InviteeDays）支付成功即发；推荐人奖励（InviterDays）到 UnlockAt 后由定时任务发。
type ReferralReward struct {
	ID            string     `gorm:"primaryKey;size:32" json:"id"`
	OrderID       string     `gorm:"uniqueIndex;size:32" json:"orderId"`
	InviterUserID string     `gorm:"size:32;index" json:"inviterUserId"`
	InviteeUserID string     `gorm:"size:32;index" json:"inviteeUserId"`
	PlanSlug      string     `gorm:"size:32" json:"planSlug"`
	InviterDays   int        `json:"inviterDays"`
	InviteeDays   int        `json:"inviteeDays"`
	Status        string     `gorm:"size:16;default:pending;index" json:"status"`
	UnlockAt      time.Time  `gorm:"index" json:"unlockAt"`
	GrantedAt     *time.Time `json:"grantedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (ReferralReward) TableName() string { return "referral_rewards" }

// ========== 代理商入驻 ==========

const (
	AgencyApplicationPending   = "pending"
	AgencyApplicationApproved  = "approved"
	AgencyApplicationRejected  = "rejected"
	AgencyApplicationSuspended = "suspended"
)

// AgencyApplication 代理商入驻资料。
// 当前采用自动通过：首次提交即 approved；运营侧仍可暂停资格，方便后续接入风控。
type AgencyApplication struct {
	ID           string     `gorm:"primaryKey;size:32" json:"id"`
	UserID       string     `gorm:"uniqueIndex;size:32" json:"userId"`
	AgencyCode   *string    `gorm:"uniqueIndex;size:16" json:"agencyCode,omitempty"`
	Name         string     `gorm:"size:64" json:"name"`
	Phone        string     `gorm:"size:32" json:"phone"`
	Region       string     `gorm:"size:128" json:"region"`
	ChannelType  string     `gorm:"size:32" json:"channelType"`
	AudienceSize string     `gorm:"size:32" json:"audienceSize"`
	Experience   string     `gorm:"type:text" json:"experience"`
	InviteCode   string     `gorm:"size:64;index" json:"inviteCode,omitempty"`
	Status       string     `gorm:"size:16;default:pending;index" json:"status"`
	ReviewNote   string     `gorm:"type:text" json:"reviewNote,omitempty"`
	ConsentAt    time.Time  `json:"consentAt"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (AgencyApplication) TableName() string { return "agency_applications" }

// AgencyUserBinding 代理商邀请归因。每个用户只绑定一次，先到先得；
// 付费统计只计算绑定时间之后完成的会员订单。
type AgencyUserBinding struct {
	ID            string    `gorm:"primaryKey;size:32" json:"id"`
	AgencyUserID  string    `gorm:"size:32;index:idx_agency_binding_owner_time" json:"agencyUserId"`
	InviteeUserID string    `gorm:"uniqueIndex;size:32" json:"inviteeUserId"`
	AgencyCode    string    `gorm:"size:16;index" json:"agencyCode"`
	NewUser       bool      `gorm:"default:false;index" json:"newUser"`
	CreatedAt     time.Time `gorm:"index:idx_agency_binding_owner_time" json:"createdAt"`
}

func (AgencyUserBinding) TableName() string { return "agency_user_bindings" }

// AllModels 用于 AutoMigrate
func AllModels() []interface{} {
	return []interface{}{
		&User{}, &Profile{}, &PersonaInput{}, &PersonaAI{},
		&KnowledgeItem{}, &KnowledgeGap{}, &Lead{}, &Connection{},
		&Visit{}, &Event{}, &ChatSession{}, &ChatMessage{},
		&Order{},
		&AsyncQuestion{},
		&ToolAgent{}, &AgentEntitlement{}, &AgentPin{}, &AgentSession{}, &AgentMessage{}, &AgentSkill{},
		&AgentConcept{}, &UserConcept{}, &LearnStreak{}, &LearnReminder{},
		&Membership{}, &MembershipPlan{}, &MembershipLead{},
		&ReferralBinding{}, &ReferralReward{},
		&AgencyApplication{}, &AgencyUserBinding{},
	}
}
