package profile

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/persona"
)

type Handler struct {
	db        *gorm.DB
	persona   *persona.Service
	jwtSecret string
}

func NewHandler(db *gorm.DB, p *persona.Service, jwtSecret string) *Handler {
	return &Handler{db: db, persona: p, jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.POST("/profile", auth, httpx.Handle(h.createOrUpdate))
	rg.POST("/profile/extract", auth, httpx.Handle(h.extract))
	rg.POST("/profile/suggest", auth, httpx.Handle(h.suggest))
	rg.POST("/profile/regenerate", auth, httpx.Handle(h.regenerate))
	rg.GET("/profile/mine", auth, httpx.Handle(h.mine))
	rg.PATCH("/profile/contact", auth, httpx.Handle(h.updateContact))
	rg.PATCH("/profile/avatar", auth, httpx.Handle(h.updateAvatar))
	rg.PATCH("/profile/persona", auth, httpx.Handle(h.updatePersona))
	rg.PATCH("/profile/discoverable", auth, httpx.Handle(h.updateDiscoverable))
	rg.GET("/profile/:id", httpx.Handle(h.findOne))
	rg.GET("/profile/:id/contact", httpx.Handle(h.contact))

	h.registerInbox(rg)
}

type createReq struct {
	RealName string `json:"realName" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Company  string `json:"company"`
	City     string `json:"city"`
	// 快速创建（裂变通道）只填 strengths 一项；recentWork/howToKnow 选填，persona 生成时跳过空项
	Strengths   string `json:"strengths" binding:"required"`
	RecentWork  string `json:"recentWork"`
	HowToKnow   string `json:"howToKnow"`
	AvatarStyle string `json:"avatarStyle"`
	Style       string `json:"style"` // 对外沟通风格，白名单见 persona.StyleDescriptions
	Ref         string `json:"ref"`   // 裂变归因：来源 Agent 的 profileId，仅创建时落库
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *Handler) createOrUpdate(c *gin.Context) error {
	auth := middleware.Current(c)
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "请完整填写信息")
	}
	req.Style = persona.NormalizeStyle(req.Style)

	var profile models.Profile
	err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error
	if err == gorm.ErrRecordNotFound {
		// 归因仅在首次创建时写入；校验来源 profile 真实存在，垃圾值直接丢弃
		var referrer *string
		if req.Ref != "" && len(req.Ref) <= 32 {
			var cnt int64
			h.db.Model(&models.Profile{}).Where("id = ?", req.Ref).Count(&cnt)
			if cnt > 0 {
				referrer = strPtr(req.Ref)
			}
		}
		profile = models.Profile{
			ID:                idgen.New(),
			UserID:            auth.UserID,
			RealName:          req.RealName,
			Title:             req.Title,
			Company:           strPtr(req.Company),
			City:              strPtr(req.City),
			AvatarStyle:       req.AvatarStyle,
			ReferrerProfileID: referrer,
			Status:            models.ProfileDraft,
		}
		if err := h.db.Create(&profile).Error; err != nil {
			return httpx.Internal("DB_ERROR", "创建失败")
		}
	} else if err != nil {
		return httpx.Internal("DB_ERROR", "查询失败")
	} else {
		updates := map[string]interface{}{
			"real_name": req.RealName,
			"title":     req.Title,
			"company":   strPtr(req.Company),
			"city":      strPtr(req.City),
		}
		if req.AvatarStyle != "" {
			updates["avatar_style"] = req.AvatarStyle
		}
		h.db.Model(&profile).Updates(updates)
	}

	// upsert PersonaInput
	var input models.PersonaInput
	err = h.db.Where("profile_id = ?", profile.ID).First(&input).Error
	if err == gorm.ErrRecordNotFound {
		h.db.Create(&models.PersonaInput{
			ID:         idgen.New(),
			ProfileID:  profile.ID,
			Strengths:  req.Strengths,
			RecentWork: req.RecentWork,
			HowToKnow:  req.HowToKnow,
			Style:      req.Style,
		})
	} else {
		h.db.Model(&input).Updates(map[string]interface{}{
			"strengths":   req.Strengths,
			"recent_work": req.RecentWork,
			"how_to_know": req.HowToKnow,
			"style":       req.Style,
		})
	}

	if _, err := h.persona.GenerateForProfile(profile.ID, auth.Openid); err != nil {
		return err
	}
	h.db.Model(&models.Profile{}).Where("id = ?", profile.ID).Update("status", models.ProfilePublished)

	data, err := h.publicByID(profile.ID)
	if err != nil {
		return err
	}
	httpx.OK(c, data)
	return nil
}

type extractMsg struct {
	Role string `json:"role"` // 'ai' | 'me'
	Text string `json:"text"`
}

type extractReq struct {
	Messages []extractMsg `json:"messages"`
}

// extract 对话式创建：从对话消息抽取建主页字段 + 自动判定沟通风格。不写库，仅返回抽取结果。
func (h *Handler) extract(c *gin.Context) error {
	auth := middleware.Current(c)
	var req extractReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数有误")
	}
	var b strings.Builder
	for _, m := range req.Messages {
		t := strings.TrimSpace(m.Text)
		if t == "" {
			continue
		}
		who := "用户"
		if m.Role == "ai" {
			who = "AI助理"
		}
		b.WriteString(who + "：" + t + "\n")
	}
	res, err := h.persona.ExtractProfileFields(auth.Openid, b.String())
	if err != nil {
		return err
	}
	httpx.OK(c, res)
	return nil
}

type suggestReq struct {
	Title    string   `json:"title" binding:"required"` // 已选/已说的职业或领域
	Audience string   `json:"audience"`                 // 主要接待谁（选填）
	Exclude  []string `json:"exclude"`                  // 已展示过的候选（换一批时传入）
}

// suggest 对话式创建「卖点」一步的点选候选：按职业+受众生成 4 条供点选，换一批带 exclude。
func (h *Handler) suggest(c *gin.Context) error {
	auth := middleware.Current(c)
	var req suggestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数有误")
	}
	opts, err := h.persona.SuggestStrengths(auth.Openid, req.Title, req.Audience, req.Exclude)
	if err != nil {
		return err
	}
	httpx.OK(c, gin.H{"options": opts})
	return nil
}

func (h *Handler) regenerate(c *gin.Context) error {
	auth := middleware.Current(c)
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建你的 AI 分身")
	}
	if _, err := h.persona.GenerateForProfile(profile.ID, auth.Openid); err != nil {
		return err
	}
	data, err := h.publicByID(profile.ID)
	if err != nil {
		return err
	}
	httpx.OK(c, data)
	return nil
}

// publicByID 返回访客视图（不含联系方式）
func (h *Handler) publicByID(profileID string) (gin.H, error) {
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", profileID).Error; err != nil {
		return nil, httpx.NotFound("PROFILE_NOT_FOUND", "AI 分身不存在")
	}
	var u models.User
	h.db.First(&u, "id = ?", profile.UserID)

	var p models.PersonaAI
	var personaData interface{}
	if err := h.db.First(&p, "profile_id = ?", profileID).Error; err == nil {
		var tags []string
		_ = json.Unmarshal(p.Tags, &tags)
		var starters []string
		_ = json.Unmarshal(p.Starters, &starters)
		personaData = gin.H{
			"oneLiner":   p.OneLiner,
			"fullIntro":  p.FullIntro,
			"tags":       tags,
			"starters":   starters,
			"greeting":   p.Greeting,
			"tone":       p.Tone,
			"voiceStyle": p.VoiceStyle,
			"avatarUrl":  p.AvatarURL,
		}
	}

	return gin.H{
		"id":             profile.ID,
		"realName":       profile.RealName,
		"title":          profile.Title,
		"company":        profile.Company,
		"city":           profile.City,
		"nickname":       u.Nickname,
		"avatarUrl":      u.AvatarURL,
		"avatarStyle":    profile.AvatarStyle,
		"status":         profile.Status,
		"contactVisible": profile.ContactVisible,
		"discoverable":   profile.Discoverable,
		"persona":        personaData,
		"trust":          h.trustFor(profileID),
	}, nil
}

// trustFor 聚合社会证明信号（全部派生自既有交易数据，无新表/无埋点）：
// 付费咨询过 TA 的人数、已答提问数、平均回答时长、答复率。
// 冷启动数字过小时由前端隐藏，仅保留"答不上全额退"保障文案。
func (h *Handler) trustFor(profileID string) gin.H {
	// 已答付费提问的提问人数（去重）
	var answeredAskers int64
	h.db.Model(&models.AsyncQuestion{}).
		Where("profile_id = ? AND status = ?", profileID, models.AsyncAnswered).
		Distinct("asker_openid").Count(&answeredAskers)

	var answeredCount, totalCount int64
	h.db.Model(&models.AsyncQuestion{}).
		Where("profile_id = ? AND status = ?", profileID, models.AsyncAnswered).Count(&answeredCount)
	h.db.Model(&models.AsyncQuestion{}).
		Where("profile_id = ?", profileID).Count(&totalCount)

	// 平均回答时长（小时）：已答提问从提出到作答的时长
	var avgHours float64
	h.db.Model(&models.AsyncQuestion{}).
		Where("profile_id = ? AND status = ? AND answered_at IS NOT NULL", profileID, models.AsyncAnswered).
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (answered_at - created_at)) / 3600.0), 0)").
		Scan(&avgHours)

	repliedRate := 0
	if totalCount > 0 {
		repliedRate = int(answeredCount * 100 / totalCount)
	}

	return gin.H{
		"consultedPeople": answeredAskers,
		"answeredCount":   answeredCount,
		"avgAnswerHours":  math.Round(avgHours*10) / 10,
		"repliedRate":     repliedRate,
	}
}

func (h *Handler) findOne(c *gin.Context) error {
	data, err := h.publicByID(c.Param("id"))
	if err != nil {
		return err
	}
	httpx.OK(c, data)
	return nil
}

func (h *Handler) mine(c *gin.Context) error {
	auth := middleware.Current(c)
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		httpx.OK(c, nil)
		return nil
	}
	data, err := h.publicByID(profile.ID)
	if err != nil {
		return err
	}
	data["contactWechat"] = profile.ContactWechat
	data["contactPhone"] = profile.ContactPhone

	var input models.PersonaInput
	if err := h.db.First(&input, "profile_id = ?", profile.ID).Error; err == nil {
		data["personaInput"] = gin.H{
			"strengths":  input.Strengths,
			"recentWork": input.RecentWork,
			"howToKnow":  input.HowToKnow,
			"style":      input.Style,
		}
	}
	httpx.OK(c, data)
	return nil
}

type contactReq struct {
	ContactWechat  *string `json:"contactWechat"`
	ContactPhone   *string `json:"contactPhone"`
	ContactVisible *bool   `json:"contactVisible"`
}

func (h *Handler) updateContact(c *gin.Context) error {
	auth := middleware.Current(c)
	var req contactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建你的 AI 分身")
	}
	updates := map[string]interface{}{}
	if req.ContactWechat != nil {
		updates["contact_wechat"] = *req.ContactWechat
	}
	if req.ContactPhone != nil {
		updates["contact_phone"] = *req.ContactPhone
	}
	if req.ContactVisible != nil {
		updates["contact_visible"] = *req.ContactVisible
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		h.db.Model(&profile).Updates(updates)
	}
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

type avatarReq struct {
	AvatarStyle string `json:"avatarStyle" binding:"required"`
}

func (h *Handler) updateAvatar(c *gin.Context) error {
	auth := middleware.Current(c)
	var req avatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	if len(req.AvatarStyle) > 32 {
		return httpx.BadRequest("INVALID_PARAMS", "形象标识过长")
	}
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建你的 AI 分身")
	}
	h.db.Model(&profile).Updates(map[string]interface{}{
		"avatar_style": req.AvatarStyle,
		"updated_at":   time.Now(),
	})
	httpx.OK(c, gin.H{"ok": true, "avatarStyle": req.AvatarStyle})
	return nil
}

type personaReq struct {
	OneLiner   *string `json:"oneLiner"`
	Greeting   *string `json:"greeting"`
	Tone       *string `json:"tone"`
	VoiceStyle *string `json:"voiceStyle"`
}

// updatePersona 让本人手动微调 AI 人设（开场白/语气/音色/一句话）——人设深度定制。
func (h *Handler) updatePersona(c *gin.Context) error {
	auth := middleware.Current(c)
	var req personaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建你的 AI 分身")
	}
	var p models.PersonaAI
	if err := h.db.First(&p, "profile_id = ?", profile.ID).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_READY", "AI 分身还没生成好")
	}

	// 对用户改写的文本做内容安全校验。
	check := ""
	updates := map[string]interface{}{}
	if req.OneLiner != nil {
		updates["one_liner"] = strings.TrimSpace(*req.OneLiner)
		check += *req.OneLiner + "\n"
	}
	if req.Greeting != nil {
		updates["greeting"] = strings.TrimSpace(*req.Greeting)
		check += *req.Greeting + "\n"
	}
	if req.Tone != nil {
		updates["tone"] = strings.TrimSpace(*req.Tone)
		check += *req.Tone + "\n"
	}
	if req.VoiceStyle != nil {
		vs := strings.TrimSpace(*req.VoiceStyle)
		if len(vs) > 32 {
			return httpx.BadRequest("INVALID_PARAMS", "音色标识过长")
		}
		updates["voice_style"] = vs
	}
	if len(updates) == 0 {
		return httpx.BadRequest("INVALID_PARAMS", "无可更新字段")
	}
	if check != "" && !h.persona.CheckText(check, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "内容包含敏感信息，请修改后重试")
	}
	if err := h.db.Model(&p).Updates(updates).Error; err != nil {
		return httpx.Internal("DB_ERROR", "更新失败")
	}
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

type discoverableReq struct {
	Discoverable bool `json:"discoverable"`
}

// updateDiscoverable 切换"公开到人物广场"（opt-in）。
func (h *Handler) updateDiscoverable(c *gin.Context) error {
	auth := middleware.Current(c)
	var req discoverableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "请先创建你的 AI 分身")
	}
	h.db.Model(&profile).Updates(map[string]interface{}{
		"discoverable": req.Discoverable,
		"updated_at":   time.Now(),
	})
	httpx.OK(c, gin.H{"ok": true, "discoverable": req.Discoverable})
	return nil
}

func (h *Handler) contact(c *gin.Context) error {
	var profile models.Profile
	if err := h.db.First(&profile, "id = ?", c.Param("id")).Error; err != nil {
		return httpx.NotFound("PROFILE_NOT_FOUND", "AI 分身不存在")
	}
	if !profile.ContactVisible {
		return httpx.Forbidden("CONTACT_HIDDEN", "本人未公开联系方式")
	}
	httpx.OK(c, gin.H{"wechat": profile.ContactWechat, "phone": profile.ContactPhone})
	return nil
}
