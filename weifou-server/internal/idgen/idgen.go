package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

// New 生成一个排序友好的唯一 ID（时间戳 + 随机），用作主键。
func New() string {
	ts := strconv.FormatInt(time.Now().UnixNano(), 36)
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return ts + hex.EncodeToString(b)
}

// WithPrefix 生成带业务前缀的单号（如订单号、退款号）。
func WithPrefix(prefix string) string {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + ts + hex.EncodeToString(b)
}
