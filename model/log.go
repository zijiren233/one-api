package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
)

type Log struct {
	Id               int       `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	Type             int       `json:"type" gorm:"index:idx_created_at_type"`
	Content          string    `json:"content"`
	Group            string    `json:"group" gorm:"index;index:idx_group_model_name,priority:2"`
	Model            string    `json:"model" gorm:"index;index:idx_group_model_name,priority:1"`
	TokenName        string    `json:"token_name" gorm:"index"`
	PromptTokens     int       `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int       `json:"completion_tokens" gorm:"default:0"`
	ChannelId        int       `json:"channel" gorm:"index"`
}

const (
	LogTypeUnknown = iota
	LogTypeConsume
	LogTypeSystem
)

func RecordLog(group string, logType int, content string) {
	if logType == LogTypeConsume {
		return
	}
	log := &Log{
		Group:     group,
		CreatedAt: time.Now(),
		Type:      logType,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.SysError("failed to record log: " + err.Error())
	}
}

func RecordConsumeLog(ctx context.Context, group string, channelId int, promptTokens int, completionTokens int, modelName string, tokenName string, quota int64, content string) {
	logger.Info(ctx, fmt.Sprintf("record consume log: group=%s, channelId=%d, promptTokens=%d, completionTokens=%d, modelName=%s, tokenName=%s, quota=%d, content=%s", group, channelId, promptTokens, completionTokens, modelName, tokenName, quota, content))
	log := &Log{
		Group:            group,
		CreatedAt:        time.Now(),
		Type:             LogTypeConsume,
		Content:          content,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TokenName:        tokenName,
		Model:            modelName,
		ChannelId:        channelId,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.Error(ctx, "failed to record log: "+err.Error())
	}
}

func GetLogs(logType int, startTimestamp time.Time, endTimestamp time.Time, modelName string, group string, tokenName string, startIdx int, num int, channel int) (logs []*Log, total int64, err error) {
	tx := LOG_DB.Model(&Log{})
	if logType != LogTypeUnknown {
		tx = tx.Where("type = ?", logType)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if group != "" {
		tx = tx.Where("`group` = ?", group)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
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

func GetGroupLogs(group string, logType int, startTimestamp time.Time, endTimestamp time.Time, modelName string, tokenName string, startIdx int, num int, channel int) (logs []*Log, total int64, err error) {
	tx := LOG_DB.Model(&Log{}).Where("`group` = ?", group)
	if logType != LogTypeUnknown {
		tx = tx.Where("type = ?", logType)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
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
			tx = tx.Where("type = ? or content ILIKE ? or `group` ILIKE ?", keyword, "%"+keyword+"%", "%"+keyword+"%")
		} else {
			tx = tx.Where("type = ? or content LIKE ? or `group` LIKE ?", keyword, "%"+keyword+"%", "%"+keyword+"%")
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
		tx = tx.Where("`group` = ?", group)
	} else {
		tx = tx.Where("`group` = ?", group)
	}
	if keyword != "" {
		if common.UsingPostgreSQL {
			tx = tx.Where("content ILIKE ?", "%"+keyword+"%")
		} else {
			tx = tx.Where("content LIKE ?", "%"+keyword+"%")
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

func SumUsedQuota(logType int, startTimestamp time.Time, endTimestamp time.Time, modelName string, group string, tokenName string, channel int) (quota int64) {
	ifnull := "ifnull"
	if common.UsingPostgreSQL {
		ifnull = "COALESCE"
	}
	tx := LOG_DB.Table("logs").Select(fmt.Sprintf("%s(sum(quota),0)", ifnull))
	if group != "" {
		tx = tx.Where("`group` = ?", group)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
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
	tx.Where("type = ?", LogTypeConsume).Scan(&quota)
	return quota
}

func SumUsedToken(logType int, startTimestamp time.Time, endTimestamp time.Time, modelName string, group string, tokenName string) (token int) {
	ifnull := "ifnull"
	if common.UsingPostgreSQL {
		ifnull = "COALESCE"
	}
	tx := LOG_DB.Table("logs").Select(fmt.Sprintf("%s(sum(prompt_tokens),0) + %s(sum(completion_tokens),0)", ifnull, ifnull))
	if group != "" {
		tx = tx.Where("`group` = ?", group)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
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
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

func DeleteOldLog(targetTimestamp time.Time) (int64, error) {
	result := LOG_DB.Where("created_at < ?", targetTimestamp).Delete(&Log{})
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
		WHERE type=2
		AND group= ?
		AND created_at BETWEEN ? AND ?
		GROUP BY day, model
		ORDER BY day, model
	`, group, start, end).Scan(&LogStatistics).Error

	return LogStatistics, err
}
