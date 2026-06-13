package deepseek

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatOptions struct {
	Temperature    float64
	MaxTokens      int
	ResponseFormat string // "json_object" 或空
}

type Client struct {
	apiKey  string
	baseURL string
	model   string
	hc      *http.Client
}

func New(apiKey, baseURL, model string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		hc:      &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) ModelVersion() string { return c.model }

func (c *Client) Chat(messages []Message, opt ChatOptions) (string, error) {
	temp := opt.Temperature
	if temp == 0 {
		temp = 0.7
	}
	maxTokens := opt.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}
	payload := map[string]interface{}{
		"model":       c.model,
		"messages":    messages,
		"temperature": temp,
		"max_tokens":  maxTokens,
		"stream":      false,
	}
	if opt.ResponseFormat != "" {
		payload["response_format"] = map[string]string{"type": opt.ResponseFormat}
	}
	buf, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("deepseek http %d: %s", resp.StatusCode, string(body))
	}
	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if len(data.Choices) == 0 {
		return "", fmt.Errorf("deepseek empty choices")
	}
	return data.Choices[0].Message.Content, nil
}
