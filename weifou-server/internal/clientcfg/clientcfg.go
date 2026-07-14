// Package clientcfg 下发会员等虚拟商品入口在当前端的可见性，是 iOS 虚拟支付红线的服务端真源。
// 前端（小程序 / App EntryGate.applyRemote）据此隐藏违规入口；各业务另有 X-Platform 兜底拒单。
package clientcfg

import (
	"strings"

	"github.com/gin-gonic/gin"

	"weifou-server/internal/httpx"
)

type Handler struct {
	vpayReady bool // 虚拟支付是否已配置就绪（决定小程序端虚拟商品入口是否下发，未就绪则隐藏避免点了报错）
}

func NewHandler(vpayReady bool) *Handler { return &Handler{vpayReady: vpayReady} }

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/config/entries", httpx.Handle(h.entries))
}

func (h *Handler) entries(c *gin.Context) error {
	isIOS := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Platform"))) == "ios"
	isApp := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Client-Type"))) == "app"
	// 虚拟商品入口可见性：
	//   小程序端 = 虚拟支付是否就绪（未配 offerId/appKey 则隐藏，避免点了才报「未开通」）；
	//   App 端  = 维持原红线（iOS 隐藏，尚未接苹果 IAP）。
	virtualVisible := !isIOS
	if !isApp {
		virtualVisible = h.vpayReady
	}
	httpx.OK(c, gin.H{
		"virtualGoods": virtualVisible,
		"agent":        virtualVisible,
		"membership":   virtualVisible,
	})
	return nil
}
