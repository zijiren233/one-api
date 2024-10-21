package common

import (
	"fmt"

	"github.com/songquanpeng/one-api/common/config"
)

func LogQuota(quota int64) string {
	return fmt.Sprintf("＄%.6f 额度", float64(quota)/config.QuotaPerUnit)
}
