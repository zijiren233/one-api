package xunfei

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/constant"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

// https://console.xfyun.cn/services/cbm
// https://www.xfyun.cn/doc/spark/Web.html

func requestOpenAI2Xunfei(request model.GeneralOpenAIRequest, xunfeiAppId string, domain string) *ChatRequest {
	messages := make([]Message, 0, len(request.Messages))
	for _, message := range request.Messages {
		messages = append(messages, Message{
			Role:    message.Role,
			Content: message.StringContent(),
		})
	}
	xunfeiRequest := ChatRequest{}
	xunfeiRequest.Header.AppId = xunfeiAppId
	xunfeiRequest.Parameter.Chat.Domain = domain
	xunfeiRequest.Parameter.Chat.Temperature = request.Temperature
	xunfeiRequest.Parameter.Chat.TopK = request.N
	xunfeiRequest.Parameter.Chat.MaxTokens = request.MaxTokens
	xunfeiRequest.Payload.Message.Text = messages

	if strings.HasPrefix(domain, "generalv3") || domain == "4.0Ultra" {
		functions := make([]model.Function, len(request.Tools))
		for i, tool := range request.Tools {
			functions[i] = tool.Function
		}
		xunfeiRequest.Payload.Functions = &Functions{
			Text: functions,
		}
	}

	return &xunfeiRequest
}

func getToolCalls(response *ChatResponse) []model.Tool {
	var toolCalls []model.Tool
	if len(response.Payload.Choices.Text) == 0 {
		return toolCalls
	}
	item := response.Payload.Choices.Text[0]
	if item.FunctionCall == nil {
		return toolCalls
	}
	toolCall := model.Tool{
		Id:       fmt.Sprintf("call_%s", random.GetUUID()),
		Type:     "function",
		Function: *item.FunctionCall,
	}
	toolCalls = append(toolCalls, toolCall)
	return toolCalls
}

func responseXunfei2OpenAI(response *ChatResponse) *openai.TextResponse {
	if len(response.Payload.Choices.Text) == 0 {
		response.Payload.Choices.Text = []ChatResponseTextItem{
			{
				Content: "",
			},
		}
	}
	choice := openai.TextResponseChoice{
		Index: 0,
		Message: model.Message{
			Role:      "assistant",
			Content:   response.Payload.Choices.Text[0].Content,
			ToolCalls: getToolCalls(response),
		},
		FinishReason: constant.StopFinishReason,
	}
	fullTextResponse := openai.TextResponse{
		Id:      fmt.Sprintf("chatcmpl-%s", random.GetUUID()),
		Object:  "chat.completion",
		Created: helper.GetTimestamp(),
		Choices: []openai.TextResponseChoice{choice},
		Usage:   response.Payload.Usage.Text,
	}
	return &fullTextResponse
}

func streamResponseXunfei2OpenAI(xunfeiResponse *ChatResponse) *openai.ChatCompletionsStreamResponse {
	if len(xunfeiResponse.Payload.Choices.Text) == 0 {
		xunfeiResponse.Payload.Choices.Text = []ChatResponseTextItem{
			{
				Content: "",
			},
		}
	}
	var choice openai.ChatCompletionsStreamResponseChoice
	choice.Delta.Content = xunfeiResponse.Payload.Choices.Text[0].Content
	choice.Delta.ToolCalls = getToolCalls(xunfeiResponse)
	if xunfeiResponse.Payload.Choices.Status == 2 {
		choice.FinishReason = &constant.StopFinishReason
	}
	response := openai.ChatCompletionsStreamResponse{
		Id:      fmt.Sprintf("chatcmpl-%s", random.GetUUID()),
		Object:  "chat.completion.chunk",
		Created: helper.GetTimestamp(),
		Model:   "SparkDesk",
		Choices: []openai.ChatCompletionsStreamResponseChoice{choice},
	}
	return &response
}

func buildXunfeiAuthUrl(hostUrl string, apiKey, apiSecret string) string {
	HmacWithShaToBase64 := func(algorithm, data, key string) string {
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(data))
		encodeData := mac.Sum(nil)
		return base64.StdEncoding.EncodeToString(encodeData)
	}
	ul, err := url.Parse(hostUrl)
	if err != nil {
		fmt.Println(err)
	}
	date := time.Now().UTC().Format(time.RFC1123)
	signString := []string{"host: " + ul.Host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	sign := strings.Join(signString, "\n")
	sha := HmacWithShaToBase64("hmac-sha256", sign, apiSecret)
	authUrl := fmt.Sprintf("hmac username=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey,
		"hmac-sha256", "host date request-line", sha)
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))
	v := url.Values{}
	v.Add("host", ul.Host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	callUrl := hostUrl + "?" + v.Encode()
	return callUrl
}

func StreamHandler(c *gin.Context, meta *meta.Meta, textRequest model.GeneralOpenAIRequest, appId string, apiSecret string, apiKey string) (*model.ErrorWithStatusCode, *model.Usage) {
	domain, authUrl, err := getXunfeiAuthUrl(meta.ActualModelName, apiKey, apiSecret)
	if err != nil {
		return openai.ErrorWrapper(err, "invalid_model_name", http.StatusBadRequest), nil
	}
	dataChan, err := xunfeiMakeRequest(textRequest, domain, authUrl, appId)
	if err != nil {
		return openai.ErrorWrapper(err, "xunfei_request_failed", http.StatusInternalServerError), nil
	}
	common.SetEventStreamHeaders(c)
	var usage model.Usage
	c.Stream(func(w io.Writer) bool {
		xunfeiResponse, ok := <-dataChan
		if !ok {
			return false
		}
		usage.PromptTokens += xunfeiResponse.Payload.Usage.Text.PromptTokens
		usage.CompletionTokens += xunfeiResponse.Payload.Usage.Text.CompletionTokens
		usage.TotalTokens += xunfeiResponse.Payload.Usage.Text.TotalTokens
		response := streamResponseXunfei2OpenAI(&xunfeiResponse)
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			logger.SysError("error marshalling stream response: " + err.Error())
			return true
		}
		c.Render(-1, common.CustomEvent{Data: "data: " + string(jsonResponse)})
		return true
	})

	c.Render(-1, common.CustomEvent{Data: "data: [DONE]"})

	return nil, &usage
}

func Handler(c *gin.Context, meta *meta.Meta, textRequest model.GeneralOpenAIRequest, appId string, apiSecret string, apiKey string) (*model.ErrorWithStatusCode, *model.Usage) {
	domain, authUrl, err := getXunfeiAuthUrl(meta.ActualModelName, apiKey, apiSecret)
	if err != nil {
		return openai.ErrorWrapper(err, "invalid_model_name", http.StatusBadRequest), nil
	}
	dataChan, err := xunfeiMakeRequest(textRequest, domain, authUrl, appId)
	if err != nil {
		return openai.ErrorWrapper(err, "xunfei_request_failed", http.StatusInternalServerError), nil
	}
	var usage model.Usage
	var content string
	for xunfeiResponse := range dataChan {
		if xunfeiResponse.Header.Status == 2 {
			return openai.ErrorWrapper(fmt.Errorf("xunfei request failed: %s, code: %d", xunfeiResponse.Header.Message, xunfeiResponse.Header.Code), fmt.Sprintf("xunfei_request_failed_%d", xunfeiResponse.Header.Status), http.StatusInternalServerError), nil
		}

		if len(xunfeiResponse.Payload.Choices.Text) == 0 {
			return openai.ErrorWrapper(errors.New("xunfei empty response detected"), "xunfei_empty_response_detected", http.StatusInternalServerError), nil
		}

		content += xunfeiResponse.Payload.Choices.Text[0].Content
		usage.PromptTokens += xunfeiResponse.Payload.Usage.Text.PromptTokens
		usage.CompletionTokens += xunfeiResponse.Payload.Usage.Text.CompletionTokens
		usage.TotalTokens += xunfeiResponse.Payload.Usage.Text.TotalTokens

		response := responseXunfei2OpenAI(&xunfeiResponse)
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
		}
		c.Writer.Header().Set("Content-Type", "application/json")
		_, _ = c.Writer.Write(jsonResponse)
	}

	return nil, &usage
}

func xunfeiMakeRequest(textRequest model.GeneralOpenAIRequest, domain, authUrl, appId string) (chan ChatResponse, error) {
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, resp, err := d.Dial(authUrl, nil)
	if err != nil || resp.StatusCode != 101 {
		return nil, err
	}
	data := requestOpenAI2Xunfei(textRequest, appId, domain)
	err = conn.WriteJSON(data)
	if err != nil {
		conn.Close()
		return nil, err
	}

	dataChan := make(chan ChatResponse)
	go func() {
		defer conn.Close()
		defer close(dataChan)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				logger.SysError("error reading stream response: " + err.Error())
				break
			}
			fmt.Println(string(msg))
			var response ChatResponse
			err = json.Unmarshal(msg, &response)
			if err != nil {
				logger.SysError("error unmarshalling stream response: " + err.Error())
				break
			}
			dataChan <- response
			if response.Header.Status == 2 {
				return
			}
		}
	}()

	return dataChan, nil
}

func getXunfeiAuthUrl(modelName string, apiKey string, apiSecret string) (string, string, error) {
	var domain string
	var path string

	_, s, ok := strings.Cut(modelName, "-")
	if !ok {
		return "", "", errors.New("invalid model name")
	}

	switch strings.ToLower(s) {
	case "lite":
		domain = "lite"
		path = "v1.1/chat"
	case "pro":
		domain = "generalv3"
		path = "v3.1/chat"
	case "pro-128k":
		domain = "pro-128k"
		path = "chat/pro-128k"
	case "max":
		domain = "generalv3.5"
		path = "v3.5/chat"
	case "max-32k":
		domain = "max-32k"
		path = "chat/max-32k"
	case "4.0-ultra":
		domain = "4.0Ultra"
		path = "v4.0/chat"
	default:
		return "", "", errors.New("invalid model name")
	}

	authUrl := buildXunfeiAuthUrl(fmt.Sprintf("wss://spark-api.xf-yun.com/%s", path), apiKey, apiSecret)
	return domain, authUrl, nil
}
