package cloudflare

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/model"
)

func ConvertRequest(textRequest model.GeneralOpenAIRequest) *Request {
	return &Request{
		Messages:    textRequest.Messages,
		MaxTokens:   textRequest.MaxTokens,
		Stream:      textRequest.Stream,
		Temperature: textRequest.Temperature,
	}
}

func ConvertCompletionsRequest(textRequest model.GeneralOpenAIRequest) *Request {
	return &Request{
		Prompt:      textRequest.Prompt.(string),
		MaxTokens:   textRequest.MaxTokens,
		Stream:      textRequest.Stream,
		Temperature: textRequest.Temperature,
	}
}

func ResponseCloudflare2OpenAI(cloudflareResponse *Response) *openai.TextResponse {
	choice := openai.TextResponseChoice{
		Index: 0,
		Message: model.Message{
			Role:    "assistant",
			Content: cloudflareResponse.Result.Response,
		},
		FinishReason: "stop",
	}
	fullTextResponse := openai.TextResponse{
		Object:  "chat.completion",
		Created: helper.GetTimestamp(),
		Choices: []openai.TextResponseChoice{choice},
	}
	return &fullTextResponse
}

func StreamResponseCloudflare2OpenAI(cloudflareResponse *StreamResponse) *openai.ChatCompletionsStreamResponse {
	var choice openai.ChatCompletionsStreamResponseChoice
	choice.Delta.Content = cloudflareResponse.Response
	choice.Delta.Role = "assistant"
	openaiResponse := openai.ChatCompletionsStreamResponse{
		Object:  "chat.completion.chunk",
		Choices: []openai.ChatCompletionsStreamResponseChoice{choice},
		Created: helper.GetTimestamp(),
	}
	return &openaiResponse
}

func StreamHandler(c *gin.Context, resp *http.Response, promptTokens int, modelName string) (*model.ErrorWithStatusCode, *model.Usage) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	common.SetEventStreamHeaders(c)
	id := helper.GetResponseID(c)
	responseModel := c.GetString("original_model")
	var responseText string
	for scanner.Scan() {
		data := scanner.Text()
		if len(data) < len("data: ") {
			continue
		}
		data = strings.TrimPrefix(strings.TrimSuffix(data, "\r"), "data: ")
		if data == "[DONE]" {
			c.Render(-1, common.CustomEvent{Data: "data: [DONE]"})
			break
		}
		var cloudflareResponse StreamResponse
		err := json.Unmarshal([]byte(data), &cloudflareResponse)
		if err != nil {
			logger.SysError("error unmarshalling stream response: " + err.Error())
			continue
		}
		response := StreamResponseCloudflare2OpenAI(&cloudflareResponse)
		if response == nil {
			continue
		}
		responseText += cloudflareResponse.Response
		response.Id = id
		response.Model = responseModel
		jsonStr, err := json.Marshal(response)
		if err != nil {
			logger.SysError("error marshalling stream response: " + err.Error())
			continue
		}
		c.Render(-1, common.CustomEvent{Data: "data: " + string(jsonStr)})
		c.Writer.Flush()
	}
	_ = resp.Body.Close()
	usage := openai.ResponseText2Usage(responseText, responseModel, promptTokens)
	return nil, usage
}

func Handler(c *gin.Context, resp *http.Response, promptTokens int, modelName string) (*model.ErrorWithStatusCode, *model.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	var cloudflareResponse Response
	err = json.Unmarshal(responseBody, &cloudflareResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	fullTextResponse := ResponseCloudflare2OpenAI(&cloudflareResponse)
	fullTextResponse.Model = modelName
	usage := openai.ResponseText2Usage(cloudflareResponse.Result.Response, modelName, promptTokens)
	fullTextResponse.Usage = *usage
	fullTextResponse.Id = helper.GetResponseID(c)
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError), nil
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = c.Writer.Write(jsonResponse)
	return nil, usage
}
