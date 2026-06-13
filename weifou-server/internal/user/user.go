package user

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
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
	rg.GET("/user/me", middleware.JWTAuth(h.jwtSecret), httpx.Handle(h.me))
}

func (h *Handler) me(c *gin.Context) error {
	auth := middleware.Current(c)
	var u models.User
	if err := h.db.First(&u, "id = ?", auth.UserID).Error; err != nil {
		return httpx.NotFound("USER_NOT_FOUND", "用户不存在")
	}
	var profile models.Profile
	var profileID interface{}
	var profileStatus interface{}
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err == nil {
		profileID = profile.ID
		profileStatus = profile.Status
	}
	httpx.OK(c, gin.H{
		"id":            u.ID,
		"nickname":      u.Nickname,
		"avatarUrl":     u.AvatarURL,
		"profileId":     profileID,
		"profileStatus": profileStatus,
	})
	return nil
}
