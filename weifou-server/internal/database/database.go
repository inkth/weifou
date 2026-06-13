package database

import (
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"weifou-server/internal/models"
)

// Connect 连接 Postgres 并执行 AutoMigrate。
// 接受 postgresql:// 或 postgres:// DSN。
func Connect(dsn string, env string) (*gorm.DB, error) {
	// gorm postgres 驱动接受标准连接串
	normalized := strings.Replace(dsn, "postgresql://", "postgres://", 1)

	logLevel := gormlogger.Warn
	if env == "development" {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(normalized), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return nil, err
	}
	// 路线 A 回填：历史用户均来自小程序，wx_mp_openid 对齐 openid，
	// 使其后续可按端 openid 命中并与 App 端通过 unionid 合并。
	if err := db.Exec(
		"UPDATE users SET wx_mp_openid = openid WHERE wx_mp_openid IS NULL AND openid <> ''",
	).Error; err != nil {
		return nil, err
	}
	log.Println("[db] connected & migrated")
	return db, nil
}
