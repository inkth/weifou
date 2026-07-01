// Package toolagent — novel.go：写小说创作型 Agent 的「作品/章节」持久化。
// 一人一 Agent 一作品（MVP）。进度不用 X/N，用完成度：阶段由作品状态推导 + 总字数。
package toolagent

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

// loadWork 取（或建）用户在该 Agent 的作品。
func (h *Handler) loadWork(userID, agentID string) *models.Work {
	var w models.Work
	if err := h.db.First(&w, "user_id = ? AND agent_id = ?", userID, agentID).Error; err == gorm.ErrRecordNotFound {
		w = models.Work{ID: idgen.New(), UserID: userID, AgentID: agentID}
		if cerr := h.db.Create(&w).Error; cerr != nil {
			h.db.First(&w, "user_id = ? AND agent_id = ?", userID, agentID) // 并发：他人先建 → 重查
		}
	}
	return &w
}

// requireNovelAgent 校验 :id 是存在且启用的写小说 Agent。
func (h *Handler) requireNovelAgent(c *gin.Context) (*models.ToolAgent, error) {
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", c.Param("id"), true).Error; err != nil {
		return nil, httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	if !a.Novel {
		return nil, httpx.BadRequest("NOT_NOVEL_AGENT", "该 Agent 不支持作品")
	}
	return &a, nil
}

// workView 序列化作品 + 章节 + 推导阶段 + 总字数。
func workView(w *models.Work, chapters []models.Chapter) gin.H {
	words := 0
	items := make([]gin.H, 0, len(chapters))
	for i := range chapters {
		words += chapters[i].WordCount
		items = append(items, gin.H{
			"id": chapters[i].ID, "idx": chapters[i].Idx, "title": chapters[i].Title,
			"content": chapters[i].Content, "wordCount": chapters[i].WordCount,
		})
	}
	// 阶段推导：立意 / 大纲 / 初稿 / 定稿。
	done := []bool{w.Logline != "", w.Outline != "", len(chapters) > 0, w.Finalized}
	labels := []string{"立意", "大纲", "初稿", "定稿"}
	stages := make([]gin.H, 0, 4)
	current := "构思"
	for i, lb := range labels {
		stages = append(stages, gin.H{"label": lb, "done": done[i]})
		if done[i] {
			current = lb
		}
	}
	return gin.H{
		"id": w.ID, "title": w.Title, "logline": w.Logline, "genre": w.Genre,
		"outline": w.Outline, "finalized": w.Finalized,
		"stage": current, "stages": stages, "wordCount": words,
		"chapterCount": len(chapters), "chapters": items,
	}
}

func (h *Handler) getWork(c *gin.Context) error {
	auth := middleware.Current(c)
	if _, err := h.requireNovelAgent(c); err != nil {
		return err
	}
	w := h.loadWork(auth.UserID, c.Param("id"))
	var chapters []models.Chapter
	h.db.Where("work_id = ?", w.ID).Order("idx asc").Find(&chapters)
	httpx.OK(c, workView(w, chapters))
	return nil
}

type updateWorkReq struct {
	Title     *string `json:"title"`
	Logline   *string `json:"logline"`
	Genre     *string `json:"genre"`
	Outline   *string `json:"outline"`
	Finalized *bool   `json:"finalized"`
}

func (h *Handler) updateWork(c *gin.Context) error {
	auth := middleware.Current(c)
	if _, err := h.requireNovelAgent(c); err != nil {
		return err
	}
	var req updateWorkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("BAD_INPUT", "参数错误")
	}
	w := h.loadWork(auth.UserID, c.Param("id"))
	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = clipText(strings.TrimSpace(*req.Title), 120)
	}
	if req.Logline != nil {
		updates["logline"] = clipText(strings.TrimSpace(*req.Logline), 255)
	}
	if req.Genre != nil {
		updates["genre"] = clipText(strings.TrimSpace(*req.Genre), 64)
	}
	if req.Outline != nil {
		updates["outline"] = *req.Outline
	}
	if req.Finalized != nil {
		updates["finalized"] = *req.Finalized
	}
	if len(updates) > 0 {
		h.db.Model(w).Updates(updates)
	}
	var chapters []models.Chapter
	h.db.Where("work_id = ?", w.ID).Order("idx asc").Find(&chapters)
	httpx.OK(c, workView(w, chapters))
	return nil
}

type chapterReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (h *Handler) addChapter(c *gin.Context) error {
	auth := middleware.Current(c)
	if _, err := h.requireNovelAgent(c); err != nil {
		return err
	}
	var req chapterReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Content) == "" {
		return httpx.BadRequest("EMPTY_CHAPTER", "章节内容不能为空")
	}
	w := h.loadWork(auth.UserID, c.Param("id"))
	var maxIdx int
	h.db.Model(&models.Chapter{}).Where("work_id = ?", w.ID).
		Select("COALESCE(MAX(idx),0)").Scan(&maxIdx)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "未命名章节"
	}
	ch := models.Chapter{
		ID: idgen.New(), WorkID: w.ID, Idx: maxIdx + 1,
		Title: clipText(title, 120), Content: req.Content,
		WordCount: len([]rune(strings.TrimSpace(req.Content))),
	}
	h.db.Create(&ch)
	h.db.Model(w).Update("updated_at", gorm.Expr("NOW()"))
	httpx.OK(c, gin.H{"chapter": gin.H{"id": ch.ID, "idx": ch.Idx, "title": ch.Title, "wordCount": ch.WordCount}})
	return nil
}

// chapterOfMine 校验 :cid 章节属于当前用户在 :id(agentId) 下的作品。
func (h *Handler) chapterOfMine(userID, agentID, chapterID string) (*models.Chapter, error) {
	w := h.loadWork(userID, agentID)
	var ch models.Chapter
	if err := h.db.First(&ch, "id = ? AND work_id = ?", chapterID, w.ID).Error; err != nil {
		return nil, httpx.NotFound("CHAPTER_NOT_FOUND", "章节不存在")
	}
	return &ch, nil
}

func (h *Handler) updateChapter(c *gin.Context) error {
	auth := middleware.Current(c)
	if _, err := h.requireNovelAgent(c); err != nil {
		return err
	}
	var req chapterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("BAD_INPUT", "参数错误")
	}
	ch, err := h.chapterOfMine(auth.UserID, c.Param("id"), c.Param("cid"))
	if err != nil {
		return err
	}
	updates := map[string]interface{}{}
	if t := strings.TrimSpace(req.Title); t != "" {
		updates["title"] = clipText(t, 120)
	}
	if req.Content != "" {
		updates["content"] = req.Content
		updates["word_count"] = len([]rune(strings.TrimSpace(req.Content)))
	}
	if len(updates) > 0 {
		h.db.Model(ch).Updates(updates)
	}
	httpx.OK(c, gin.H{"ok": true})
	return nil
}

func (h *Handler) deleteChapter(c *gin.Context) error {
	auth := middleware.Current(c)
	if _, err := h.requireNovelAgent(c); err != nil {
		return err
	}
	ch, err := h.chapterOfMine(auth.UserID, c.Param("id"), c.Param("cid"))
	if err != nil {
		return err
	}
	h.db.Delete(ch)
	httpx.OK(c, gin.H{"ok": true})
	return nil
}
