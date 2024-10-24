package config

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/songquanpeng/one-api/common/env"
)

var (
	OptionMap        map[string]string
	OptionMapRWMutex sync.RWMutex
)

var (
	DebugEnabled    = strings.ToLower(os.Getenv("DEBUG")) == "true"
	DebugSQLEnabled = strings.ToLower(os.Getenv("DEBUG_SQL")) == "true"
)

var (
	// 当测试或请求的时候发生错误是否自动禁用渠道
	automaticDisableChannelEnabled uint32 = 0
	// 当测试成功是否自动启用渠道
	automaticEnableChannelWhenTestSucceedEnabled uint32 = 0
	// 是否近似计算token
	approximateTokenEnabled uint32 = 0
	// 重试次数
	retryTimes int64 = 0
)

func GetAutomaticDisableChannelEnabled() bool {
	return atomic.LoadUint32(&automaticDisableChannelEnabled) == 1
}

func SetAutomaticDisableChannelEnabled(enabled bool) {
	if enabled {
		atomic.StoreUint32(&automaticDisableChannelEnabled, 1)
	} else {
		atomic.StoreUint32(&automaticDisableChannelEnabled, 0)
	}
}

func GetAutomaticEnableChannelWhenTestSucceedEnabled() bool {
	return atomic.LoadUint32(&automaticEnableChannelWhenTestSucceedEnabled) == 1
}

func SetAutomaticEnableChannelWhenTestSucceedEnabled(enabled bool) {
	if enabled {
		atomic.StoreUint32(&automaticEnableChannelWhenTestSucceedEnabled, 1)
	} else {
		atomic.StoreUint32(&automaticEnableChannelWhenTestSucceedEnabled, 0)
	}
}

func GetApproximateTokenEnabled() bool {
	return atomic.LoadUint32(&approximateTokenEnabled) == 1
}

func SetApproximateTokenEnabled(enabled bool) {
	if enabled {
		atomic.StoreUint32(&approximateTokenEnabled, 1)
	} else {
		atomic.StoreUint32(&approximateTokenEnabled, 0)
	}
}

func GetRetryTimes() int64 {
	return atomic.LoadInt64(&retryTimes)
}

func SetRetryTimes(times int64) {
	atomic.StoreInt64(&retryTimes, times)
}

var DisableAutoMigrateDB = os.Getenv("DISABLE_AUTO_MIGRATE_DB") == "true"

var RelayTimeout = env.Int("RELAY_TIMEOUT", 0) // unit is second

var RateLimitKeyExpirationDuration = 20 * time.Minute

var (
	// 是否根据请求成功率禁用渠道，默认不开启
	EnableMetric = env.Bool("ENABLE_METRIC", false)
	// 指标队列大小
	MetricQueueSize = env.Int("METRIC_QUEUE_SIZE", 10)
	// 请求成功率阈值，默认80%
	MetricSuccessRateThreshold = env.Float64("METRIC_SUCCESS_RATE_THRESHOLD", 0.8)
	// 请求成功率指标队列大小
	MetricSuccessChanSize = env.Int("METRIC_SUCCESS_CHAN_SIZE", 1024)
	// 请求失败率指标队列大小
	MetricFailChanSize = env.Int("METRIC_FAIL_CHAN_SIZE", 128)
)

var OnlyOneLogFile = env.Bool("ONLY_ONE_LOG_FILE", false)

var (
	// 代理地址
	RelayProxy = env.String("RELAY_PROXY", "")
	// 用户内容请求代理地址
	UserContentRequestProxy = env.String("USER_CONTENT_REQUEST_PROXY", "")
	// 用户内容请求超时时间，单位为秒
	UserContentRequestTimeout = env.Int("USER_CONTENT_REQUEST_TIMEOUT", 30)
)

var AdminKey = env.String("ADMIN_KEY", "")

var (
	globalApiRateLimitNum      int64 = 0
	defaultChannelModels       atomic.Value
	defaultChannelModelMapping atomic.Value
	defaultGroupQPM            int64 = 120
	groupMaxTokenNum           int32 = 0
)

func init() {
	defaultChannelModels.Store(make(map[int][]string))
	defaultChannelModelMapping.Store(make(map[int]map[string]string))
}

func GetGlobalApiRateLimitNum() int64 {
	return atomic.LoadInt64(&globalApiRateLimitNum)
}

func SetGlobalApiRateLimitNum(num int64) {
	atomic.StoreInt64(&globalApiRateLimitNum, num)
}

func GetDefaultGroupQPM() int64 {
	return atomic.LoadInt64(&defaultGroupQPM)
}

func SetDefaultGroupQPM(qpm int64) {
	atomic.StoreInt64(&defaultGroupQPM, qpm)
}

func GetDefaultChannelModels() map[int][]string {
	return defaultChannelModels.Load().(map[int][]string)
}

func SetDefaultChannelModels(models map[int][]string) {
	defaultChannelModels.Store(models)
}

func GetDefaultChannelModelMapping() map[int]map[string]string {
	return defaultChannelModelMapping.Load().(map[int]map[string]string)
}

func SetDefaultChannelModelMapping(mapping map[int]map[string]string) {
	defaultChannelModelMapping.Store(mapping)
}

func GetGroupMaxTokenNum() int32 {
	return atomic.LoadInt32(&groupMaxTokenNum)
}

func SetGroupMaxTokenNum(num int32) {
	atomic.StoreInt32(&groupMaxTokenNum, num)
}

var (
	geminiSafetySetting atomic.Value
	geminiVersion       atomic.Value
)

func init() {
	geminiSafetySetting.Store("BLOCK_NONE")
	geminiVersion.Store("v1")
}

func GetGeminiSafetySetting() string {
	return geminiSafetySetting.Load().(string)
}

func SetGeminiSafetySetting(setting string) {
	geminiSafetySetting.Store(setting)
}

func GetGeminiVersion() string {
	return geminiVersion.Load().(string)
}

func SetGeminiVersion(version string) {
	geminiVersion.Store(version)
}
