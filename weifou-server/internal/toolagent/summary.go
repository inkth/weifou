package toolagent

import (
	"math"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/httpx"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

// learningSummary 为“我的”页提供一次性摘要，避免客户端为每门课程分别拉取进度。
// 最近课程以真实发生过学习对话的 session 为准；全局掌握数来自 UserConcept。
func (h *Handler) learningSummary(c *gin.Context) error {
	auth := middleware.Current(c)

	streak, todayDone := streakOf(h.db, auth.UserID)

	var mastered int64
	h.db.Model(&models.UserConcept{}).
		Where("user_id = ? AND level >= ?", auth.UserID, 2).
		Count(&mastered)

	var learningCourses int64
	h.db.Table("agent_sessions AS s").
		Joins("JOIN tool_agents AS a ON a.id = s.agent_id").
		Where("s.user_id = ? AND a.enabled = ? AND (a.assess = ? OR a.concept = ?)", auth.UserID, true, true, true).
		Distinct("s.agent_id").
		Count(&learningCourses)

	resp := gin.H{
		"streak": gin.H{
			"days": streak.Current, "best": streak.Best, "todayDone": todayDone,
		},
		"mastered":        mastered,
		"learningCourses": learningCourses,
		"current":         nil,
	}

	var session models.AgentSession
	err := h.db.Table("agent_sessions AS s").
		Select("s.*").
		Joins("JOIN tool_agents AS a ON a.id = s.agent_id").
		Where("s.user_id = ? AND a.enabled = ? AND (a.assess = ? OR a.concept = ?)", auth.UserID, true, true, true).
		Order("s.updated_at DESC").
		First(&session).Error
	if err != nil {
		httpx.OK(c, resp)
		return nil
	}

	var agent models.ToolAgent
	if h.db.First(&agent, "id = ? AND enabled = ?", session.AgentID, true).Error != nil {
		httpx.OK(c, resp)
		return nil
	}

	var total, lit, courseMastered int64
	if agent.Concept {
		h.db.Model(&models.AgentConcept{}).Where("agent_id = ?", agent.ID).Count(&total)
		h.db.Model(&models.UserConcept{}).
			Where("user_id = ? AND agent_id = ? AND level >= ?", auth.UserID, agent.ID, 1).
			Count(&lit)
		h.db.Model(&models.UserConcept{}).
			Where("user_id = ? AND agent_id = ? AND level >= ?", auth.UserID, agent.ID, 2).
			Count(&courseMastered)
	}

	percent := 0
	if total > 0 {
		percent = int(math.Round(float64(lit) * 100 / float64(total)))
		if percent > 100 {
			percent = 100
		}
	}

	resp["current"] = gin.H{
		"id": agent.ID, "name": agent.Name, "subject": agent.Subject, "guide": agent.Guide,
		"tagline": agent.Tagline, "icon": agent.Icon, "accent": agent.Accent,
		"lit": lit, "total": total, "mastered": courseMastered, "progressPercent": percent,
		"updatedAt": session.UpdatedAt,
	}
	httpx.OK(c, resp)
	return nil
}
