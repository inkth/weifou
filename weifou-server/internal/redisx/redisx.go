package redisx

import (
	"log"

	"github.com/redis/go-redis/v9"
)

// New 解析 redis:// URL 并返回客户端。失败时返回可用但连接懒加载的客户端。
func New(url string) *redis.Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Printf("[redis] parse url failed, fallback to localhost: %v", err)
		opt = &redis.Options{Addr: "localhost:6379"}
	}
	return redis.NewClient(opt)
}
