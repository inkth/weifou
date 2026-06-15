package app

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"weifou-server/internal/asyncq"
	"weifou-server/internal/auth"
	"weifou-server/internal/chat"
	"weifou-server/internal/config"
	"weifou-server/internal/consult"
	"weifou-server/internal/deepseek"
	"weifou-server/internal/payment"
	"weifou-server/internal/persona"
	"weifou-server/internal/plaza"
	"weifou-server/internal/profile"
	"weifou-server/internal/rtc"
	"weifou-server/internal/share"
	"weifou-server/internal/tasks"
	"weifou-server/internal/user"
	"weifou-server/internal/visit"
	"weifou-server/internal/wechat"
	"weifou-server/internal/wxpay"
)

// App 持有所有 handler 与定时任务调度器。
type App struct {
	cfg *config.Config

	authH    *auth.Handler
	userH    *user.Handler
	profileH *profile.Handler
	chatH    *chat.Handler
	visitH   *visit.Handler
	shareH   *share.Handler
	consultH *consult.Handler
	paymentH *payment.Handler
	asyncqH  *asyncq.Handler
	rtcH     *rtc.Handler
	plazaH   *plaza.Handler

	scheduler *tasks.Scheduler
}

func New(cfg *config.Config, db *gorm.DB, rdb *redis.Client) *App {
	// 外部客户端
	loginClient := wechat.NewLoginClient(cfg.WxAppID, cfg.WxAppSecret)
	// 移动应用 oauth2 客户端（未配置时为 nil，App 登录会返回明确错误）
	var appLoginClient *wechat.LoginClient
	if cfg.WxMobileAppID != "" && cfg.WxMobileSecret != "" {
		appLoginClient = wechat.NewLoginClient(cfg.WxMobileAppID, cfg.WxMobileSecret)
	}
	security := wechat.NewSecurityService(loginClient)
	subscribe := wechat.NewSubscribeService(loginClient, cfg.SubscribeNewQuestionTmpl, cfg.SubscribeAnsweredTmpl, cfg.SubscribeRefundedTmpl, cfg.SubscribeMiniState)
	ds := deepseek.New(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)
	payClient := wxpay.New(wxpay.Config{
		AppID:            cfg.WxAppID,
		MchID:            cfg.WxPayMchID,
		SerialNo:         cfg.WxPayCertSerial,
		APIV3Key:         cfg.WxPayAPIV3Key,
		NotifyURL:        cfg.WxPayNotifyURL,
		PrivateKeyPath:   cfg.WxPayPrivateKeyPath,
		PlatformCertPath: cfg.WxPayPlatformCert,
	})

	// 业务服务
	personaSvc := persona.NewService(db, ds, security)
	profitShare := payment.NewProfitShareService(db, payClient, cfg.ProfitSharing, cfg.PlatformFeeRate)
	paymentH := payment.NewHandler(db, payClient, security, profitShare, subscribe, cfg.JWTSecret, cfg.TipMaxAmount, cfg.AsyncQSLAHours)

	a := &App{
		cfg:      cfg,
		authH:    auth.NewHandler(db, loginClient, appLoginClient, cfg.JWTSecret, cfg.JWTExpiresHours, cfg.Env),
		userH:    user.NewHandler(db, cfg.JWTSecret),
		profileH: profile.NewHandler(db, personaSvc, cfg.JWTSecret),
		chatH:    chat.NewHandler(db, rdb, ds, security, cfg.JWTSecret, cfg.ChatFreeQuotaPerDay),
		visitH:   visit.NewHandler(db, cfg.JWTSecret),
		shareH:   share.NewHandler(db, loginClient),
		consultH: consult.NewHandler(db, cfg.JWTSecret),
		paymentH: paymentH,
		asyncqH:  asyncq.NewHandler(db, paymentH, security, subscribe, cfg.JWTSecret),
		rtcH: rtc.NewHandler(db, profitShare, rtc.Config{
			JWTSecret:    cfg.JWTSecret,
			SdkAppID:     cfg.TRTCSdkAppID,
			SecretKey:    cfg.TRTCSecret,
			SigExpire:    cfg.TRTCSigExp,
			EarlyJoinMin: cfg.CallEarlyJoinMin,
			GraceMin:     cfg.CallGraceMin,
		}),
		plazaH:    plaza.NewHandler(db),
		scheduler: tasks.NewScheduler(db, paymentH, cfg.OrderTimeoutMin, cfg.CallGraceMin),
	}
	return a
}

func (a *App) RegisterRoutes(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	api := r.Group("/api")
	a.authH.Register(api)
	a.userH.Register(api)
	a.profileH.Register(api)
	a.chatH.Register(api)
	a.visitH.Register(api)
	a.shareH.Register(api)
	a.consultH.Register(api)
	a.paymentH.Register(api)
	a.asyncqH.Register(api)
	a.rtcH.Register(api)
	a.plazaH.Register(api)
}

func (a *App) StartCron() {
	a.scheduler.Start()
}
