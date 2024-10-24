package controller

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/balance"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	billingPrice "github.com/songquanpeng/one-api/relay/billing/price"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller/validator"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func getAndValidateTextRequest(c *gin.Context, relayMode int) (*relaymodel.GeneralOpenAIRequest, error) {
	textRequest := &relaymodel.GeneralOpenAIRequest{}
	err := common.UnmarshalBodyReusable(c, textRequest)
	if err != nil {
		return nil, err
	}
	if relayMode == relaymode.Moderations && textRequest.Model == "" {
		textRequest.Model = "text-moderation-latest"
	}
	if relayMode == relaymode.Embeddings && textRequest.Model == "" {
		textRequest.Model = c.Param("model")
	}
	err = validator.ValidateTextRequest(textRequest, relayMode)
	if err != nil {
		return nil, err
	}
	return textRequest, nil
}

func getPromptTokens(textRequest *relaymodel.GeneralOpenAIRequest, relayMode int) int {
	switch relayMode {
	case relaymode.ChatCompletions:
		return openai.CountTokenMessages(textRequest.Messages, textRequest.Model)
	case relaymode.Completions:
		return openai.CountTokenInput(textRequest.Prompt, textRequest.Model)
	case relaymode.Moderations:
		return openai.CountTokenInput(textRequest.Input, textRequest.Model)
	}
	return 0
}

func getPreConsumedAmount(textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, price float64) float64 {
	preConsumedTokens := int64(promptTokens)
	if textRequest.MaxTokens != 0 {
		preConsumedTokens += int64(textRequest.MaxTokens)
	}
	return float64(preConsumedTokens) * price / billingPrice.PriceUnit
}

func preCheckGroupBalance(ctx context.Context, textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, price float64, meta *meta.Meta) (bool, *relaymodel.ErrorWithStatusCode) {
	preConsumedAmount := getPreConsumedAmount(textRequest, promptTokens, price)

	groupRemainBalance, err := balance.Default.GetGroupRemainBalance(ctx, meta.Group)
	if err != nil {
		return false, openai.ErrorWrapper(err, "get_group_quota_failed", http.StatusInternalServerError)
	}
	if groupRemainBalance < preConsumedAmount {
		return false, nil
	}
	return true, nil
}

func postConsumeAmount(ctx context.Context, usage *relaymodel.Usage, meta *meta.Meta, textRequest *relaymodel.GeneralOpenAIRequest, price float64) {
	if usage == nil {
		logger.Error(ctx, "usage is nil, which is unexpected")
		return
	}
	completionPrice := billingPrice.GetCompletionPrice(textRequest.Model, meta.ChannelType)
	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens
	amount := (float64(promptTokens) + float64(completionTokens)*completionPrice) * price / billingPrice.PriceUnit
	totalTokens := promptTokens + completionTokens
	if totalTokens == 0 {
		// in this case, must be some error happened
		// we cannot just return, because we may have to return the pre-consumed amount
		amount = 0
	}
	err := balance.Default.PostGroupConsume(ctx, meta.Group, amount)
	if err != nil {
		logger.Error(ctx, "error consuming token remain amount: "+err.Error())
	}
	model.RecordConsumeLog(ctx, meta.Group, meta.ChannelId, promptTokens, completionTokens, textRequest.Model, meta.TokenRemark, amount, price, completionPrice, "")
	model.UpdateGroupUsedAmountAndRequestCount(meta.Group, amount, 1)
	model.UpdateTokenUsedAmount(meta.TokenId, amount, 1)
	model.UpdateChannelUsedAmount(meta.ChannelId, amount, 1)
}

func getMappedModelName(modelName string, mapping map[string]string) (string, bool) {
	if mapping == nil {
		return modelName, false
	}
	mappedModelName := mapping[modelName]
	if mappedModelName != "" {
		return mappedModelName, true
	}
	return modelName, false
}

func isErrorHappened(meta *meta.Meta, resp *http.Response) bool {
	if resp == nil {
		if meta.ChannelType == channeltype.AwsClaude {
			return false
		}
		return true
	}
	if resp.StatusCode != http.StatusOK {
		return true
	}
	if meta.ChannelType == channeltype.DeepL {
		// skip stream check for deepl
		return false
	}
	if meta.IsStream && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return true
	}
	return false
}
