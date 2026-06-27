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
	Unionid         *string   `gorm:"uniqueIndex;size:64" json:"unionid,omitempty"`
	Nickname        *string   `gorm:"size:128" json:"nickname,omitempty"`
	AvatarURL       *string   `gorm:"size:512" json:"avatarUrl,omitempty"`
	PsReceiverAdded bool      `gorm:"default:false" json:"psReceiverAdded"`
	// 最近一次小程序 session_key（虚拟支付发货确认 NotifyProvideGoods 的离线兜底；会过期）。
	WxSessionKey    *string   `gorm:"size:64" json:"-"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
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

// ========== 付费 ==========

type ConsultSetting struct {
	ID      string  `gorm:"primaryKey;size:32" json:"id"`
	UserID  string  `gorm:"uniqueIndex;size:32" json:"userId"`
	Enabled bool    `gorm:"default:false" json:"enabled"`
	Price30 int     `gorm:"default:9900" json:"price30"`
	Price60 int     `gorm:"default:19900" json:"price60"`
	Intro   *string `gorm:"type:text" json:"intro,omitempty"`
	// 付费异步咨询（独立于音视频咨询开关，可只开异步）。
	AsyncEnabled bool      `gorm:"default:false" json:"asyncEnabled"`
	AsyncPrice   int       `gorm:"default:4900" json:"asyncPrice"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (ConsultSetting) TableName() string { return "consult_settings" }

const (
	OrderTip           = "tip"
	OrderConsult       = "consult"
	OrderAsyncQuestion = "async_question"
	OrderAgent         = "agent"      // 旧：单 Agent 次卡（已被会员制取代，保留兼容）
	OrderMembership    = "membership" // 会员：一价解锁全部工具 Agent（虚拟商品，平台自营、不分账）

	OrderPending   = "pending"
	OrderPaid      = "paid"
	OrderRefunding = "refunding"
	OrderRefunded  = "refunded"
	OrderClosed    = "closed"
)

type Order struct {
	ID            string     `gorm:"primaryKey;size:32" json:"id"`
	OutTradeNo    string     `gorm:"uniqueIndex;size:64" json:"outTradeNo"`
	Type          string     `gorm:"size:16" json:"type"`
	Status        string     `gorm:"size:16;default:pending;index:idx_order_status_time" json:"status"`
	Amount        int        `json:"amount"`
	ProfileID     string     `gorm:"size:32" json:"profileId"`
	PayerOpenid   string     `gorm:"size:64;index:idx_order_payer" json:"payerOpenid"`
	PayerUserID   *string    `gorm:"size:32" json:"payerUserId,omitempty"`
	PayeeUserID   string     `gorm:"size:32;index:idx_order_payee" json:"payeeUserId"`
	DurationMin   *int       `json:"durationMin,omitempty"`
	SlotID        *string    `gorm:"uniqueIndex;size:32" json:"slotId,omitempty"`
	ScheduledAt   *time.Time `json:"scheduledAt,omitempty"`
	Message       *string    `gorm:"type:text" json:"message,omitempty"`
	Source        string     `gorm:"size:16;default:profile;index" json:"source"` // 成交来源：profile / chat_card
	AgentID       *string    `gorm:"size:32" json:"agentId,omitempty"`            // OrderAgent：购买的工具 Agent
	PlanID        *string    `gorm:"size:32" json:"planId,omitempty"`             // OrderMembership：购买的会员套餐
	Platform      string     `gorm:"size:16" json:"platform"`                     // 下单端 ios/android/devtools（iOS 虚拟支付红线分流/兜底）
	PrepayID      *string    `gorm:"size:128" json:"prepayId,omitempty"`
	TransactionID *string    `gorm:"size:64" json:"transactionId,omitempty"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	CreatedAt     time.Time  `gorm:"index:idx_order_status_time" json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (Order) TableName() string { return "orders" }

const (
	ConsultPending  = "pending"
	ConsultOngoing  = "ongoing"
	ConsultEnded    = "ended"
	ConsultCanceled = "canceled"
)

type ConsultSession struct {
	ID          string     `gorm:"primaryKey;size:32" json:"id"`
	OrderID     string     `gorm:"uniqueIndex;size:32" json:"orderId"`
	ProfileID   string     `gorm:"size:32" json:"profileId"`
	HostUserID  string     `gorm:"size:32;index:idx_cs_host" json:"hostUserId"`
	GuestOpenid string     `gorm:"size:64;index:idx_cs_guest" json:"guestOpenid"`
	TrtcRoomID  string     `gorm:"uniqueIndex;size:64" json:"trtcRoomId"`
	Status      string     `gorm:"size:16;default:pending" json:"status"`
	DurationMin int        `json:"durationMin"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	DurationSec *int       `json:"durationSec,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (ConsultSession) TableName() string { return "consult_sessions" }

const (
	SlotOpen     = "open"
	SlotBooked   = "booked"
	SlotCanceled = "canceled"
)

type ConsultSlot struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	HostUserID  string    `gorm:"size:32;index:idx_slot_host" json:"hostUserId"`
	StartAt     time.Time `gorm:"index:idx_slot_host" json:"startAt"`
	DurationMin int       `json:"durationMin"`
	Status      string    `gorm:"size:16;default:open" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (ConsultSlot) TableName() string { return "consult_slots" }

// ========== 付费异步咨询（付费向本人提问，本人异步作答） ==========

const (
	AsyncPendingPayment = "pending_payment" // 已创建，待支付
	AsyncPaid           = "paid"            // 已支付，待主人作答
	AsyncAnswered       = "answered"        // 主人已作答
	AsyncRefunded       = "refunded"        // 超时未答自动退款 / 退款
)

// AsyncQuestion 访客付费向主人提问，主人在 SLA 时限内异步作答；
// 超时未答由定时任务自动全额退款。付费购买的是「主人本人作答」（真人服务），非 AI 回答。
type AsyncQuestion struct {
	ID             string     `gorm:"primaryKey;size:32" json:"id"`
	OrderID        string     `gorm:"uniqueIndex;size:32" json:"orderId"`
	ProfileID      string     `gorm:"size:32;index" json:"profileId"`
	HostUserID     string     `gorm:"size:32;index:idx_aq_host_status" json:"hostUserId"`
	AskerOpenid    string     `gorm:"size:64;index:idx_aq_asker" json:"askerOpenid"`
	AskerUserID    *string    `gorm:"size:32" json:"askerUserId,omitempty"`
	Question       string     `gorm:"type:text" json:"question"`
	Price          int        `json:"price"` // 下单时主人定价快照（分）
	Status         string     `gorm:"size:16;default:pending_payment;index:idx_aq_host_status" json:"status"`
	PaidAt         *time.Time `json:"paidAt,omitempty"`
	AnswerDeadline *time.Time `json:"answerDeadline,omitempty"`
	Answer         string     `gorm:"type:text" json:"answer"`   // 文字回答（可与语音并存，也可为空）
	VoiceURL       string     `gorm:"type:text" json:"voiceUrl"` // 语音回答的公开 URL（空=无语音）
	VoiceDuration  int        `json:"voiceDuration"`             // 语音时长（秒）
	AnsweredAt     *time.Time `json:"answeredAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (AsyncQuestion) TableName() string { return "async_questions" }

const (
	RefundProcessing = "processing"
	RefundSuccess    = "success"
	RefundFail       = "fail"
)

type Refund struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	OrderID     string    `gorm:"size:32;index" json:"orderId"`
	OutRefundNo string    `gorm:"uniqueIndex;size:64" json:"outRefundNo"`
	Amount      int       `json:"amount"`
	Reason      *string   `gorm:"size:128" json:"reason,omitempty"`
	Status      string    `gorm:"size:16;default:processing" json:"status"`
	RefundID    *string   `gorm:"size:64" json:"refundId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (Refund) TableName() string { return "refunds" }

const (
	PSProcessing = "processing"
	PSSuccess    = "success"
	PSFinished   = "finished"
	PSFail       = "fail"
)

type ProfitShare struct {
	ID          string    `gorm:"primaryKey;size:32" json:"id"`
	OrderID     string    `gorm:"uniqueIndex;size:32" json:"orderId"`
	OutOrderNo  string    `gorm:"uniqueIndex;size:64" json:"outOrderNo"`
	PlatformFee int       `json:"platformFee"`
	PayeeAmount int       `json:"payeeAmount"`
	Status      string    `gorm:"size:16;default:processing" json:"status"`
	WxOrderID   *string   `gorm:"size:64" json:"wxOrderId,omitempty"`
	Finished    bool      `gorm:"default:false" json:"finished"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (ProfitShare) TableName() string { return "profit_shares" }

// ========== AI 工具 Agent（平台预设、付费使用的虚拟商品；与"代表真人的对外助理"并存） ==========
//
// 与对外助理的本质差异：ToolAgent 不绑定任何真人 Profile，平台是卖家（无真人 payee、不分账）。
// 卖的是 AI 生成内容=虚拟商品，受 iOS 虚拟支付红线约束（仅 Android 端可购买，见 clientcfg）。

const (
	AgentCatEducation = "education"
	AgentCatCareer    = "career"
	AgentCatLife      = "life"
)

// ToolAgent 平台自编的工具型 AI 角色（如英语陪练）。SystemPrompt 不下发前端。
type ToolAgent struct {
	ID           string    `gorm:"primaryKey;size:32" json:"id"`
	Slug         string    `gorm:"uniqueIndex;size:48" json:"slug"`
	Name         string    `gorm:"size:64" json:"name"`
	Tagline      string    `gorm:"size:128" json:"tagline"`
	Description  string    `gorm:"type:text" json:"description"`
	Category     string    `gorm:"size:24;index" json:"category"`
	Icon         string    `gorm:"size:16" json:"icon"`   // emoji
	Accent       string    `gorm:"size:16" json:"accent"` // 主题色（前端氛围）
	Greeting     string    `gorm:"type:text" json:"greeting"`
	SystemPrompt string    `gorm:"type:text" json:"-"`
	Price        int       `json:"price"`        // 一次购买价格（分）
	QuotaPerPack int       `json:"quotaPerPack"` // 每次购买发放的对话条数
	FreeTrial    int       `json:"freeTrial"`    // 首次免费体验条数
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

// AgentSession 用户与某工具 Agent 的对话会话（一人一 Agent 一会话，持续累积）。
type AgentSession struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	AgentID   string    `gorm:"size:32;index:idx_asess_agent_user" json:"agentId"`
	UserID    string    `gorm:"size:32;index:idx_asess_agent_user" json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (AgentSession) TableName() string { return "agent_sessions" }

type AgentMessage struct {
	ID              string    `gorm:"primaryKey;size:32" json:"id"`
	SessionID       string    `gorm:"size:32;index:idx_amsg_session_time" json:"sessionId"`
	Role            string    `gorm:"size:16" json:"role"`
	Content         string    `gorm:"type:text" json:"content"`
	SafeCheckStatus string    `gorm:"size:16;default:pending" json:"safeCheckStatus"`
	CreatedAt       time.Time `gorm:"index:idx_amsg_session_time" json:"createdAt"`
}

func (AgentMessage) TableName() string { return "agent_messages" }

// ========== 会员（一价解锁全部工具 Agent；虚拟商品，平台自营、不分账） ==========

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
	Days      int       `json:"days"`  // 时长（天）
	Price     int       `json:"price"` // 价格（分）
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

// AllModels 用于 AutoMigrate
func AllModels() []interface{} {
	return []interface{}{
		&User{}, &Profile{}, &PersonaInput{}, &PersonaAI{},
		&KnowledgeItem{}, &KnowledgeGap{}, &Lead{},
		&Visit{}, &Event{}, &ChatSession{}, &ChatMessage{},
		&ConsultSetting{}, &Order{}, &ConsultSession{}, &ConsultSlot{},
		&AsyncQuestion{},
		&Refund{}, &ProfitShare{},
		&ToolAgent{}, &AgentEntitlement{}, &AgentSession{}, &AgentMessage{},
		&Membership{}, &MembershipPlan{}, &MembershipLead{},
	}
}
