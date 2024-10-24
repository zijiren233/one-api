package model

import (
	"context"
	"encoding"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
)

const (
	SyncFrequency = time.Minute
	TokenCacheKey = "token:%s"
	GroupCacheKey = "group:%s"
)

var (
	_ encoding.BinaryMarshaler = (*redisStringSlice)(nil)
	_ redis.Scanner            = (*redisStringSlice)(nil)
)

type redisStringSlice []string

func (r *redisStringSlice) ScanRedis(value string) error {
	return json.Unmarshal(common.StringToBytes(value), r)
}

func (r redisStringSlice) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

type TokenCache struct {
	Id         int              `json:"id" redis:"i"`
	Group      string           `json:"group" redis:"g"`
	Key        string           `json:"-" redis:"-"`
	Remark     string           `json:"remark" redis:"r"`
	Models     redisStringSlice `json:"models" redis:"m"`
	Subnet     string           `json:"subnet" redis:"s"`
	Status     int              `json:"status" redis:"st"`
	ExpiredAt  time.Time        `json:"expired_at" redis:"e"`
	Quota      float64          `json:"quota" redis:"q"`
	UsedAmount float64          `json:"used_amount" redis:"u"`
}

func (t *Token) ToTokenCache() *TokenCache {
	return &TokenCache{
		Id:         t.Id,
		Group:      t.GroupId,
		Remark:     t.Remark.String(),
		Models:     t.Models,
		Subnet:     t.Subnet,
		Status:     t.Status,
		ExpiredAt:  t.ExpiredAt,
		Quota:      t.Quota,
		UsedAmount: t.UsedAmount,
	}
}

func CacheDeleteToken(key string) error {
	if !common.RedisEnabled {
		return nil
	}
	return common.RedisDel(fmt.Sprintf(TokenCacheKey, key))
}

func CacheSetToken(token *Token) error {
	if !common.RedisEnabled {
		return nil
	}
	key := fmt.Sprintf(TokenCacheKey, token.Key)
	pipe := common.RDB.Pipeline()
	pipe.HSet(context.Background(), key, token.ToTokenCache())
	pipe.Expire(context.Background(), key, SyncFrequency)
	_, err := pipe.Exec(context.Background())
	return err
}

func CacheGetTokenByKey(key string) (*TokenCache, error) {
	if !common.RedisEnabled {
		token, err := GetTokenByKey(key)
		if err != nil {
			return nil, err
		}
		return token.ToTokenCache(), nil
	}

	cacheKey := fmt.Sprintf(TokenCacheKey, key)
	tokenCache := &TokenCache{}
	err := common.RDB.HGetAll(context.Background(), cacheKey).Scan(tokenCache)
	if err == nil && tokenCache.Id != 0 {
		tokenCache.Key = key
		return tokenCache, nil
	}

	logger.SysLog("token not found in redis, getting from database")

	token, err := GetTokenByKey(key)
	if err != nil {
		return nil, err
	}

	if err := CacheSetToken(token); err != nil {
		logger.SysError("Redis set token error: " + err.Error())
	}

	return token.ToTokenCache(), nil
}

func CacheUpdateTokenUsedAmount(key string, amount float64) error {
	if !common.RedisEnabled {
		return nil
	}
	cacheKey := fmt.Sprintf(TokenCacheKey, key)
	pipe := common.RDB.Pipeline()
	pipe.HSet(context.Background(), cacheKey, "used_amount", amount)
	pipe.Expire(context.Background(), cacheKey, SyncFrequency)
	_, err := pipe.Exec(context.Background())
	return err
}

type GroupCache struct {
	Id     string `json:"-" redis:"-"`
	Status int    `json:"status" redis:"st"`
	QPM    int64  `json:"qpm" redis:"q"`
}

func (g *Group) ToGroupCache() *GroupCache {
	return &GroupCache{
		Id:     g.Id,
		Status: g.Status,
		QPM:    g.QPM,
	}
}

func CacheDeleteGroup(id string) error {
	if !common.RedisEnabled {
		return nil
	}
	return common.RedisDel(fmt.Sprintf(GroupCacheKey, id))
}

func CacheSetGroup(group *Group) error {
	if !common.RedisEnabled {
		return nil
	}
	key := fmt.Sprintf(GroupCacheKey, group.Id)
	pipe := common.RDB.Pipeline()
	pipe.HSet(context.Background(), key, group.ToGroupCache())
	pipe.Expire(context.Background(), key, SyncFrequency)
	_, err := pipe.Exec(context.Background())
	return err
}

func CacheGetGroup(id string) (*GroupCache, error) {
	if !common.RedisEnabled {
		return getGroupFromDB(id)
	}

	cacheKey := fmt.Sprintf(GroupCacheKey, id)
	groupCache := &GroupCache{}
	err := common.RDB.HGetAll(context.Background(), cacheKey).Scan(groupCache)
	if err == nil && groupCache.Status != 0 {
		groupCache.Id = id
		return groupCache, nil
	}

	group, err := getGroupFromDB(id)
	if err != nil {
		return nil, err
	}

	if err := CacheSetGroup(&Group{Id: id, Status: group.Status, QPM: group.QPM}); err != nil {
		logger.SysError("Redis set group error: " + err.Error())
	}

	return group, nil
}

func getGroupFromDB(id string) (*GroupCache, error) {
	group, err := GetGroupById(id)
	if err != nil {
		return nil, err
	}
	return group.ToGroupCache(), nil
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
		if len(channel.Models) == 0 {
			channel.Models = config.GetDefaultChannelModels()[channel.Type]
		}
		if len(channel.ModelMapping) == 0 {
			channel.ModelMapping = config.GetDefaultChannelModelMapping()[channel.Type]
		}
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
