package app

import (
	"testing"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/config"
)

// 冒烟测试：用真实 App.New + RegisterRoutes 注册全部路由，
// 捕获 Gin 路由冲突（会在注册时 panic）。不触达 DB/Redis。
func TestRegisterRoutesNoConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test"}
	a := New(cfg, nil, nil)
	r := gin.New()
	a.RegisterRoutes(r)
}
