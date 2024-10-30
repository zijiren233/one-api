package model

type ResponseFormat struct {
	JsonSchema *JSONSchema `json:"json_schema,omitempty"`
	Type       string      `json:"type,omitempty"`
}

type JSONSchema struct {
	Schema      map[string]interface{} `json:"schema,omitempty"`
	Strict      *bool                  `json:"strict,omitempty"`
	Description string                 `json:"description,omitempty"`
	Name        string                 `json:"name"`
}

type Audio struct {
	Voice  string `json:"voice,omitempty"`
	Format string `json:"format,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type GeneralOpenAIRequest struct {
	Input            any             `json:"input,omitempty"`
	FunctionCall     any             `json:"function_call,omitempty"`
	Functions        any             `json:"functions,omitempty"`
	ToolChoice       any             `json:"tool_choice,omitempty"`
	Prompt           any             `json:"prompt,omitempty"`
	Stop             any             `json:"stop,omitempty"`
	StreamOptions    *StreamOptions  `json:"stream_options,omitempty"`
	ResponseFormat   *ResponseFormat `json:"response_format,omitempty"`
	Audio            *Audio          `json:"audio,omitempty"`
	EncodingFormat   string          `json:"encoding_format,omitempty"`
	Instruction      string          `json:"instruction,omitempty"`
	Size             string          `json:"size,omitempty"`
	User             string          `json:"user,omitempty"`
	Model            string          `json:"model,omitempty"`
	Messages         []Message       `json:"messages,omitempty"`
	Modalities       []string        `json:"modalities,omitempty"`
	Tools            []Tool          `json:"tools,omitempty"`
	PresencePenalty  float64         `json:"presence_penalty,omitempty"`
	TopK             int             `json:"top_k,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
	Temperature      float64         `json:"temperature,omitempty"`
	Seed             float64         `json:"seed,omitempty"`
	N                int             `json:"n,omitempty"`
	Dimensions       int             `json:"dimensions,omitempty"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	FrequencyPenalty float64         `json:"frequency_penalty,omitempty"`
	NumCtx           int             `json:"num_ctx,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
}

func (r GeneralOpenAIRequest) ParseInput() []string {
	if r.Input == nil {
		return nil
	}
	var input []string
	switch r.Input.(type) {
	case string:
		input = []string{r.Input.(string)}
	case []any:
		input = make([]string, 0, len(r.Input.([]any)))
		for _, item := range r.Input.([]any) {
			if str, ok := item.(string); ok {
				input = append(input, str)
			}
		}
	}
	return input
}
