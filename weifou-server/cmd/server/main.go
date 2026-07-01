package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/app"
	"weifou-server/internal/config"
	"weifou-server/internal/database"
	"weifou-server/internal/membership"
	"weifou-server/internal/redisx"
	"weifou-server/internal/toolagent"
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
	toolagent.Seed(db)         // 首启写入平台自编的工具 Agent（按 slug 幂等）
	toolagent.SeedConcepts(db) // 须在 Seed 之后：写入概念型 Agent 的 100 概念课程表（按 agent+slug 幂等）
	membership.Seed(db)        // 首启写入会员套餐
	application.StartCron()

	r := gin.Default()
	application.RegisterRoutes(r)

	addr := ":" + cfg.Port
	log.Printf("微否 (Go) 监听 http://localhost%s/api", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
