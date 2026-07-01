// Package musicgen 封装「文本/歌词 → 带人声完整歌」的音乐生成 provider。
// 做成可插拔接口：默认 fal（复用 FALAI_API_KEY），将来换国内合规 provider 只改本包，不动上层。
package musicgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Provider 生成一首歌，返回**远端**音频 URL（上层负责下载并重存到本站）。
type Provider interface {
	Ready() bool
	Generate(lyrics, style string) (remoteAudioURL string, err error)
}

// New 按 provider 名构造。目前只有 fal；未配 key 时返回始终报错的 disabled provider。
func New(provider, key, baseURL, model string) Provider {
	if key == "" {
		return disabled{}
	}
	if baseURL == "" {
		baseURL = "https://fal.run"
	}
	if model == "" {
		model = "fal-ai/ace-step"
	}
	switch strings.ToLower(provider) {
	case "", "fal":
		return &falProvider{key: key, baseURL: strings.TrimRight(baseURL, "/"), model: model,
			hc: &http.Client{Timeout: 180 * time.Second}} // 生成慢，给足超时
	default:
		return disabled{}
	}
}

type disabled struct{}

func (disabled) Ready() bool                          { return false }
func (disabled) Generate(_, _ string) (string, error) { return "", fmt.Errorf("music provider 未配置") }

// falProvider 调 fal 的 lyrics→song 模型（默认 fal-ai/ace-step：tags=曲风, lyrics=歌词）。
type falProvider struct {
	key, baseURL, model string
	hc                  *http.Client
}

func (p *falProvider) Ready() bool { return true }

func (p *falProvider) Generate(lyrics, style string) (string, error) {
	body := map[string]interface{}{
		"tags":   style,  // ace-step：曲风/情绪标签
		"lyrics": lyrics, // 歌词（含 [verse]/[chorus] 结构更好）
	}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", p.baseURL+"/"+p.model, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Key "+p.key)

	resp, err := p.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("fal 返回 %d: %s", resp.StatusCode, clip(string(raw), 200))
	}

	// 兼容多种返回结构：audio.url / audio_file.url / audio_url。
	var out struct {
		Audio     *struct{ URL string `json:"url"` } `json:"audio"`
		AudioFile *struct{ URL string `json:"url"` } `json:"audio_file"`
		AudioURL  string                             `json:"audio_url"`
	}
	if jerr := json.Unmarshal(raw, &out); jerr != nil {
		return "", fmt.Errorf("解析 fal 返回失败: %v", jerr)
	}
	switch {
	case out.Audio != nil && out.Audio.URL != "":
		return out.Audio.URL, nil
	case out.AudioFile != nil && out.AudioFile.URL != "":
		return out.AudioFile.URL, nil
	case out.AudioURL != "":
		return out.AudioURL, nil
	}
	return "", fmt.Errorf("fal 返回里没有音频 URL: %s", clip(string(raw), 200))
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
