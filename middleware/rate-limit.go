package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
)

var inMemoryRateLimiter common.InMemoryRateLimiter

var luaScript = `
local key = KEYS[1]
local max_requests = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local current_time = tonumber(ARGV[3])

local count = redis.call('LLEN', key)

if count < max_requests then
    redis.call('LPUSH', key, current_time)
    redis.call('EXPIRE', key, window)
    return 1
else
    local oldest = redis.call('LINDEX', key, -1)
    if current_time - tonumber(oldest) >= window then
        redis.call('LPUSH', key, current_time)
        redis.call('LTRIM', key, 0, max_requests - 1)
        redis.call('EXPIRE', key, window)
        return 1
    else
        return 0
    end
end
`

func redisRateLimitRequest(ctx context.Context, key string, maxRequestNum int, duration time.Duration) (bool, error) {
	rdb := common.RDB
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)
	result, err := rdb.Eval(ctx, luaScript, []string{key}, maxRequestNum, int64(duration/time.Millisecond), currentTime).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func RateLimit(ctx context.Context, key string, maxRequestNum int, duration time.Duration) (bool, error) {
	if maxRequestNum == 0 {
		return true, nil
	}
	if common.RedisEnabled {
		return redisRateLimitRequest(ctx, key, maxRequestNum, duration)
	} else {
		// It's safe to call multi times.
		inMemoryRateLimiter.Init(config.RateLimitKeyExpirationDuration)
		return inMemoryRateLimiter.Request(key, maxRequestNum, duration), nil
	}
}

func rateLimitFactory(maxRequestNum int, duration time.Duration) func(c *gin.Context) {
	return func(c *gin.Context) {
		ok, err := RateLimit(c.Request.Context(), "ip"+c.ClientIP(), maxRequestNum, duration)
		if err != nil {
			fmt.Println(err.Error())
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
		if !ok {
			c.Status(http.StatusTooManyRequests)
			c.Abort()
		}
		c.Next()
	}
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	return rateLimitFactory(config.GlobalApiRateLimitNum, time.Minute)
}
