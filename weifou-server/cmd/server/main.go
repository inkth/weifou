package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/app"
	"weifou-server/internal/config"
	"weifou-server/internal/database"
	"weifou-server/internal/redisx"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL, cfg.Env)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	rdb := redisx.New(cfg.RedisURL)

	if cfg.Env != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	application := app.New(cfg, db, rdb)
	application.StartCron()

	r := gin.Default()
	application.RegisterRoutes(r)

	addr := ":" + cfg.Port
	log.Printf("微否 (Go) 监听 http://localhost%s/api", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
