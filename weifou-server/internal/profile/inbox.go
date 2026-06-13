package profile

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

// registerInbox 注册主人侧"收件箱"相关路由：知识缺口、知识库、访客线索。
// 这些是对话飞轮的主人端：把访客问倒的缺口补成知识，把访客线索沉淀成 CRM。
func (h *Handler) registerInbox(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.GET("/profile/gaps", auth, httpx.Handle(h.listGaps))
	rg.POST("/profile/gaps/:id/answer", auth, httpx.Handle(h.answerGap))
	rg.POST("/profile/gaps/:id/dismiss", auth, httpx.Handle(h.dismissGap))
	rg.GET("/profile/knowledge", auth, httpx.Handle(h.listKnowledge))
	rg.POST("/profile/knowledge", auth, httpx.Handle(h.addKnowledge))
	rg.POST("/profile/knowledge/ingest", auth, httpx.Handle(h.ingestKnowledge))
	rg.DELETE("/profile/knowledge/:id", auth, httpx.Handle(h.deleteKnowledge))
	rg.GET("/profile/leads", auth, httpx.Handle(h.listLeads))
}

// myProfile 取当前登录用户的主页，找不到返回错误。
func (h *Handler) myProfile(c *gin.Context) (*models.Profile, error) {
	auth := middleware.Current(c)
	var profile models.Profile
	if err := h.db.Where("user_id = ?", auth.UserID).First(&profile).Error; err != nil {
		return nil, httpx.NotFound("PROFILE_NOT_FOUND", "请先创建主页")
	}
	return &profile, nil
}

func (h *Handler) listGaps(c *gin.Context) error {
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	var gaps []models.KnowledgeGap
	h.db.Where("profile_id = ? AND status = ?", profile.ID, models.GapOpen).
		Order("asked_count desc, last_asked_at desc").Limit(100).Find(&gaps)
	httpx.OK(c, gaps)
	return nil
}

type answerGapReq struct {
	Topic   string `json:"topic"`
	Content string `json:"content" binding:"required"`
}

// answerGap 主人回答一条缺口：写入知识库 + 标记缺口已答。Agent 下次即变聪明。
func (h *Handler) answerGap(c *gin.Context) error {
	auth := middleware.Current(c)
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	var req answerGapReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "请填写回答内容")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return httpx.BadRequest("EMPTY_INPUT", "请填写回答内容")
	}
	if len([]rune(content)) > 1000 {
		return httpx.BadRequest("INPUT_TOO_LONG", "回答太长（限 1000 字）")
	}

	var gap models.KnowledgeGap
	if err := h.db.First(&gap, "id = ?", c.Param("id")).Error; err != nil {
		return httpx.NotFound("GAP_NOT_FOUND", "问题不存在")
	}
	if gap.ProfileID != profile.ID {
		return httpx.Forbidden("FORBIDDEN", "无权操作")
	}

	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		topic = gap.Question
	}
	if len([]rune(topic)) > 128 {
		topic = string([]rune(topic)[:128])
	}
	if !h.persona.CheckText(topic+"\n"+content, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "内容包含敏感信息，请修改后重试")
	}

	h.db.Create(&models.KnowledgeItem{
		ID: idgen.New(), ProfileID: profile.ID, Topic: topic,
		Content: content, Source: models.KnowledgeSourceGap, Enabled: true,
	})
	h.db.Model(&gap).Update("status", models.GapAnswered)
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

func (h *Handler) dismissGap(c *gin.Context) error {
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	var gap models.KnowledgeGap
	if err := h.db.First(&gap, "id = ?", c.Param("id")).Error; err != nil {
		return httpx.NotFound("GAP_NOT_FOUND", "问题不存在")
	}
	if gap.ProfileID != profile.ID {
		return httpx.Forbidden("FORBIDDEN", "无权操作")
	}
	h.db.Model(&gap).Update("status", models.GapDismissed)
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

func (h *Handler) listKnowledge(c *gin.Context) error {
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	var items []models.KnowledgeItem
	h.db.Where("profile_id = ?", profile.ID).
		Order("updated_at desc").Limit(200).Find(&items)
	httpx.OK(c, items)
	return nil
}

type addKnowledgeReq struct {
	Topic   string `json:"topic"`
	Content string `json:"content" binding:"required"`
}

func (h *Handler) addKnowledge(c *gin.Context) error {
	auth := middleware.Current(c)
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	var req addKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("INVALID_PARAMS", "请填写内容")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return httpx.BadRequest("EMPTY_INPUT", "请填写内容")
	}
	if len([]rune(content)) > 1000 {
		return httpx.BadRequest("INPUT_TOO_LONG", "内容太长（限 1000 字）")
	}
	topic := strings.TrimSpace(req.Topic)
	if len([]rune(topic)) > 128 {
		topic = string([]rune(topic)[:128])
	}
	if !h.persona.CheckText(topic+"\n"+content, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "内容包含敏感信息，请修改后重试")
	}
	item := models.KnowledgeItem{
		ID: idgen.New(), ProfileID: profile.ID, Topic: topic,
		Content: content, Source: models.KnowledgeSourceManual, Enabled: true,
	}
	h.db.Create(&item)
	httpx.OK(c, item)
	return nil
}

type ingestReq struct {
	Text string `json:"text" binding:"required"`
}

// ingestKnowledge 粘贴一段原始文本，由 AI 自动拆成多条知识入库（轻量灌入）。
func (h *Handler) ingestKnowledge(c *gin.Context) error {
	auth := middleware.Current(c)
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	var req ingestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("EMPTY_INPUT", "请粘贴要整理的内容")
	}
	count, err := h.persona.ExtractKnowledge(profile.ID, auth.Openid, req.Text)
	if err != nil {
		return err
	}
	httpx.OK(c, gin.H{"ok": true, "count": count})
	return nil
}

func (h *Handler) deleteKnowledge(c *gin.Context) error {
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	res := h.db.Where("id = ? AND profile_id = ?", c.Param("id"), profile.ID).
		Delete(&models.KnowledgeItem{})
	if res.RowsAffected == 0 {
		return httpx.NotFound("KNOWLEDGE_NOT_FOUND", "知识不存在")
	}
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

type leadItem struct {
	ID            string    `json:"id"`
	Note          string    `json:"note"`
	Contact       *string   `json:"contact,omitempty"`
	Status        string    `json:"status"`
	SessionID     *string   `json:"sessionId,omitempty"`
	VisitorName   *string   `json:"visitorName,omitempty"`
	VisitorAvatar *string   `json:"visitorAvatar,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// listLeads 列出本主页收到的访客线索，解析访客昵称/头像（CRM 雏形）。
func (h *Handler) listLeads(c *gin.Context) error {
	profile, err := h.myProfile(c)
	if err != nil {
		return err
	}
	var leads []models.Lead
	h.db.Where("profile_id = ?", profile.ID).
		Order("created_at desc").Limit(100).Find(&leads)

	items := make([]leadItem, 0, len(leads))
	for _, l := range leads {
		it := leadItem{
			ID: l.ID, Note: l.Note, Contact: l.Contact, Status: l.Status,
			SessionID: l.SessionID, CreatedAt: l.CreatedAt,
		}
		var u models.User
		if err := h.db.Where("openid = ? OR wx_mp_openid = ? OR wx_app_openid = ?",
			l.VisitorOpenid, l.VisitorOpenid, l.VisitorOpenid).First(&u).Error; err == nil {
			it.VisitorName = u.Nickname
			it.VisitorAvatar = u.AvatarURL
		}
		items = append(items, it)
	}
	httpx.OK(c, items)
	return nil
}
