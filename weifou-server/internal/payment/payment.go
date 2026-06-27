package payment

import (
	"encoding/json"
	"fmt"
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
	"weifou-server/internal/wechat"
	"weifou-server/internal/wxpay"
)

type Handler struct {
	db           *gorm.DB
	pay          *wxpay.Client
	security     *wechat.SecurityService
	profitShare  *ProfitShareService
	subscribe    *wechat.SubscribeService
	jwtSecret    string
	tipMaxAmount int
	asyncSLAHrs  int
}

func NewHandler(db *gorm.DB, pay *wxpay.Client, security *wechat.SecurityService, ps *ProfitShareService, subscribe *wechat.SubscribeService, jwtSecret string, tipMax, asyncSLAHrs int) *Handler {
	return &Handler{db: db, pay: pay, security: security, profitShare: ps, subscribe: subscribe, jwtSecret: jwtSecret, tipMaxAmount: tipMax, asyncSLAHrs: asyncSLAHrs}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.POST("/payment/tip", auth, httpx.Handle(h.tip))
	rg.POST("/payment/consult", auth, httpx.Handle(h.consult))
	rg.GET("/payment/orders/:id", auth, httpx.Handle(h.getOrder))
	rg.POST("/payment/refund", auth, httpx.Handle(h.refund))
	rg.POST("/payment/notify", httpx.Handle(h.notify))
	rg.POST("/payment/refund-notify", httpx.Handle(h.refundNotify))
}

// normSource 将客户端传入的成交来源收敛到白名单，默认 profile。
func normSource(s string) string {
	if s == "chat_card" {
		return "chat_card"
	}
	return "profile"
}

// ---------- 打赏 ----------

type tipReq struct {
	ProfileID string `json:"profileId" binding:"required"`
	Amount    int    `json:"amount" binding:"required"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

func (h *Handler) tip(c *gin.Context) error {
	auth := middleware.Current(c)
	var req tipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	if req.Amount < 100 {
		return httpx.BadRequest("INVALID_AMOUNT", "金额过小")
	}
	if req.Amount > h.tipMaxAmount {
		return httpx.BadRequest("TIP_TOO_LARGE", "打赏金额超出上限")
	}
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", req.ProfileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "主页不存在")
	}
	if req.Message != "" && !h.security.CheckText(req.Message, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "留言包含不当内容")
	}

	order := models.Order{
		ID: idgen.New(), OutTradeNo: idgen.WithPrefix("TIP"), Type: models.OrderTip,
		Amount: req.Amount, ProfileID: profile.ID, PayerOpenid: auth.Openid,
		PayerUserID: &auth.UserID, PayeeUserID: profile.UserID,
		Source: normSource(req.Source),
	}
	if req.Message != "" {
		order.Message = &req.Message
	}
	h.db.Create(&order)

	return h.prepay(c, &order, fmt.Sprintf("打赏 %s", profile.RealName), false)
}

// ---------- 付费咨询（选档） ----------

type consultReq struct {
	ProfileID string `json:"profileId" binding:"required"`
	SlotID    string `json:"slotId" binding:"required"`
	Source    string `json:"source"`
}

func (h *Handler) consult(c *gin.Context) error {
	auth := middleware.Current(c)
	var req consultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", req.ProfileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "主页不存在")
	}
	if profile.UserID == auth.UserID {
		return httpx.BadRequest("CANNOT_CONSULT_SELF", "不能预约自己")
	}
	var setting models.ConsultSetting
	if err := h.db.First(&setting, "user_id = ?", profile.UserID).Error; err != nil || !setting.Enabled {
		return httpx.Forbidden("CONSULT_DISABLED", "对方未开放付费咨询")
	}
	var slot models.ConsultSlot
	if err := h.db.First(&slot, "id = ?", req.SlotID).Error; err != nil || slot.HostUserID != profile.UserID {
		return httpx.BadRequest("SLOT_INVALID", "档期不存在")
	}
	if slot.Status != models.SlotOpen {
		return httpx.BadRequest("SLOT_TAKEN", "该档期已被预约")
	}
	if slot.StartAt.Before(time.Now()) {
		return httpx.BadRequest("SLOT_EXPIRED", "该档期已过期")
	}

	amount := setting.Price30
	if slot.DurationMin == 60 {
		amount = setting.Price60
	}

	var order models.Order
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.ConsultSlot{}).
			Where("id = ? AND status = ?", slot.ID, models.SlotOpen).
			Update("status", models.SlotBooked)
		if res.RowsAffected == 0 {
			return httpx.BadRequest("SLOT_TAKEN", "该档期刚被预约")
		}
		dm := slot.DurationMin
		order = models.Order{
			ID: idgen.New(), OutTradeNo: idgen.WithPrefix("CST"), Type: models.OrderConsult,
			Amount: amount, ProfileID: profile.ID, PayerOpenid: auth.Openid,
			PayerUserID: &auth.UserID, PayeeUserID: profile.UserID,
			DurationMin: &dm, SlotID: &slot.ID, ScheduledAt: &slot.StartAt,
			Source: normSource(req.Source),
		}
		return tx.Create(&order).Error
	})
	if txErr != nil {
		return txErr
	}

	return h.prepay(c, &order, fmt.Sprintf("预约 %s %d 分钟咨询", profile.RealName, slot.DurationMin), true)
}

func (h *Handler) prepay(c *gin.Context, order *models.Order, desc string, profitSharing bool) error {
	// 记录下单端（合规分流/兜底）。业务方已设置则不覆盖。
	if order.Platform == "" {
		if p := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Platform"))); p != "" {
			order.Platform = p
			h.db.Model(order).Update("platform", p)
		}
	}
	prepayID, err := h.pay.CreateJsapiOrder(wxpay.JsapiOrder{
		OutTradeNo:    order.OutTradeNo,
		Description:   desc,
		Amount:        order.Amount,
		PayerOpenid:   order.PayerOpenid,
		Attach:        order.ID,
		ProfitSharing: profitSharing && h.profitShare.Enabled(),
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

// PrepayOrder 通用预下单：业务方建好 order 后调用，复用下单 + 返回 payParams（如付费提问）。
func (h *Handler) PrepayOrder(c *gin.Context, order *models.Order, desc string, profitSharing bool) error {
	return h.prepay(c, order, desc, profitSharing)
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

// Settle 触发分账（供异步咨询等业务在「服务交付」时刻调用）。
func (h *Handler) Settle(orderID string) {
	h.profitShare.SettleForOrder(orderID)
}

// notifyHostNewQuestion 给主人下发「有新的付费提问」订阅消息（按小程序 openid，纯 App 主人会静默失败）。
func (h *Handler) notifyHostNewQuestion(orderID string) {
	if h.subscribe == nil {
		return
	}
	var q models.AsyncQuestion
	if h.db.First(&q, "order_id = ?", orderID).Error != nil {
		return
	}
	var host models.User
	if h.db.First(&host, "id = ?", q.HostUserID).Error != nil {
		return
	}
	openid := host.Openid
	if host.WxMpOpenid != nil && *host.WxMpOpenid != "" {
		openid = *host.WxMpOpenid
	}
	deadline := time.Now()
	if q.AnswerDeadline != nil {
		deadline = *q.AnswerDeadline
	}
	h.subscribe.NotifyNewQuestion(openid, q.Question, q.Price, deadline, "pages/inbox/index")
}

func (h *Handler) getOrder(c *gin.Context) error {
	var order models.Order
	if err := h.db.First(&order, "id = ?", c.Param("id")).Error; err != nil {
		return httpx.NotFound("ORDER_NOT_FOUND", "订单不存在")
	}
	var consultSessionID interface{}
	var session models.ConsultSession
	if err := h.db.First(&session, "order_id = ?", order.ID).Error; err == nil {
		consultSessionID = session.ID
	}
	var asyncQuestionID interface{}
	var aq models.AsyncQuestion
	if err := h.db.First(&aq, "order_id = ?", order.ID).Error; err == nil {
		asyncQuestionID = aq.ID
	}
	httpx.OK(c, gin.H{
		"id": order.ID, "type": order.Type, "status": order.Status,
		"amount": order.Amount, "durationMin": order.DurationMin,
		"consultSessionId": consultSessionID,
		"asyncQuestionId":  asyncQuestionID,
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

	if order.Type == models.OrderConsult {
		var existing models.ConsultSession
		if h.db.First(&existing, "order_id = ?", order.ID).Error == gorm.ErrRecordNotFound {
			dm := 30
			if order.DurationMin != nil {
				dm = *order.DurationMin
			}
			h.db.Create(&models.ConsultSession{
				ID: idgen.New(), OrderID: order.ID, ProfileID: order.ProfileID,
				HostUserID: order.PayeeUserID, GuestOpenid: order.PayerOpenid,
				TrtcRoomID: "consult_" + order.ID, DurationMin: dm,
				ScheduledAt: order.ScheduledAt, Status: models.ConsultPending,
			})
		}
	}

	if order.Type == models.OrderAsyncQuestion {
		deadline := now.Add(time.Duration(h.asyncSLAHrs) * time.Hour)
		res := h.db.Model(&models.AsyncQuestion{}).
			Where("order_id = ? AND status = ?", order.ID, models.AsyncPendingPayment).
			Updates(map[string]interface{}{
				"status": models.AsyncPaid, "paid_at": now, "answer_deadline": deadline,
			})
		if res.RowsAffected > 0 {
			go h.notifyHostNewQuestion(order.ID)
		}
	}

	if order.Type == models.OrderMembership && order.PlanID != nil {
		var plan models.MembershipPlan
		if h.db.First(&plan, "id = ?", *order.PlanID).Error == nil {
			h.grantMembership(order.PayerUserID, plan.Days)
		}
	}
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

// GrantMembershipByOrder 供虚拟支付发货回调复用：按已支付的会员订单开通/续费。
// 幂等由调用方的「订单 paid 守卫」保证（同一订单只发货一次）。
func (h *Handler) GrantMembershipByOrder(order *models.Order) {
	if order.Type != models.OrderMembership || order.PlanID == nil {
		return
	}
	var plan models.MembershipPlan
	if h.db.First(&plan, "id = ?", *order.PlanID).Error == nil {
		h.grantMembership(order.PayerUserID, plan.Days)
	}
}

// ---------- 退款 ----------

type refundReq struct {
	OrderID string `json:"orderId" binding:"required"`
	Reason  string `json:"reason"`
}

func (h *Handler) refund(c *gin.Context) error {
	auth := middleware.Current(c)
	var req refundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	if err := h.RefundConsult(req.OrderID, auth.UserID, auth.Openid, req.Reason); err != nil {
		return err
	}
	httpx.OK(c, gin.H{"orderId": req.OrderID, "refundStatus": "PROCESSING"})
	return nil
}

// RefundConsult 退款（供接口与定时任务复用）。仅 consult 且通话未开始可退。
func (h *Handler) RefundConsult(orderID, userID, openid, reason string) error {
	var order models.Order
	if err := h.db.First(&order, "id = ?", orderID).Error; err != nil {
		return httpx.NotFound("ORDER_NOT_FOUND", "订单不存在")
	}
	if order.Type != models.OrderConsult {
		return httpx.BadRequest("TIP_NOT_REFUNDABLE", "打赏为自愿赠予，不支持退款")
	}
	if order.PayerOpenid != openid && order.PayeeUserID != userID {
		return httpx.Forbidden("NOT_PARTICIPANT", "无权操作该订单")
	}
	if order.Status == models.OrderRefunded || order.Status == models.OrderRefunding {
		return httpx.BadRequest("ALREADY_REFUNDING", "订单已在退款流程中")
	}
	if order.Status != models.OrderPaid {
		return httpx.BadRequest("ORDER_NOT_PAID", "订单不可退款")
	}
	var session models.ConsultSession
	if h.db.First(&session, "order_id = ?", order.ID).Error == nil {
		if session.Status != models.ConsultPending {
			return httpx.BadRequest("CALL_STARTED", "通话已开始或结束，不支持退款")
		}
	}
	return h.doRefund(&order, reason)
}

// doRefund 执行退款（建退款单→置订单退款中→调微信退款→成功则 finalize）。
// 调用方负责业务前置校验与并发认领。
func (h *Handler) doRefund(order *models.Order, reason string) error {
	outRefundNo := idgen.WithPrefix("RFD")
	refund := models.Refund{
		ID: idgen.New(), OrderID: order.ID, OutRefundNo: outRefundNo,
		Amount: order.Amount, Status: models.RefundProcessing,
	}
	if reason != "" {
		refund.Reason = &reason
	}
	h.db.Create(&refund)
	h.db.Model(order).Update("status", models.OrderRefunding)

	resp, err := h.pay.Refund(wxpay.RefundReq{
		OutTradeNo: order.OutTradeNo, OutRefundNo: outRefundNo,
		Refund: order.Amount, Total: order.Amount, Reason: reason,
	})
	if err != nil {
		h.db.Model(&refund).Update("status", models.RefundFail)
		h.db.Model(order).Update("status", models.OrderPaid)
		return httpx.Internal("WXPAY_REFUND_FAILED", "退款失败，请稍后重试")
	}
	h.db.Model(&refund).Update("refund_id", resp.RefundID)
	if resp.Status == "SUCCESS" {
		h.finalizeRefund(order.ID, refund.ID)
	}
	return nil
}

// RefundAsyncQuestion 付费提问退款（超时未答自动退 / 主人主动退）。供定时任务复用。
func (h *Handler) RefundAsyncQuestion(orderID, reason string) error {
	var order models.Order
	if err := h.db.First(&order, "id = ?", orderID).Error; err != nil {
		return httpx.NotFound("ORDER_NOT_FOUND", "订单不存在")
	}
	if order.Type != models.OrderAsyncQuestion {
		return httpx.BadRequest("NOT_REFUNDABLE", "该订单不支持退款")
	}
	if order.Status != models.OrderPaid {
		return httpx.BadRequest("ORDER_NOT_PAID", "订单不可退款")
	}
	// 原子认领 paid→refunded，胜出者执行退款，防止与「主人作答」竞态。
	claim := h.db.Model(&models.AsyncQuestion{}).
		Where("order_id = ? AND status = ?", order.ID, models.AsyncPaid).
		Update("status", models.AsyncRefunded)
	if claim.RowsAffected == 0 {
		return httpx.BadRequest("NOT_REFUNDABLE", "提问当前不可退款")
	}
	if err := h.doRefund(&order, reason); err != nil {
		// 退款发起失败：回滚认领，让下一轮可重试。
		h.db.Model(&models.AsyncQuestion{}).
			Where("order_id = ? AND status = ?", order.ID, models.AsyncRefunded).
			Update("status", models.AsyncPaid)
		return err
	}
	if h.subscribe != nil {
		var q models.AsyncQuestion
		if h.db.First(&q, "order_id = ?", order.ID).Error == nil && q.AskerOpenid != "" {
			go h.subscribe.NotifyRefunded(q.AskerOpenid, q.Question, order.Amount, reason, "pages/my-questions/index")
		}
	}
	return nil
}

func (h *Handler) finalizeRefund(orderID, refundID string) {
	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&models.Refund{}).Where("id = ?", refundID).Update("status", models.RefundSuccess)
		var order models.Order
		tx.First(&order, "id = ?", orderID)
		tx.Model(&order).Update("status", models.OrderRefunded)
		tx.Model(&models.ConsultSession{}).Where("order_id = ?", orderID).Update("status", models.ConsultCanceled)
		tx.Model(&models.AsyncQuestion{}).Where("order_id = ?", orderID).Update("status", models.AsyncRefunded)
		if order.SlotID != nil {
			tx.Model(&models.ConsultSlot{}).Where("id = ?", *order.SlotID).Update("status", models.SlotOpen)
			tx.Model(&models.Order{}).Where("id = ?", orderID).Update("slot_id", nil)
		}
		return nil
	})
}

func (h *Handler) refundNotify(c *gin.Context) error {
	raw, _ := c.GetRawData()
	rawBody := string(raw)
	if !h.pay.VerifyNotifySignature(c.Request.Header, rawBody) {
		c.JSON(500, gin.H{"code": "FAIL", "message": "invalid signature"})
		return nil
	}
	var body struct {
		Resource wxpay.NotifyResource `json:"resource"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		c.JSON(500, gin.H{"code": "FAIL", "message": "bad body"})
		return nil
	}
	plain, err := h.pay.DecryptNotify(body.Resource)
	if err == nil {
		var d struct {
			OutRefundNo  string `json:"out_refund_no"`
			RefundStatus string `json:"refund_status"`
		}
		if json.Unmarshal(plain, &d) == nil {
			var refund models.Refund
			if h.db.First(&refund, "out_refund_no = ?", d.OutRefundNo).Error == nil {
				switch d.RefundStatus {
				case "SUCCESS":
					h.finalizeRefund(refund.OrderID, refund.ID)
				case "ABNORMAL", "CLOSED":
					h.db.Model(&refund).Update("status", models.RefundFail)
					h.db.Model(&models.Order{}).Where("id = ?", refund.OrderID).Update("status", models.OrderPaid)
				}
			}
		}
	}
	c.JSON(200, gin.H{"code": "SUCCESS", "message": "OK"})
	return nil
}

// CloseOrder 关闭超时未支付订单并释放档期（供定时任务调用）。
func (h *Handler) CloseOrder(orderID string) {
	var order models.Order
	if err := h.db.First(&order, "id = ?", orderID).Error; err != nil || order.Status != models.OrderPending {
		return
	}
	if err := h.pay.CloseOrder(order.OutTradeNo); err != nil {
		log.Printf("[payment] 关单失败(可能已支付) %s: %v", order.OutTradeNo, err)
	}
	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&order).Update("status", models.OrderClosed)
		if order.SlotID != nil {
			tx.Model(&models.ConsultSlot{}).Where("id = ?", *order.SlotID).Update("status", models.SlotOpen)
			tx.Model(&models.Order{}).Where("id = ?", orderID).Update("slot_id", nil)
		}
		return nil
	})
}
