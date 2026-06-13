package rtc

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/payment"
	"weifou-server/internal/trtc"
)

type Handler struct {
	db           *gorm.DB
	profitShare  *payment.ProfitShareService
	jwtSecret    string
	sdkAppID     int
	secretKey    string
	sigExpire    int
	earlyJoinMin int
	graceMin     int
}

type Config struct {
	JWTSecret    string
	SdkAppID     int
	SecretKey    string
	SigExpire    int
	EarlyJoinMin int
	GraceMin     int
}

func NewHandler(db *gorm.DB, ps *payment.ProfitShareService, cfg Config) *Handler {
	return &Handler{
		db: db, profitShare: ps, jwtSecret: cfg.JWTSecret,
		sdkAppID: cfg.SdkAppID, secretKey: cfg.SecretKey, sigExpire: cfg.SigExpire,
		earlyJoinMin: cfg.EarlyJoinMin, graceMin: cfg.GraceMin,
	}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.POST("/rtc/consult/:sessionId/join", auth, httpx.Handle(h.join))
	rg.POST("/rtc/consult/:sessionId/start", auth, httpx.Handle(h.start))
	rg.POST("/rtc/consult/:sessionId/end", auth, httpx.Handle(h.end))
}

func (h *Handler) loadParticipant(c *gin.Context) (*models.ConsultSession, *middleware.AuthUser, bool, error) {
	auth := middleware.Current(c)
	var session models.ConsultSession
	if err := h.db.First(&session, "id = ?", c.Param("sessionId")).Error; err != nil {
		return nil, nil, false, httpx.NotFound("SESSION_NOT_FOUND", "通话不存在")
	}
	isHost := session.HostUserID == auth.UserID
	isGuest := session.GuestOpenid == auth.Openid
	if !isHost && !isGuest {
		return nil, nil, false, httpx.Forbidden("NOT_PARTICIPANT", "你不是本次通话成员")
	}
	return &session, auth, isHost, nil
}

func (h *Handler) join(c *gin.Context) error {
	if h.sdkAppID == 0 || h.secretKey == "" {
		return httpx.Internal("TRTC_NOT_CONFIGURED", "音视频未配置")
	}
	session, auth, isHost, err := h.loadParticipant(c)
	if err != nil {
		return err
	}

	var order models.Order
	h.db.First(&order, "id = ?", session.OrderID)
	if order.Status != models.OrderPaid {
		return httpx.Forbidden("ORDER_UNPAID", "订单未支付")
	}
	if session.Status == models.ConsultEnded || session.Status == models.ConsultCanceled {
		return httpx.Forbidden("SESSION_CLOSED", "通话已结束")
	}

	// 进房时间窗
	if session.ScheduledAt != nil {
		start := session.ScheduledAt.UnixMilli()
		now := time.Now().UnixMilli()
		openFrom := start - int64(h.earlyJoinMin)*60000
		openUntil := start + int64(session.DurationMin+h.graceMin)*60000
		if now < openFrom {
			return httpx.Forbidden("CALL_NOT_OPEN", "通话将于预约时间前开放进入")
		}
		if now > openUntil {
			return httpx.Forbidden("CALL_WINDOW_PASSED", "通话时间窗已过")
		}
	}

	trtcUserID := "g_" + auth.Openid
	role := "guest"
	if isHost {
		trtcUserID = "h_" + auth.UserID
		role = "host"
	}
	sig, err := trtc.GenUserSig(h.sdkAppID, h.secretKey, trtcUserID, h.sigExpire)
	if err != nil {
		return httpx.Internal("TRTC_SIG_FAILED", "签发失败")
	}

	httpx.OK(c, gin.H{
		"sdkAppId": h.sdkAppID, "userId": trtcUserID, "userSig": sig,
		"roomId": session.TrtcRoomID, "role": role, "durationMin": session.DurationMin,
		"status": session.Status, "startedAt": session.StartedAt, "scheduledAt": session.ScheduledAt,
	})
	return nil
}

func (h *Handler) start(c *gin.Context) error {
	session, _, _, err := h.loadParticipant(c)
	if err != nil {
		return err
	}
	if session.Status == models.ConsultPending {
		now := time.Now()
		h.db.Model(session).Updates(map[string]interface{}{
			"status": models.ConsultOngoing, "started_at": now,
		})
	}
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

func (h *Handler) end(c *gin.Context) error {
	session, _, _, err := h.loadParticipant(c)
	if err != nil {
		return err
	}
	if session.Status == models.ConsultEnded {
		httpx.OK(c, gin.H{"ok": true})
		return nil
	}
	now := time.Now()
	durationSec := 0
	if session.StartedAt != nil {
		durationSec = int(now.Sub(*session.StartedAt).Seconds())
	}
	h.db.Model(session).Updates(map[string]interface{}{
		"status": models.ConsultEnded, "ended_at": now, "duration_sec": durationSec,
	})
	if session.StartedAt != nil {
		go h.profitShare.SettleForOrder(session.OrderID)
	}
	httpx.OK(c, gin.H{"ok": true, "durationSec": durationSec})
	return nil
}
