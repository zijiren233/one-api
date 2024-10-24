package price

import (
	"fmt"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/songquanpeng/one-api/common/logger"
)

const (
	// /1K tokens
	PriceUnit = 1000
)

// ModelPrice
// https://platform.openai.com/docs/models/model-endpoint-compatibility
// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Blfmc9dlf
// https://openai.com/pricing
// 价格单位：人民币/1K tokens
var ModelPrice = map[string]float64{
	// https://openai.com/pricing
	"gpt-4":                   0.21,
	"gpt-4-0314":              0.21,
	"gpt-4-0613":              0.21,
	"gpt-4-32k":               0.42,
	"gpt-4-32k-0314":          0.42,
	"gpt-4-32k-0613":          0.42,
	"gpt-4-1106-preview":      0.07,
	"gpt-4-0125-preview":      0.07,
	"gpt-4-turbo-preview":     0.07,
	"gpt-4-turbo":             0.07,
	"gpt-4-turbo-2024-04-09":  0.07,
	"gpt-4o":                  0.035,
	"chatgpt-4o-latest":       0.035,
	"gpt-4o-2024-05-13":       0.035,
	"gpt-4o-2024-08-06":       0.0175,
	"gpt-4o-mini":             0.00105,
	"gpt-4o-mini-2024-07-18":  0.00105,
	"gpt-4-vision-preview":    0.07,
	"gpt-3.5-turbo":           0.0035,
	"gpt-3.5-turbo-0301":      0.0105,
	"gpt-3.5-turbo-0613":      0.0105,
	"gpt-3.5-turbo-16k":       0.021,
	"gpt-3.5-turbo-16k-0613":  0.021,
	"gpt-3.5-turbo-instruct":  0.0105,
	"gpt-3.5-turbo-1106":      0.007,
	"gpt-3.5-turbo-0125":      0.0035,
	"davinci-002":             0.014,
	"babbage-002":             0.0028,
	"text-ada-001":            0.0028,
	"text-babbage-001":        0.0035,
	"text-curie-001":          0.014,
	"text-davinci-002":        0.14,
	"text-davinci-003":        0.14,
	"text-davinci-edit-001":   0.14,
	"code-davinci-edit-001":   0.14,
	"whisper-1":               0.21,
	"tts-1":                   0.105,
	"tts-1-1106":              0.105,
	"tts-1-hd":                0.21,
	"tts-1-hd-1106":           0.21,
	"davinci":                 0.14,
	"curie":                   0.14,
	"babbage":                 0.14,
	"ada":                     0.14,
	"text-embedding-ada-002":  0.0007,
	"text-embedding-3-small":  0.00014,
	"text-embedding-3-large":  0.00091,
	"text-search-ada-doc-001": 0.14,
	"text-moderation-stable":  0.0014,
	"text-moderation-latest":  0.0014,
	"dall-e-2":                0.112,
	"dall-e-3":                0.224,
	// https://www.anthropic.com/api#pricing
	"claude-instant-1.2":         0.0056,
	"claude-2.0":                 0.056,
	"claude-2.1":                 0.056,
	"claude-3-haiku-20240307":    0.00175,
	"claude-3-sonnet-20240229":   0.021,
	"claude-3-5-sonnet-20240620": 0.021,
	"claude-3-opus-20240229":     0.105,
	// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/hlrk4akp7
	"ERNIE-4.0-8K":       0.12,
	"ERNIE-3.5-8K":       0.012,
	"ERNIE-3.5-8K-0205":  0.024,
	"ERNIE-3.5-8K-1222":  0.012,
	"ERNIE-Bot-8K":       0.024,
	"ERNIE-3.5-4K-0205":  0.012,
	"ERNIE-Speed-8K":     0.004,
	"ERNIE-Speed-128K":   0.004,
	"ERNIE-Lite-8K-0922": 0.008,
	"ERNIE-Lite-8K-0308": 0.003,
	"ERNIE-Tiny-8K":      0.001,
	"BLOOMZ-7B":          0.004,
	"Embedding-V1":       0.002,
	"bge-large-zh":       0.002,
	"bge-large-en":       0.002,
	"tao-8k":             0.002,
	// https://ai.google.dev/pricing
	"gemini-pro":       0.014,
	"gemini-1.0-pro":   0.014,
	"gemini-1.5-flash": 0.014,
	"gemini-1.5-pro":   0.014,
	"aqa":              0.014,
	// https://open.bigmodel.cn/pricing
	"glm-4":         0.1,
	"glm-4v":        0.1,
	"glm-3-turbo":   0.005,
	"embedding-2":   0.0005,
	"chatglm_turbo": 0.005,
	"chatglm_pro":   0.01,
	"chatglm_std":   0.005,
	"chatglm_lite":  0.002,
	"cogview-3":     0.25,
	// https://help.aliyun.com/zh/dashscope/developer-reference/tongyi-thousand-questions-metering-and-billing
	"qwen-turbo":                0.008,
	"qwen-plus":                 0.02,
	"qwen-max":                  0.02,
	"qwen-max-longcontext":      0.02,
	"text-embedding-v1":         0.0007,
	"ali-stable-diffusion-xl":   0.112,
	"ali-stable-diffusion-v1.5": 0.112,
	"wanx-v1":                   0.112,
	"SparkDesk":                 0.018,
	"SparkDesk-v1.1":            0.018,
	"SparkDesk-v2.1":            0.018,
	"SparkDesk-v3.1":            0.018,
	"SparkDesk-v3.1-128K":       0.018,
	"SparkDesk-v3.5":            0.018,
	"SparkDesk-v4.0":            0.018,
	"360GPT_S2_V9":              0.012,
	"embedding-bert-512-v1":     0.001,
	"embedding_s1_v1":           0.001,
	"semantic_similarity_s1_v1": 0.001,
	"hunyuan":                   0.1,
	"ChatStd":                   0.01,
	"ChatPro":                   0.1,
	// https://platform.moonshot.cn/pricing
	"moonshot-v1-8k":   0.012,
	"moonshot-v1-32k":  0.024,
	"moonshot-v1-128k": 0.06,
	// https://platform.baichuan-ai.com/price
	"Baichuan2-Turbo":      0.008,
	"Baichuan2-Turbo-192k": 0.016,
	"Baichuan2-53B":        0.02,
	// https://api.minimax.chat/document/price
	"abab6.5-chat":  0.03,
	"abab6.5s-chat": 0.01,
	"abab6-chat":    0.1,
	"abab5.5-chat":  0.015,
	"abab5.5s-chat": 0.005,
	// https://docs.mistral.ai/platform/pricing/
	"open-mistral-7b":       0.00175,
	"open-mixtral-8x7b":     0.0049,
	"mistral-small-latest":  0.014,
	"mistral-medium-latest": 0.0189,
	"mistral-large-latest":  0.056,
	"mistral-embed":         0.0007,
	// https://wow.groq.com/#:~:text=inquiries%C2%A0here.-,Model,-Current%20Speed
	"gemma-7b-it":                           0.00000049,
	"mixtral-8x7b-32768":                    0.00000168,
	"llama3-8b-8192":                        0.00000035,
	"llama3-70b-8192":                       0.00000413,
	"gemma2-9b-it":                          0.0000014,
	"llama-3.1-405b-reasoning":              0.00000623,
	"llama-3.1-70b-versatile":               0.00000413,
	"llama-3.1-8b-instant":                  0.00000035,
	"llama3-groq-70b-8192-tool-use-preview": 0.00000623,
	"llama3-groq-8b-8192-tool-use-preview":  0.00000133,
	// https://platform.lingyiwanwu.com/docs#-计费单元
	"yi-34b-chat-0205": 0.0025,
	"yi-34b-chat-200k": 0.012,
	"yi-vl-plus":       0.006,
	// https://platform.stepfun.com/docs/pricing/details
	"step-1-8k":    0.000005,
	"step-1-32k":   0.000015,
	"step-1-128k":  0.00004,
	"step-1-256k":  0.000095,
	"step-1-flash": 0.000001,
	"step-2-16k":   0.000038,
	"step-1v-8k":   0.000005,
	"step-1v-32k":  0.000015,
	// aws llama3 https://aws.amazon.com/cn/bedrock/pricing/
	"llama3-8b-8192(33)":  0.0021,
	"llama3-70b-8192(33)": 0.01855,
	// https://cohere.com/pricing
	"command":               0.007,
	"command-nightly":       0.007,
	"command-light":         0.007,
	"command-light-nightly": 0.007,
	"command-r":             0.0035,
	"command-r-plus":        0.021,
	// https://platform.deepseek.com/api-docs/pricing/
	"deepseek-chat":  0.001,
	"deepseek-coder": 0.001,
	// https://www.deepl.com/pro?cta=header-prices
	"deepl-zh": 0.175,
	"deepl-en": 0.175,
	"deepl-ja": 0.175,
}

var CompletionPrice = map[string]float64{
	// aws llama3
	"llama3-8b-8192(33)":  0.0042,
	"llama3-70b-8192(33)": 0.0245,
}

var (
	DefaultModelPrice      map[string]float64
	DefaultCompletionPrice map[string]float64
)

func init() {
	DefaultModelPrice = make(map[string]float64)
	for k, v := range ModelPrice {
		DefaultModelPrice[k] = v
	}
	DefaultCompletionPrice = make(map[string]float64)
	for k, v := range CompletionPrice {
		DefaultCompletionPrice[k] = v
	}
}

func AddNewMissingPrice(oldPrice string) string {
	newPrice := make(map[string]float64)
	err := json.Unmarshal([]byte(oldPrice), &newPrice)
	if err != nil {
		logger.SysError("error unmarshalling old price: " + err.Error())
		return oldPrice
	}
	for k, v := range DefaultModelPrice {
		if _, ok := newPrice[k]; !ok {
			newPrice[k] = v
		}
	}
	jsonBytes, err := json.Marshal(newPrice)
	if err != nil {
		logger.SysError("error marshalling new price: " + err.Error())
		return oldPrice
	}
	return string(jsonBytes)
}

func ModelPrice2JSONString() string {
	jsonBytes, err := json.Marshal(ModelPrice)
	if err != nil {
		logger.SysError("error marshalling model price: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateModelPriceByJSONString(jsonStr string) error {
	newModelPrice := make(map[string]float64)
	err := json.Unmarshal([]byte(jsonStr), &newModelPrice)
	if err != nil {
		logger.SysError("error unmarshalling model price: " + err.Error())
		return err
	}
	ModelPrice = newModelPrice
	return nil
}

func GetModelPrice(name string, channelType int) float64 {
	if strings.HasPrefix(name, "qwen-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}
	if strings.HasPrefix(name, "command-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}
	model := fmt.Sprintf("%s(%d)", name, channelType)
	if price, ok := ModelPrice[model]; ok {
		return price
	}
	if price, ok := DefaultModelPrice[model]; ok {
		return price
	}
	if price, ok := ModelPrice[name]; ok {
		return price
	}
	if price, ok := DefaultModelPrice[name]; ok {
		return price
	}
	logger.SysError("model price not found: " + name)
	return 0.42
}

func CompletionPrice2JSONString() string {
	jsonBytes, err := json.Marshal(CompletionPrice)
	if err != nil {
		logger.SysError("error marshalling completion price: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateCompletionPriceByJSONString(jsonStr string) error {
	newCompletionPrice := make(map[string]float64)
	err := json.Unmarshal([]byte(jsonStr), &newCompletionPrice)
	if err != nil {
		logger.SysError("error unmarshalling completion price: " + err.Error())
		return err
	}
	CompletionPrice = newCompletionPrice
	return nil
}

func GetCompletionPrice(name string, channelType int) float64 {
	if strings.HasPrefix(name, "qwen-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}
	model := fmt.Sprintf("%s(%d)", name, channelType)
	if price, ok := CompletionPrice[model]; ok {
		return price
	}
	if price, ok := DefaultCompletionPrice[model]; ok {
		return price
	}
	if price, ok := CompletionPrice[name]; ok {
		return price
	}
	if price, ok := DefaultCompletionPrice[name]; ok {
		return price
	}
	if strings.HasPrefix(name, "gpt-3.5") {
		if name == "gpt-3.5-turbo" || strings.HasSuffix(name, "0125") {
			return 0.0105
		}
		if strings.HasSuffix(name, "1106") {
			return 0.014
		}
		return 0.0187
	}
	if strings.HasPrefix(name, "gpt-4") {
		if strings.HasPrefix(name, "gpt-4o-mini") || name == "gpt-4o-2024-08-06" {
			return 0.007
		}
		if strings.HasPrefix(name, "gpt-4-turbo") ||
			strings.HasPrefix(name, "gpt-4o") ||
			strings.HasSuffix(name, "preview") {
			return 0.21
		}
		return 0.42
	}
	if name == "chatgpt-4o-latest" {
		return 0.105
	}
	if strings.HasPrefix(name, "claude-3") {
		return 0.105
	}
	if strings.HasPrefix(name, "claude-") {
		return 0.168
	}
	if strings.HasPrefix(name, "mistral-") {
		return 0.042
	}
	if strings.HasPrefix(name, "gemini-") {
		return 0.042
	}
	if strings.HasPrefix(name, "deepseek-") {
		return 0.002
	}
	switch name {
	case "llama2-70b-4096":
		return 0.0175
	case "llama3-8b-8192":
		return 0.0042
	case "llama3-70b-8192":
		return 0.0187
	case "command", "command-light", "command-nightly", "command-light-nightly":
		return 0.014
	case "command-r":
		return 0.0105
	case "command-r-plus":
		return 0.105
	}
	return GetModelPrice(name, channelType)
}
