package plaza

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"weifou-server/internal/httpx"
	"weifou-server/internal/models"
)

// Handler 人物广场（发现）。仅展示 opt-in 公开且已发布的主页。
type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	// 无需登录，访客可浏览发现页。
	rg.GET("/plaza", httpx.Handle(h.list))
}

type cardRow struct {
	ProfileID string         `gorm:"column:profile_id"`
	RealName  string         `gorm:"column:real_name"`
	Title     string         `gorm:"column:title"`
	Nickname  *string        `gorm:"column:nickname"`
	AvatarURL *string        `gorm:"column:avatar_url"`
	OneLiner  string         `gorm:"column:one_liner"`
	Tags      datatypes.JSON `gorm:"column:tags"`
}

// list 支持 ?sort=hot|new & ?q=关键词 & ?page= & ?pageSize=
func (h *Handler) list(c *gin.Context) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	q := h.db.Table("profiles p").
		Select("p.id as profile_id, p.real_name, p.title, u.nickname, u.avatar_url, pa.one_liner, pa.tags").
		Joins("JOIN persona_ai pa ON pa.profile_id = p.id").
		Joins("LEFT JOIN users u ON u.id = p.user_id").
		Where("p.status = ? AND p.discoverable = ?", models.ProfilePublished, true)

	if kw := strings.TrimSpace(c.Query("q")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"p.real_name ILIKE ? OR p.title ILIKE ? OR pa.one_liner ILIKE ?",
			like, like, like,
		)
	}

	// 热度：按访问数倒序；最新：按更新时间倒序。
	if c.Query("sort") == "hot" {
		q = q.Order("(SELECT count(*) FROM visits v WHERE v.profile_id = p.id) DESC, p.updated_at DESC")
	} else {
		q = q.Order("p.updated_at DESC")
	}

	var rows []cardRow
	if err := q.Limit(pageSize).Offset(offset).Scan(&rows).Error; err != nil {
		return httpx.Internal("DB_ERROR", "加载失败")
	}

	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var tags []string
		_ = json.Unmarshal(r.Tags, &tags)
		items = append(items, gin.H{
			"profileId": r.ProfileID,
			"realName":  r.RealName,
			"title":     r.Title,
			"nickname":  r.Nickname,
			"avatarUrl": r.AvatarURL,
			"oneLiner":  r.OneLiner,
			"tags":      tags,
		})
	}
	httpx.OK(c, gin.H{"items": items, "page": page, "pageSize": pageSize})
	return nil
}
