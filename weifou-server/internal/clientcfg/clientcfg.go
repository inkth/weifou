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
	// 订阅消息模板 ID（服务端 .env 真源，随 /config/entries 下发给前端）：
	// 前端不再硬编码模板 ID——公众平台申请到模板后填服务器 .env 重启即全链路生效，无需小程序发版。
	subscribeTmpls SubscribeTmpls
}

// SubscribeTmpls 各订阅消息模板 ID；空串=未配置（前端静默降级，不弹授权）。
type SubscribeTmpls struct {
	Answered    string `json:"answered"`    // 访客：你的提问已回答
	NewQuestion string `json:"newQuestion"` // 主人：有人问了你的问答箱
	Lead        string `json:"lead"`        // 主人：有新的访客线索
	LearnRemind string `json:"learnRemind"` // 学员：学习提醒（明天叫你继续）
}

func NewHandler(vpayReady bool, tmpls SubscribeTmpls) *Handler {
	return &Handler{vpayReady: vpayReady, subscribeTmpls: tmpls}
}

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
		"virtualGoods":   virtualVisible,
		"agent":          virtualVisible,
		"membership":     virtualVisible,
		"subscribeTmpls": h.subscribeTmpls,
	})
	return nil
}
