package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	quotaIf "github.com/songquanpeng/one-api/common/quota"
	"github.com/songquanpeng/one-api/common/random"
)

const (
	SyncFrequency = time.Minute * 10
	TokenCacheKey = "token:%s"
)

func CacheGetTokenByKey(key string) (*Token, error) {
	var token Token

	if !common.RedisEnabled {
		keyCol := "`key`"
		if common.UsingPostgreSQL {
			keyCol = `"key"`
		}
		return &token, DB.Where(keyCol+" = ?", key).First(&token).Error
	}

	cacheKey := fmt.Sprintf(TokenCacheKey, key)
	tokenObjectString, err := common.RedisGet(cacheKey)
	if err == nil {
		return &token, json.Unmarshal([]byte(tokenObjectString), &token)
	}

	// Cache miss, fetch from database
	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	if err := DB.Where(keyCol+" = ?", key).First(&token).Error; err != nil {
		return nil, err
	}

	// Update cache
	if jsonBytes, err := json.Marshal(token); err == nil {
		if err := common.RedisSet(cacheKey, string(jsonBytes), SyncFrequency); err != nil {
			logger.SysError("Redis set token error: " + err.Error())
		}
	}

	return &token, nil
}

func fetchAndUpdateGroupQuota(ctx context.Context, id string) (quota int64, err error) {
	err = common.RedisSet(fmt.Sprintf("group_quota:%s", id), fmt.Sprintf("%d", quota), SyncFrequency)
	if err != nil {
		logger.Error(ctx, "Redis set group quota error: "+err.Error())
	}
	return
}

func CacheGetGroupQuota(ctx context.Context, id string) (quota int64, err error) {
	if !common.RedisEnabled {
		return quotaIf.DefaultMockGroupQuota.GetGroupQuota(id)
	}
	quotaString, err := common.RedisGet(fmt.Sprintf("group_quota:%s", id))
	if err != nil {
		return fetchAndUpdateGroupQuota(ctx, id)
	}
	quota, err = strconv.ParseInt(quotaString, 10, 64)
	if err != nil {
		return 0, nil
	}
	if quota <= config.PreConsumedQuota { // when user's quota is less than pre-consumed quota, we need to fetch from db
		logger.Infof(ctx, "group %s's cached quota is too low: %d, refreshing from db", id, quota)
		return fetchAndUpdateGroupQuota(ctx, id)
	}
	return quota, nil
}

func CacheUpdateGroupQuota(ctx context.Context, id string) error {
	if !common.RedisEnabled {
		return nil
	}
	quota, err := CacheGetGroupQuota(ctx, id)
	if err != nil {
		return err
	}
	err = common.RedisSet(fmt.Sprintf("group_quota:%s", id), fmt.Sprintf("%d", quota), SyncFrequency)
	return err
}

func CacheDecreaseGroupQuota(id string, quota int64) error {
	if !common.RedisEnabled {
		return nil
	}
	err := common.RedisDecrease(fmt.Sprintf("group_quota:%s", id), int64(quota))
	return err
}

func CacheIsGroupEnabled(ctx context.Context, id string) (bool, error) {
	if !common.RedisEnabled {
		return IsGroupEnabled(id)
	}
	enabled, err := common.RedisGet(fmt.Sprintf("group_enabled:%s", id))
	if err == nil {
		return enabled == "1", nil
	}

	userEnabled, err := IsGroupEnabled(id)
	if err != nil {
		return false, err
	}
	enabled = "0"
	if userEnabled {
		enabled = "1"
	}
	err = common.RedisSet(fmt.Sprintf("group_enabled:%s", id), enabled, SyncFrequency)
	if err != nil {
		logger.SysError("Redis set group enabled error: " + err.Error())
	}
	return userEnabled, err
}

var (
	model2channels  map[string][]*Channel
	allModels       []string
	type2Models     map[int][]string
	channelSyncLock sync.RWMutex
)

func CacheGetAllModels() []string {
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	return allModels
}

func CacheGetType2Models() map[int][]string {
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	return type2Models
}

func CacheGetModelsByType(channelType int) []string {
	return CacheGetType2Models()[channelType]
}

func InitChannelCache() {
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Where("status = ?", ChannelStatusEnabled).Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}
	newModel2channels := make(map[string][]*Channel)
	for _, channel := range channels {
		for _, model := range channel.Models {
			newModel2channels[model] = append(newModel2channels[model], channel)
		}
	}

	// sort by priority
	for _, channels := range newModel2channels {
		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Priority > channels[j].Priority
		})
	}

	models := make([]string, 0, len(newModel2channels))
	for model := range newModel2channels {
		models = append(models, model)
	}

	newType2ModelsMap := make(map[int]map[string]struct{})
	for _, channel := range channels {
		newType2ModelsMap[channel.Type] = make(map[string]struct{})
		for _, model := range channel.Models {
			newType2ModelsMap[channel.Type][model] = struct{}{}
		}
	}
	newType2Models := make(map[int][]string)
	for k, v := range newType2ModelsMap {
		newType2Models[k] = make([]string, 0, len(v))
		for model := range v {
			newType2Models[k] = append(newType2Models[k], model)
		}
	}

	channelSyncLock.Lock()
	model2channels = newModel2channels
	allModels = models
	type2Models = newType2Models
	channelSyncLock.Unlock()
	logger.SysLog("channels synced from database")
}

func SyncChannelCache(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for range ticker.C {
		logger.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func CacheGetRandomSatisfiedChannel(model string, ignoreFirstPriority bool) (*Channel, error) {
	channelSyncLock.RLock()
	channels := model2channels[model]
	channelSyncLock.RUnlock()
	if len(channels) == 0 {
		return nil, errors.New("channel not found")
	}
	endIdx := len(channels)
	// choose by priority
	firstChannel := channels[0]
	if firstChannel.Priority > 0 {
		for i := range channels {
			if channels[i].Priority != firstChannel.Priority {
				endIdx = i
				break
			}
		}
	}
	idx := rand.Intn(endIdx)
	if ignoreFirstPriority {
		if endIdx < len(channels) { // which means there are more than one priority
			idx = random.RandRange(endIdx, len(channels))
		}
	}
	return channels[idx], nil
}
