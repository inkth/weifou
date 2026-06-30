// Package home 聚合「我的首页 Agent 小队」：把三种底层数据源（PersonaAI 实例 /
// 平台 ToolAgent / 玩法）抹平成一个统一的卡片列表，由服务端决定阵容、顺序与每张卡的状态。
// 前端只渲染、按 type 路由——加减换 Agent、调顺序、做会员灰度都改这里，不用前端发版。
package home

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/membership"
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
	rg.GET("/home/agents", auth, httpx.Handle(h.agents))
}

// 首页工具分身阵容：展示名/文案与 ToolAgent 内部名解耦（slug 取真实 Agent 与状态）。
// 顺序即展示顺序；要加/减/换工具分身，改这张表即可。
var homeTools = []struct{ Slug, Name, Line, Initial, Tier string }{
	{"spoken-english", "学英语分身", "陪你开口练——纠音、对话、一段段升级。", "EN", "cool"},
	{"business-coach", "学商业分身", "生意卡哪了？我陪你拆，给能落地的下一步。", "商", "lively"},
}

// agents 返回当前用户的首页 Agent 小队（主分身 + 工具分身 + 找对象）。
func (h *Handler) agents(c *gin.Context) error {
	auth := middleware.Current(c)
	out := make([]gin.H, 0, len(homeTools)+2)

	// 1) 主分身 = 用户的 PersonaAI 实例（属于你、内容是你）。
	var profile models.Profile
	ready := h.db.First(&profile, "user_id = ?", auth.UserID).Error == nil
	persona := gin.H{
		"key": "persona", "type": "persona", "primary": true,
		"tier": "warm", "ready": ready,
	}
	if ready {
		name := "我的主分身"
		if profile.RealName != "" {
			name = profile.RealName + " 的主分身"
		}
		persona["name"] = name
		persona["initial"] = firstRune(profile.RealName, "否")
		persona["line"] = "我替你把对外的事看着，有结果就喊你。"
		persona["profileId"] = profile.ID
	} else {
		persona["name"] = "我的主分身"
		persona["initial"] = "+"
		persona["line"] = "先建一个，替你对外接待、有结果喊你。"
	}
	out = append(out, persona)

	// 2) 工具分身 = 平台 ToolAgent（你「用」不「拥有」），带会员/免费状态。
	member := membership.IsActive(h.db, auth.UserID)
	for _, t := range homeTools {
		card := gin.H{
			"key": t.Slug, "type": "tool", "name": t.Name, "line": t.Line,
			"initial": t.Initial, "tier": t.Tier, "member": member,
		}
		var a models.ToolAgent
		if h.db.First(&a, "slug = ? AND enabled = ?", t.Slug, true).Error == nil {
			card["agentId"] = a.ID
			remaining := a.FreeTrial
			var ent models.AgentEntitlement
			if h.db.First(&ent, "user_id = ? AND agent_id = ?", auth.UserID, a.ID).Error == nil {
				remaining = ent.Remaining
			}
			card["freeRemaining"] = remaining
		} else {
			card["agentId"] = "" // 该 Agent 还没 seed → 前端提示「正在上线」
		}
		out = append(out, card)
	}

	// 3) 找对象 = 玩法/工具（结果回喂主分身画像）。
	out = append(out, gin.H{
		"key": "dating", "type": "dating", "name": "找对象分身",
		"line": "测测你和谁最配，顺手喂懂我的主分身。", "initial": "❤", "tier": "lively",
	})

	httpx.OK(c, out)
	return nil
}

func firstRune(s, fallback string) string {
	r := []rune(s)
	if len(r) == 0 {
		return fallback
	}
	return string(r[0])
}
