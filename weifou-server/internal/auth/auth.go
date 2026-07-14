package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db          *gorm.DB
	login       *wechat.LoginClient // 小程序 jscode2session
	appLogin    *wechat.LoginClient // 移动应用 oauth2（可为 nil，未配置移动应用时）
	jwtSecret   string
	jwtExpHours int
	env         string // 环境（production 时禁用测试登录）
}

func NewHandler(db *gorm.DB, login, appLogin *wechat.LoginClient, jwtSecret string, jwtExpHours int, env string) *Handler {
	return &Handler{db: db, login: login, appLogin: appLogin, jwtSecret: jwtSecret, jwtExpHours: jwtExpHours, env: env}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/auth/login", httpx.Handle(h.loginHandler))
	rg.POST("/auth/test-login", httpx.Handle(h.testLoginHandler))
}

type loginReq struct {
	Code      string `json:"code" binding:"required"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

func (h *Handler) issueToken(userID, openid string) (string, error) {
	claims := jwt.MapClaims{
		"sub":    userID,
		"openid": openid,
		"exp":    time.Now().Add(time.Duration(h.jwtExpHours) * time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func (h *Handler) loginHandler(c *gin.Context) error {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}

	// 按客户端来源分流：app → 移动应用 oauth2；其余 → 小程序 jscode2session。
	isApp := c.GetHeader("X-Client-Type") == "app"
	var session *wechat.Session
	var err error
	if isApp {
		if h.appLogin == nil {
			return httpx.BadRequest("WX_APP_NOT_CONFIGURED", "移动应用登录未配置")
		}
		session, err = h.appLogin.OAuth2Code2Session(req.Code)
	} else {
		session, err = h.login.Code2Session(req.Code)
	}
	if err != nil {
		return httpx.BadRequest("WX_LOGIN_FAILED", err.Error())
	}

	// 昵称/头像：优先用客户端传入，回退微信 userinfo（App oauth 会带）。
	nickname := req.Nickname
	if nickname == "" {
		nickname = session.Nickname
	}
	avatar := req.AvatarURL
	if avatar == "" {
		avatar = session.AvatarURL
	}

	user, err := h.findOrCreateUser(session, isApp, nickname, avatar)
	if err != nil {
		return err
	}

	// JWT 携带「当前端」openid，保证 JSAPI 支付 payer 与访客身份归因正确。
	endOpenid := session.Openid
	token, err := h.issueToken(user.ID, endOpenid)
	if err != nil {
		return httpx.Internal("TOKEN_ERROR", "签发失败")
	}
	httpx.OK(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":        user.ID,
			"nickname":  user.Nickname,
			"avatarUrl": user.AvatarURL,
		},
	})
	return nil
}

// testLoginCode 是测试专用的固定验证码，任意手机号 + 该码即可登录。
// 仅非生产环境放行，便于客户端在未接入微信开放平台时联调。
const testLoginCode = "654321"

type testLoginReq struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// testLoginHandler 测试登录：手机号 + 固定验证码 654321。
// 每个手机号对应一个稳定的测试账号（合成 openid），方便多账号互测。
func (h *Handler) testLoginHandler(c *gin.Context) error {
	if h.env == "production" {
		return httpx.BadRequest("TEST_LOGIN_DISABLED", "测试登录在生产环境已禁用")
	}
	var req testLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	if req.Code != testLoginCode {
		return httpx.BadRequest("CODE_INVALID", "验证码错误")
	}

	// 用手机号合成稳定身份，复用 App 端账号链路（落 wx_app_openid）。
	session := &wechat.Session{Openid: "test_app_" + req.Phone}
	nickname := "测试用户" + phoneTail(req.Phone)
	user, err := h.findOrCreateUser(session, true, nickname, "")
	if err != nil {
		return err
	}

	token, err := h.issueToken(user.ID, session.Openid)
	if err != nil {
		return httpx.Internal("TOKEN_ERROR", "签发失败")
	}
	httpx.OK(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":        user.ID,
			"nickname":  user.Nickname,
			"avatarUrl": user.AvatarURL,
		},
	})
	return nil
}

// phoneTail 取手机号后 4 位作昵称后缀，不足 4 位则用全量。
func phoneTail(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	return phone[len(phone)-4:]
}

// findOrCreateUser 路线 A：优先按 unionid 合并同一真人的小程序/App 账号，
// 否则按当前端 openid 查；都没有则新建。
func (h *Handler) findOrCreateUser(session *wechat.Session, isApp bool, nickname, avatar string) (*models.User, error) {
	var user models.User

	// 1) 优先 unionid 合并。
	found := false
	if session.Unionid != "" {
		if err := h.db.Where("unionid = ?", session.Unionid).First(&user).Error; err == nil {
			found = true
		} else if err != gorm.ErrRecordNotFound {
			return nil, httpx.Internal("DB_ERROR", "查询用户失败")
		}
	}
	// 2) 回退按当前端 openid 查。
	if !found {
		col := "wx_mp_openid"
		if isApp {
			col = "wx_app_openid"
		}
		// 同时兼容历史数据：老用户仅有 openid 字段。
		q := h.db.Where(col+" = ?", session.Openid)
		if !isApp {
			q = q.Or("openid = ?", session.Openid)
		}
		if err := q.First(&user).Error; err == nil {
			found = true
		} else if err != gorm.ErrRecordNotFound {
			return nil, httpx.Internal("DB_ERROR", "查询用户失败")
		}
	}

	if !found {
		user = models.User{ID: idgen.New(), Openid: session.Openid}
		setEndOpenid(&user, isApp, session.Openid)
		if session.Unionid != "" {
			user.Unionid = &session.Unionid
		}
		if nickname != "" {
			user.Nickname = &nickname
		}
		if avatar != "" {
			user.AvatarURL = &avatar
		}
		if err := h.db.Create(&user).Error; err != nil {
			return nil, httpx.Internal("DB_ERROR", "创建用户失败")
		}
		return &user, nil
	}

	// 命中已有账号：补齐当前端 openid / unionid / 资料。
	updates := map[string]interface{}{}
	if isApp && (user.WxAppOpenid == nil || *user.WxAppOpenid == "") {
		updates["wx_app_openid"] = session.Openid
	}
	if !isApp && (user.WxMpOpenid == nil || *user.WxMpOpenid == "") {
		updates["wx_mp_openid"] = session.Openid
	}
	if session.Unionid != "" && (user.Unionid == nil || *user.Unionid == "") {
		updates["unionid"] = session.Unionid
	}
	if nickname != "" && user.Nickname == nil {
		updates["nickname"] = nickname
	}
	if avatar != "" && user.AvatarURL == nil {
		updates["avatar_url"] = avatar
	}
	if len(updates) > 0 {
		if err := h.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, httpx.Internal("DB_ERROR", "更新用户失败")
		}
	}
	return &user, nil
}

func setEndOpenid(u *models.User, isApp bool, openid string) {
	if isApp {
		u.WxAppOpenid = &openid
	} else {
		u.WxMpOpenid = &openid
	}
}
