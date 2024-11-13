package cloudflare

import "github.com/songquanpeng/one-api/relay/model"

type Request struct {
	Temperature *float64        `json:"temperature,omitempty"`
	Lora        string          `json:"lora,omitempty"`
	Prompt      string          `json:"prompt,omitempty"`
	Messages    []model.Message `json:"messages,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Raw         bool            `json:"raw,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}
