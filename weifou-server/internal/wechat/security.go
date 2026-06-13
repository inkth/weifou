package wechat

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// SecurityService 封装 msg_sec_check 文本内容安全。
// 未配置 appid/secret 或调用失败时降级为 pass=true（开发环境可用）。
type SecurityService struct {
	login *LoginClient
	hc    *http.Client
}

func NewSecurityService(login *LoginClient) *SecurityService {
	return &SecurityService{login: login, hc: &http.Client{Timeout: 8 * time.Second}}
}

// CheckText 返回是否通过。content 为空直接通过。
func (s *SecurityService) CheckText(content, openid string) bool {
	if strings.TrimSpace(content) == "" {
		return true
	}
	token, err := s.login.AccessToken()
	if err != nil || token == "" {
		// 开发环境降级
		return true
	}
	payload := map[string]interface{}{
		"version": 2,
		"scene":   1,
		"content": content,
	}
	if openid != "" {
		payload["openid"] = openid
	}
	buf, _ := json.Marshal(payload)
	resp, err := s.hc.Post(
		"https://api.weixin.qq.com/wxa/msg_sec_check?access_token="+token,
		"application/json",
		bytes.NewReader(buf),
	)
	if err != nil {
		log.Printf("[security] msg_sec_check error: %v", err)
		return true
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data struct {
		Errcode int `json:"errcode"`
		Result  struct {
			Suggest string `json:"suggest"`
			Label   int    `json:"label"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return true
	}
	return data.Errcode == 0 && data.Result.Suggest == "pass"
}
