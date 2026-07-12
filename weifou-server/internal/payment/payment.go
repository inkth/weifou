package payment

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/wxpay"
)

type Handler struct {
	db        *gorm.DB
	pay       *wxpay.Client
	jwtSecret string
	// onMembershipPaid 会员订单支付成功钩子（邀请返奖等；app 装配时注入，可为 nil）。
	onMembershipPaid func(*models.Order)
}

func NewHandler(db *gorm.DB, pay *wxpay.Client, jwtSecret string) *Handler {
	return &Handler{db: db, pay: pay, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/payment/orders/:id", auth, httpx.Handle(h.getOrder))
	rg.POST("/payment/notify", httpx.Handle(h.notify))
}

func (h *Handler) prepay(c *gin.Context, order *models.Order, desc string) error {
	// 记录下单端（合规分流/兜底）。业务方已设置则不覆盖。
	if order.Platform == "" {
		if p := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Platform"))); p != "" {
			order.Platform = p
			h.db.Model(order).Update("platform", p)
		}
	}
	prepayID, err := h.pay.CreateJsapiOrder(wxpay.JsapiOrder{
		OutTradeNo:  order.OutTradeNo,
		Description: desc,
		Amount:      order.Amount,
		PayerOpenid: order.PayerOpenid,
		Attach:      order.ID,
	})
	if err != nil {
		return httpx.Internal("WXPAY_ORDER_FAILED", "下单失败，请稍后重试")
	}
	h.db.Model(order).Update("prepay_id", prepayID)
	params, err := h.pay.BuildPayParams(prepayID)
	if err != nil {
		return httpx.Internal("WXPAY_SIGN_FAILED", "下单失败")
	}
	httpx.OK(c, gin.H{"orderId": order.ID, "outTradeNo": order.OutTradeNo, "payParams": params})
	return nil
}

// PrepayOrder 通用预下单：业务方建好 order 后调用，复用下单 + 返回 payParams。
func (h *Handler) PrepayOrder(c *gin.Context, order *models.Order, desc string) error {
	return h.prepay(c, order, desc)
}

// PrepayH5 H5(MWEB) 下单：业务方建好 order 后调用，返回外部浏览器跳转的 h5_url（已附 redirect_url）。
// 用于 iOS 等外部 Safari 收款（微信外浏览器），不被 Apple 抽成。
func (h *Handler) PrepayH5(order *models.Order, desc, clientIP, returnURL string) (string, error) {
	h5url, err := h.pay.CreateH5Order(wxpay.H5Order{
		OutTradeNo: order.OutTradeNo, Description: desc, Amount: order.Amount,
		Attach: order.ID, ClientIP: clientIP,
	})
	if err != nil {
		return "", err
	}
	if returnURL != "" && h5url != "" {
		sep := "?"
		if strings.Contains(h5url, "?") {
			sep = "&"
		}
		h5url += sep + "redirect_url=" + url.QueryEscape(returnURL)
	}
	return h5url, nil
}

func (h *Handler) getOrder(c *gin.Context) error {
	var order models.Order
	if err := h.db.First(&order, "id = ?", c.Param("id")).Error; err != nil {
		return httpx.NotFound("ORDER_NOT_FOUND", "订单不存在")
	}
	httpx.OK(c, gin.H{
		"id": order.ID, "type": order.Type, "status": order.Status,
		"amount": order.Amount,
	})
	return nil
}

// ---------- 支付回调 ----------

func (h *Handler) notify(c *gin.Context) error {
	raw, _ := c.GetRawData()
	rawBody := string(raw)
	if !h.pay.VerifyNotifySignature(c.Request.Header, rawBody) {
		c.JSON(500, gin.H{"code": "FAIL", "message": "invalid signature"})
		return nil
	}
	var body struct {
		EventType string               `json:"event_type"`
		Resource  wxpay.NotifyResource `json:"resource"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		c.JSON(500, gin.H{"code": "FAIL", "message": "bad body"})
		return nil
	}
	if body.EventType == "TRANSACTION.SUCCESS" {
		if err := h.handlePaid(body.Resource); err != nil {
			log.Printf("[payment] notify handle error: %v", err)
		}
	}
	c.JSON(200, gin.H{"code": "SUCCESS", "message": "OK"})
	return nil
}

func (h *Handler) handlePaid(res wxpay.NotifyResource) error {
	plain, err := h.pay.DecryptNotify(res)
	if err != nil {
		return err
	}
	var d struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		Amount        struct {
			Total int `json:"total"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(plain, &d); err != nil {
		return err
	}
	if d.TradeState != "SUCCESS" {
		return nil
	}
	var order models.Order
	if err := h.db.First(&order, "out_trade_no = ?", d.OutTradeNo).Error; err != nil {
		return nil
	}
	if order.Status == models.OrderPaid {
		return nil // 幂等
	}
	if d.Amount.Total != order.Amount {
		log.Printf("[payment] 金额不一致 %s: 期望 %d 实收 %d", order.OutTradeNo, order.Amount, d.Amount.Total)
		return nil
	}
	now := time.Now()
	h.db.Model(&order).Updates(map[string]interface{}{
		"status": models.OrderPaid, "transaction_id": d.TransactionID, "paid_at": now,
	})

	order.PaidAt = &now
	h.GrantMembershipByOrder(&order)
	return nil
}

// grantMembership 会员购买成功后开通/续费（未过期则在原到期日上顺延；平台自营、不分账）。
// 幂等由 handlePaid 顶部的 paid 守卫保证：同一订单只会进来一次。
func (h *Handler) grantMembership(userID *string, days int) {
	if userID == nil || *userID == "" || days <= 0 {
		return
	}
	now := time.Now()
	var m models.Membership
	if err := h.db.First(&m, "user_id = ?", *userID).Error; err == gorm.ErrRecordNotFound {
		h.db.Create(&models.Membership{ID: idgen.New(), UserID: *userID, ExpiresAt: now.AddDate(0, 0, days)})
		return
	}
	base := now
	if m.ExpiresAt.After(now) {
		base = m.ExpiresAt // 未过期 → 在原到期日上叠加
	}
	h.db.Model(&models.Membership{}).Where("user_id = ?", *userID).
		Update("expires_at", base.AddDate(0, 0, days))
}

// GrantDays 往用户会员状态上直接叠加天数（邀请返奖等非订单场景用）。
func (h *Handler) GrantDays(userID string, days int) {
	h.grantMembership(&userID, days)
}

// SetMembershipPaidHook 注册「会员订单支付成功」钩子（app 装配时注入 referral，避免包循环依赖）。
func (h *Handler) SetMembershipPaidHook(fn func(*models.Order)) {
	h.onMembershipPaid = fn
}

// GrantMembershipByOrder 按已支付的会员订单开通/续费（JSAPI/H5 回调与虚拟支付发货共用入口）。
// 幂等由调用方的「订单 paid 守卫」保证（同一订单只发货一次）。
func (h *Handler) GrantMembershipByOrder(order *models.Order) {
	if order.Type != models.OrderMembership || order.PlanID == nil {
		return
	}
	var plan models.MembershipPlan
	if h.db.First(&plan, "id = ?", *order.PlanID).Error == nil {
		h.grantMembership(order.PayerUserID, plan.Days)
	}
	if h.onMembershipPaid != nil {
		h.onMembershipPaid(order)
	}
}

// CloseOrder 关闭超时未支付订单（供定时任务调用）。
func (h *Handler) CloseOrder(orderID string) {
	var order models.Order
	if err := h.db.First(&order, "id = ?", orderID).Error; err != nil || order.Status != models.OrderPending {
		return
	}
	if err := h.pay.CloseOrder(order.OutTradeNo); err != nil {
		log.Printf("[payment] 关单失败(可能已支付) %s: %v", order.OutTradeNo, err)
	}
	h.db.Model(&order).Update("status", models.OrderClosed)
}
