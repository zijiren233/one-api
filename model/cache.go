package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	json "github.com/json-iterator/go"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
)

const (
	SyncFrequency           = time.Minute
	TokenCacheKey           = "token:%s"
	TokenUsedAmountCacheKey = "token_used_amount:%d"
	GroupCacheKey           = "group:%s"
)

type TokenCache struct {
	Id        int       `json:"id"`
	Group     string    `json:"group"`
	Key       string    `json:"-"`
	Name      string    `json:"name"`
	Models    []string  `json:"models"`
	Subnet    string    `json:"subnet"`
	Status    int       `json:"status"`
	ExpiredAt time.Time `json:"expired_at"`
	Quota     float64   `json:"quota"`
}

func (t *Token) ToTokenCache() *TokenCache {
	return &TokenCache{
		Id:        t.Id,
		Group:     t.GroupId,
		Name:      t.Name,
		Models:    t.Models,
		Subnet:    t.Subnet,
		Status:    t.Status,
		ExpiredAt: t.ExpiredAt,
		Quota:     t.Quota,
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
	jsonBytes, err := json.Marshal(token.ToTokenCache())
	if err != nil {
		return err
	}
	return common.RedisSet(fmt.Sprintf(TokenCacheKey, token.Key), common.BytesToString(jsonBytes), SyncFrequency)
}

func CacheGetTokenByKey(key string) (*TokenCache, error) {
	if !common.RedisEnabled {
		return getTokenFromDB(key)
	}

	cacheKey := fmt.Sprintf(TokenCacheKey, key)
	tokenObjectString, err := common.RedisGet(cacheKey)
	if err == nil {
		return unmarshalToken(tokenObjectString, key)
	}

	token, err := getTokenFromDB(key)
	if err != nil {
		return nil, err
	}

	if err := cacheToken(cacheKey, token); err != nil {
		logger.SysError("Redis set token error: " + err.Error())
	}

	return token, nil
}

func getTokenFromDB(key string) (*TokenCache, error) {
	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	var token Token
	if err := DB.Where(keyCol+" = ?", key).First(&token).Error; err != nil {
		return nil, err
	}
	return token.ToTokenCache(), nil
}

func unmarshalToken(tokenString, key string) (*TokenCache, error) {
	var token TokenCache
	if err := json.Unmarshal(common.StringToBytes(tokenString), &token); err != nil {
		return nil, err
	}
	token.Key = key
	return &token, nil
}

func cacheToken(cacheKey string, token *TokenCache) error {
	jsonBytes, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return common.RedisSet(cacheKey, common.BytesToString(jsonBytes), SyncFrequency)
}

func CacheGetTokenUsedAmount(id int) (float64, error) {
	if !common.RedisEnabled {
		return GetTokenUsedAmount(id)
	}
	amountString, err := common.RedisGet(fmt.Sprintf(TokenUsedAmountCacheKey, id))
	if err == nil {
		return strconv.ParseFloat(amountString, 64)
	}
	amount, err := GetTokenUsedAmount(id)
	if err != nil {
		return 0, err
	}
	if err := CacheUpdateTokenUsedAmount(id, amount); err != nil {
		logger.SysError("Redis set token used amount error: " + err.Error())
	}
	return amount, nil
}

func CacheUpdateTokenUsedAmount(id int, amount float64) error {
	if !common.RedisEnabled {
		return nil
	}
	return common.RedisSet(fmt.Sprintf(TokenUsedAmountCacheKey, id), fmt.Sprintf("%f", amount), SyncFrequency)
}

func CacheDeleteTokenUsedAmount(id int) error {
	if !common.RedisEnabled {
		return nil
	}
	return common.RedisDel(fmt.Sprintf(TokenUsedAmountCacheKey, id))
}

type GroupCache struct {
	Id     string `json:"-"`
	Status int    `json:"status"`
	QPM    int64  `json:"qpm"`
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
	jsonBytes, err := json.Marshal(group.ToGroupCache())
	if err != nil {
		return err
	}
	return common.RedisSet(fmt.Sprintf(GroupCacheKey, group.Id), common.BytesToString(jsonBytes), SyncFrequency)
}

func CacheGetGroup(id string) (*GroupCache, error) {
	if !common.RedisEnabled {
		return getGroupFromDB(id)
	}

	cacheKey := fmt.Sprintf(GroupCacheKey, id)
	groupObjectString, err := common.RedisGet(cacheKey)
	if err == nil {
		return unmarshalGroup(groupObjectString, id)
	}

	group, err := getGroupFromDB(id)
	if err != nil {
		return nil, err
	}

	if err := cacheGroup(cacheKey, group); err != nil {
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

func unmarshalGroup(groupString, id string) (*GroupCache, error) {
	var group GroupCache
	if err := json.Unmarshal(common.StringToBytes(groupString), &group); err != nil {
		return nil, err
	}
	group.Id = id
	return &group, nil
}

func cacheGroup(cacheKey string, group *GroupCache) error {
	jsonBytes, err := json.Marshal(group)
	if err != nil {
		return err
	}
	return common.RedisSet(cacheKey, common.BytesToString(jsonBytes), SyncFrequency)
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
