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
	PlatformFeeRate     float64
	ProfitSharing       bool
	OrderTimeoutMin     int
	CallEarlyJoinMin    int
	CallGraceMin        int

	// TRTC
	TRTCSdkAppID int
	TRTCSecret   string
	TRTCSigExp   int

	// 付费异步咨询
	AsyncQSLAHours           int    // 主人作答时限（小时），超时自动退款
	SubscribeNewQuestionTmpl string // 订阅消息模板：新付费提问→主人
	SubscribeAnsweredTmpl    string // 订阅消息模板：已回答→访客
	SubscribeRefundedTmpl    string // 订阅消息模板：已退款→访客
	SubscribeMiniState       string // formal / trial / developer
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

func getFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
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
		PlatformFeeRate:     getFloat("PLATFORM_FEE_RATE", 0),
		ProfitSharing:       getBool("PROFIT_SHARING_ENABLED", false),
		OrderTimeoutMin:     getInt("ORDER_TIMEOUT_MIN", 15),
		CallEarlyJoinMin:    getInt("CALL_EARLY_JOIN_MIN", 5),
		CallGraceMin:        getInt("CALL_GRACE_MIN", 15),

		TRTCSdkAppID: getInt("TRTC_SDK_APPID", 0),
		TRTCSecret:   os.Getenv("TRTC_SECRET_KEY"),
		TRTCSigExp:   getInt("TRTC_SIG_EXPIRE", 86400),

		AsyncQSLAHours:           getInt("ASYNCQ_SLA_HOURS", 48),
		SubscribeNewQuestionTmpl: os.Getenv("WX_SUBSCRIBE_NEW_QUESTION_TMPL_ID"),
		SubscribeAnsweredTmpl:    os.Getenv("WX_SUBSCRIBE_ANSWERED_TMPL_ID"),
		SubscribeRefundedTmpl:    os.Getenv("WX_SUBSCRIBE_REFUNDED_TMPL_ID"),
		SubscribeMiniState:       getEnv("WX_SUBSCRIBE_STATE", "formal"),
	}
}
