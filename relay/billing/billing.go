package billing

import (
	"context"
	"fmt"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/quota"
	"github.com/songquanpeng/one-api/model"
)

func ReturnPreConsumedQuota(ctx context.Context, preConsumedQuota int64, group string) {
	if preConsumedQuota != 0 {
		go func(ctx context.Context) {
			// return pre-consumed quota
			err := quota.Default.PostGroupConsume(group, -preConsumedQuota)
			if err != nil {
				logger.Error(ctx, "error return pre-consumed quota: "+err.Error())
			}
		}(ctx)
	}
}

func PostConsumeQuota(ctx context.Context, tokenId int, quotaDelta int64, totalQuota int64, group string, channelId int, modelRatio float64, modelName string, tokenName string) {
	// quotaDelta is remaining quota to be consumed
	err := quota.Default.PostGroupConsume(group, quotaDelta)
	if err != nil {
		logger.SysError("error consuming token remain quota: " + err.Error())
	}
	// totalQuota is total quota consumed
	if totalQuota != 0 {
		logContent := fmt.Sprintf("模型倍率 %.2f", modelRatio)
		model.RecordConsumeLog(ctx, group, channelId, int(totalQuota), 0, modelName, tokenName, totalQuota, logContent)
		model.UpdateGroupUsedQuotaAndRequestCount(group, totalQuota, 1)
		model.UpdateTokenUsedQuota(tokenId, totalQuota, 1)
		model.UpdateChannelUsedQuota(channelId, totalQuota, 1)
	}
	if totalQuota <= 0 {
		logger.Error(ctx, fmt.Sprintf("totalQuota consumed is %d, something is wrong", totalQuota))
	}
}
