package billing

import (
	"context"

	"github.com/songquanpeng/one-api/common/balance"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

func PostConsumeAmount(ctx context.Context, postGroupConsumer balance.PostGroupConsumer, code int, tokenId int, amount float64, group string, channelId int, modelPrice float64, modelName string, tokenName string, endpoint string, content string) {
	if amount > 0 {
		// amountDelta is remaining amount to be consumed
		_amount, err := postGroupConsumer.PostGroupConsume(ctx, tokenName, amount)
		if err != nil {
			logger.SysError("error consuming token remain quota: " + err.Error())
		} else {
			amount = _amount
		}
	}
	err := model.BatchRecordConsume(ctx, group, code, channelId, 0, 0, modelName, tokenId, tokenName, amount, modelPrice, 0, endpoint, content)
	if err != nil {
		logger.Error(ctx, "error batch record consume: "+err.Error())
	}
}
