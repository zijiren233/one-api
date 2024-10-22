package model

import (
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ErrChannelNotFound = "channel"
)

const (
	ChannelStatusUnknown          = 0
	ChannelStatusEnabled          = 1 // don't use 0, 0 is the default value!
	ChannelStatusManuallyDisabled = 2 // also don't use 0
	ChannelStatusAutoDisabled     = 3
)

type Channel struct {
	Id               int               `json:"id"`
	Type             int               `json:"type" gorm:"default:0"`
	Key              string            `json:"key" gorm:"type:text"`
	Status           int               `json:"status" gorm:"default:1"`
	Name             string            `json:"name" gorm:"index"`
	CreatedAt        time.Time         `json:"created_at"`
	TestAt           time.Time         `json:"test_at"`
	ResponseDuration int64             `gorm:"bigint" json:"response_duration"` // in milliseconds
	BaseURL          string            `json:"base_url"`
	Other            string            `json:"other"`   // DEPRECATED: please save config to field Config
	Balance          float64           `json:"balance"` // in USD
	BalanceUpdatedAt time.Time         `json:"balance_updated_at"`
	Models           []string          `gorm:"serializer:json;type:text" json:"models"`
	UsedQuota        int64             `gorm:"bigint" json:"used_quota"`
	ModelMapping     map[string]string `gorm:"serializer:fastjson;type:text" json:"model_mapping"`
	Priority         int32             `json:"priority"`
	Config           ChannelConfig     `gorm:"serializer:json;type:text" json:"config"`
}

type ChannelConfig struct {
	Region            string `json:"region,omitempty"`
	SK                string `json:"sk,omitempty"`
	AK                string `json:"ak,omitempty"`
	UserID            string `json:"user_id,omitempty"`
	APIVersion        string `json:"api_version,omitempty"`
	LibraryID         string `json:"library_id,omitempty"`
	Plugin            string `json:"plugin,omitempty"`
	VertexAIProjectID string `json:"vertex_ai_project_id,omitempty"`
	VertexAIADC       string `json:"vertex_ai_adc,omitempty"`
}

func GetAllChannels(onlyDisabled bool, omitKey bool) (channels []*Channel, err error) {
	tx := DB.Model(&Channel{})
	if onlyDisabled {
		tx = tx.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled)
	}
	if omitKey {
		tx = tx.Omit("key")
	}
	err = tx.Order("id desc").Find(&channels).Error
	return channels, err
}

func GetChannels(startIdx int, num int, onlyDisabled bool, omitKey bool) (channels []*Channel, total int64, err error) {
	tx := DB.Model(&Channel{})
	if onlyDisabled {
		tx = tx.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled)
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if omitKey {
		tx = tx.Omit("key")
	}
	if total <= 0 {
		return nil, 0, nil
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&channels).Error
	return channels, total, err
}

func SearchChannels(keyword string, startIdx int, num int, onlyDisabled bool, omitKey bool) (channels []*Channel, total int64, err error) {
	tx := DB.Model(&Channel{})
	if onlyDisabled {
		tx = tx.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled)
	}
	if common.UsingPostgreSQL {
		tx = tx.Where("id = ? or name ILIKE ?", helper.String2Int(keyword), "%"+keyword+"%")
	} else {
		tx = tx.Where("id = ? or name LIKE ?", helper.String2Int(keyword), "%"+keyword+"%")
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if omitKey {
		tx = tx.Omit("key")
	}
	if total <= 0 {
		return nil, 0, nil
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&channels).Error
	return channels, total, err
}

func GetChannelById(id int, omitKey bool) (*Channel, error) {
	channel := Channel{Id: id}
	var err error
	if omitKey {
		err = DB.Omit("key").First(&channel, "id = ?", id).Error
	} else {
		err = DB.First(&channel, "id = ?", id).Error
	}
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func BatchInsertChannels(channels []*Channel) error {
	return DB.Create(&channels).Error
}

func UpdateChannel(channel *Channel) error {
	result := DB.Model(channel).Clauses(clause.Returning{}).Updates(channel)
	return HandleUpdateResult(result, ErrChannelNotFound)
}

func (channel *Channel) UpdateResponseTime(responseTime int64) {
	err := DB.Model(channel).Select("test_at", "response_duration").Updates(Channel{
		TestAt:           time.Now(),
		ResponseDuration: responseTime,
	}).Error
	if err != nil {
		logger.SysError("failed to update response time: " + err.Error())
	}
}

func (channel *Channel) UpdateBalance(balance float64) {
	err := DB.Model(channel).Select("balance_updated_at", "balance").Updates(Channel{
		BalanceUpdatedAt: time.Now(),
		Balance:          balance,
	}).Error
	if err != nil {
		logger.SysError("failed to update balance: " + err.Error())
	}
}

func DeleteChannelById(id int) error {
	result := DB.Delete(&Channel{Id: id})
	return HandleUpdateResult(result, ErrChannelNotFound)
}

func UpdateChannelStatusById(id int, status int) error {
	result := DB.Model(&Channel{}).Where("id = ?", id).Update("status", status)
	return HandleUpdateResult(result, ErrChannelNotFound)
}

func DisableChannelById(id int) error {
	return UpdateChannelStatusById(id, ChannelStatusAutoDisabled)
}

func EnableChannelById(id int) error {
	return UpdateChannelStatusById(id, ChannelStatusEnabled)
}

func UpdateChannelUsedQuota(id int, quota int64) error {
	result := DB.Model(&Channel{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota))
	return HandleUpdateResult(result, ErrChannelNotFound)
}

func DeleteDisabledChannel() error {
	result := DB.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled).Delete(&Channel{})
	return HandleUpdateResult(result, ErrChannelNotFound)
}
