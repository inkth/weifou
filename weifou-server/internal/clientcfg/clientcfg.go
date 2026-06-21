// Package clientcfg 下发各付费入口在当前端的可见性，是 iOS 虚拟支付红线的服务端真源。
// 前端（小程序 / App EntryGate.applyRemote）据此隐藏违规入口；各业务另有 X-Platform 兜底拒单。
package clientcfg

import (
	"strings"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/httpx"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/config/entries", httpx.Handle(h.entries))
}

func (h *Handler) entries(c *gin.Context) error {
	isIOS := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Platform"))) == "ios"
	// 键名与 App 端 PayEntry 对齐（tip/consult/virtualGoods），并新增 agent。
	httpx.OK(c, gin.H{
		"tip":           !isIOS, // 打赏：iOS 合规灰区，隐藏
		"consult":       true,   // 真人一对一服务：iOS 允许第三方支付
		"asyncQuestion": true,   // 真人作答：允许
		"virtualGoods":  !isIOS, // 虚拟商品：iOS 必须 IAP，隐藏
		"agent":         !isIOS, // AI 工具 Agent = 虚拟商品，iOS 隐藏
	})
	return nil
}
