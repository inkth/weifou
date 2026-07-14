// Package mp 实现服务号（公众号）承接 + 召回：把 iOS 会员开通的"外部支付引导"从小程序
// 移到服务号（消掉小程序"诱导外部支付"风险）。关注 / 点菜单 / 发"会员"关键词时，按 unionid
// 匹配到微否账号，用客服消息把"在浏览器开通"的 H5 链接推给用户（会员入同一账号）。
//
// 全链路配置门控：未配 WX_MP_APPID/SECRET/TOKEN 则回调校验失败、不处理；缺 H5_BASE_URL
// 则只发引导文案。需在公众平台把消息模式配为「明文/兼容模式」（本实现不含安全模式 AES 解密）。
package mp

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/models"
)

// mbrLinker 由 membership.Handler 实现（构造 H5 开通链接）。
type mbrLinker interface {
	H5URL(base, userID, openid string) (string, error)
}

// tokener 由 wechat.LoginClient 实现（服务号 access_token 缓存）。
type tokener interface {
	AccessToken() (string, error)
}

type Handler struct {
	db     *gorm.DB
	login  tokener   // 服务号凭证（appid/secret）
	mbr    mbrLinker // 构造 H5 开通链接
	token  string    // 回调校验 token
	h5Base string
	hc     *http.Client
}

func NewHandler(db *gorm.DB, login tokener, mbr mbrLinker, token, h5Base string) *Handler {
	return &Handler{db: db, login: login, mbr: mbr, token: token, h5Base: h5Base, hc: &http.Client{Timeout: 8 * time.Second}}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/wx/mp/callback", h.verify)
	rg.POST("/wx/mp/callback", h.event)
}

// checkSign 校验 signature = sha1(sort(token,timestamp,nonce))。
func (h *Handler) checkSign(c *gin.Context) bool {
	if h.token == "" {
		return false
	}
	arr := []string{h.token, c.Query("timestamp"), c.Query("nonce")}
	sort.Strings(arr)
	sum := sha1.Sum([]byte(strings.Join(arr, "")))
	return hex.EncodeToString(sum[:]) == c.Query("signature")
}

// verify 服务器配置校验（返回 echostr）。
func (h *Handler) verify(c *gin.Context) {
	if h.checkSign(c) {
		c.String(http.StatusOK, c.Query("echostr"))
		return
	}
	c.String(http.StatusForbidden, "")
}

type wxMsg struct {
	FromUserName string `xml:"FromUserName"`
	MsgType      string `xml:"MsgType"`
	Event        string `xml:"Event"`
	EventKey     string `xml:"EventKey"`
	Content      string `xml:"Content"`
}

// event 处理消息/事件。被动回复留空，主动用客服消息推送（48h 内可达）。
func (h *Handler) event(c *gin.Context) {
	if !h.checkSign(c) {
		c.String(http.StatusForbidden, "")
		return
	}
	raw, _ := io.ReadAll(c.Request.Body)
	var m wxMsg
	if xml.Unmarshal(raw, &m) != nil {
		c.String(http.StatusOK, "success")
		return
	}
	if m.FromUserName != "" && h.isMembershipTrigger(&m) {
		go h.pushMembership(m.FromUserName)
	}
	c.String(http.StatusOK, "success")
}

// isMembershipTrigger 关注 / 点「开通会员」菜单 / 发含"会员|开通"的文字 → 触发开通引导。
func (h *Handler) isMembershipTrigger(m *wxMsg) bool {
	switch m.MsgType {
	case "event":
		return m.Event == "subscribe" || (m.Event == "CLICK" && m.EventKey == "OPEN_MEMBERSHIP")
	case "text":
		return strings.Contains(m.Content, "会员") || strings.Contains(m.Content, "开通")
	}
	return false
}

// pushMembership 按 unionid 匹配账号 → 客服消息推「在浏览器开通」链接。
func (h *Handler) pushMembership(mpOpenid string) {
	if h.login == nil {
		return
	}
	userID, payOpenid := h.matchAccount(mpOpenid)
	if userID == "" {
		// 关注者还没用过小程序：引导先去体验（无法把会员发放给未知账号）。
		h.sendText(mpOpenid, "欢迎！先在「微否」小程序体验人类基本功计划的第一幕，想继续全部能力路径就回这里开通全课会员～")
		return
	}
	if h.h5Base == "" || h.mbr == nil {
		h.sendText(mpOpenid, "会员开通入口准备中，稍后再来～")
		return
	}
	link, err := h.mbr.H5URL(h.h5Base, userID, payOpenid)
	if err != nil {
		return
	}
	h.sendText(mpOpenid, "加入「微否·人类基本功计划」，解锁 15 门完整课程与能力路径 👉\n"+link+"\n\n在浏览器打开此链接完成开通，开通后回小程序自动解锁。")
}

// matchAccount 服务号 openid → unionid → 微否 User。返回 (userID, 该用户的锚点 openid)。
func (h *Handler) matchAccount(mpOpenid string) (string, string) {
	unionid := h.unionidOf(mpOpenid)
	if unionid == "" {
		return "", ""
	}
	var u models.User
	if h.db.Where("unionid = ?", unionid).First(&u).Error != nil {
		return "", ""
	}
	return u.ID, u.Openid
}

// unionidOf 取关注者 unionid（需服务号已绑定微信开放平台）。
func (h *Handler) unionidOf(openid string) string {
	token, err := h.login.AccessToken()
	if err != nil || token == "" {
		return ""
	}
	q := url.Values{}
	q.Set("access_token", token)
	q.Set("openid", openid)
	q.Set("lang", "zh_CN")
	resp, err := h.hc.Get("https://api.weixin.qq.com/cgi-bin/user/info?" + q.Encode())
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d struct {
		Unionid string `json:"unionid"`
	}
	_ = json.Unmarshal(body, &d)
	return d.Unionid
}

// sendText 客服消息发文本（用户与服务号交互后 48h 内可达）。
func (h *Handler) sendText(openid, content string) {
	token, err := h.login.AccessToken()
	if err != nil || token == "" {
		return
	}
	payload := map[string]interface{}{
		"touser":  openid,
		"msgtype": "text",
		"text":    map[string]string{"content": content},
	}
	buf, _ := json.Marshal(payload)
	resp, err := h.hc.Post(
		"https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token="+token,
		"application/json", bytes.NewReader(buf),
	)
	if err != nil {
		log.Printf("[mp] kf send error: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(body, &r)
	if r.Errcode != 0 {
		log.Printf("[mp] kf send failed errcode=%d errmsg=%s", r.Errcode, r.Errmsg)
	}
}
