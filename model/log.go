package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	json "github.com/json-iterator/go"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
)

type Log struct {
	Id               int       `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	Code             int       `json:"code"`
	GroupId          string    `gorm:"index;index:idx_group_model_name,priority:2" json:"group"`
	Group            *Group    `gorm:"foreignKey:GroupId" json:"-"`
	Model            string    `gorm:"index;index:idx_group_model_name,priority:1" json:"model"`
	UsedAmount       float64   `json:"used_amount"`
	Price            float64   `json:"price"`
	CompletionPrice  float64   `json:"completion_price"`
	TokenRemark      string    `gorm:"index" json:"token_remark"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	ChannelId        int       `gorm:"index" json:"channel"`
	Endpoint         string    `gorm:"index" json:"endpoint"`
}

func (l *Log) MarshalJSON() ([]byte, error) {
	type Alias Log
	return json.Marshal(&struct {
		Alias
		CreatedAt int64 `json:"created_at"`
	}{
		Alias:     (Alias)(*l),
		CreatedAt: l.CreatedAt.UnixMilli(),
	})
}

func RecordConsumeLog(ctx context.Context, group string, code int, channelId int, promptTokens int, completionTokens int, modelName string, tokenRemark string, usedAmount float64, price float64, completionPrice float64, endpoint string) {
	logger.Info(ctx, fmt.Sprintf("record consume log: group=%s, code=%d, channelId=%d, promptTokens=%d, completionTokens=%d, modelName=%s, tokenRemark=%s, usedAmount=%f, price=%f, completionPrice=%f, endpoint=%s", group, code, channelId, promptTokens, completionTokens, modelName, tokenRemark, usedAmount, price, completionPrice, endpoint))
	log := &Log{
		GroupId:          group,
		CreatedAt:        time.Now(),
		Code:             code,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TokenRemark:      tokenRemark,
		Model:            modelName,
		UsedAmount:       usedAmount,
		Price:            price,
		CompletionPrice:  completionPrice,
		ChannelId:        channelId,
		Endpoint:         endpoint,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.Error(ctx, "failed to record log: "+err.Error())
	}
}

func GetLogs(startTimestamp time.Time, endTimestamp time.Time, code int, modelName string, group string, tokenRemark string, startIdx int, num int, channel int, endpoint string) (logs []*Log, total int64, err error) {
	tx := LOG_DB.Model(&Log{})
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if group != "" {
		tx = tx.Where("group_id = ?", group)
	}
	if tokenRemark != "" {
		tx = tx.Where("token_remark = ?", tokenRemark)
	}
	if !startTimestamp.IsZero() {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if !endTimestamp.IsZero() {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	if endpoint != "" {
		tx = tx.Where("endpoint = ?", endpoint)
	}
	if code != 0 {
		tx = tx.Where("code = ?", code)
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	return logs, total, err
}

func GetGroupLogs(group string, startTimestamp time.Time, endTimestamp time.Time, code int, modelName string, tokenRemark string, startIdx int, num int, channel int, endpoint string) (logs []*Log, total int64, err error) {
	tx := LOG_DB.Model(&Log{}).Where("group_id = ?", group)
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if tokenRemark != "" {
		tx = tx.Where("token_remark = ?", tokenRemark)
	}
	if !startTimestamp.IsZero() {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if !endTimestamp.IsZero() {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	if endpoint != "" {
		tx = tx.Where("endpoint = ?", endpoint)
	}
	if code != 0 {
		tx = tx.Where("code = ?", code)
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Omit("id").Find(&logs).Error
	return logs, total, err
}

func SearchLogs(keyword string, page int, perPage int) (logs []*Log, total int64, err error) {
	tx := LOG_DB.Model(&Log{})
	if keyword != "" {
		if common.UsingPostgreSQL {
			tx = tx.Where("code = ? or group_id ILIKE ? or endpoint ILIKE ?", keyword, "%"+keyword+"%", "%"+keyword+"%")
		} else {
			tx = tx.Where("code = ? or group_id LIKE ? or endpoint LIKE ?", keyword, "%"+keyword+"%", "%"+keyword+"%")
		}
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	err = tx.Order("id desc").Limit(perPage).Offset(page * perPage).Find(&logs).Error
	return logs, total, err
}

func SearchGroupLogs(group string, keyword string, page int, perPage int) (logs []*Log, total int64, err error) {
	if group == "" {
		return nil, 0, errors.New("group is empty")
	}
	tx := LOG_DB.Model(&Log{})
	if common.UsingPostgreSQL {
		tx = tx.Where("group_id = ?", group)
	} else {
		tx = tx.Where("group_id = ?", group)
	}
	if keyword != "" {
		if common.UsingPostgreSQL {
			tx = tx.Where("code = ? or group_id ILIKE ? or endpoint ILIKE ?", keyword, "%"+keyword+"%", "%"+keyword+"%")
		} else {
			tx = tx.Where("code = ? or group_id LIKE ? or endpoint LIKE ?", keyword, "%"+keyword+"%", "%"+keyword+"%")
		}
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	err = tx.Order("id desc").Limit(perPage).Offset(page * perPage).Find(&logs).Error
	return logs, total, err
}

func SumUsedQuota(startTimestamp time.Time, endTimestamp time.Time, modelName string, group string, tokenRemark string, channel int, endpoint string) (quota int64) {
	ifnull := "ifnull"
	if common.UsingPostgreSQL {
		ifnull = "COALESCE"
	}
	tx := LOG_DB.Table("logs").Select(fmt.Sprintf("%s(sum(quota),0)", ifnull))
	if group != "" {
		tx = tx.Where("group_id = ?", group)
	}
	if tokenRemark != "" {
		tx = tx.Where("token_remark = ?", tokenRemark)
	}
	if !startTimestamp.IsZero() {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if !endTimestamp.IsZero() {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	if endpoint != "" {
		tx = tx.Where("endpoint = ?", endpoint)
	}
	tx.Scan(&quota)
	return quota
}

func SumUsedToken(startTimestamp time.Time, endTimestamp time.Time, modelName string, group string, tokenRemark string, endpoint string) (token int) {
	ifnull := "ifnull"
	if common.UsingPostgreSQL {
		ifnull = "COALESCE"
	}
	tx := LOG_DB.Table("logs").Select(fmt.Sprintf("%s(sum(prompt_tokens),0) + %s(sum(completion_tokens),0)", ifnull, ifnull))
	if group != "" {
		tx = tx.Where("group_id = ?", group)
	}
	if tokenRemark != "" {
		tx = tx.Where("token_remark = ?", tokenRemark)
	}
	if !startTimestamp.IsZero() {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if !endTimestamp.IsZero() {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if endpoint != "" {
		tx = tx.Where("endpoint = ?", endpoint)
	}
	tx.Scan(&token)
	return token
}

func DeleteOldLog(timestamp time.Time) (int64, error) {
	result := LOG_DB.Where("created_at < ?", timestamp).Delete(&Log{})
	return result.RowsAffected, result.Error
}

type LogStatistic struct {
	Day              string `gorm:"column:day"`
	Model            string `gorm:"column:model"`
	RequestCount     int    `gorm:"column:request_count"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
}

func SearchLogsByDayAndModel(group string, start time.Time, end time.Time) (LogStatistics []*LogStatistic, err error) {
	groupSelect := "DATE_FORMAT(FROM_UNIXTIME(created_at), '%Y-%m-%d') as day"

	if common.UsingPostgreSQL {
		groupSelect = "TO_CHAR(date_trunc('day', to_timestamp(created_at)), 'YYYY-MM-DD') as day"
	}

	if common.UsingSQLite {
		groupSelect = "strftime('%Y-%m-%d', datetime(created_at, 'unixepoch')) as day"
	}

	err = LOG_DB.Raw(`
		SELECT `+groupSelect+`,
		model, count(1) as request_count,
		sum(prompt_tokens) as prompt_tokens,
		sum(completion_tokens) as completion_tokens
		FROM logs
		AND group_id= ?
		AND created_at BETWEEN ? AND ?
		GROUP BY day, model
		ORDER BY day, model
	`, group, start, end).Scan(&LogStatistics).Error

	return LogStatistics, err
}
