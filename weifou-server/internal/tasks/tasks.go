package tasks

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"weifou-server/internal/models"
	"weifou-server/internal/payment"
)

type Scheduler struct {
	db              *gorm.DB
	payment         *payment.Handler
	cron            *cron.Cron
	orderTimeoutMin int
	graceMin        int
}

func NewScheduler(db *gorm.DB, pay *payment.Handler, orderTimeoutMin, graceMin int) *Scheduler {
	return &Scheduler{
		db: db, payment: pay, cron: cron.New(),
		orderTimeoutMin: orderTimeoutMin, graceMin: graceMin,
	}
}

func (s *Scheduler) Start() {
	// 每分钟关闭超时未支付订单
	s.cron.AddFunc("@every 1m", s.closeTimeoutOrders)
	// 每 5 分钟清理过期档期
	s.cron.AddFunc("@every 5m", s.releaseExpiredSlots)
	// 每 5 分钟爽约自动退款
	s.cron.AddFunc("@every 5m", s.autoRefundNoShow)
	s.cron.Start()
	log.Println("[tasks] cron started")
}

func (s *Scheduler) closeTimeoutOrders() {
	deadline := time.Now().Add(-time.Duration(s.orderTimeoutMin) * time.Minute)
	var stale []models.Order
	s.db.Where("status = ? AND created_at < ?", models.OrderPending, deadline).Limit(100).Find(&stale)
	for _, o := range stale {
		s.payment.CloseOrder(o.ID)
	}
	if len(stale) > 0 {
		log.Printf("[tasks] 关闭超时订单 %d 笔", len(stale))
	}
}

func (s *Scheduler) releaseExpiredSlots() {
	res := s.db.Model(&models.ConsultSlot{}).
		Where("status = ? AND start_at < ?", models.SlotOpen, time.Now()).
		Update("status", models.SlotCanceled)
	if res.RowsAffected > 0 {
		log.Printf("[tasks] 清理过期档期 %d 个", res.RowsAffected)
	}
}

func (s *Scheduler) autoRefundNoShow() {
	now := time.Now()
	var candidates []models.ConsultSession
	s.db.Where("status = ? AND scheduled_at IS NOT NULL", models.ConsultPending).Limit(100).Find(&candidates)
	for _, sess := range candidates {
		if sess.ScheduledAt == nil {
			continue
		}
		end := sess.ScheduledAt.Add(time.Duration(sess.DurationMin+s.graceMin) * time.Minute)
		if now.Before(end) {
			continue
		}
		var order models.Order
		if s.db.First(&order, "id = ?", sess.OrderID).Error != nil || order.Status != models.OrderPaid {
			continue
		}
		if err := s.payment.RefundConsult(sess.OrderID, sess.HostUserID, sess.GuestOpenid, "通话时间窗已过，系统自动退款"); err != nil {
			log.Printf("[tasks] 自动退款失败 session=%s: %v", sess.ID, err)
		} else {
			log.Printf("[tasks] 爽约自动退款 session=%s", sess.ID)
		}
	}
}
