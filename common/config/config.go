package config

import (
	"os"
	"strings"
	"sync"
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
	// 渠道超时时的禁用阈值
	ChannelDisableThreshold        = 5.0
	AutomaticDisableChannelEnabled = false
	AutomaticEnableChannelEnabled  = false
	ApproximateTokenEnabled        = false
	RetryTimes                     = 0
)

var DisableAutoMigrateDB = os.Getenv("DISABLE_AUTO_MIGRATE_DB") == "true"

var RelayTimeout = env.Int("RELAY_TIMEOUT", 0) // unit is second

var GeminiSafetySetting = env.String("GEMINI_SAFETY_SETTING", "BLOCK_NONE")

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitNum = env.Int("GLOBAL_API_RATE_LIMIT", 240)
)

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

var GeminiVersion = env.String("GEMINI_VERSION", "v1")

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
	DefaultChannelModels       map[int][]string
	DefaultChannelModelMapping map[int]map[string]string
)
