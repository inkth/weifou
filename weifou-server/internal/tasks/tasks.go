package tasks

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"weifou-server/internal/models"
	"weifou-server/internal/payment"
	"weifou-server/internal/referral"
)

type Scheduler struct {
	db              *gorm.DB
	payment         *payment.Handler
	referral        *referral.Handler
	cron            *cron.Cron
	orderTimeoutMin int
}

func NewScheduler(db *gorm.DB, pay *payment.Handler, ref *referral.Handler, orderTimeoutMin int) *Scheduler {
	return &Scheduler{
		db: db, payment: pay, referral: ref, cron: cron.New(),
		orderTimeoutMin: orderTimeoutMin,
	}
}

func (s *Scheduler) Start() {
	// 每分钟关闭超时未支付订单
	s.cron.AddFunc("@every 1m", s.closeTimeoutOrders)
	// 每 10 分钟发放已过观察期的邀请奖励
	s.cron.AddFunc("@every 10m", s.referral.GrantDueRewards)
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
