// Package home 聚合「我的首页」：AI 名片（PersonaAI 实例）+ 用户添加到首页的工具 Agent（AgentPin）。
// 阵容由用户自定义（市场添加/移除），服务端组装统一卡片列表；前端只渲染、按 type 路由。
package home

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/membership"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/toolagent"
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
	rg.GET("/home/agents", auth, httpx.Handle(h.agents))
	rg.POST("/home/agents/pin", auth, httpx.Handle(h.pin))              // 添加到首页
	rg.DELETE("/home/agents/pin/:agentId", auth, httpx.Handle(h.unpin)) // 从首页移除
}

// 新用户首页默认放上去的工具 Agent（slug）；只在用户从未种过时种一次。
var defaultPinSlugs = []string{"spoken-english", "business-coach"}

// 工具卡的温度档循环（视觉用）。
var toolTiers = []string{"cool", "lively", "warm"}

// agents 返回当前用户的首页：AI 名片（primary）+ 用户添加的工具 Agent。
func (h *Handler) agents(c *gin.Context) error {
	auth := middleware.Current(c)
	out := make([]gin.H, 0, 6)

	// 1) AI 名片 = 代表你的对外分身（PersonaAI 实例：属于你、内容是你、会对话）。
	var profile models.Profile
	ready := h.db.First(&profile, "user_id = ?", auth.UserID).Error == nil
	persona := gin.H{
		"key": "persona", "type": "persona", "primary": true,
		"tier": "warm", "ready": ready,
	}
	if ready {
		name := "我的 AI 名片"
		if profile.RealName != "" {
			name = profile.RealName + " 的 AI 名片"
		}
		persona["name"] = name
		persona["initial"] = firstRune(profile.RealName, "否")
		persona["line"] = "我替你把对外的事看着，有结果就喊你。"
		persona["profileId"] = profile.ID
	} else {
		persona["name"] = "我的 AI 名片"
		persona["initial"] = "+"
		persona["line"] = "先建一个，别人点开就能直接问你、和你聊。"
	}
	out = append(out, persona)

	// 2) 工具 Agent = 用户添加到首页的（AgentPin）；新用户无 pin 则种默认一次。
	h.ensureDefaultPins(auth.UserID)
	var pins []models.AgentPin
	h.db.Where("user_id = ?", auth.UserID).Order("sort asc, created_at asc").Find(&pins)
	member := membership.IsActive(h.db, auth.UserID)
	for i := range pins {
		var a models.ToolAgent
		if h.db.First(&a, "id = ? AND enabled = ?", pins[i].AgentID, true).Error != nil {
			continue // Agent 下架/不存在 → 跳过
		}
		out = append(out, h.toolCard(&a, auth.UserID, member, len(out)))
	}

	httpx.OK(c, out)
	return nil
}

// ensureDefaultPins 新用户首次进首页：若从未种过，按默认 slug 自动 pin 一次（不空场、语义统一；
// 用 User.HomeSeeded 记录，保证「移除全部」后默认不会再回来）。
func (h *Handler) ensureDefaultPins(userID string) {
	var u models.User
	if h.db.First(&u, "id = ?", userID).Error != nil || u.HomeSeeded {
		return
	}
	for i, slug := range defaultPinSlugs {
		var a models.ToolAgent
		if h.db.First(&a, "slug = ? AND enabled = ?", slug, true).Error == nil {
			h.db.Create(&models.AgentPin{ID: idgen.New(), UserID: userID, AgentID: a.ID, Sort: i})
		}
	}
	h.db.Model(&models.User{}).Where("id = ?", userID).Update("home_seeded", true)
}

// toolCard 把一个 ToolAgent + 用户状态序列化成首页工具卡。
func (h *Handler) toolCard(a *models.ToolAgent, userID string, member bool, idx int) gin.H {
	initial := a.Icon
	if initial == "" {
		initial = firstRune(a.Name, "A")
	}
	remaining := a.FreeTrial
	var ent models.AgentEntitlement
	if h.db.First(&ent, "user_id = ? AND agent_id = ?", userID, a.ID).Error == nil {
		remaining = ent.Remaining
	}
	// L2 催课条：学过的 Agent 用动态学习状态替代静态 tagline（用户没点进来，主动性已发生）。
	line, nudge := a.Tagline, false
	if s, ok := toolagent.NudgeLine(h.db, a, userID); ok && s != "" {
		line, nudge = s, true
	}
	return gin.H{
		"key": a.Slug, "type": "tool", "name": a.Name, "line": line, "nudge": nudge,
		"initial": initial, "tier": toolTiers[idx%len(toolTiers)], "agentId": a.ID,
		"member": member, "freeRemaining": remaining,
	}
}

// ---------- 添加 / 移除 ----------

type pinReq struct {
	AgentID string `json:"agentId" binding:"required"`
}

func (h *Handler) pin(c *gin.Context) error {
	auth := middleware.Current(c)
	var req pinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "参数错误")
	}
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", req.AgentID, true).Error; err != nil {
		return httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	var existing models.AgentPin
	if h.db.First(&existing, "user_id = ? AND agent_id = ?", auth.UserID, a.ID).Error == nil {
		httpx.OK(c, gin.H{"pinned": true}) // 幂等
		return nil
	}
	var maxSort struct{ M int }
	h.db.Model(&models.AgentPin{}).
		Select("COALESCE(MAX(sort), -1) as m").
		Where("user_id = ?", auth.UserID).Scan(&maxSort)
	h.db.Create(&models.AgentPin{ID: idgen.New(), UserID: auth.UserID, AgentID: a.ID, Sort: maxSort.M + 1})
	httpx.OK(c, gin.H{"pinned": true})
	return nil
}

func (h *Handler) unpin(c *gin.Context) error {
	auth := middleware.Current(c)
	h.db.Where("user_id = ? AND agent_id = ?", auth.UserID, c.Param("agentId")).Delete(&models.AgentPin{})
	httpx.OK(c, gin.H{"pinned": false})
	return nil
}

func firstRune(s, fallback string) string {
	r := []rune(s)
	if len(r) == 0 {
		return fallback
	}
	return string(r[0])
}
