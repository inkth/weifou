package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// SubscribeService 封装微信小程序「一次性订阅消息」下发（subscribeMessage.send），
// 复用 LoginClient 的 access_token。未配置模板 ID / openid 为空 / 拿不到 token 时全链路 no-op
// （对齐 SecurityService 的降级风格），保证开发环境与未申请模板时不报错。
//
// ⚠️ 模板字段 key（thingN / amountN / timeN）必须与你在公众平台「功能→订阅消息」申请到的
// 模板字段顺序一致：按各 Notify 方法注释里的字段顺序申请，平台会自动生成 thing1/amount2/time3…，
// 顺序不同就改对应方法里的 key。
type SubscribeService struct {
	login           *LoginClient
	hc              *http.Client
	newQuestionTmpl string
	answeredTmpl    string
	refundedTmpl    string
	miniState       string
}

func NewSubscribeService(login *LoginClient, newQuestionTmpl, answeredTmpl, refundedTmpl, miniState string) *SubscribeService {
	if miniState == "" {
		miniState = "formal"
	}
	return &SubscribeService{
		login:           login,
		hc:              &http.Client{Timeout: 8 * time.Second},
		newQuestionTmpl: newQuestionTmpl,
		answeredTmpl:    answeredTmpl,
		refundedTmpl:    refundedTmpl,
		miniState:       miniState,
	}
}

// send 下发一条订阅消息。templateID / openid 为空、或拿不到 token 时静默 no-op。
func (s *SubscribeService) send(openid, templateID, page string, data map[string]string) {
	if templateID == "" || strings.TrimSpace(openid) == "" {
		return
	}
	token, err := s.login.AccessToken()
	if err != nil || token == "" {
		return // 开发环境/未配置降级
	}
	dataField := make(map[string]map[string]string, len(data))
	for k, v := range data {
		dataField[k] = map[string]string{"value": v}
	}
	payload := map[string]interface{}{
		"touser":            openid,
		"template_id":       templateID,
		"miniprogram_state": s.miniState,
		"lang":              "zh_CN",
		"data":              dataField,
	}
	if page != "" {
		payload["page"] = page
	}
	buf, _ := json.Marshal(payload)
	resp, err := s.hc.Post(
		"https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token="+token,
		"application/json",
		bytes.NewReader(buf),
	)
	if err != nil {
		log.Printf("[subscribe] send error: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(body, &r)
	// 43101 = 用户未授权/未订阅该模板（正常，静默）；其余打日志便于排查。
	if r.Errcode != 0 && r.Errcode != 43101 {
		log.Printf("[subscribe] send failed tmpl=%s errcode=%d errmsg=%s", templateID, r.Errcode, r.Errmsg)
	}
}

func yuan(fen int) string { return fmt.Sprintf("%.2f", float64(fen)/100) + "元" }

// clip 截断到 n 个字符（thing 类型上限 20）。
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// NotifyNewQuestion 通知主人「有新的付费提问」。
// 模板字段顺序：thing1=提问内容  amount2=金额  time3=回答截止时间
func (s *SubscribeService) NotifyNewQuestion(openid, question string, amountFen int, deadline time.Time, page string) {
	s.send(openid, s.newQuestionTmpl, page, map[string]string{
		"thing1":  clip(question, 20),
		"amount2": yuan(amountFen),
		"time3":   deadline.Format("2006-01-02 15:04"),
	})
}

// NotifyAnswered 通知访客「你的提问已回答」。
// 模板字段顺序：thing1=回答人  thing2=回答摘要  time3=回答时间
func (s *SubscribeService) NotifyAnswered(openid, hostName, answer string, answeredAt time.Time, page string) {
	s.send(openid, s.answeredTmpl, page, map[string]string{
		"thing1": clip(hostName, 20),
		"thing2": clip(answer, 20),
		"time3":  answeredAt.Format("2006-01-02 15:04"),
	})
}

// NotifyRefunded 通知访客「提问已退款」。
// 模板字段顺序：thing1=提问内容  amount2=退款金额  thing3=退款原因
func (s *SubscribeService) NotifyRefunded(openid, question string, amountFen int, reason, page string) {
	s.send(openid, s.refundedTmpl, page, map[string]string{
		"thing1":  clip(question, 20),
		"amount2": yuan(amountFen),
		"thing3":  clip(reason, 20),
	})
}
