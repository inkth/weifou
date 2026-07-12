// Package referral 好友邀请返奖：推荐好友首次开通会员，双边送会员时长。
//
// 合规边界（不可破）：
//   - 奖励只挂「好友完成支付」，绝不挂分享/转发动作（微信利诱分享红线）；
//   - 只做一级（推荐人→好友），不做多级分销；
//   - 只送会员时长，不返现金、不可提现。
//
// 防刷：
//   - 每个被邀人一生只能绑定一位推荐人（先到先得），且绑定时必须尚无已支付会员订单；
//   - 自邀（同账号 / 同 unionid）拒绝；
//   - 推荐人奖励过退款窗口（7 天）后由定时任务发放，期间退款则作废；
//   - 被邀人加赠随首单立即发（退款时整单会员时长本身即失去意义，不单独追回）。
package referral

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

// RefundWindowDays 推荐人奖励的解锁等待期（覆盖退款窗口）。
const RefundWindowDays = 7

// planReward 各套餐的双边奖励天数（按 plan slug 配置；未列出的套餐不产生奖励）。
type planReward struct {
	InviterDays int // 推荐人：好友买月付送 1 个月、年付送 2 个月
	InviteeDays int // 被邀人首开加赠：立即到账，让分享话术是「给好友的福利」
}

var planRewards = map[string]planReward{
	"month": {InviterDays: 31, InviteeDays: 7},
	"year":  {InviterDays: 62, InviteeDays: 31},
}

type Handler struct {
	db        *gorm.DB
	jwtSecret string
	// grantDays 往用户会员状态上叠加天数（由 payment.GrantDays 注入，避免包循环依赖）。
	grantDays func(userID string, days int)
}

func NewHandler(db *gorm.DB, jwtSecret string, grantDays func(userID string, days int)) *Handler {
	return &Handler{db: db, jwtSecret: jwtSecret, grantDays: grantDays}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/referral/summary", auth, httpx.Handle(h.summary))
	rg.POST("/referral/bind", auth, httpx.Handle(h.bind))
}

// summary 我的邀请概况：邀请参数（refCode=userID）+ 战报（已邀人数/待到账/已到账天数）。
func (h *Handler) summary(c *gin.Context) error {
	auth := middleware.Current(c)
	var invitedCount int64
	h.db.Model(&models.ReferralBinding{}).Where("inviter_user_id = ?", auth.UserID).Count(&invitedCount)

	var rewards []models.ReferralReward
	h.db.Where("inviter_user_id = ?", auth.UserID).Find(&rewards)
	pendingDays, grantedDays := 0, 0
	for i := range rewards {
		switch rewards[i].Status {
		case models.ReferralRewardPending:
			pendingDays += rewards[i].InviterDays
		case models.ReferralRewardGranted:
			grantedDays += rewards[i].InviterDays
		}
	}

	// 被邀人视角：是否已通过别人的邀请进入（用于前端展示首开加赠横幅）。
	var binding models.ReferralBinding
	isInvitee := h.db.First(&binding, "invitee_user_id = ?", auth.UserID).Error == nil

	httpx.OK(c, gin.H{
		"refCode":      auth.UserID,
		"invitedCount": invitedCount,
		"pendingDays":  pendingDays,
		"grantedDays":  grantedDays,
		"isInvitee":    isInvitee,
	})
	return nil
}

// bind 被邀人绑定推荐人（通过邀请链接进入会员页时调用；幂等，失败静默不打断购买流程）。
func (h *Handler) bind(c *gin.Context) error {
	auth := middleware.Current(c)
	var req struct {
		InviterUserID string `json:"inviterUserId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	if req.InviterUserID == auth.UserID {
		httpx.OK(c, gin.H{"bound": false, "reason": "SELF"})
		return nil
	}
	// 已绑定过（先到先得）：幂等返回。
	var ex models.ReferralBinding
	if h.db.First(&ex, "invitee_user_id = ?", auth.UserID).Error == nil {
		httpx.OK(c, gin.H{"bound": ex.InviterUserID == req.InviterUserID, "reason": "ALREADY_BOUND"})
		return nil
	}
	var inviter models.User
	if h.db.First(&inviter, "id = ?", req.InviterUserID).Error != nil {
		httpx.OK(c, gin.H{"bound": false, "reason": "INVITER_NOT_FOUND"})
		return nil
	}
	// 同一真人跨端小号（unionid 相同）不算邀请。
	var invitee models.User
	if h.db.First(&invitee, "id = ?", auth.UserID).Error == nil &&
		invitee.Unionid != nil && inviter.Unionid != nil && *invitee.Unionid == *inviter.Unionid {
		httpx.OK(c, gin.H{"bound": false, "reason": "SELF"})
		return nil
	}
	// 只认新客：已有已支付会员订单的老客不再产生邀请归因。
	if h.hasPaidMembershipOrder(auth.UserID) {
		httpx.OK(c, gin.H{"bound": false, "reason": "NOT_FIRST_PURCHASE"})
		return nil
	}
	b := models.ReferralBinding{ID: idgen.New(), InviteeUserID: auth.UserID, InviterUserID: req.InviterUserID}
	if err := h.db.Create(&b).Error; err != nil {
		// 并发下撞唯一索引 = 已被绑定，按幂等处理。
		httpx.OK(c, gin.H{"bound": false, "reason": "ALREADY_BOUND"})
		return nil
	}
	httpx.OK(c, gin.H{"bound": true})
	return nil
}

func (h *Handler) hasPaidMembershipOrder(userID string) bool {
	var n int64
	h.db.Model(&models.Order{}).
		Where("payer_user_id = ? AND type = ? AND status = ?", userID, models.OrderMembership, models.OrderPaid).
		Count(&n)
	return n > 0
}

// OnMembershipPaid 会员订单支付成功钩子（payment 回调注入；JSAPI/H5/虚拟支付三通道统一走这里）。
// 被邀人首单：立即发被邀人加赠，并为推荐人记一笔待解锁奖励。幂等由 OrderID 唯一索引保证。
func (h *Handler) OnMembershipPaid(order *models.Order) {
	if order == nil || order.Type != models.OrderMembership || order.PayerUserID == nil || order.PlanID == nil {
		return
	}
	inviteeID := *order.PayerUserID
	var binding models.ReferralBinding
	if h.db.First(&binding, "invitee_user_id = ?", inviteeID).Error != nil {
		return // 非受邀用户
	}
	// 每个被邀人一生只产生一笔奖励（防同人反复首购刷奖，退款重买也不再触发）。
	var existed int64
	h.db.Model(&models.ReferralReward{}).Where("invitee_user_id = ?", inviteeID).Count(&existed)
	if existed > 0 {
		return
	}
	var plan models.MembershipPlan
	if h.db.First(&plan, "id = ?", *order.PlanID).Error != nil {
		return
	}
	rw, ok := planRewards[plan.Slug]
	if !ok {
		return
	}
	paidAt := time.Now()
	if order.PaidAt != nil {
		paidAt = *order.PaidAt
	}
	reward := models.ReferralReward{
		ID: idgen.New(), OrderID: order.ID,
		InviterUserID: binding.InviterUserID, InviteeUserID: inviteeID,
		PlanSlug: plan.Slug, InviterDays: rw.InviterDays, InviteeDays: rw.InviteeDays,
		Status: models.ReferralRewardPending, UnlockAt: paidAt.AddDate(0, 0, RefundWindowDays),
	}
	if err := h.db.Create(&reward).Error; err != nil {
		return // 撞 OrderID 唯一索引 = 回调重放，幂等退出
	}
	if rw.InviteeDays > 0 && h.grantDays != nil {
		h.grantDays(inviteeID, rw.InviteeDays)
	}
	log.Printf("[referral] 邀请成交 inviter=%s invitee=%s plan=%s 加赠=%dd 待发=%dd",
		binding.InviterUserID, inviteeID, plan.Slug, rw.InviteeDays, rw.InviterDays)
}

// GrantDueRewards 发放已过退款窗口的推荐人奖励（定时任务调用）。
// 订单已退款（处理中/成功）则作废该笔奖励。
func (h *Handler) GrantDueRewards() {
	var due []models.ReferralReward
	h.db.Where("status = ? AND unlock_at <= ?", models.ReferralRewardPending, time.Now()).
		Limit(100).Find(&due)
	for i := range due {
		r := due[i]
		var refunded int64
		h.db.Model(&models.Refund{}).
			Where("order_id = ? AND status IN ?", r.OrderID, []string{models.RefundProcessing, models.RefundSuccess}).
			Count(&refunded)
		if refunded > 0 {
			h.db.Model(&models.ReferralReward{}).Where("id = ? AND status = ?", r.ID, models.ReferralRewardPending).
				Update("status", models.ReferralRewardCancelled)
			log.Printf("[referral] 订单已退款，奖励作废 reward=%s order=%s", r.ID, r.OrderID)
			continue
		}
		// 先占坑再发放：状态条件更新保证并发/重启下只发一次。
		res := h.db.Model(&models.ReferralReward{}).Where("id = ? AND status = ?", r.ID, models.ReferralRewardPending).
			Updates(map[string]interface{}{"status": models.ReferralRewardGranted, "granted_at": time.Now()})
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		if h.grantDays != nil {
			h.grantDays(r.InviterUserID, r.InviterDays)
		}
		log.Printf("[referral] 推荐奖励到账 inviter=%s +%dd (order=%s)", r.InviterUserID, r.InviterDays, r.OrderID)
	}
}
