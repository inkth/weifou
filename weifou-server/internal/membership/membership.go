// Package membership 实现账号级会员：一价解锁全部工具 Agent。
// 卖的是 AI 服务=虚拟商品，平台自营、不分账，受 iOS 虚拟支付红线约束：
//   - 小程序：仅 Android 端可开通（buy 对 iOS 兜底拒单）；
//   - iOS / 跨端：走 H5 收银（外部 Safari 的 H5支付，不被 Apple 抽成）——
//     小程序登录态换「交接令牌」h5-link → 外部浏览器打开 h5page → h5/order 下单。
//
// 渠道无关：JSAPI / H5 都通过 payment.grantMembership 往同一会员状态叠加。
package membership

import (
	"encoding/json"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/payment"
)

type Handler struct {
	db        *gorm.DB
	pay       *payment.Handler
	jwtSecret string
}

func NewHandler(db *gorm.DB, pay *payment.Handler, jwtSecret string) *Handler {
	return &Handler{db: db, pay: pay, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/membership/status", auth, httpx.Handle(h.status))
	rg.GET("/membership/h5page", h.h5Page) // 外部浏览器打开的收银页（HTML，凭 ?t= 令牌）
	rg.POST("/membership/buy", auth, httpx.Handle(h.buy))
	rg.POST("/membership/intent", auth, httpx.Handle(h.intent))
	rg.POST("/membership/h5-link", auth, httpx.Handle(h.h5Link))
	rg.POST("/membership/h5/order", httpx.Handle(h.h5Order))
}

func platformOf(c *gin.Context) string {
	return strings.ToLower(strings.TrimSpace(c.GetHeader("X-Platform")))
}

func reqBaseURL(c *gin.Context) string {
	host := c.Request.Host
	scheme := "https"
	if c.GetHeader("X-Forwarded-Proto") == "http" ||
		(c.Request.TLS == nil && (strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127."))) {
		scheme = "http"
	}
	return scheme + "://" + host
}

// IsActive 供 toolagent 等模块判断会员是否有效。
func IsActive(db *gorm.DB, userID string) bool {
	var m models.Membership
	if db.First(&m, "user_id = ?", userID).Error != nil {
		return false
	}
	return m.ExpiresAt.After(time.Now())
}

// status 返回我的会员状态 + 可购买套餐。
func (h *Handler) status(c *gin.Context) error {
	auth := middleware.Current(c)
	isMember := false
	var expiresAt *time.Time
	var m models.Membership
	if h.db.First(&m, "user_id = ?", auth.UserID).Error == nil {
		if m.ExpiresAt.After(time.Now()) {
			isMember = true
		}
		e := m.ExpiresAt
		expiresAt = &e
	}
	httpx.OK(c, gin.H{"isMember": isMember, "expiresAt": expiresAt, "plans": h.planList()})
	return nil
}

func (h *Handler) planList() []gin.H {
	var plans []models.MembershipPlan
	h.db.Where("enabled = ?", true).Order("sort asc").Find(&plans)
	out := make([]gin.H, 0, len(plans))
	for i := range plans {
		out = append(out, gin.H{
			"id": plans[i].ID, "slug": plans[i].Slug, "name": plans[i].Name,
			"days": plans[i].Days, "price": plans[i].Price,
		})
	}
	return out
}

func (h *Handler) findPlan(id string) (*models.MembershipPlan, error) {
	var plan models.MembershipPlan
	if err := h.db.First(&plan, "id = ? AND enabled = ?", id, true).Error; err != nil {
		return nil, httpx.NotFound("PLAN_NOT_FOUND", "套餐不存在")
	}
	if plan.Price < 100 || plan.Days <= 0 {
		return nil, httpx.BadRequest("PLAN_INVALID", "套餐配置异常")
	}
	return &plan, nil
}

// buy 开通/续费会员（小程序内微信支付 JSAPI，安卓）。iOS 兜底拒单。
func (h *Handler) buy(c *gin.Context) error {
	auth := middleware.Current(c)
	if platformOf(c) == "ios" {
		return httpx.Forbidden("IOS_VIRTUAL_PAY_BLOCKED", "iOS 暂不支持在此开通，请在浏览器中开通")
	}
	var req struct {
		PlanID string `json:"planId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	plan, err := h.findPlan(req.PlanID)
	if err != nil {
		return err
	}
	planID := plan.ID
	order := models.Order{
		ID: idgen.New(), OutTradeNo: idgen.WithPrefix("MBR"), Type: models.OrderMembership,
		Amount: plan.Price, PayerOpenid: auth.Openid, PayerUserID: &auth.UserID,
		PlanID: &planID, Platform: platformOf(c),
	}
	if err := h.db.Create(&order).Error; err != nil {
		return httpx.Internal("ORDER_CREATE_FAILED", "下单失败，请稍后重试")
	}
	return h.pay.PrepayOrder(c, &order, "微否会员 · "+plan.Name, false) // 平台自营、不分账
}

// intent 留资意向（多为 iOS 用户：当下不能在小程序内开通，记录意向）。
func (h *Handler) intent(c *gin.Context) error {
	auth := middleware.Current(c)
	h.db.Create(&models.MembershipLead{
		ID: idgen.New(), UserID: auth.UserID, Openid: auth.Openid, Platform: platformOf(c),
	})
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

// ---------- H5 收银（外部浏览器） ----------

// h5Link 小程序登录态换一个短时「交接令牌」+ 外部浏览器收银页 URL。
// 用户在 Safari 打开该 URL 即可识别身份、完成开通，会员入同一账号。
func (h *Handler) h5Link(c *gin.Context) error {
	auth := middleware.Current(c)
	tok, err := h.mintH5Token(auth.UserID, auth.Openid)
	if err != nil {
		return httpx.Internal("TOKEN_FAILED", "生成失败")
	}
	httpx.OK(c, gin.H{"url": reqBaseURL(c) + "/api/membership/h5page?t=" + tok})
	return nil
}

// h5Page 渲染外部浏览器收银页（HTML）。微信内置浏览器引导「在浏览器打开」。
func (h *Handler) h5Page(c *gin.Context) {
	b, _ := json.Marshal(h.planList())
	paid := ""
	if c.Query("paid") == "1" {
		paid = "1"
	}
	html := strings.ReplaceAll(h5Template, "__PLANS_JSON__", string(b))
	html = strings.ReplaceAll(html, "__TOKEN__", template.JSEscapeString(c.Query("t")))
	html = strings.ReplaceAll(html, "__PAID__", paid)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, html)
}

// h5Order 凭交接令牌创建会员订单并返回 H5支付跳转 URL。
func (h *Handler) h5Order(c *gin.Context) error {
	var req struct {
		Token  string `json:"token" binding:"required"`
		PlanID string `json:"planId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	userID, openid, ok := h.parseH5Token(req.Token)
	if !ok {
		return httpx.BadRequest("LINK_EXPIRED", "链接已失效，请回小程序重新获取")
	}
	plan, err := h.findPlan(req.PlanID)
	if err != nil {
		return err
	}
	planID := plan.ID
	uid := userID
	order := models.Order{
		ID: idgen.New(), OutTradeNo: idgen.WithPrefix("MBR"), Type: models.OrderMembership,
		Amount: plan.Price, PayerOpenid: openid, PayerUserID: &uid,
		PlanID: &planID, Platform: "h5",
	}
	if err := h.db.Create(&order).Error; err != nil {
		return httpx.Internal("ORDER_CREATE_FAILED", "下单失败，请稍后重试")
	}
	returnURL := reqBaseURL(c) + "/api/membership/h5page?t=" + req.Token + "&paid=1"
	mwebURL, perr := h.pay.PrepayH5(&order, "微否会员 · "+plan.Name, c.ClientIP(), returnURL)
	if perr != nil {
		return httpx.Internal("WXPAY_H5_FAILED", "下单失败，请稍后重试")
	}
	httpx.OK(c, gin.H{"mwebUrl": mwebURL})
	return nil
}

// H5URL 服务端构造「在浏览器开通」链接（服务号推送等无 request 上下文时用）。
func (h *Handler) H5URL(base, userID, openid string) (string, error) {
	tok, err := h.mintH5Token(userID, openid)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base, "/") + "/api/membership/h5page?t=" + tok, nil
}

// mintH5Token 短时（30 分钟）交接令牌：与登录 JWT 同密钥，带 purpose=h5pay 防混用。
func (h *Handler) mintH5Token(userID, openid string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID, "openid": openid, "purpose": "h5pay",
		"exp": time.Now().Add(30 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.jwtSecret))
}

func (h *Handler) parseH5Token(tokenStr string) (userID, openid string, ok bool) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, hm := t.Method.(*jwt.SigningMethodHMAC); !hm {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", false
	}
	claims, cok := token.Claims.(jwt.MapClaims)
	if !cok {
		return "", "", false
	}
	if p, _ := claims["purpose"].(string); p != "h5pay" {
		return "", "", false
	}
	sub, _ := claims["sub"].(string)
	oid, _ := claims["openid"].(string)
	if sub == "" {
		return "", "", false
	}
	return sub, oid, true
}

// Seed 默认会员套餐（按 slug 幂等；改价改这里）。
func Seed(db *gorm.DB) {
	if db == nil {
		return
	}
	plans := []models.MembershipPlan{
		{Slug: "month", Name: "月卡", Days: 31, Price: 2900, Sort: 1},
		{Slug: "quarter", Name: "季卡", Days: 93, Price: 6900, Sort: 2},
		{Slug: "year", Name: "年卡", Days: 366, Price: 19900, Sort: 3},
	}
	for i := range plans {
		p := plans[i]
		var ex models.MembershipPlan
		if db.Where("slug = ?", p.Slug).First(&ex).Error == gorm.ErrRecordNotFound {
			p.ID = idgen.New()
			p.Enabled = true
			db.Create(&p)
			continue
		}
		db.Model(&ex).Updates(map[string]interface{}{
			"name": p.Name, "days": p.Days, "price": p.Price, "sort": p.Sort,
		})
	}
}

// h5Template 自包含收银页：微信内引导「在浏览器打开」，外部浏览器渲染套餐并 H5支付。
const h5Template = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
<title>微否会员</title>
<style>
* { box-sizing:border-box; -webkit-tap-highlight-color:transparent; }
body { margin:0; font-family:-apple-system,BlinkMacSystemFont,"PingFang SC",sans-serif; background:#faf7f2; color:#4a4133; }
.hero { text-align:center; padding:48px 24px 28px; background:linear-gradient(160deg,#fff3e3,#faf7f2); }
.crown { font-size:48px; }
.title { font-size:26px; font-weight:800; color:#6b4a1f; margin-top:6px; }
.sub { font-size:14px; color:#b08a5a; margin-top:6px; }
.card { margin:16px; background:#fff; border-radius:16px; padding:16px 20px; box-shadow:0 4px 20px rgba(180,140,80,.06); }
.bf { display:flex; align-items:center; font-size:15px; padding:8px 0; }
.bf i { width:28px; font-style:normal; }
.plans { display:flex; gap:12px; margin:8px 16px; }
.plan { flex:1; background:#fff; border:2px solid #f0e6d6; border-radius:14px; padding:18px 8px; text-align:center; }
.plan:active { border-color:#fb923c; }
.pn { font-size:14px; color:#6b4a1f; font-weight:600; }
.pp { font-size:24px; font-weight:800; color:#fb923c; margin-top:6px; }
.pd { font-size:12px; color:#b08a5a; margin-top:4px; }
.tip { text-align:center; font-size:12px; color:#b08a5a; margin:14px 24px; }
.overlay { position:fixed; inset:0; background:rgba(20,18,14,.85); color:#fff; display:flex; flex-direction:column; align-items:center; justify-content:center; padding:40px; text-align:center; }
.overlay .arr { position:absolute; top:8px; right:18px; font-size:40px; }
.overlay .big { font-size:18px; font-weight:700; }
.done { text-align:center; padding:64px 24px; }
.done .ok { font-size:52px; }
.done .t { font-size:18px; font-weight:700; margin-top:12px; }
.done .s { font-size:14px; color:#8a7a60; margin-top:8px; line-height:1.6; }
</style>
</head>
<body>
<div id="app"></div>
<script>
var PLANS = __PLANS_JSON__;
var TOKEN = "__TOKEN__";
var PAID = "__PAID__";
var isWeChat = /MicroMessenger/i.test(navigator.userAgent);
var app = document.getElementById('app');
function yuan(fen){ return (fen/100).toFixed(2).replace(/\.00$/,''); }
function render(){
  if (PAID === "1"){
    app.innerHTML = '<div class="done"><div class="ok">✅</div><div class="t">支付完成</div><div class="s">会员已开通，返回微否小程序即可畅用全部 AI 助手</div></div>';
    return;
  }
  if (isWeChat){
    app.innerHTML = '<div class="overlay"><div class="arr">↗</div><div class="big">请在浏览器中打开</div><div style="margin-top:14px;font-size:14px;opacity:.85">点右上角「···」→「在 Safari / 浏览器打开」，再开通会员</div></div>';
    return;
  }
  var h = '<div class="hero"><div class="crown">👑</div><div class="title">微否会员</div><div class="sub">一价畅用全部 AI 助手</div></div>';
  h += '<div class="card"><div class="bf"><i>🗣️</i>全部精选 Agent 畅用</div><div class="bf"><i>∞</i>不限次数，随时开练</div><div class="bf"><i>🆕</i>新上线 Agent 自动包含</div></div>';
  h += '<div class="plans">';
  for (var i=0;i<PLANS.length;i++){
    var p = PLANS[i]; var d = p.days>0 ? (p.price/100/p.days).toFixed(1) : '';
    h += '<div class="plan" onclick="pay(\'' + p.id + '\')"><div class="pn">' + p.name + '</div><div class="pp">¥' + yuan(p.price) + '</div>' + (d?'<div class="pd">约 ¥'+d+'/天</div>':'') + '</div>';
  }
  h += '</div><div class="tip">点套餐用微信支付开通</div>';
  app.innerHTML = h;
}
function pay(planId){
  if (!TOKEN){ alert('链接已失效，请回小程序重新获取'); return; }
  fetch('/api/membership/h5/order', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ token: TOKEN, planId: planId }) })
    .then(function(r){ return r.json(); })
    .then(function(res){
      if (res && res.success !== false && res.data && res.data.mwebUrl){ location.href = res.data.mwebUrl; }
      else { alert((res && res.message) || '下单失败，请重试'); }
    })
    .catch(function(){ alert('网络异常，请重试'); });
}
render();
</script>
</body>
</html>`
