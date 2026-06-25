// Package upload 提供登录后的小文件上传，当前用于付费提问的「语音回答」。
// 存储经 storage.Store 抽象（现为本地盘）；返回可公开访问的绝对 URL。
package upload

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/httpx"
	"weifou-server/internal/idgen"
	"weifou-server/internal/middleware"
	"weifou-server/internal/storage"
)

const maxVoiceBytes = 5 << 20 // 5MB，约 5 分钟 mp3，足够异步语音答

var voiceExt = map[string]bool{".mp3": true, ".aac": true, ".m4a": true, ".wav": true}

type Handler struct {
	store      storage.Store
	publicBase string // 形如 https://api.weifou.com/api/uploads（末尾无斜杠）
	jwtSecret  string
}

func NewHandler(store storage.Store, publicBase, jwtSecret string) *Handler {
	return &Handler{store: store, publicBase: strings.TrimRight(publicBase, "/"), jwtSecret: jwtSecret}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := middleware.JWTAuth(h.jwtSecret)
	rg.POST("/upload/voice", auth, httpx.Handle(h.voice))
}

func (h *Handler) voice(c *gin.Context) error {
	_ = middleware.Current(c) // 仅需登录态
	fh, err := c.FormFile("file")
	if err != nil {
		return httpx.BadRequest("NO_FILE", "未收到文件")
	}
	if fh.Size <= 0 || fh.Size > maxVoiceBytes {
		return httpx.BadRequest("FILE_TOO_LARGE", "语音文件为空或超过 5MB")
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !voiceExt[ext] {
		ext = ".mp3"
	}
	src, err := fh.Open()
	if err != nil {
		return httpx.Internal("OPEN_FAILED", "读取文件失败")
	}
	defer src.Close()
	rel, err := h.store.Save("voices", idgen.New()+ext, src)
	if err != nil {
		return httpx.Internal("SAVE_FAILED", "保存失败")
	}
	httpx.OK(c, gin.H{"url": h.publicBase + "/" + rel})
	return nil
}
