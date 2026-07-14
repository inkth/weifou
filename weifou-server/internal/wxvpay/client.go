// Package wxvpay 实现微信小程序「虚拟支付」(wx.requestVirtualPayment) 的服务端配合逻辑。
// 纯标准库自实现，风格对齐 internal/wxpay（不引第三方支付库，便于审计）。
//
// 为什么独立成一套（而非扩展 wxpay）：虚拟支付与微信支付 V3 是两个体系。
// 虚拟支付使用米大师 offerId、AppKey 与用户 session_key 双签名，承载会员虚拟权益。
//
// 签名（对齐官方虚拟支付签名规则）：
//
//	前端拉起 ：paySig    = HMAC_SHA256(AppKey,     "requestVirtualPayment&" + signData)
//	          signature = HMAC_SHA256(sessionKey, signData)
//	服务端接口：pay_sig   = HMAC_SHA256(AppKey,     uri + "&" + body)
//	          signature = HMAC_SHA256(sessionKey, body)
//
// ⚠️ 落地前需用官方文档 / 开通后控制台校准：
//
//	① signData 的键名与序列化要求；② xpay 各接口确切 path；
//	③ productId 来自米大师后台「上传 / 发布商品」；
//	④ 发货回调走小程序「消息推送」(xpay_goods_deliver_notify，EncodingAESKey 加密)。
package wxvpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const xpayBase = "https://api.weixin.qq.com"

// ErrNotConfigured 未配置虚拟支付（offerId/appKey 缺失）时返回，调用方据此降级。
var ErrNotConfigured = errors.New("WXVPAY_NOT_CONFIGURED")

// tokenSource 复用 wechat.LoginClient（小程序 access_token，与内容安全 / 订阅消息共用）。
type tokenSource interface {
	AccessToken() (string, error)
}

// Client 虚拟支付客户端。无状态、可并发复用；session_key 为 per-user，作方法入参传入。
type Client struct {
	appID   string
	offerID string // 米大师应用 ID（开通虚拟支付后获得）
	appKey  string // 现网 AppKey（米大师，区别于 V3 的 APIv3Key）
	sandbox bool
	login   tokenSource
	hc      *http.Client
}

func New(appID, offerID, appKey string, sandbox bool, login tokenSource) *Client {
	return &Client{
		appID:   appID,
		offerID: offerID,
		appKey:  appKey,
		sandbox: sandbox,
		login:   login,
		hc:      &http.Client{Timeout: 10 * time.Second},
	}
}

// Ready 是否具备调起虚拟支付的最小配置。未就绪时上层应降级（不暴露购买入口）。
func (c *Client) Ready() bool {
	return c.offerID != "" && c.appKey != "" && c.login != nil
}

func (c *Client) env() int {
	if c.sandbox {
		return 1
	}
	return 0
}

// ---------- 前端拉起参数（wx.requestVirtualPayment） ----------

// 业务模式。会员 = 商品直购。
const (
	ModeGoods = "short_series_goods" // 商品直购
	ModeCoin  = "short_series_coin"  // 代币充值（暂未用）
)

// GoodsOrder 一次商品直购（如会员套餐）。
type GoodsOrder struct {
	OutTradeNo string // 业务订单号（8-32 字符：数字 / 大小写字母 / _-|*@）
	ProductID  string // 米大师商品 ID（后台上传 / 发布后获得；会员每档一个）
	GoodsPrice int    // 单价（分）
	Quantity   int    // 数量（会员 = 1）
	Platform   string // "android" | "ios"
	Attach     string // 透传（放 order.ID，回调里取回）
}

// PaymentParams 返回小程序，前端原样传给 wx.requestVirtualPayment。
type PaymentParams struct {
	SignData  string `json:"signData"`
	Mode      string `json:"mode"`
	PaySig    string `json:"paySig"`
	Signature string `json:"signature"`
	Env       int    `json:"env"`
}

// BuildGoodsPayment 生成「商品直购」前端拉起参数。
// sessionKey：该用户当前有效的微信 session_key（来自 jscode2session，会过期；
// 建议下单时用前端新鲜的 wx.login code 实时换取，见 membership 下单端点）。
func (c *Client) BuildGoodsPayment(o GoodsOrder, sessionKey string) (*PaymentParams, error) {
	if !c.Ready() {
		return nil, ErrNotConfigured
	}
	if sessionKey == "" {
		return nil, errors.New("WXVPAY_NO_SESSION_KEY")
	}
	qty := o.Quantity
	if qty <= 0 {
		qty = 1
	}
	// signData 字段对齐官方 SignData 契约（键名 / 序列化以官方文档为准）。
	sd, _ := json.Marshal(map[string]interface{}{
		"offerId":      c.offerID,
		"buyQuantity":  qty,
		"env":          c.env(),
		"currencyType": "CNY",
		"platform":     o.Platform,
		"productId":    o.ProductID,
		"goodsPrice":   o.GoodsPrice,
		"outTradeNo":   o.OutTradeNo,
		"attach":       o.Attach,
	})
	signData := string(sd)
	return &PaymentParams{
		SignData:  signData,
		Mode:      ModeGoods,
		PaySig:    hmacHex(c.appKey, "requestVirtualPayment&"+signData),
		Signature: hmacHex(sessionKey, signData),
		Env:       c.env(),
	}, nil
}

// ---------- 服务端接口（access_token + 双签名） ----------

// xpayPost 调用米大师服务端接口。uri 形如 "/xpay/notify_provide_goods"。
// sessionKey 可空（部分接口仅需 pay_sig）。
func (c *Client) xpayPost(uri string, payload map[string]interface{}, sessionKey string) ([]byte, error) {
	if !c.Ready() {
		return nil, ErrNotConfigured
	}
	token, err := c.login.AccessToken()
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(payload)
	q := url.Values{}
	q.Set("access_token", token)
	q.Set("pay_sig", hmacHex(c.appKey, uri+"&"+string(body)))
	if sessionKey != "" {
		q.Set("signature", hmacHex(sessionKey, string(body)))
	}
	req, _ := http.NewRequest("POST", xpayBase+uri+"?"+q.Encode(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("wxvpay %s -> %d: %s", uri, resp.StatusCode, string(respBody))
	}
	var head struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(respBody, &head)
	if head.ErrCode != 0 {
		return respBody, fmt.Errorf("wxvpay %s errcode=%d: %s", uri, head.ErrCode, head.ErrMsg)
	}
	return respBody, nil
}

// NotifyProvideGoods 确认发货：收到发货回调、完成开通后告知米大师已发货（避免自动退款）。
// 商品直购 path / 字段以官方文档为准。发货失败不应回滚已开通的会员，交由对账补偿。
func (c *Client) NotifyProvideGoods(openid, outTradeNo, sessionKey string) error {
	_, err := c.xpayPost("/xpay/notify_provide_goods", map[string]interface{}{
		"openid":       openid,
		"env":          c.env(),
		"out_trade_no": outTradeNo,
	}, sessionKey)
	return err
}

// QueryOrder 查询订单（对账 / 回调丢失时补发货）。返回原始 JSON，由调用方解析。
func (c *Client) QueryOrder(openid, outTradeNo, sessionKey string) ([]byte, error) {
	return c.xpayPost("/xpay/query_order", map[string]interface{}{
		"openid":       openid,
		"env":          c.env(),
		"out_trade_no": outTradeNo,
	}, sessionKey)
}

// ---------- 发货回调（小程序消息推送 xpay_goods_deliver_notify） ----------

// DeliverNotify 发货通知业务字段（调用方先完成消息解密，再 json 反序列化到此）。
// 字段名以官方推送格式为准，落地时按真实样例校准。
type DeliverNotify struct {
	ToUserName   string `json:"ToUserName"`
	FromUserName string `json:"FromUserName"`
	MsgType      string `json:"MsgType"` // "event"
	Event        string `json:"Event"`   // "xpay_goods_deliver_notify"
	OpenID       string `json:"OpenId"`
	OutTradeNo   string `json:"OutTradeNo"`
	Env          int    `json:"Env"`
	GoodsInfo    struct {
		ProductID   string `json:"ProductId"`
		Quantity    int    `json:"Quantity"`
		OrigPrice   int    `json:"OrigPrice"`
		ActualPrice int    `json:"ActualPrice"` // 实付（分）
		Attach      string `json:"Attach"`
	} `json:"GoodsInfo"`
	WeChatPayInfo struct {
		MchOrderNo    string `json:"MchOrderNo"`
		TransactionID string `json:"TransactionId"`
	} `json:"WeChatPayInfo"`
}

// IsGoodsDeliver 是否为商品直购发货事件。
func (n *DeliverNotify) IsGoodsDeliver() bool {
	return n.MsgType == "event" && n.Event == "xpay_goods_deliver_notify"
}

func hmacHex(key, data string) string {
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(data))
	return hex.EncodeToString(m.Sum(nil))
}
