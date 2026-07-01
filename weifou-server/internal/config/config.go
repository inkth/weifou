package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env        string
	Port       string
	AppBaseURL string

	WxAppID     string
	WxAppSecret string

	// 微信开放平台「移动应用」凭证（原生 App 登录/支付/分享用，区别于小程序）
	WxMobileAppID  string
	WxMobileSecret string

	JWTSecret       string
	JWTExpiresHours int

	DatabaseURL string
	RedisURL    string

	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string

	ChatFreeQuotaPerDay int

	// 微信支付
	WxPayMchID          string
	WxPayAPIV3Key       string
	WxPayCertSerial     string
	WxPayPrivateKeyPath string
	WxPayPlatformCert   string
	WxPayNotifyURL      string
	TipMaxAmount        int
	OrderTimeoutMin     int
	CallEarlyJoinMin    int
	CallGraceMin        int

	// 微信小程序虚拟支付（虚拟商品=会员；iOS 自动走苹果 IAP，2026-04 起强制接入）。
	// appID 复用 WxAppID（同一小程序）；offerId/appKey 来自开通虚拟支付后的米大师控制台。
	WxvOfferID string
	WxvAppKey  string
	WxvSandbox bool

	// TRTC
	TRTCSdkAppID int
	TRTCSecret   string
	TRTCSigExp   int

	// 付费异步咨询
	AsyncQSLAHours           int    // 主人作答时限（小时），超时自动退款
	SubscribeNewQuestionTmpl string // 订阅消息模板：新付费提问→主人
	SubscribeAnsweredTmpl    string // 订阅消息模板：已回答→访客
	SubscribeRefundedTmpl    string // 订阅消息模板：已退款→访客
	SubscribeLeadTmpl        string // 订阅消息模板：新访客线索→主人
	SubscribeMiniState       string // formal / trial / developer

	// 服务号（公众号）：承接 iOS 会员开通引导 + 召回。未配则相关功能 no-op。
	MpAppID   string
	MpSecret  string
	MpToken   string // 服务号消息回调校验 token（公众平台配明文/兼容模式）
	H5BaseURL string // 服务端构造 H5 链接的公网基址（服务号推送无 request 上下文）

	// 文件上传 / 静态服务（当前：付费提问语音回答，存本地盘命名卷；未来可换 COS）
	UploadDir  string // 容器内可写目录（挂 docker 命名卷持久化）
	PublicHost string // 生成公开 URL 的公网基址；空则回落 AppBaseURL

	// 音乐生成（做音乐 Agent；默认 fal，复用 FALAI_API_KEY）。未配 key 则做音乐 no-op。
	MusicProvider string // fal（默认）
	MusicAPIKey   string // FALAI_API_KEY
	MusicBaseURL  string // https://fal.run
	MusicModel    string // fal-ai/ace-step
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1"
	}
	return def
}

// Load 从 .env（若存在）与环境变量加载配置
func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Env:        getEnv("ENV", "development"),
		Port:       getEnv("PORT", "3000"),
		AppBaseURL: getEnv("APP_BASE_URL", "http://localhost:3000"),

		WxAppID:     os.Getenv("WX_APPID"),
		WxAppSecret: os.Getenv("WX_APPSECRET"),

		WxMobileAppID:  os.Getenv("WX_MOBILE_APPID"),
		WxMobileSecret: os.Getenv("WX_MOBILE_APPSECRET"),

		JWTSecret:       getEnv("JWT_SECRET", "please-change-me"),
		JWTExpiresHours: getInt("JWT_EXPIRES_HOURS", 720),

		DatabaseURL: getEnv("DATABASE_URL", "postgresql://weifou:weifou@localhost:5432/weifou?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),

		DeepSeekAPIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:   getEnv("DEEPSEEK_MODEL", "deepseek-chat"),

		ChatFreeQuotaPerDay: getInt("CHAT_FREE_QUOTA_PER_DAY", 10),

		WxPayMchID:          os.Getenv("WXPAY_MCHID"),
		WxPayAPIV3Key:       os.Getenv("WXPAY_API_V3_KEY"),
		WxPayCertSerial:     os.Getenv("WXPAY_CERT_SERIAL"),
		WxPayPrivateKeyPath: os.Getenv("WXPAY_PRIVATE_KEY_PATH"),
		WxPayPlatformCert:   os.Getenv("WXPAY_PLATFORM_CERT_PATH"),
		WxPayNotifyURL:      os.Getenv("WXPAY_NOTIFY_URL"),
		TipMaxAmount:        getInt("TIP_MAX_AMOUNT", 50000),
		OrderTimeoutMin:     getInt("ORDER_TIMEOUT_MIN", 15),
		CallEarlyJoinMin:    getInt("CALL_EARLY_JOIN_MIN", 5),
		CallGraceMin:        getInt("CALL_GRACE_MIN", 15),

		WxvOfferID: os.Getenv("WXV_OFFER_ID"),
		WxvAppKey:  os.Getenv("WXV_APP_KEY"),
		WxvSandbox: getBool("WXV_SANDBOX", false),

		TRTCSdkAppID: getInt("TRTC_SDK_APPID", 0),
		TRTCSecret:   os.Getenv("TRTC_SECRET_KEY"),
		TRTCSigExp:   getInt("TRTC_SIG_EXPIRE", 86400),

		AsyncQSLAHours:           getInt("ASYNCQ_SLA_HOURS", 48),
		SubscribeNewQuestionTmpl: os.Getenv("WX_SUBSCRIBE_NEW_QUESTION_TMPL_ID"),
		SubscribeAnsweredTmpl:    os.Getenv("WX_SUBSCRIBE_ANSWERED_TMPL_ID"),
		SubscribeRefundedTmpl:    os.Getenv("WX_SUBSCRIBE_REFUNDED_TMPL_ID"),
		SubscribeLeadTmpl:        os.Getenv("WX_SUBSCRIBE_LEAD_TMPL_ID"),
		SubscribeMiniState:       getEnv("WX_SUBSCRIBE_STATE", "formal"),

		MpAppID:   os.Getenv("WX_MP_APPID"),
		MpSecret:  os.Getenv("WX_MP_SECRET"),
		MpToken:   os.Getenv("WX_MP_TOKEN"),
		H5BaseURL: getEnv("H5_BASE_URL", ""),

		UploadDir:  getEnv("UPLOAD_DIR", "./uploads"),
		PublicHost: getEnv("PUBLIC_HOST", ""),

		MusicProvider: getEnv("MUSIC_PROVIDER", "fal"),
		MusicAPIKey:   getEnv("MUSIC_API_KEY", os.Getenv("FALAI_API_KEY")), // 复用 fal key
		MusicBaseURL:  getEnv("MUSIC_BASE_URL", "https://fal.run"),
		MusicModel:    getEnv("MUSIC_MODEL", "fal-ai/ace-step"),
	}
}
