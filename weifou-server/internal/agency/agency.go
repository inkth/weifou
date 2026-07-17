// Package agency 提供独立的代理商注册入驻流程。
package agency

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

var mainlandPhone = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

var channelTypes = map[string]bool{
	"enterprise": true,
	"creator":    true,
	"community":  true,
	"consultant": true,
	"local":      true,
	"other":      true,
}

var audienceSizes = map[string]bool{
	"under_500":   true,
	"500_2000":    true,
	"2000_10000":  true,
	"10000_50000": true,
	"over_50000":  true,
}

type Handler struct {
	db        *gorm.DB
	jwtSecret string
}

func NewHandler(db *gorm.DB, jwtSecret string) *Handler {
	return &Handler{db: db, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/agency/application", auth, httpx.Handle(h.getApplication))
	rg.POST("/agency/application", auth, httpx.Handle(h.submitApplication))
}

type applicationInput struct {
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Region       string `json:"region"`
	ChannelType  string `json:"channelType"`
	AudienceSize string `json:"audienceSize"`
	Experience   string `json:"experience"`
	InviteCode   string `json:"inviteCode"`
	Consent      bool   `json:"consent"`
}

func cleanInput(in applicationInput) applicationInput {
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Region = strings.TrimSpace(in.Region)
	in.ChannelType = strings.TrimSpace(in.ChannelType)
	in.AudienceSize = strings.TrimSpace(in.AudienceSize)
	in.Experience = strings.TrimSpace(in.Experience)
	in.InviteCode = strings.TrimSpace(in.InviteCode)
	return in
}

func validateInput(in applicationInput) *httpx.APIError {
	if utf8.RuneCountInString(in.Name) < 2 || utf8.RuneCountInString(in.Name) > 20 {
		return httpx.BadRequest("INVALID_NAME", "请填写 2～20 个字的真实姓名")
	}
	if !mainlandPhone.MatchString(in.Phone) {
		return httpx.BadRequest("INVALID_PHONE", "请填写正确的中国大陆手机号")
	}
	if in.Region == "" || utf8.RuneCountInString(in.Region) > 60 {
		return httpx.BadRequest("INVALID_REGION", "请选择所在地区")
	}
	if !channelTypes[in.ChannelType] {
		return httpx.BadRequest("INVALID_CHANNEL_TYPE", "请选择主要渠道类型")
	}
	if !audienceSizes[in.AudienceSize] {
		return httpx.BadRequest("INVALID_AUDIENCE_SIZE", "请选择可触达用户规模")
	}
	if utf8.RuneCountInString(in.Experience) < 10 || utf8.RuneCountInString(in.Experience) > 500 {
		return httpx.BadRequest("INVALID_EXPERIENCE", "请用 10～500 个字介绍你的渠道与推广经验")
	}
	if utf8.RuneCountInString(in.InviteCode) > 32 {
		return httpx.BadRequest("INVALID_INVITE_CODE", "邀请码格式不正确")
	}
	if !in.Consent {
		return httpx.BadRequest("CONSENT_REQUIRED", "请确认资料真实并同意用于代理商注册")
	}
	return nil
}

func publicApplication(a *models.AgencyApplication) gin.H {
	return gin.H{
		"id":           a.ID,
		"name":         a.Name,
		"phone":        a.Phone,
		"region":       a.Region,
		"channelType":  a.ChannelType,
		"audienceSize": a.AudienceSize,
		"experience":   a.Experience,
		"inviteCode":   a.InviteCode,
		"status":       a.Status,
		"reviewNote":   a.ReviewNote,
		"createdAt":    a.CreatedAt,
		"updatedAt":    a.UpdatedAt,
	}
}

func (h *Handler) getApplication(c *gin.Context) error {
	auth := middleware.Current(c)
	var application models.AgencyApplication
	if err := h.db.First(&application, "user_id = ?", auth.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.OK(c, nil)
			return nil
		}
		return err
	}
	httpx.OK(c, publicApplication(&application))
	return nil
}

func (h *Handler) submitApplication(c *gin.Context) error {
	auth := middleware.Current(c)
	var in applicationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		return httpx.BadRequest("INVALID_BODY", "申请资料格式不正确")
	}
	in = cleanInput(in)
	if err := validateInput(in); err != nil {
		return err
	}

	var application models.AgencyApplication
	err := h.db.First(&application, "user_id = ?", auth.UserID).Error
	now := time.Now()
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		application = models.AgencyApplication{
			ID:           idgen.New(),
			UserID:       auth.UserID,
			Status:       models.AgencyApplicationApproved,
			ConsentAt:    now,
			ReviewedAt:   &now,
			Name:         in.Name,
			Phone:        in.Phone,
			Region:       in.Region,
			ChannelType:  in.ChannelType,
			AudienceSize: in.AudienceSize,
			Experience:   in.Experience,
			InviteCode:   in.InviteCode,
		}
		if err := h.db.Create(&application).Error; err != nil {
			return err
		}
	case err != nil:
		return err
	case application.Status == models.AgencyApplicationApproved || application.Status == models.AgencyApplicationSuspended:
		return httpx.NewError(http.StatusConflict, "APPLICATION_LOCKED", "当前代理商状态不可修改")
	default:
		application.Name = in.Name
		application.Phone = in.Phone
		application.Region = in.Region
		application.ChannelType = in.ChannelType
		application.AudienceSize = in.AudienceSize
		application.Experience = in.Experience
		application.InviteCode = in.InviteCode
		application.Status = models.AgencyApplicationApproved
		application.ReviewNote = ""
		application.ReviewedAt = &now
		application.ConsentAt = now
		if err := h.db.Save(&application).Error; err != nil {
			return err
		}
	}

	httpx.OK(c, publicApplication(&application))
	return nil
}
