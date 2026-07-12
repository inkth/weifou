package app

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"weifou-server/internal/answer"
	"weifou-server/internal/asyncq"
	"weifou-server/internal/auth"
	"weifou-server/internal/chat"
	"weifou-server/internal/clientcfg"
	"weifou-server/internal/config"
	"weifou-server/internal/connection"
	"weifou-server/internal/deepseek"
	"weifou-server/internal/home"
	"weifou-server/internal/membership"
	"weifou-server/internal/mp"
	"weifou-server/internal/payment"
	"weifou-server/internal/persona"
	"weifou-server/internal/plaza"
	"weifou-server/internal/profile"
	"weifou-server/internal/referral"
	"weifou-server/internal/share"
	"weifou-server/internal/storage"
	"weifou-server/internal/tasks"
	"weifou-server/internal/toolagent"
	"weifou-server/internal/upload"
	"weifou-server/internal/user"
	"weifou-server/internal/visit"
	"weifou-server/internal/wechat"
	"weifou-server/internal/wxpay"
	"weifou-server/internal/wxvpay"
)

// App 持有所有 handler 与定时任务调度器。
type App struct {
	cfg *config.Config

	authH       *auth.Handler
	userH       *user.Handler
	profileH    *profile.Handler
	chatH       *chat.Handler
	connectionH *connection.Handler
	visitH      *visit.Handler
	shareH      *share.Handler
	paymentH    *payment.Handler
	asyncqH     *asyncq.Handler
	plazaH      *plaza.Handler
	toolagentH  *toolagent.Handler
	homeH       *home.Handler
	membershipH *membership.Handler
	referralH   *referral.Handler
	mpH         *mp.Handler
	clientcfgH  *clientcfg.Handler
	uploadH     *upload.Handler

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
	subscribe := wechat.NewSubscribeService(loginClient, cfg.SubscribeNewQuestionTmpl, cfg.SubscribeAnsweredTmpl, cfg.SubscribeRefundedTmpl, cfg.SubscribeLeadTmpl, cfg.SubscribeLearnRemindTmpl, cfg.SubscribeMiniState)
	// 学习提醒承诺发送循环（模板未配时内部直接 no-op 不启动）
	toolagent.StartLearnRemindLoop(db, subscribe)
	ds := deepseek.New(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel)
	// 分身作答共享内核：chat（对话）与 asyncq（问答箱）共用，避免 prompt/知识注入逻辑分叉。
	ansEngine := answer.NewEngine(db, ds)
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
	paymentH := payment.NewHandler(db, payClient, cfg.JWTSecret)
	// 邀请返奖：挂到「会员订单支付成功」钩子上（三条支付通道统一触发）。
	referralH := referral.NewHandler(db, cfg.JWTSecret, paymentH.GrantDays)
	paymentH.SetMembershipPaidHook(referralH.OnMembershipPaid)
	vpayClient := wxvpay.New(cfg.WxAppID, cfg.WxvOfferID, cfg.WxvAppKey, cfg.WxvSandbox, loginClient)
	mbrH := membership.NewHandler(db, paymentH, vpayClient, loginClient, cfg.JWTSecret)
	mpLogin := wechat.NewLoginClient(cfg.MpAppID, cfg.MpSecret)

	// 公网基址：生成上传文件的可访问 URL；未配 PUBLIC_HOST 时回落 AppBaseURL（dev）。
	publicHost := cfg.PublicHost
	if publicHost == "" {
		publicHost = cfg.AppBaseURL
	}
	uploadStore := storage.NewLocal(cfg.UploadDir)

	a := &App{
		cfg:         cfg,
		authH:       auth.NewHandler(db, loginClient, appLoginClient, cfg.JWTSecret, cfg.JWTExpiresHours, cfg.Env),
		userH:       user.NewHandler(db, cfg.JWTSecret),
		profileH:    profile.NewHandler(db, personaSvc, cfg.JWTSecret),
		chatH:       chat.NewHandler(db, rdb, ansEngine, security, subscribe, cfg.JWTSecret, cfg.ChatFreeQuotaPerDay),
		connectionH: connection.NewHandler(db, security, subscribe, cfg.JWTSecret),
		visitH:      visit.NewHandler(db, cfg.JWTSecret),
		shareH:      share.NewHandler(db, loginClient),
		paymentH:    paymentH,
		asyncqH:     asyncq.NewHandler(db, ansEngine, security, subscribe, cfg.JWTSecret),
		plazaH:      plaza.NewHandler(db),
		toolagentH:  toolagent.NewHandler(db, ds, security, cfg.JWTSecret),
		homeH:       home.NewHandler(db, cfg.JWTSecret),
		membershipH: mbrH,
		referralH:   referralH,
		mpH:         mp.NewHandler(db, mpLogin, mbrH, cfg.MpToken, cfg.H5BaseURL),
		clientcfgH:  clientcfg.NewHandler(vpayClient.Ready()),
		uploadH:     upload.NewHandler(uploadStore, publicHost+"/api/uploads", cfg.JWTSecret),
		scheduler:   tasks.NewScheduler(db, paymentH, referralH, cfg.OrderTimeoutMin, cfg.CallGraceMin),
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
	a.connectionH.Register(api)
	a.visitH.Register(api)
	a.shareH.Register(api)
	a.paymentH.Register(api)
	a.asyncqH.Register(api)
	a.plazaH.Register(api)
	a.toolagentH.Register(api)
	a.homeH.Register(api)
	a.membershipH.Register(api)
	a.referralH.Register(api)
	a.mpH.Register(api)
	a.clientcfgH.Register(api)
	a.uploadH.Register(api)
	// 静态服务上传的语音文件（公开，文件名为随机 id；落点 /api/uploads/...）。
	api.Static("/uploads", a.cfg.UploadDir)
}

func (a *App) StartCron() {
	a.scheduler.Start()
}
