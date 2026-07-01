// Package toolagent — music.go：做音乐创作型 Agent 的「生成带人声歌曲」。
// 生成慢（几十秒），走异步：建 Song(pending) → 起 goroutine 调 provider → 下载音频重存本站 → 轮询取回。
package toolagent

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/models"
)

type genMusicReq struct {
	Lyrics string `json:"lyrics" binding:"required"`
	Style  string `json:"style"`
	Title  string `json:"title"`
}

// genMusic 发起一次生成：扣额度 → 建 Song(pending) → 异步生成 → 立即返回 songId。
func (h *Handler) genMusic(c *gin.Context) error {
	auth := middleware.Current(c)
	var a models.ToolAgent
	if err := h.db.First(&a, "id = ? AND enabled = ?", c.Param("id"), true).Error; err != nil {
		return httpx.NotFound("AGENT_NOT_FOUND", "该 Agent 不存在或已下架")
	}
	if !a.Music {
		return httpx.BadRequest("NOT_MUSIC_AGENT", "该 Agent 不支持生成音乐")
	}
	if h.music == nil || !h.music.Ready() {
		return httpx.Internal("MUSIC_NOT_READY", "音乐生成暂未开通，请稍后再试")
	}
	var req genMusicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return httpx.BadRequest("EMPTY_LYRICS", "请先写好歌词")
	}
	lyrics := strings.TrimSpace(req.Lyrics)
	if lyrics == "" {
		return httpx.BadRequest("EMPTY_LYRICS", "请先写好歌词")
	}
	if len([]rune(lyrics)) > 2000 {
		return httpx.BadRequest("LYRICS_TOO_LONG", "歌词太长（限 2000 字）")
	}
	if !h.security.CheckText(lyrics, auth.Openid) {
		return httpx.BadRequest("CONTENT_UNSAFE", "歌词包含不当信息")
	}

	// 准入：会员畅用；非会员扣一次免费体验（生成贵，1 次=1 条）。
	member, _, aerr := h.checkAccess(auth.UserID, &a)
	if aerr != nil {
		return aerr
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "未命名歌曲"
	}
	song := models.Song{
		ID: idgen.New(), UserID: auth.UserID, AgentID: a.ID,
		Title: clipText(title, 120), Lyrics: lyrics, Style: strings.TrimSpace(req.Style),
		Status: models.SongPending,
	}
	h.db.Create(&song)

	go h.runMusicGen(song.ID, song.Lyrics, song.Style, auth.UserID, a.ID, member)

	httpx.OK(c, gin.H{"songId": song.ID, "status": song.Status})
	return nil
}

// runMusicGen 后台生成：provider 出远端音频 → 下载重存本站 → 落 done/failed。失败且非会员则退还额度。
func (h *Handler) runMusicGen(songID, lyrics, style, userID, agentID string, member bool) {
	h.db.Model(&models.Song{}).Where("id = ?", songID).Update("status", models.SongGenerating)

	fail := func(reason string) {
		h.db.Model(&models.Song{}).Where("id = ?", songID).
			Updates(map[string]interface{}{"status": models.SongFailed, "err": clipText(reason, 255)})
		if !member {
			h.db.Model(&models.AgentEntitlement{}).
				Where("user_id = ? AND agent_id = ?", userID, agentID).
				UpdateColumn("remaining", gorm.Expr("remaining + 1"))
		}
	}

	remoteURL, err := h.music.Generate(lyrics, style)
	if err != nil {
		fail("生成失败：" + err.Error())
		return
	}
	audioURL, err := h.rehostAudio(remoteURL)
	if err != nil {
		fail("音频保存失败：" + err.Error())
		return
	}
	h.db.Model(&models.Song{}).Where("id = ?", songID).
		Updates(map[string]interface{}{"status": models.SongDone, "audio_url": audioURL, "err": ""})
}

// rehostAudio 下载远端音频并存到本站（小程序域名白名单只放本站，必须重存），返回可播 URL。
func (h *Handler) rehostAudio(remoteURL string) (string, error) {
	hc := &http.Client{Timeout: 120 * time.Second}
	resp, err := hc.Get(remoteURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", httpx.Internal("AUDIO_DOWNLOAD", "下载音频失败")
	}
	ext := strings.ToLower(path.Ext(remoteURL))
	if ext != ".mp3" && ext != ".wav" && ext != ".m4a" && ext != ".ogg" {
		ext = ".mp3"
	}
	rel, err := h.store.Save("music", idgen.New()+ext, resp.Body)
	if err != nil {
		return "", err
	}
	return h.publicBase + "/" + rel, nil
}

// musicStatus 前端轮询：:id = songId。
func (h *Handler) musicStatus(c *gin.Context) error {
	auth := middleware.Current(c)
	var song models.Song
	if err := h.db.First(&song, "id = ? AND user_id = ?", c.Param("id"), auth.UserID).Error; err != nil {
		return httpx.NotFound("SONG_NOT_FOUND", "歌曲不存在")
	}
	httpx.OK(c, songView(&song))
	return nil
}

// myMusic 我的曲库：:id = agentId，最近生成在前，只列已成功的可回放歌曲。
func (h *Handler) myMusic(c *gin.Context) error {
	auth := middleware.Current(c)
	var songs []models.Song
	h.db.Where("user_id = ? AND agent_id = ?", auth.UserID, c.Param("id")).
		Order("created_at desc").Limit(50).Find(&songs)
	out := make([]gin.H, 0, len(songs))
	for i := range songs {
		out = append(out, songView(&songs[i]))
	}
	httpx.OK(c, out)
	return nil
}

func songView(s *models.Song) gin.H {
	return gin.H{
		"songId": s.ID, "title": s.Title, "style": s.Style,
		"status": s.Status, "audioUrl": s.AudioURL, "err": s.Err,
		"createdAt": s.CreatedAt,
	}
}
