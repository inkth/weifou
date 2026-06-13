package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// LoginClient 处理 code2Session 与 access_token 缓存
type LoginClient struct {
	appID     string
	appSecret string
	hc        *http.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

func NewLoginClient(appID, appSecret string) *LoginClient {
	return &LoginClient{
		appID:     appID,
		appSecret: appSecret,
		hc:        &http.Client{Timeout: 8 * time.Second},
	}
}

type Session struct {
	Openid     string
	Unionid    string
	SessionKey string
	// 仅 App OAuth2 登录会填充（来自 sns/userinfo）
	Nickname  string
	AvatarURL string
}

func (c *LoginClient) Code2Session(code string) (*Session, error) {
	if c.appID == "" || c.appSecret == "" {
		return nil, fmt.Errorf("微信小程序配置缺失")
	}
	q := url.Values{}
	q.Set("appid", c.appID)
	q.Set("secret", c.appSecret)
	q.Set("js_code", code)
	q.Set("grant_type", "authorization_code")

	resp, err := c.hc.Get("https://api.weixin.qq.com/sns/jscode2session?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data struct {
		Openid     string `json:"openid"`
		Unionid    string `json:"unionid"`
		SessionKey string `json:"session_key"`
		Errcode    int    `json:"errcode"`
		Errmsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if data.Errcode != 0 {
		return nil, fmt.Errorf("微信登录失败: %s", data.Errmsg)
	}
	return &Session{Openid: data.Openid, Unionid: data.Unionid, SessionKey: data.SessionKey}, nil
}

// OAuth2Code2Session 处理原生 App（移动应用）授权登录：
// 用 fluwx sendWeChatAuth 拿到的 code 换 openid/unionid，再拉 userinfo。
// 与小程序 jscode2session 不同，走 sns/oauth2/access_token。
func (c *LoginClient) OAuth2Code2Session(code string) (*Session, error) {
	if c.appID == "" || c.appSecret == "" {
		return nil, fmt.Errorf("微信移动应用配置缺失")
	}
	q := url.Values{}
	q.Set("appid", c.appID)
	q.Set("secret", c.appSecret)
	q.Set("code", code)
	q.Set("grant_type", "authorization_code")

	resp, err := c.hc.Get("https://api.weixin.qq.com/sns/oauth2/access_token?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data struct {
		AccessToken string `json:"access_token"`
		Openid      string `json:"openid"`
		Unionid     string `json:"unionid"`
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if data.Errcode != 0 {
		return nil, fmt.Errorf("微信登录失败: %s", data.Errmsg)
	}

	s := &Session{Openid: data.Openid, Unionid: data.Unionid}
	// 拉取昵称/头像（失败不阻断登录）。
	if info, err := c.userInfo(data.AccessToken, data.Openid); err == nil {
		s.Nickname = info.Nickname
		s.AvatarURL = info.HeadImgURL
	}
	return s, nil
}

type wxUserInfo struct {
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
}

func (c *LoginClient) userInfo(accessToken, openid string) (*wxUserInfo, error) {
	q := url.Values{}
	q.Set("access_token", accessToken)
	q.Set("openid", openid)
	q.Set("lang", "zh_CN")
	resp, err := c.hc.Get("https://api.weixin.qq.com/sns/userinfo?" + q.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var info wxUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// AccessToken 获取并缓存接口调用凭证（内容安全、小程序码等共用）
func (c *LoginClient) AccessToken() (string, error) {
	if c.appID == "" || c.appSecret == "" {
		return "", fmt.Errorf("no wx config")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp.Add(-time.Minute)) {
		return c.token, nil
	}
	q := url.Values{}
	q.Set("grant_type", "client_credential")
	q.Set("appid", c.appID)
	q.Set("secret", c.appSecret)
	resp, err := c.hc.Get("https://api.weixin.qq.com/cgi-bin/token?" + q.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data.AccessToken == "" {
		return "", fmt.Errorf("get access_token failed: %s", data.Errmsg)
	}
	c.token = data.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(data.ExpiresIn) * time.Second)
	return c.token, nil
}
