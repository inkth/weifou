package wxpay

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const base = "https://api.mch.weixin.qq.com"

// Client 微信支付 V3：纯标准库实现签名/验签/AES-GCM 解密。
type Client struct {
	appID       string
	mchID       string
	serialNo    string
	apiV3Key    string
	notifyURL   string
	privateKey  *rsa.PrivateKey
	platformKey *rsa.PublicKey // 平台证书公钥（验签用）
	hc          *http.Client
}

type Config struct {
	AppID            string
	MchID            string
	SerialNo         string
	APIV3Key         string
	NotifyURL        string
	PrivateKeyPath   string
	PlatformCertPath string
}

func New(cfg Config) *Client {
	c := &Client{
		appID:     cfg.AppID,
		mchID:     cfg.MchID,
		serialNo:  cfg.SerialNo,
		apiV3Key:  cfg.APIV3Key,
		notifyURL: cfg.NotifyURL,
		hc:        &http.Client{Timeout: 10 * time.Second},
	}
	if cfg.PrivateKeyPath != "" {
		if pk, err := loadPrivateKey(cfg.PrivateKeyPath); err == nil {
			c.privateKey = pk
		}
	}
	if cfg.PlatformCertPath != "" {
		if pub, err := loadPublicKeyFromCert(cfg.PlatformCertPath); err == nil {
			c.platformKey = pub
		}
	}
	return c
}

func (c *Client) Ready() bool {
	return c.mchID != "" && c.serialNo != "" && c.apiV3Key != "" && c.privateKey != nil
}

func (c *Client) AppID() string { return c.appID }

// notifyBase 用于推导退款/分账回调地址（与 payment/notify 同域名）
func (c *Client) notifyBase() string {
	return strings.TrimSuffix(c.notifyURL, "/payment/notify")
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid private key pem")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := key.(*rsa.PrivateKey); ok {
			return rk, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func loadPublicKeyFromCert(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid cert pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	if pub, ok := cert.PublicKey.(*rsa.PublicKey); ok {
		return pub, nil
	}
	return nil, fmt.Errorf("not rsa public key")
}

// ---------- 签名 ----------

func (c *Client) authHeader(method, urlPath, body string) (string, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randHex(16)
	message := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", method, urlPath, ts, nonce, body)
	h := sha256.Sum256([]byte(message))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	sig := base64.StdEncoding.EncodeToString(sigBytes)
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		c.mchID, nonce, sig, ts, c.serialNo,
	), nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	const hexd = "0123456789abcdef"
	out := make([]byte, n*2)
	for i, v := range b {
		out[i*2] = hexd[v>>4]
		out[i*2+1] = hexd[v&0x0f]
	}
	return string(out)
}

func (c *Client) doV3(method, urlPath string, payload interface{}) ([]byte, error) {
	if !c.Ready() {
		return nil, fmt.Errorf("WXPAY_NOT_CONFIGURED")
	}
	var bodyStr string
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyStr = string(b)
	}
	auth, err := c.authHeader(method, urlPath, bodyStr)
	if err != nil {
		return nil, err
	}
	var reqBody io.Reader
	if bodyStr != "" {
		reqBody = bytes.NewReader([]byte(bodyStr))
	}
	req, _ := http.NewRequest(method, base+urlPath, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", auth)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("wxpay %s %s -> %d: %s", method, urlPath, resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// ---------- 下单 ----------

type JsapiOrder struct {
	OutTradeNo    string
	Description   string
	Amount        int
	PayerOpenid   string
	Attach        string
	ProfitSharing bool
}

func (c *Client) CreateJsapiOrder(o JsapiOrder) (string, error) {
	payload := map[string]interface{}{
		"appid":        c.appID,
		"mchid":        c.mchID,
		"description":  o.Description,
		"out_trade_no": o.OutTradeNo,
		"notify_url":   c.notifyURL,
		"amount":       map[string]interface{}{"total": o.Amount, "currency": "CNY"},
		"payer":        map[string]string{"openid": o.PayerOpenid},
	}
	if o.Attach != "" {
		payload["attach"] = o.Attach
	}
	if o.ProfitSharing {
		payload["settle_info"] = map[string]bool{"profit_sharing": true}
	}
	body, err := c.doV3("POST", "/v3/pay/transactions/jsapi", payload)
	if err != nil {
		return "", err
	}
	var data struct {
		PrepayID string `json:"prepay_id"`
	}
	_ = json.Unmarshal(body, &data)
	return data.PrepayID, nil
}

// PayParams 小程序 wx.requestPayment 所需参数
type PayParams struct {
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

func (c *Client) BuildPayParams(prepayID string) (*PayParams, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := randHex(16)
	pkg := "prepay_id=" + prepayID
	message := fmt.Sprintf("%s\n%s\n%s\n%s\n", c.appID, ts, nonce, pkg)
	h := sha256.Sum256([]byte(message))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, h[:])
	if err != nil {
		return nil, err
	}
	return &PayParams{
		AppID:     c.appID,
		TimeStamp: ts,
		NonceStr:  nonce,
		Package:   pkg,
		SignType:  "RSA",
		PaySign:   base64.StdEncoding.EncodeToString(sigBytes),
	}, nil
}

// ---------- 关单 ----------

func (c *Client) CloseOrder(outTradeNo string) error {
	_, err := c.doV3("POST", fmt.Sprintf("/v3/pay/transactions/out-trade-no/%s/close", outTradeNo),
		map[string]string{"mchid": c.mchID})
	return err
}

// ---------- 退款 ----------

type RefundReq struct {
	OutTradeNo  string
	OutRefundNo string
	Refund      int
	Total       int
	Reason      string
}

type RefundResp struct {
	RefundID string `json:"refund_id"`
	Status   string `json:"status"`
}

func (c *Client) Refund(r RefundReq) (*RefundResp, error) {
	payload := map[string]interface{}{
		"out_trade_no":  r.OutTradeNo,
		"out_refund_no": r.OutRefundNo,
		"notify_url":    c.notifyBase() + "/payment/refund-notify",
		"amount":        map[string]interface{}{"refund": r.Refund, "total": r.Total, "currency": "CNY"},
	}
	if r.Reason != "" {
		payload["reason"] = r.Reason
	}
	body, err := c.doV3("POST", "/v3/refund/domestic/refunds", payload)
	if err != nil {
		return nil, err
	}
	var data RefundResp
	_ = json.Unmarshal(body, &data)
	return &data, nil
}

// ---------- 分账 ----------

func (c *Client) AddProfitShareReceiver(openid, name string) error {
	payload := map[string]interface{}{
		"appid":         c.appID,
		"type":          "PERSONAL_OPENID",
		"account":       openid,
		"relation_type": "PARTNER",
	}
	if name != "" {
		payload["name"] = name
	}
	_, err := c.doV3("POST", "/v3/profitsharing/receivers/add", payload)
	return err
}

type ProfitShareReq struct {
	TransactionID  string
	OutOrderNo     string
	ReceiverOpenid string
	Amount         int
	Description    string
}

type ProfitShareResp struct {
	OrderID string `json:"order_id"`
	State   string `json:"state"`
}

func (c *Client) CreateProfitShare(r ProfitShareReq) (*ProfitShareResp, error) {
	payload := map[string]interface{}{
		"appid":          c.appID,
		"transaction_id": r.TransactionID,
		"out_order_no":   r.OutOrderNo,
		"receivers": []map[string]interface{}{
			{
				"type":        "PERSONAL_OPENID",
				"account":     r.ReceiverOpenid,
				"amount":      r.Amount,
				"description": r.Description,
			},
		},
		"unfreeze_unsplit": true,
	}
	body, err := c.doV3("POST", "/v3/profitsharing/orders", payload)
	if err != nil {
		return nil, err
	}
	var data ProfitShareResp
	_ = json.Unmarshal(body, &data)
	return &data, nil
}

// ---------- 回调验签 + 解密 ----------

func (c *Client) VerifyNotifySignature(headers http.Header, rawBody string) bool {
	if c.platformKey == nil {
		// 未配置平台证书，开发环境放行
		return true
	}
	ts := headers.Get("Wechatpay-Timestamp")
	nonce := headers.Get("Wechatpay-Nonce")
	sig := headers.Get("Wechatpay-Signature")
	if ts == "" || nonce == "" || sig == "" {
		return false
	}
	message := fmt.Sprintf("%s\n%s\n%s\n", ts, nonce, rawBody)
	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	h := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(c.platformKey, crypto.SHA256, h[:], sigBytes) == nil
}

type NotifyResource struct {
	Ciphertext     string `json:"ciphertext"`
	Nonce          string `json:"nonce"`
	AssociatedData string `json:"associated_data"`
}

// DecryptNotify AES-256-GCM 解密回调 resource，返回明文 JSON 字节。
func (c *Client) DecryptNotify(res NotifyResource) ([]byte, error) {
	cipherData, err := base64.StdEncoding.DecodeString(res.Ciphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher([]byte(c.apiV3Key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, []byte(res.Nonce), cipherData, []byte(res.AssociatedData))
	if err != nil {
		return nil, err
	}
	return plain, nil
}
