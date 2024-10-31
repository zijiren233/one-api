package model

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ErrNotFound string

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found", string(e))
}

func HandleNotFound(err error, errMsg ...string) error {
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound(strings.Join(errMsg, " "))
	}
	return err
}

// Helper function to handle update results
func HandleUpdateResult(result *gorm.DB, entityName string) error {
	if result.Error != nil {
		return HandleNotFound(result.Error, entityName)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound(entityName)
	}
	return nil
}

func OnConflictDoNothing() *gorm.DB {
	return DB.Clauses(clause.OnConflict{
		DoNothing: true,
	})
}

func BatchRecordConsume(ctx context.Context, group string, code int, channelId int, promptTokens int, completionTokens int, modelName string, tokenId int, tokenName string, amount float64, price float64, completionPrice float64, endpoint string, content string) (err error) {
	token := &Token{}
	defer func() {
		if err == nil && token.Quota > 0 {
			if err := CacheUpdateTokenUsedAmountOnlyIncrease(token.Key, token.UsedAmount); err != nil {
				logger.SysError("CacheUpdateTokenUsedAmountOnlyIncrease failed: " + err.Error())
			}
		}
	}()
	now := time.Now()
	return DB.Transaction(func(tx *gorm.DB) error {
		log := &Log{
			CreatedAt:        now,
			GroupId:          group,
			TokenId:          tokenId,
			TokenName:        tokenName,
			Model:            modelName,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			ChannelId:        channelId,
			Content:          content,
			Code:             code,
			Price:            price,
			CompletionPrice:  completionPrice,
			UsedAmount:       amount,
			Endpoint:         endpoint,
		}
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		result := tx.Model(token).
			Clauses(clause.Returning{
				Columns: []clause.Column{
					{Name: "key"},
					{Name: "quota"},
					{Name: "used_amount"},
				},
			}).
			Where("id = ?", tokenId).
			Updates(map[string]interface{}{
				"used_amount":   gorm.Expr("used_amount + ?", amount),
				"request_count": gorm.Expr("request_count + ?", 1),
				"accessed_at":   now,
			})
		if err := HandleUpdateResult(result, ErrTokenNotFound); err != nil {
			return err
		}

		result = tx.Model(&Group{}).Where("id = ?", group).Updates(map[string]interface{}{
			"used_amount":   gorm.Expr("used_amount + ?", amount),
			"request_count": gorm.Expr("request_count + ?", 1),
			"accessed_at":   now,
		})
		if err := HandleUpdateResult(result, ErrGroupNotFound); err != nil {
			return err
		}

		result = tx.Model(&Channel{}).Where("id = ?", channelId).Updates(map[string]interface{}{
			"used_amount":   gorm.Expr("used_amount + ?", amount),
			"request_count": gorm.Expr("request_count + ?", 1),
			"accessed_at":   now,
		})
		if err := HandleUpdateResult(result, ErrChannelNotFound); err != nil {
			return err
		}
		return nil
	})
}

type EmptyNullString string

func (ns EmptyNullString) String() string {
	return string(ns)
}

// Scan implements the [Scanner] interface.
func (ns *EmptyNullString) Scan(value any) error {
	if value == nil {
		*ns = ""
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*ns = EmptyNullString(v)
	case string:
		*ns = EmptyNullString(v)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return nil
}

// Value implements the [driver.Valuer] interface.
func (ns EmptyNullString) Value() (driver.Value, error) {
	if ns == "" {
		return nil, nil
	}
	return string(ns), nil
}
