package controller

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/balance"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/conv"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	billingprice "github.com/songquanpeng/one-api/relay/price"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func RelayAudioHelper(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	if c.Request.ContentLength <= 0 {
		return openai.ErrorWrapper(errors.New("request body is empty"), "request_body_empty", http.StatusBadRequest)
	}

	meta := meta.GetByContext(c)

	channelType := c.GetInt(ctxkey.Channel)
	group := c.GetString(ctxkey.Group)

	var ttsRequest openai.TextToSpeechRequest
	switch relayMode {
	case relaymode.AudioSpeech:
		// Read JSON
		err := common.UnmarshalBodyReusable(c, &ttsRequest)
		// Check if JSON is valid
		if err != nil {
			return openai.ErrorWrapper(err, "invalid_json", http.StatusBadRequest)
		}
		meta.OriginModelName = ttsRequest.Model
		// Check if text is too long 4096
		if len(ttsRequest.Input) > 4096 {
			return openai.ErrorWrapper(errors.New("input is too long (over 4096 characters)"), "text_too_long", http.StatusBadRequest)
		}
	default:
		meta.OriginModelName = "whisper-1"
	}

	// map model name
	meta.ActualModelName, _ = getMappedModelName(meta.OriginModelName, c.GetStringMapString(ctxkey.ModelMapping))

	price, ok := billingprice.GetModelPrice(meta.OriginModelName, meta.ActualModelName, channelType)
	if !ok {
		return openai.ErrorWrapper(fmt.Errorf("model price not found: %s", meta.OriginModelName), "model_price_not_found", http.StatusInternalServerError)
	}
	completionPrice, ok := billingprice.GetModelPrice(meta.OriginModelName, meta.ActualModelName, channelType)
	if !ok {
		return openai.ErrorWrapper(fmt.Errorf("model price not found: %s", meta.OriginModelName), "model_price_not_found", http.StatusInternalServerError)
	}

	groupRemainBalance, postGroupConsumer, err := balance.Default.GetGroupRemainBalance(c.Request.Context(), group)
	if err != nil {
		return openai.ErrorWrapper(err, "get_group_balance_failed", http.StatusInternalServerError)
	}

	var preConsumedAmount float64
	var promptTokens int
	switch relayMode {
	case relaymode.AudioSpeech:
		promptTokens = openai.CountTokenText(ttsRequest.Input, meta.ActualModelName)
		preConsumedAmount = decimal.NewFromInt(int64(promptTokens)).
			Mul(decimal.NewFromFloat(price)).
			Div(decimal.NewFromInt(billingprice.PriceUnit)).
			InexactFloat64()
	}

	// Check if group balance is enough
	if groupRemainBalance < preConsumedAmount {
		return openai.ErrorWrapper(errors.New("group balance is not enough"), "insufficient_group_balance", http.StatusForbidden)
	}

	baseURL := channeltype.ChannelBaseURLs[channelType]
	requestURL := c.Request.URL.String()
	if c.GetString(ctxkey.BaseURL) != "" {
		baseURL = c.GetString(ctxkey.BaseURL)
	}

	fullRequestURL := openai.GetFullRequestURL(baseURL, requestURL, channelType)
	if channelType == channeltype.Azure {
		apiVersion := meta.Config.APIVersion
		switch relayMode {
		case relaymode.AudioTranscription:
			// https://learn.microsoft.com/en-us/azure/ai-services/openai/whisper-quickstart?tabs=command-line#rest-api
			fullRequestURL = fmt.Sprintf("%s/openai/deployments/%s/audio/transcriptions?api-version=%s", baseURL, meta.ActualModelName, apiVersion)
		case relaymode.AudioSpeech:
			// https://learn.microsoft.com/en-us/azure/ai-services/openai/text-to-speech-quickstart?tabs=command-line#rest-api
			fullRequestURL = fmt.Sprintf("%s/openai/deployments/%s/audio/speech?api-version=%s", baseURL, meta.ActualModelName, apiVersion)
		}
	}

	buf := make([]byte, c.Request.ContentLength)
	_, err = io.ReadFull(c.Request.Body, buf)
	if err != nil {
		return openai.ErrorWrapper(err, "new_request_body_failed", http.StatusInternalServerError)
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(buf))
	responseFormat := c.DefaultPostForm("response_format", "json")

	req, err := http.NewRequest(c.Request.Method, fullRequestURL, bytes.NewReader(buf))
	if err != nil {
		return openai.ErrorWrapper(err, "new_request_failed", http.StatusInternalServerError)
	}

	if (relayMode == relaymode.AudioTranscription || relayMode == relaymode.AudioSpeech) && channelType == channeltype.Azure {
		// https://learn.microsoft.com/en-us/azure/ai-services/openai/whisper-quickstart?tabs=command-line#rest-api
		apiKey := c.Request.Header.Get("Authorization")
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		req.Header.Set("api-key", apiKey)
		req.ContentLength = c.Request.ContentLength
	} else {
		req.Header.Set("Authorization", c.Request.Header.Get("Authorization"))
	}
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}

	var completionTokens int
	switch relayMode {
	case relaymode.AudioSpeech:
		defer resp.Body.Close()
	default:
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		}
		resp.Body.Close()

		resp.Body = io.NopCloser(bytes.NewReader(responseBody))

		var openAIErr openai.SlimTextResponse
		if err = json.Unmarshal(responseBody, &openAIErr); err == nil {
			if openAIErr.Error.Message != "" {
				return openai.ErrorWrapper(fmt.Errorf("type %s, code %v, message %s", openAIErr.Error.Type, openAIErr.Error.Code, openAIErr.Error.Message), "request_error", http.StatusInternalServerError)
			}
		}

		var text string
		switch responseFormat {
		case "json":
			text, err = getTextFromJSON(responseBody)
		case "text":
			text, err = getTextFromText(responseBody)
		case "srt":
			text, err = getTextFromSRT(responseBody)
		case "verbose_json":
			text, err = getTextFromVerboseJSON(responseBody)
		case "vtt":
			text, err = getTextFromVTT(responseBody)
		default:
			return openai.ErrorWrapper(errors.New("unexpected_response_format"), "unexpected_response_format", http.StatusInternalServerError)
		}
		if err != nil {
			return openai.ErrorWrapper(err, "get_text_from_body_err", http.StatusInternalServerError)
		}
		completionTokens = openai.CountTokenText(text, meta.ActualModelName)
	}

	if resp.StatusCode != http.StatusOK {
		err := RelayErrorHandler(resp)
		go postConsumeAmount(context.Background(), postGroupConsumer, resp.StatusCode, c.Request.URL.Path, &relaymodel.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
		}, meta, price, completionPrice, err.Message)
		return err
	}

	go postConsumeAmount(context.Background(), postGroupConsumer, resp.StatusCode, c.Request.URL.Path, &relaymodel.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}, meta, price, completionPrice, "")

	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
	}
	return nil
}

func getTextFromVTT(body []byte) (string, error) {
	return getTextFromSRT(body)
}

func getTextFromVerboseJSON(body []byte) (string, error) {
	var whisperResponse openai.WhisperVerboseJSONResponse
	if err := json.Unmarshal(body, &whisperResponse); err != nil {
		return "", fmt.Errorf("unmarshal_response_body_failed err :%w", err)
	}
	return whisperResponse.Text, nil
}

func getTextFromSRT(body []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	var builder strings.Builder
	var textLine bool
	for scanner.Scan() {
		line := scanner.Text()
		if textLine {
			builder.WriteString(line)
			textLine = false
			continue
		} else if strings.Contains(line, "-->") {
			textLine = true
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func getTextFromText(body []byte) (string, error) {
	return strings.TrimSuffix(conv.BytesToString(body), "\n"), nil
}

func getTextFromJSON(body []byte) (string, error) {
	var whisperResponse openai.WhisperJSONResponse
	if err := json.Unmarshal(body, &whisperResponse); err != nil {
		return "", fmt.Errorf("unmarshal_response_body_failed err :%w", err)
	}
	return whisperResponse.Text, nil
}
