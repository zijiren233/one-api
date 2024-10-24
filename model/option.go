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
	config.OptionMap["AutomaticDisableChannelEnabled"] = strconv.FormatBool(config.AutomaticDisableChannelEnabled)
	config.OptionMap["AutomaticEnableChannelEnabled"] = strconv.FormatBool(config.AutomaticEnableChannelEnabled)
	config.OptionMap["ApproximateTokenEnabled"] = strconv.FormatBool(config.ApproximateTokenEnabled)
	config.OptionMap["ChannelDisableThreshold"] = strconv.FormatFloat(config.ChannelDisableThreshold, 'f', -1, 64)
	config.OptionMap["ModelPrice"] = billingprice.ModelPrice2JSONString()
	config.OptionMap["CompletionPrice"] = billingprice.CompletionPrice2JSONString()
	config.OptionMap["RetryTimes"] = strconv.Itoa(config.RetryTimes)
	config.OptionMap["GlobalApiRateLimitNum"] = strconv.Itoa(config.GlobalApiRateLimitNum)
	config.OptionMap["DefaultGroupQPM"] = strconv.Itoa(config.DefaultGroupQPM)
	defaultChannelModelsJSON, _ := json.Marshal(config.DefaultChannelModels)
	config.OptionMap["DefaultChannelModels"] = common.BytesToString(defaultChannelModelsJSON)
	defaultChannelModelMappingJSON, _ := json.Marshal(config.DefaultChannelModelMapping)
	config.OptionMap["DefaultChannelModelMapping"] = common.BytesToString(defaultChannelModelMappingJSON)
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
			config.AutomaticDisableChannelEnabled = boolValue
		case "AutomaticEnableChannelEnabled":
			config.AutomaticEnableChannelEnabled = boolValue
		case "ApproximateTokenEnabled":
			config.ApproximateTokenEnabled = boolValue
		}
	}
	switch key {
	case "GlobalApiRateLimitNum":
		config.GlobalApiRateLimitNum, _ = strconv.Atoi(value)
	case "DefaultGroupQPM":
		config.DefaultGroupQPM, _ = strconv.Atoi(value)
	case "DefaultChannelModels":
		var newModules map[int][]string
		err := json.Unmarshal(common.StringToBytes(value), &newModules)
		if err != nil {
			return err
		}
		config.DefaultChannelModels = newModules
	case "DefaultChannelModelMapping":
		var newMapping map[int]map[string]string
		err := json.Unmarshal(common.StringToBytes(value), &newMapping)
		if err != nil {
			return err
		}
		config.DefaultChannelModelMapping = newMapping
	case "RetryTimes":
		config.RetryTimes, _ = strconv.Atoi(value)
	case "ModelPrice":
		err = billingprice.UpdateModelPriceByJSONString(value)
	case "CompletionPrice":
		err = billingprice.UpdateCompletionPriceByJSONString(value)
	case "ChannelDisableThreshold":
		config.ChannelDisableThreshold, _ = strconv.ParseFloat(value, 64)
	}
	return err
}
