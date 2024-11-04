package xunfei

import (
	"github.com/songquanpeng/one-api/relay/model"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Functions struct {
	Text []model.Function `json:"text,omitempty"`
}

type ChatRequest struct {
	Payload struct {
		Functions *Functions `json:"functions,omitempty"`
		Message   struct {
			Text []Message `json:"text"`
		} `json:"message"`
	} `json:"payload"`
	Header struct {
		AppId string `json:"app_id"`
	} `json:"header"`
	Parameter struct {
		Chat struct {
			Domain      string  `json:"domain,omitempty"`
			Temperature float64 `json:"temperature,omitempty"`
			TopK        int     `json:"top_k,omitempty"`
			MaxTokens   int     `json:"max_tokens,omitempty"`
			Auditing    bool    `json:"auditing,omitempty"`
		} `json:"chat"`
	} `json:"parameter"`
}

type ChatResponseTextItem struct {
	FunctionCall *model.Function `json:"function_call"`
	Content      string          `json:"content"`
	Role         string          `json:"role"`
	ContentType  string          `json:"content_type"`
	Index        int             `json:"index"`
}

type ChatResponse struct {
	Header struct {
		Message string `json:"message"`
		Sid     string `json:"sid"`
		Code    int    `json:"code"`
		Status  int    `json:"status"`
	} `json:"header"`
	Payload struct {
		Choices struct {
			Text   []ChatResponseTextItem `json:"text"`
			Status int                    `json:"status"`
			Seq    int                    `json:"seq"`
		} `json:"choices"`
		Usage struct {
			//Text struct {
			//	QuestionTokens   string `json:"question_tokens"`
			//	PromptTokens     string `json:"prompt_tokens"`
			//	CompletionTokens string `json:"completion_tokens"`
			//	TotalTokens      string `json:"total_tokens"`
			//} `json:"text"`
			Text model.Usage `json:"text"`
		} `json:"usage"`
	} `json:"payload"`
}
