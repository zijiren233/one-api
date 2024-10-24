package model

import (
	"strconv"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	billingprice "github.com/songquanpeng/one-api/relay/billing/price"
)

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

func AllOption() ([]*Option, error) {
	var options []*Option
	err := DB.Find(&options).Error
	return options, err
}

func InitOptionMap() {
	config.OptionMapRWMutex.Lock()
	config.OptionMap = make(map[string]string)
	config.OptionMap["AutomaticDisableChannelEnabled"] = strconv.FormatBool(config.GetAutomaticDisableChannelEnabled())
	config.OptionMap["AutomaticEnableChannelWhenTestSucceedEnabled"] = strconv.FormatBool(config.GetAutomaticEnableChannelWhenTestSucceedEnabled())
	config.OptionMap["ApproximateTokenEnabled"] = strconv.FormatBool(config.GetApproximateTokenEnabled())
	config.OptionMap["ModelPrice"] = billingprice.ModelPrice2JSONString()
	config.OptionMap["CompletionPrice"] = billingprice.CompletionPrice2JSONString()
	config.OptionMap["RetryTimes"] = strconv.FormatInt(config.GetRetryTimes(), 10)
	config.OptionMap["GlobalApiRateLimitNum"] = strconv.FormatInt(config.GetGlobalApiRateLimitNum(), 10)
	config.OptionMap["DefaultGroupQPM"] = strconv.FormatInt(config.GetDefaultGroupQPM(), 10)
	defaultChannelModelsJSON, _ := json.Marshal(config.GetDefaultChannelModels())
	config.OptionMap["DefaultChannelModels"] = common.BytesToString(defaultChannelModelsJSON)
	defaultChannelModelMappingJSON, _ := json.Marshal(config.GetDefaultChannelModelMapping())
	config.OptionMap["DefaultChannelModelMapping"] = common.BytesToString(defaultChannelModelMappingJSON)
	config.OptionMap["GeminiSafetySetting"] = config.GetGeminiSafetySetting()
	config.OptionMap["GeminiVersion"] = config.GetGeminiVersion()
	config.OptionMapRWMutex.Unlock()
	loadOptionsFromDatabase()
}

func loadOptionsFromDatabase() {
	options, _ := AllOption()
	for _, option := range options {
		if option.Key == "ModelPrice" {
			option.Value = billingprice.AddNewMissingPrice(option.Value)
		}
		err := updateOptionMap(option.Key, option.Value)
		if err != nil {
			logger.SysError("failed to update option map: " + err.Error())
		}
	}
	logger.SysLog("options synced from database")
}

func SyncOptions(frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for range ticker.C {
		logger.SysLog("syncing options from database")
		loadOptionsFromDatabase()
	}
}

func UpdateOption(key string, value string) error {
	// Save to database first
	option := Option{
		Key: key,
	}
	// https://gorm.io/docs/update.html#Save-All-Fields
	err := DB.Assign(Option{Key: key, Value: value}).FirstOrCreate(&option).Error
	if err != nil {
		return err
	}
	// Update OptionMap
	return updateOptionMap(key, value)
}

func updateOptionMap(key string, value string) (err error) {
	config.OptionMapRWMutex.Lock()
	defer config.OptionMapRWMutex.Unlock()
	config.OptionMap[key] = value
	if strings.HasSuffix(key, "Enabled") {
		boolValue := value == "true"
		switch key {
		case "AutomaticDisableChannelEnabled":
			config.SetAutomaticDisableChannelEnabled(boolValue)
		case "AutomaticEnableChannelWhenTestSucceedEnabled":
			config.SetAutomaticEnableChannelWhenTestSucceedEnabled(boolValue)
		case "ApproximateTokenEnabled":
			config.SetApproximateTokenEnabled(boolValue)
		}
	}
	switch key {
	case "GeminiSafetySetting":
		config.SetGeminiSafetySetting(value)
	case "GeminiVersion":
		config.SetGeminiVersion(value)
	case "GlobalApiRateLimitNum":
		globalApiRateLimitNum, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		config.SetGlobalApiRateLimitNum(globalApiRateLimitNum)
	case "DefaultGroupQPM":
		defaultGroupQPM, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		config.SetDefaultGroupQPM(defaultGroupQPM)
	case "DefaultChannelModels":
		var newModules map[int][]string
		err := json.Unmarshal(common.StringToBytes(value), &newModules)
		if err != nil {
			return err
		}
		config.SetDefaultChannelModels(newModules)
	case "DefaultChannelModelMapping":
		var newMapping map[int]map[string]string
		err := json.Unmarshal(common.StringToBytes(value), &newMapping)
		if err != nil {
			return err
		}
		config.SetDefaultChannelModelMapping(newMapping)
	case "RetryTimes":
		retryTimes, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return err
		}
		config.SetRetryTimes(retryTimes)
	case "ModelPrice":
		err = billingprice.UpdateModelPriceByJSONString(value)
	case "CompletionPrice":
		err = billingprice.UpdateCompletionPriceByJSONString(value)
	}
	return err
}
