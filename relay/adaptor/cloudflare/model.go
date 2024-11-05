package cloudflare

import "github.com/songquanpeng/one-api/relay/model"

type Request struct {
	Lora        string          `json:"lora,omitempty"`
	Prompt      string          `json:"prompt,omitempty"`
	Messages    []model.Message `json:"messages,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Raw         bool            `json:"raw,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}
