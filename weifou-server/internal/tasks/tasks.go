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
