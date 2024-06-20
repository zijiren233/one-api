package cloudflare

import "github.com/songquanpeng/one-api/relay/model"

type Request struct {
	Messages    []model.Message `json:"messages,omitempty"`
	Lora        string          `json:"lora,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Prompt      string          `json:"prompt,omitempty"`
	Raw         bool            `json:"raw,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type ImageRequest struct {
	Prompt   string  `json:"prompt"`
	Image    any     `json:"image,omitempty"`
	Mask     any     `json:"mask,omitempty"`
	NumSteps int     `json:"num_steps,omitempty"`
	Strength float64 `json:"strength,omitempty"`
	Guidance float64 `json:"guidance,omitempty"`
}
