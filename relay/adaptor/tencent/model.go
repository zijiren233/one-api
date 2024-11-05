package tencent

type Message struct {
	Role    string `json:"Role"`
	Content string `json:"Content"`
}

type ChatRequest struct {
	Model       string     `json:"Model"`
	Messages    []*Message `json:"Messages"`
	TopP        float64    `json:"TopP,omitempty"`
	Temperature float64    `json:"Temperature,omitempty"`
	Stream      bool       `json:"Stream,omitempty"`
}

type Error struct {
	Message string `json:"Message"`
	Code    int    `json:"Code"`
}

type Usage struct {
	PromptTokens     int `json:"PromptTokens"`
	CompletionTokens int `json:"CompletionTokens"`
	TotalTokens      int `json:"TotalTokens"`
}

type ResponseChoices struct {
	FinishReason string  `json:"FinishReason,omitempty"` // 流式结束标志位，为 stop 则表示尾包
	Messages     Message `json:"Message,omitempty"`      // 内容，同步模式返回内容，流模式为 null 输出 content 内容总数最多支持 1024token。
	Delta        Message `json:"Delta,omitempty"`        // 内容，流模式返回内容，同步模式为 null 输出 content 内容总数最多支持 1024token。
}

type ChatResponse struct {
	Id      string            `json:"Id,omitempty"`
	Note    string            `json:"Note,omitempty"`
	ReqID   string            `json:"Req_id,omitempty"`
	Choices []ResponseChoices `json:"Choices,omitempty"`
	Error   Error             `json:"Error,omitempty"`
	Usage   Usage             `json:"Usage,omitempty"`
	Created int64             `json:"Created,omitempty"`
}

type ChatResponseP struct {
	Response ChatResponse `json:"Response,omitempty"`
}
