package billing

import (
	"context"
	"fmt"

	"github.com/songquanpeng/one-api/common/balance"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

func ReturnPreConsumedAmount(ctx context.Context, preConsumedAmount float64, group string) {
	if preConsumedAmount != 0 {
		go func(ctx context.Context) {
			// return pre-consumed amount
			err := balance.Default.PostGroupConsume(group, preConsumedAmount)
			if err != nil {
				logger.Error(ctx, "error return pre-consumed amount: "+err.Error())
			}
		}(ctx)
	}
}

func PostConsumeAmount(ctx context.Context, tokenId int, amountDelta float64, totalAmount float64, group string, channelId int, modelPrice float64, modelName string, tokenName string) {
	// amountDelta is remaining amount to be consumed
	err := balance.Default.PostGroupConsume(group, amountDelta)
	if err != nil {
		logger.SysError("error consuming token remain quota: " + err.Error())
	}
	// totalAmount is total amount consumed
	if totalAmount != 0 {
		logContent := fmt.Sprintf("模型价格 %.6f", modelPrice)
		model.RecordConsumeLog(ctx, group, channelId, int(totalAmount), 0, modelName, tokenName, totalAmount, modelPrice, 0, logContent)
		model.UpdateGroupUsedAmountAndRequestCount(group, totalAmount, 1)
		model.UpdateTokenUsedAmount(tokenId, totalAmount, 1)
		model.UpdateChannelUsedAmount(channelId, totalAmount, 1)
	}
	if totalAmount <= 0 {
		logger.Error(ctx, fmt.Sprintf("totalAmount consumed is %d, something is wrong", totalAmount))
	}
}
