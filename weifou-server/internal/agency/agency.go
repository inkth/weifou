// Package agency 提供独立的代理商注册入驻流程。
package agency

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
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
	login     *wechat.LoginClient
	hc        *http.Client
	miniEnv   string
}

func NewHandler(db *gorm.DB, jwtSecret string, login *wechat.LoginClient, env string) *Handler {
	miniEnv := "trial"
	if env == "production" {
		miniEnv = "release"
	}
	return &Handler{db: db, jwtSecret: jwtSecret, login: login, hc: &http.Client{Timeout: 10 * time.Second}, miniEnv: miniEnv}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/agency/application", auth, httpx.Handle(h.getApplication))
	rg.POST("/agency/application", auth, httpx.Handle(h.submitApplication))
	rg.GET("/agency/dashboard", auth, httpx.Handle(h.dashboard))
	rg.POST("/agency/bind", auth, httpx.Handle(h.bindInvitee))
	rg.GET("/agency/qrcode", auth, httpx.Handle(h.qrcode))
}

const (
	firstAgencyCode = 1112
	lastAgencyCode  = 9999
	// PostgreSQL 事务级 advisory lock：串行化邀请码分配，不阻塞其他业务表。
	agencyCodeLockKey = 11129999
)

var agencyCodePattern = regexp.MustCompile(`^[0-9]{4}$`)

func (h *Handler) ensureAgencyCode(application *models.AgencyApplication) error {
	if application.AgencyCode != nil && agencyCodePattern.MatchString(*application.AgencyCode) {
		return nil
	}
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", agencyCodeLockKey).Error; err != nil {
			return err
		}

		// 拿锁后重新读取，兼容同一代理商被两个请求同时补码。
		var fresh models.AgencyApplication
		if err := tx.First(&fresh, "id = ?", application.ID).Error; err != nil {
			return err
		}
		if fresh.AgencyCode != nil && agencyCodePattern.MatchString(*fresh.AgencyCode) {
			application.AgencyCode = fresh.AgencyCode
			return nil
		}

		var highest int
		if err := tx.Raw(`
			SELECT COALESCE(MAX(CAST(agency_code AS INTEGER)), ?)
			FROM agency_applications
			WHERE agency_code ~ '^[0-9]{4}$'`, firstAgencyCode-1).Scan(&highest).Error; err != nil {
			return err
		}
		next := highest + 1
		if next < firstAgencyCode {
			next = firstAgencyCode
		}
		if next > lastAgencyCode {
			return errors.New("agency code capacity exhausted")
		}
		code := strconv.Itoa(next)
		if err := tx.Model(&fresh).Update("agency_code", code).Error; err != nil {
			return err
		}
		application.AgencyCode = &code
		return nil
	})
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
	agencyCode := ""
	if a.AgencyCode != nil {
		agencyCode = *a.AgencyCode
	}
	return gin.H{
		"id":           a.ID,
		"agencyCode":   agencyCode,
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
	if application.Status == models.AgencyApplicationApproved {
		if err := h.ensureAgencyCode(&application); err != nil {
			return err
		}
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
	if application.Status == models.AgencyApplicationApproved {
		if err := h.ensureAgencyCode(&application); err != nil {
			return err
		}
	}

	httpx.OK(c, publicApplication(&application))
	return nil
}

type bindInput struct {
	AgencyCode string `json:"agencyCode"`
}

func (h *Handler) bindInvitee(c *gin.Context) error {
	auth := middleware.Current(c)
	var in bindInput
	if err := c.ShouldBindJSON(&in); err != nil {
		return httpx.BadRequest("INVALID_BODY", "邀请参数格式不正确")
	}
	code := strings.ToUpper(strings.TrimSpace(in.AgencyCode))
	if !agencyCodePattern.MatchString(code) {
		return httpx.BadRequest("INVALID_AGENCY_CODE", "代理商邀请码无效")
	}

	var agency models.AgencyApplication
	if err := h.db.Where("agency_code = ? AND status = ?", code, models.AgencyApplicationApproved).First(&agency).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.NotFound("AGENCY_NOT_FOUND", "代理商邀请码无效")
		}
		return err
	}
	if agency.UserID == auth.UserID {
		httpx.OK(c, gin.H{"bound": false, "reason": "self"})
		return nil
	}

	var existing models.AgencyUserBinding
	if err := h.db.First(&existing, "invitee_user_id = ?", auth.UserID).Error; err == nil {
		httpx.OK(c, gin.H{"bound": false, "reason": "already_bound"})
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", auth.UserID).Error; err != nil {
		return err
	}
	accountAge := time.Since(user.CreatedAt)
	binding := models.AgencyUserBinding{
		ID:            idgen.New(),
		AgencyUserID:  agency.UserID,
		InviteeUserID: auth.UserID,
		AgencyCode:    code,
		NewUser:       accountAge >= 0 && accountAge <= 10*time.Minute,
	}
	if err := h.db.Create(&binding).Error; err != nil {
		// 多端同时打开邀请时可能触发唯一键竞争；归因已经存在即视为幂等成功。
		if h.db.First(&existing, "invitee_user_id = ?", auth.UserID).Error == nil {
			httpx.OK(c, gin.H{"bound": false, "reason": "already_bound"})
			return nil
		}
		return err
	}
	httpx.OK(c, gin.H{"bound": true, "newUser": binding.NewUser})
	return nil
}

type recentBinding struct {
	InviteeUserID string
	NewUser       bool
	CreatedAt     time.Time
	Paid          bool
}

func maskedUser(userID string) string {
	if len(userID) > 6 {
		userID = userID[len(userID)-6:]
	}
	return "用户 " + strings.ToUpper(userID)
}

var chinaLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func (h *Handler) dashboard(c *gin.Context) error {
	auth := middleware.Current(c)
	var application models.AgencyApplication
	if err := h.db.Where("user_id = ? AND status = ?", auth.UserID, models.AgencyApplicationApproved).First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.OK(c, gin.H{"isAgency": false})
			return nil
		}
		return err
	}
	if err := h.ensureAgencyCode(&application); err != nil {
		return err
	}

	now := time.Now().In(chinaLocation)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, chinaLocation)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, chinaLocation)
	base := h.db.Model(&models.AgencyUserBinding{}).Where("agency_user_id = ?", auth.UserID)
	var totalInvited, newUserCount, todayCount, monthCount int64
	if err := base.Count(&totalInvited).Error; err != nil {
		return err
	}
	if err := h.db.Model(&models.AgencyUserBinding{}).Where("agency_user_id = ? AND new_user = ?", auth.UserID, true).Count(&newUserCount).Error; err != nil {
		return err
	}
	if err := h.db.Model(&models.AgencyUserBinding{}).Where("agency_user_id = ? AND created_at >= ?", auth.UserID, todayStart).Count(&todayCount).Error; err != nil {
		return err
	}
	if err := h.db.Model(&models.AgencyUserBinding{}).Where("agency_user_id = ? AND created_at >= ?", auth.UserID, monthStart).Count(&monthCount).Error; err != nil {
		return err
	}

	var paidCount int64
	if err := h.db.Raw(`
		SELECT COUNT(DISTINCT b.invitee_user_id)
		FROM agency_user_bindings b
		JOIN orders o ON o.payer_user_id = b.invitee_user_id
		WHERE b.agency_user_id = ? AND o.type = ? AND o.status = ? AND o.paid_at >= b.created_at`,
		auth.UserID, models.OrderMembership, models.OrderPaid).Scan(&paidCount).Error; err != nil {
		return err
	}

	var rows []recentBinding
	if err := h.db.Raw(`
		SELECT b.invitee_user_id, b.new_user, b.created_at,
			EXISTS (
				SELECT 1 FROM orders o
				WHERE o.payer_user_id = b.invitee_user_id
					AND o.type = ? AND o.status = ? AND o.paid_at >= b.created_at
			) AS paid
		FROM agency_user_bindings b
		WHERE b.agency_user_id = ?
		ORDER BY b.created_at DESC
		LIMIT 20`, models.OrderMembership, models.OrderPaid, auth.UserID).Scan(&rows).Error; err != nil {
		return err
	}
	recent := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		recent = append(recent, gin.H{
			"displayName": maskedUser(row.InviteeUserID),
			"newUser":     row.NewUser,
			"paid":        row.Paid,
			"invitedAt":   row.CreatedAt,
		})
	}

	httpx.OK(c, gin.H{
		"isAgency":     true,
		"agencyCode":   *application.AgencyCode,
		"totalInvited": totalInvited,
		"newUserCount": newUserCount,
		"paidCount":    paidCount,
		"todayCount":   todayCount,
		"monthCount":   monthCount,
		"recent":       recent,
	})
	return nil
}

func (h *Handler) qrcode(c *gin.Context) error {
	auth := middleware.Current(c)
	var application models.AgencyApplication
	if err := h.db.Where("user_id = ? AND status = ?", auth.UserID, models.AgencyApplicationApproved).First(&application).Error; err != nil {
		return httpx.Forbidden("NOT_AGENCY", "当前账号不是有效代理商")
	}
	if err := h.ensureAgencyCode(&application); err != nil {
		return err
	}
	code, err := h.genWxacode(*application.AgencyCode)
	if err != nil {
		return httpx.Internal("QRCODE_ERROR", "小程序码生成失败，请稍后重试")
	}
	httpx.OK(c, gin.H{"wxacodeBase64": code})
	return nil
}

func (h *Handler) genWxacode(agencyCode string) (string, error) {
	if h.login == nil {
		return "", errors.New("wechat client not configured")
	}
	token, err := h.login.AccessToken()
	if err != nil {
		return "", err
	}
	payload := map[string]interface{}{
		"scene":       "ac=" + agencyCode,
		"page":        "pages/agency-invite/index",
		"check_path":  false,
		"env_version": h.miniEnv,
		"width":       280,
	}
	body, _ := json.Marshal(payload)
	resp, err := h.hc.Post("https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token="+token, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return "", errors.New("wechat qrcode response is not an image")
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(result), nil
}
