package payment

import (
	"log"
	"math"

	"gorm.io/gorm"

	"weifou-server/internal/idgen"
	"weifou-server/internal/models"
	"weifou-server/internal/wxpay"
)

// ProfitShareService 微信分账：通话结束后给 host 分账并完结解冻。
type ProfitShareService struct {
	db      *gorm.DB
	pay     *wxpay.Client
	enabled bool
	feeRate float64
}

func NewProfitShareService(db *gorm.DB, pay *wxpay.Client, enabled bool, feeRate float64) *ProfitShareService {
	return &ProfitShareService{db: db, pay: pay, enabled: enabled, feeRate: feeRate}
}

func (s *ProfitShareService) Enabled() bool { return s.enabled }

func (s *ProfitShareService) ensureReceiver(hostUserID string) *models.User {
	var host models.User
	if err := s.db.First(&host, "id = ?", hostUserID).Error; err != nil {
		return nil
	}
	if !host.PsReceiverAdded {
		name := ""
		if host.Nickname != nil {
			name = *host.Nickname
		}
		if err := s.pay.AddProfitShareReceiver(host.Openid, name); err != nil {
			log.Printf("[profitshare] add receiver (可忽略已存在): %v", err)
		}
		s.db.Model(&host).Update("ps_receiver_added", true)
	}
	return &host
}

// SettleForOrder 通话结束触发：分账给 host 并完结。
func (s *ProfitShareService) SettleForOrder(orderID string) {
	if !s.enabled {
		return
	}
	var order models.Order
	if err := s.db.First(&order, "id = ?", orderID).Error; err != nil {
		return
	}
	if order.Type != models.OrderConsult || order.Status != models.OrderPaid || order.TransactionID == nil {
		return
	}

	var existing models.ProfitShare
	hasExisting := s.db.First(&existing, "order_id = ?", orderID).Error == nil
	if hasExisting && existing.Finished {
		return
	}

	host := s.ensureReceiver(order.PayeeUserID)
	if host == nil {
		return
	}

	platformFee := int(math.Floor(float64(order.Amount) * s.feeRate))
	payeeAmount := order.Amount - platformFee
	outOrderNo := ""
	if hasExisting {
		outOrderNo = existing.OutOrderNo
	} else {
		outOrderNo = idgen.WithPrefix("PS")
	}

	resp, err := s.pay.CreateProfitShare(wxpay.ProfitShareReq{
		TransactionID:  *order.TransactionID,
		OutOrderNo:     outOrderNo,
		ReceiverOpenid: host.Openid,
		Amount:         payeeAmount,
		Description:    "咨询通话分成",
	})
	if err != nil {
		log.Printf("[profitshare] 分账失败 order=%s: %v", orderID, err)
		s.upsert(orderID, outOrderNo, platformFee, payeeAmount, models.PSFail, nil, false)
		return
	}
	s.upsert(orderID, outOrderNo, platformFee, payeeAmount, models.PSFinished, &resp.OrderID, true)
}

func (s *ProfitShareService) upsert(orderID, outOrderNo string, fee, payee int, status string, wxOrderID *string, finished bool) {
	var existing models.ProfitShare
	if s.db.First(&existing, "order_id = ?", orderID).Error == gorm.ErrRecordNotFound {
		s.db.Create(&models.ProfitShare{
			ID: idgen.New(), OrderID: orderID, OutOrderNo: outOrderNo,
			PlatformFee: fee, PayeeAmount: payee, Status: status,
			WxOrderID: wxOrderID, Finished: finished,
		})
	} else {
		s.db.Model(&existing).Updates(map[string]interface{}{
			"status": status, "wx_order_id": wxOrderID, "finished": finished,
		})
	}
}
