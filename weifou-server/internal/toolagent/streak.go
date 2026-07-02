// Package toolagent — streak.go：连续学习天数（streak）+ 学习提醒承诺（L3 离线主动）。
//
// streak 纪律（轻而真、温和版）：
//   - 记一天的条件 = 与任一「学习型」Agent（Assess/Concept）真实对话过一轮，跨 Agent 全局一条；
//   - 断一天不清零：每自然月可自动补签 1 次（freeze），拒绝多邻国式焦虑轰炸；
//   - 日期按东八区（用户在国内，容器时区不可依赖）。
//
// 提醒承诺（implementation intention）：课后学员点「明天这个点叫你继续吗」并授权一次性
// 订阅消息 → 建一条 LearnReminder(+24h)；后台循环到点发送。一次授权=一次发送（微信硬约束），
// 发送内容复用 NudgeLine（提醒里就带着「下一个概念」）。模板未配时全链路静默 no-op。
package toolagent

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
	"weifou-server/internal/wechat"
)

// cnLoc 东八区：streak 的"一天"以中国时间为准。
var cnLoc = time.FixedZone("CST", 8*3600)

func dayStr(t time.Time) string   { return t.In(cnLoc).Format("2006-01-02") }
func monthStr(t time.Time) string { return t.In(cnLoc).Format("2006-01") }

// bumpStreak 记一次学习活动。返回（当前连续天数, 是否今天首次学习, 是否消耗了本月补签）。
func bumpStreak(db *gorm.DB, userID string) (int, bool, bool) {
	now := time.Now()
	today := dayStr(now)
	yesterday := dayStr(now.AddDate(0, 0, -1))
	beforeYest := dayStr(now.AddDate(0, 0, -2))

	var s models.LearnStreak
	if err := db.First(&s, "user_id = ?", userID).Error; err == gorm.ErrRecordNotFound {
		s = models.LearnStreak{ID: idgen.New(), UserID: userID, Current: 1, Best: 1, LastDay: today}
		if cerr := db.Create(&s).Error; cerr == nil {
			return 1, true, false
		}
		db.First(&s, "user_id = ?", userID) // 并发兜底：他人先建 → 走更新路径
	}
	if s.LastDay == today {
		return s.Current, false, false // 今天已记过
	}
	usedFreeze := false
	switch s.LastDay {
	case yesterday:
		s.Current++
	case beforeYest:
		// 恰好断了一天：本月还没补签过 → 自动补签续上
		if s.FreezeMonth != monthStr(now) {
			s.Current++
			s.FreezeMonth = monthStr(now)
			usedFreeze = true
		} else {
			s.Current = 1
		}
	default:
		s.Current = 1
	}
	if s.Current > s.Best {
		s.Best = s.Current
	}
	s.LastDay = today
	db.Model(&s).Updates(map[string]interface{}{
		"current": s.Current, "best": s.Best, "last_day": s.LastDay, "freeze_month": s.FreezeMonth,
	})
	return s.Current, true, usedFreeze
}

// streakOf 读取 streak（不存在返回零值）。todayDone=今天是否已学。
func streakOf(db *gorm.DB, userID string) (models.LearnStreak, bool) {
	var s models.LearnStreak
	if db.First(&s, "user_id = ?", userID).Error != nil {
		return s, false
	}
	return s, s.LastDay == dayStr(time.Now())
}

// streakAtRisk 昨天学了、今天还没学、连续≥2 天 → 该催了（首页催课条用）。
func streakAtRisk(db *gorm.DB, userID string) (int, bool) {
	s, todayDone := streakOf(db, userID)
	if todayDone || s.Current < 2 {
		return 0, false
	}
	if s.LastDay == dayStr(time.Now().AddDate(0, 0, -1)) {
		return s.Current, true
	}
	return 0, false
}

// ---------- 接口 ----------

// streakInfo GET /agents/streak：连续学习天数（发现页头部/庆祝用）。
func (h *Handler) streakInfo(c *gin.Context) error {
	auth := middleware.Current(c)
	s, todayDone := streakOf(h.db, auth.UserID)
	httpx.OK(c, gin.H{"days": s.Current, "best": s.Best, "todayDone": todayDone})
	return nil
}

// remind POST /agents/:id/remind：学员授权订阅消息后落一条「明天这个点」的提醒承诺。
// 同一 user+agent 只保留一条未发送的（重复承诺=顺延到最新时间）。
func (h *Handler) remind(c *gin.Context) error {
	auth := middleware.Current(c)
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", c.Param("id"), true).Error; err != nil {
		return httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	sendAt := time.Now().Add(24 * time.Hour)
	h.db.Where("user_id = ? AND agent_id = ? AND sent = ?", auth.UserID, a.ID, false).
		Delete(&models.LearnReminder{})
	h.db.Create(&models.LearnReminder{
		ID: idgen.New(), UserID: auth.UserID, AgentID: a.ID, SendAt: sendAt,
	})
	httpx.OK(c, gin.H{"sendAt": sendAt.In(cnLoc).Format("01-02 15:04")})
	return nil
}

// ---------- 提醒发送循环 ----------

// StartLearnRemindLoop 后台循环：每分钟扫到期未发的提醒，逐条发订阅消息。
// 提醒文案优先用 NudgeLine（带着「下一个概念」去提醒，而不是干巴巴的"该学习了"）。
// 发送即标记 Sent（一次授权一次额度，失败不重试不轰炸）。模板未配时循环不启动。
func StartLearnRemindLoop(db *gorm.DB, subs *wechat.SubscribeService) {
	if db == nil || subs == nil || !subs.LearnRemindReady() {
		return
	}
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for range t.C {
			var due []models.LearnReminder
			db.Where("sent = ? AND send_at <= ?", false, time.Now()).Limit(50).Find(&due)
			for i := range due {
				r := due[i]
				db.Model(&models.LearnReminder{}).Where("id = ?", r.ID).Update("sent", true)
				var u models.User
				if db.First(&u, "id = ?", r.UserID).Error != nil {
					continue
				}
				openid := u.Openid
				if u.WxMpOpenid != nil && *u.WxMpOpenid != "" {
					openid = *u.WxMpOpenid
				}
				var a models.ToolAgent
				if db.First(&a, "id = ?", r.AgentID).Error != nil {
					continue
				}
				note := "继续昨天的进度，几分钟就好"
				if s, ok := NudgeLine(db, &a, r.UserID); ok && s != "" {
					note = s
				}
				subs.NotifyLearnRemind(openid, a.Name, note, time.Now().In(cnLoc),
					"pages/agent-chat/index?id="+a.ID)
			}
			if len(due) > 0 {
				log.Printf("[learn-remind] sent %d reminder(s)", len(due))
			}
		}
	}()
}
