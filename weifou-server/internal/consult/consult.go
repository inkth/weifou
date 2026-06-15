package consult

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

type Handler struct {
	db        *gorm.DB
	jwtSecret string
}

func NewHandler(db *gorm.DB, jwtSecret string) *Handler {
	return &Handler{db: db, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/consult/setting/mine", auth, httpx.Handle(h.mySetting))
	rg.PATCH("/consult/setting", auth, httpx.Handle(h.updateSetting))
	rg.GET("/consult/sessions/mine", auth, httpx.Handle(h.mySessions))
	rg.GET("/consult/pricing/:profileId", httpx.Handle(h.pricing))
	rg.POST("/consult/slots", auth, httpx.Handle(h.createSlots))
	rg.GET("/consult/slots/mine", auth, httpx.Handle(h.mySlots))
	rg.DELETE("/consult/slots/:slotId", auth, httpx.Handle(h.deleteSlot))
	rg.GET("/consult/slots/public/:profileId", httpx.Handle(h.publicSlots))
}

func (h *Handler) mySetting(c *gin.Context) error {
	auth := middleware.Current(c)
	var s models.ConsultSetting
	if err := h.db.First(&s, "user_id = ?", auth.UserID).Error; err != nil {
		httpx.OK(c, gin.H{"enabled": false, "price30": 9900, "price60": 19900, "intro": nil, "asyncEnabled": false, "asyncPrice": 4900})
		return nil
	}
	httpx.OK(c, s)
	return nil
}

type settingReq struct {
	Enabled      *bool   `json:"enabled"`
	Price30      *int    `json:"price30"`
	Price60      *int    `json:"price60"`
	Intro        *string `json:"intro"`
	AsyncEnabled *bool   `json:"asyncEnabled"`
	AsyncPrice   *int    `json:"asyncPrice"`
}

func (h *Handler) updateSetting(c *gin.Context) error {
	auth := middleware.Current(c)
	var req settingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	var s models.ConsultSetting
	err := h.db.First(&s, "user_id = ?", auth.UserID).Error
	if err == gorm.ErrRecordNotFound {
		s = models.ConsultSetting{
			ID: idgen.New(), UserID: auth.UserID,
			Price30: 9900, Price60: 19900, AsyncPrice: 4900,
		}
		if req.Enabled != nil {
			s.Enabled = *req.Enabled
		}
		if req.Price30 != nil {
			s.Price30 = *req.Price30
		}
		if req.Price60 != nil {
			s.Price60 = *req.Price60
		}
		if req.AsyncEnabled != nil {
			s.AsyncEnabled = *req.AsyncEnabled
		}
		if req.AsyncPrice != nil {
			s.AsyncPrice = *req.AsyncPrice
		}
		s.Intro = req.Intro
		h.db.Create(&s)
	} else {
		updates := map[string]interface{}{}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if req.Price30 != nil {
			updates["price30"] = *req.Price30
		}
		if req.Price60 != nil {
			updates["price60"] = *req.Price60
		}
		if req.Intro != nil {
			updates["intro"] = *req.Intro
		}
		if req.AsyncEnabled != nil {
			updates["async_enabled"] = *req.AsyncEnabled
		}
		if req.AsyncPrice != nil {
			updates["async_price"] = *req.AsyncPrice
		}
		if len(updates) > 0 {
			h.db.Model(&s).Updates(updates)
		}
	}
	httpx.OK(c, s)
	return nil
}

func (h *Handler) pricing(c *gin.Context) error {
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", c.Param("profileId")).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "主页不存在")
	}
	resp := gin.H{"enabled": false, "asyncEnabled": false}
	var s models.ConsultSetting
	if err := h.db.First(&s, "user_id = ?", profile.UserID).Error; err == nil {
		if s.Enabled {
			resp["enabled"] = true
			resp["price30"] = s.Price30
			resp["price60"] = s.Price60
			resp["intro"] = s.Intro
		}
		if s.AsyncEnabled {
			resp["asyncEnabled"] = true
			resp["asyncPrice"] = s.AsyncPrice
		}
	}
	httpx.OK(c, resp)
	return nil
}

// ---------- 档期 ----------

type slotItem struct {
	StartAt     string `json:"startAt" binding:"required"`
	DurationMin int    `json:"durationMin" binding:"required"`
}
type createSlotsReq struct {
	Slots []slotItem `json:"slots" binding:"required"`
}

func (h *Handler) createSlots(c *gin.Context) error {
	auth := middleware.Current(c)
	var req createSlotsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	now := time.Now()
	var toCreate []models.ConsultSlot
	for _, s := range req.Slots {
		t, err := time.Parse(time.RFC3339, s.StartAt)
		if err != nil || !t.After(now) {
			continue
		}
		if s.DurationMin != 30 && s.DurationMin != 60 {
			continue
		}
		toCreate = append(toCreate, models.ConsultSlot{
			ID: idgen.New(), HostUserID: auth.UserID, StartAt: t,
			DurationMin: s.DurationMin, Status: models.SlotOpen,
		})
	}
	if len(toCreate) == 0 {
		return httpx.BadRequest("NO_VALID_SLOT", "没有有效的未来档期")
	}
	h.db.Create(&toCreate)
	return h.respondMySlots(c, auth.UserID)
}

func (h *Handler) mySlots(c *gin.Context) error {
	auth := middleware.Current(c)
	return h.respondMySlots(c, auth.UserID)
}

func (h *Handler) respondMySlots(c *gin.Context, userID string) error {
	var slots []models.ConsultSlot
	h.db.Where("host_user_id = ? AND start_at >= ?", userID, time.Now()).
		Order("start_at asc").Find(&slots)
	httpx.OK(c, slots)
	return nil
}

func (h *Handler) deleteSlot(c *gin.Context) error {
	auth := middleware.Current(c)
	var slot models.ConsultSlot
	if err := h.db.First(&slot, "id = ?", c.Param("slotId")).Error; err != nil || slot.HostUserID != auth.UserID {
		return httpx.NotFound("SLOT_NOT_FOUND", "档期不存在")
	}
	if slot.Status == models.SlotBooked {
		return httpx.Forbidden("SLOT_BOOKED", "该档期已被预约，不能删除")
	}
	h.db.Delete(&slot)
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

func (h *Handler) publicSlots(c *gin.Context) error {
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", c.Param("profileId")).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "主页不存在")
	}
	var slots []models.ConsultSlot
	h.db.Where("host_user_id = ? AND status = ? AND start_at >= ?",
		profile.UserID, models.SlotOpen, time.Now()).
		Order("start_at asc").Limit(50).Find(&slots)
	httpx.OK(c, slots)
	return nil
}

// ---------- 通话记录 ----------

func (h *Handler) mySessions(c *gin.Context) error {
	auth := middleware.Current(c)
	var sessions []models.ConsultSession
	h.db.Where("host_user_id = ? OR guest_openid = ?", auth.UserID, auth.Openid).
		Order("created_at desc").Limit(50).Find(&sessions)

	out := make([]gin.H, 0, len(sessions))
	for _, s := range sessions {
		var profile models.Profile
		h.db.Select("real_name").First(&profile, "id = ?", s.ProfileID)
		role := "guest"
		if s.HostUserID == auth.UserID {
			role = "host"
		}
		out = append(out, gin.H{
			"id": s.ID, "orderId": s.OrderID, "profileId": s.ProfileID,
			"realName": profile.RealName, "role": role, "status": s.Status,
			"durationMin": s.DurationMin, "durationSec": s.DurationSec,
			"scheduledAt": s.ScheduledAt, "createdAt": s.CreatedAt,
		})
	}
	httpx.OK(c, out)
	return nil
}
