package share

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

type Handler struct {
	db    *gorm.DB
	login *wechat.LoginClient
	hc    *http.Client
}

func NewHandler(db *gorm.DB, login *wechat.LoginClient) *Handler {
	return &Handler{db: db, login: login, hc: &http.Client{Timeout: 10 * time.Second}}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/share/bundle/:profileId", httpx.Handle(h.bundle))
}

func (h *Handler) bundle(c *gin.Context) error {
	profileID := c.Param("profileId")
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", profileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_READY", "主页未生成")
	}
	var p models.PersonaAI
	if err := h.db.First(&p, "profile_id = ?", profileID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_READY", "主页未生成")
	}
	var u models.User
	h.db.First(&u, "id = ?", profile.UserID)

	var tags []string
	_ = json.Unmarshal(p.Tags, &tags)

	wxacode := h.genWxacode(profileID)

	httpx.OK(c, gin.H{
		"profileId":     profileID,
		"nickname":      u.Nickname,
		"realName":      profile.RealName,
		"avatarUrl":     u.AvatarURL,
		"avatarStyle":   profile.AvatarStyle,
		"oneLiner":      p.OneLiner,
		"tags":          tags,
		"wxacodeBase64": wxacode,
	})
	return nil
}

// genWxacode 调 getwxacodeunlimit，返回 data URI；失败返回空串
func (h *Handler) genWxacode(profileID string) interface{} {
	token, err := h.login.AccessToken()
	if err != nil || token == "" {
		return nil
	}
	payload := map[string]interface{}{
		"scene": "id=" + profileID,
		// TODO(chat-first): 支持 scene 解析的小程序新版本全量后，把 page 切到 "pages/chat/index"
		// （老客户端 chat 页不解析 scene，提前切会导致新码在老版本上拿不到 id）。
		// 在此之前 profile 页已做访客分流，扫码进 profile 也会被送进对话，体验无损。
		"page":        "pages/profile/index",
		"check_path":  false,
		"env_version": "trial", // 体验版；正式发布前需配置化
		"width":       280,
	}
	buf, _ := json.Marshal(payload)
	resp, err := h.hc.Post(
		"https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token="+token,
		"application/json", bytes.NewReader(buf))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	body, _ := io.ReadAll(resp.Body)
	if strings.HasPrefix(ct, "image/") {
		return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(body)
	}
	return nil
}
