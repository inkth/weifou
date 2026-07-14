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
	leadTmpl        string
	learnRemindTmpl string
	miniState       string
}

func NewSubscribeService(login *LoginClient, newQuestionTmpl, answeredTmpl, leadTmpl, learnRemindTmpl, miniState string) *SubscribeService {
	if miniState == "" {
		miniState = "formal"
	}
	return &SubscribeService{
		login:           login,
		hc:              &http.Client{Timeout: 8 * time.Second},
		newQuestionTmpl: newQuestionTmpl,
		answeredTmpl:    answeredTmpl,
		leadTmpl:        leadTmpl,
		learnRemindTmpl: learnRemindTmpl,
		miniState:       miniState,
	}
}

// LearnRemindReady 学习提醒模板是否已配置（未配时提醒承诺链路整体 no-op）。
func (s *SubscribeService) LearnRemindReady() bool { return s.learnRemindTmpl != "" }

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

// NotifyNewQuestion 通知主人「有新的提问」。
// 模板字段顺序：thing1=提问内容  time2=提问时间
func (s *SubscribeService) NotifyNewQuestion(openid, question string, askedAt time.Time, page string) {
	s.send(openid, s.newQuestionTmpl, page, map[string]string{
		"thing1": clip(question, 20),
		"time2":  askedAt.Format("2006-01-02 15:04"),
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

// NotifyNewLead 通知主人「有新访客线索」。
// 模板字段顺序：thing1=留言内容  thing2=访客  time3=留言时间
func (s *SubscribeService) NotifyNewLead(openid, note, visitorName string, at time.Time, page string) {
	s.send(openid, s.leadTmpl, page, map[string]string{
		"thing1": clip(note, 20),
		"thing2": clip(visitorName, 20),
		"time3":  at.Format("2006-01-02 15:04"),
	})
}

// NotifyLearnRemind 学习提醒承诺：用户昨天课后点了「明天叫我」，现在到点了。
// 建议申请「学习提醒/上课提醒」类目模板，字段顺序：thing1=课程  thing2=提醒内容  time3=时间。
func (s *SubscribeService) NotifyLearnRemind(openid, agentName, note string, at time.Time, page string) {
	s.send(openid, s.learnRemindTmpl, page, map[string]string{
		"thing1": clip(agentName, 20),
		"thing2": clip(note, 20),
		"time3":  at.Format("2006-01-02 15:04"),
	})
}
