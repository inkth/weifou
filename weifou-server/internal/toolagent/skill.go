package toolagent

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/deepseek"
	"weifou-server/internal/idgen"
	"weifou-server/internal/models"
)

// 三维 → 总段位（5 档）。段位由三维均分派生，名字是「身份」而非数字，主打升级感。
// 因三维只升不降，段位也只升不降。
var skillTiers = []string{"敢开口", "撑得住", "接得上", "谈得拢", "镇得场"}

// levelFromDims 由三维均分映射到 1-5 段位。
func levelFromDims(f, a, e int) int {
	avg := (f + a + e) / 3
	switch {
	case avg >= 80:
		return 5
	case avg >= 60:
		return 4
	case avg >= 40:
		return 3
	case avg >= 20:
		return 2
	default:
		return 1
	}
}

func tierName(level int) string {
	if level >= 1 && level <= len(skillTiers) {
		return skillTiers[level-1]
	}
	return skillTiers[0]
}

// ratchet 把单维分数向 target 爬升，「只升不降」+ 前段快后段慢（每轮小幅，保证次次有动、且高分越来越难）。
func ratchet(old, target int) int {
	if target <= old {
		return old // 状态差的一轮不掉分
	}
	var step int
	switch {
	case old < 40:
		step = 10
	case old < 70:
		step = 6
	default:
		step = 3
	}
	if old+step > target {
		return target
	}
	return old + step
}

// 判断用户这一轮是否给出了「足够的英文」值得评估（纯中文提问/极短只言片语不评）。
var latinRe = regexp.MustCompile(`[A-Za-z]`)

func hasEnoughEnglish(s string) bool {
	letters := len(latinRe.FindAllString(s, -1))
	if letters < 10 {
		return false
	}
	// 至少 3 个英文词，避免 "ok"、"yes" 之类单词被当成一次有效口语样本。
	words := 0
	for _, w := range strings.Fields(s) {
		if latinRe.MatchString(w) {
			words++
		}
	}
	return words >= 3
}

// loadSkill 取（或建）用户在该 Agent 的能力档案。
func (h *Handler) loadSkill(userID, agentID string) *models.AgentSkill {
	var sk models.AgentSkill
	if err := h.db.First(&sk, "user_id = ? AND agent_id = ?", userID, agentID).Error; err == gorm.ErrRecordNotFound {
		sk = models.AgentSkill{ID: idgen.New(), UserID: userID, AgentID: agentID}
		if cerr := h.db.Create(&sk).Error; cerr != nil {
			h.db.First(&sk, "user_id = ? AND agent_id = ?", userID, agentID) // 并发下他人先建 → 重查
		}
	}
	return &sk
}

type assessResult struct {
	Fluency    int    `json:"fluency"`
	Accuracy   int    `json:"accuracy"`
	Expression int    `json:"expression"`
	Note       string `json:"note"`
}

const skillAssessPrompt = `你是英语口语水平评估器。根据【用户这次的英文表达样本】，给出对该用户【当前真实英语能力】的三维估分（0-100 绝对水平，不是本句打分）：
- fluency 流利度：是否不卡顿、不犹豫、能成段表达、少填充词。
- accuracy 准确度：语法与用词的正确程度（错得越少越高）。
- expression 表达力：是否地道、用词丰富、句式有变化（Chinglish/简单堆砌则低）。
评分参考：0-19 初学只蹦词；20-39 能拼短句但磕巴/多错；40-59 能日常成段、偶有错；60-79 较流畅地道、错少；80-100 接近母语者。
再给一句 note：用中文、20 字以内、口吻鼓励，**具体指出这次做得好的一个点或一个可升级的表达**（如「把 very like 换成 really into，更地道了」）。
只输出 JSON：{"fluency":<int>,"accuracy":<int>,"expression":<int>,"note":"<中文一句>"}`

// assessAndUpdate 对用户本轮英文样本评估，按「首轮定级、其后只升不降」更新档案，返回是否升级。
// 评估失败/无英文样本时不动档案、不报错（不拖累主对话）。
func (h *Handler) assessAndUpdate(sk *models.AgentSkill, sample string) (changed, leveledUp bool) {
	if !hasEnoughEnglish(sample) {
		return false, false
	}
	raw, err := h.ds.Chat(
		[]deepseek.Message{
			{Role: "system", Content: skillAssessPrompt},
			{Role: "user", Content: sample},
		},
		deepseek.ChatOptions{Temperature: 0, MaxTokens: 220, ResponseFormat: "json_object"},
	)
	if err != nil {
		return false, false
	}
	var r assessResult
	if jerr := json.Unmarshal([]byte(strings.TrimSpace(raw)), &r); jerr != nil {
		return false, false
	}
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}
		if v > 100 {
			return 100
		}
		return v
	}
	tf, ta, te := clamp(r.Fluency), clamp(r.Accuracy), clamp(r.Expression)

	oldLevel := levelFromDims(sk.Fluency, sk.Accuracy, sk.Expression)
	if sk.Assessed == 0 {
		// 首轮 = 定级时刻：直接落到测出的水平，给一个清晰的「起点」。
		sk.Fluency, sk.Accuracy, sk.Expression = tf, ta, te
	} else {
		sk.Fluency = ratchet(sk.Fluency, tf)
		sk.Accuracy = ratchet(sk.Accuracy, ta)
		sk.Expression = ratchet(sk.Expression, te)
	}
	sk.Assessed++
	if note := strings.TrimSpace(r.Note); note != "" {
		sk.LastNote = clipText(note, 80)
	}
	newLevel := levelFromDims(sk.Fluency, sk.Accuracy, sk.Expression)
	h.db.Model(sk).Updates(map[string]interface{}{
		"fluency": sk.Fluency, "accuracy": sk.Accuracy, "expression": sk.Expression,
		"assessed": sk.Assessed, "last_note": sk.LastNote,
	})
	return true, newLevel > oldLevel
}

// skillView 序列化能力档案给前端（含派生段位名）。
func skillView(sk *models.AgentSkill) gin.H {
	level := levelFromDims(sk.Fluency, sk.Accuracy, sk.Expression)
	return gin.H{
		"level": level, "levelName": tierName(level),
		"fluency": sk.Fluency, "accuracy": sk.Accuracy, "expression": sk.Expression,
		"assessed": sk.Assessed, "note": sk.LastNote,
	}
}
