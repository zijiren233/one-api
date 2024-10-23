package billing

import (
	"context"

	"github.com/songquanpeng/one-api/common/balance"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

func PostConsumeAmount(ctx context.Context, tokenId int, amount float64, group string, channelId int, modelPrice float64, modelName string, tokenName string) {
	// amountDelta is remaining amount to be consumed
	err := balance.Default.PostGroupConsume(ctx, group, amount)
	if err != nil {
		logger.SysError("error consuming token remain quota: " + err.Error())
	}
	// totalAmount is total amount consumed
	model.RecordConsumeLog(ctx, group, channelId, int(amount), 0, modelName, tokenName, amount, modelPrice, 0, "")
	model.UpdateGroupUsedAmountAndRequestCount(group, amount, 1)
	model.UpdateTokenUsedAmount(tokenId, amount, 1)
	model.UpdateChannelUsedAmount(channelId, amount, 1)
}
