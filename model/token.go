package model

import (
	"errors"
	"fmt"
	"time"

	json "github.com/json-iterator/go"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ErrTokenNotFound = "token"
)

const (
	TokenStatusEnabled   = 1 // don't use 0, 0 is the default value!
	TokenStatusDisabled  = 2 // also don't use 0
	TokenStatusExpired   = 3
	TokenStatusExhausted = 4
)

type Token struct {
	Id           int             `gorm:"primaryKey" json:"id"`
	GroupId      string          `gorm:"index;uniqueIndex:idx_group_remark" json:"group"`
	Group        *Group          `gorm:"foreignKey:GroupId" json:"-"`
	Key          string          `gorm:"type:char(48);uniqueIndex" json:"key"`
	Status       int             `gorm:"default:1;index" json:"status"`
	Remark       EmptyNullString `gorm:"uniqueIndex:idx_group_remark" json:"remark"`
	CreatedAt    time.Time       `json:"created_at"`
	AccessedAt   time.Time       `json:"accessed_at"`
	ExpiredAt    time.Time       `json:"expired_at"`
	Quota        float64         `gorm:"bigint" json:"quota"`
	UsedAmount   float64         `gorm:"bigint" json:"used_amount"` // used amount
	RequestCount int             `gorm:"type:int" json:"request_count"`
	Models       []string        `gorm:"serializer:json;type:text" json:"models"` // allowed models
	Subnet       string          `json:"subnet"`                                  // allowed subnet
}

func (t *Token) MarshalJSON() ([]byte, error) {
	type Alias Token
	return json.Marshal(&struct {
		Alias
		CreatedAt  int64 `json:"created_at"`
		AccessedAt int64 `json:"accessed_at"`
		ExpiredAt  int64 `json:"expired_at"`
	}{
		Alias:      (Alias)(*t),
		CreatedAt:  t.CreatedAt.UnixMilli(),
		AccessedAt: t.AccessedAt.UnixMilli(),
		ExpiredAt:  t.ExpiredAt.UnixMilli(),
	})
}

func InsertToken(token *Token, autoCreateGroup bool) error {
	if autoCreateGroup {
		group := &Group{
			Id: token.GroupId,
		}
		if err := OnConflictDoNothing().Create(group).Error; err != nil {
			return err
		}
	}
	maxTokenNum := config.GetGroupMaxTokenNum()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if maxTokenNum > 0 {
			var count int64
			err := tx.Model(&Token{}).Where("group_id = ?", token.GroupId).Count(&count).Error
			if err != nil {
				return err
			}
			if count >= int64(maxTokenNum) {
				return errors.New("group max token num reached")
			}
		}
		return tx.Create(token).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("token remark already exists in this group")
		}
		return err
	}
	return nil
}

func GetTokens(startIdx int, num int, order string, group string) (tokens []*Token, total int64, err error) {
	tx := DB.Model(&Token{})

	if group != "" {
		tx = tx.Where("group_id = ?", group)
	}

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}
	switch order {
	case "used_amount":
		tx = tx.Order("used_amount desc")
	default:
		tx = tx.Order("id desc")
	}
	err = tx.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, total, err
}

func GetGroupTokens(group string, startIdx int, num int, order string) (tokens []*Token, total int64, err error) {
	if group == "" {
		return nil, 0, errors.New("group is empty")
	}

	tx := DB.Model(&Token{}).Where("group_id = ?", group)

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}
	switch order {
	case "used_amount":
		tx = tx.Order("used_amount desc")
	default:
		tx = tx.Order("id desc")
	}
	err = tx.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, total, err
}

func SearchTokens(keyword string, startIdx int, num int, order string) (tokens []*Token, total int64, err error) {
	tx := DB.Model(&Token{})
	if common.UsingPostgreSQL {
		tx = tx.Where("remark ILIKE ? or key ILIKE ? or group_id ILIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	} else {
		tx = tx.Where("remark LIKE ? or key LIKE ? or group_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	switch order {
	case "used_amount":
		tx = tx.Order("used_amount desc")
	default:
		tx = tx.Order("id desc")
	}
	err = tx.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, total, err
}

func SearchGroupTokens(group string, keyword string, startIdx int, num int, order string) (tokens []*Token, total int64, err error) {
	if group == "" {
		return nil, 0, errors.New("group is empty")
	}
	tx := DB.Model(&Token{}).Where("group_id = ?", group)
	if common.UsingPostgreSQL {
		tx = tx.Where("remark ILIKE ? or key ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	} else {
		tx = tx.Where("remark LIKE ? or key LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	switch order {
	case "used_amount":
		tx = tx.Order("used_amount desc")
	default:
		tx = tx.Order("id desc")
	}
	err = tx.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, total, err
}

func GetTokenByKey(key string) (*Token, error) {
	var token Token
	err := DB.Where("key = ?", key).First(&token).Error
	return &token, HandleNotFound(err, ErrTokenNotFound)
}

func GetTokenUsedAmount(id int) (float64, error) {
	var amount float64
	err := DB.Model(&Token{}).Where("id = ?", id).Select("used_amount").Scan(&amount).Error
	return amount, HandleNotFound(err, ErrTokenNotFound)
}

func GetTokenUsedAmountByKey(key string) (float64, error) {
	var amount float64
	err := DB.Model(&Token{}).Where("key = ?", key).Select("used_amount").Scan(&amount).Error
	return amount, HandleNotFound(err, ErrTokenNotFound)
}

func ValidateAndGetToken(key string) (token *TokenCache, err error) {
	if key == "" {
		return nil, errors.New("未提供令牌")
	}
	token, err = CacheGetTokenByKey(key)
	if err != nil {
		logger.SysError("CacheGetTokenByKey failed: " + err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无效的令牌")
		}
		return nil, fmt.Errorf("令牌验证失败")
	}
	if token.Status == TokenStatusExhausted {
		return nil, fmt.Errorf("令牌 (%d) 额度已用尽", token.Id)
	} else if token.Status == TokenStatusExpired {
		return nil, fmt.Errorf("令牌 (%d) 已过期", token.Id)
	}
	if token.Status != TokenStatusEnabled {
		return nil, fmt.Errorf("令牌 (%d) 状态不可用", token.Id)
	}
	if !token.ExpiredAt.IsZero() && token.ExpiredAt.Before(time.Now()) {
		if !common.RedisEnabled {
			err := UpdateTokenStatusAndAccessedAt(token.Id, TokenStatusExpired)
			if err != nil {
				logger.SysError("failed to update token status" + err.Error())
			}
		}
		return nil, fmt.Errorf("令牌 (%d) 已过期", token.Id)
	}
	if token.Quota > 0 && token.UsedAmount >= token.Quota {
		// in this case, we can make sure the token is exhausted
		err := UpdateTokenStatusAndAccessedAt(token.Id, TokenStatusExhausted)
		if err != nil {
			logger.SysError("failed to update token status" + err.Error())
		}
		return nil, fmt.Errorf("令牌 (%d) 额度已用尽", token.Id)
	}
	return token, nil
}

func GetGroupTokenById(group string, id int) (*Token, error) {
	if id == 0 || group == "" {
		return nil, errors.New("id 或 group 为空！")
	}
	token := Token{}
	err := DB.
		Where("id = ? and group_id = ?", id, group).
		First(&token).Error
	return &token, HandleNotFound(err, ErrTokenNotFound)
}

func GetTokenById(id int) (*Token, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	token := Token{Id: id}
	err := DB.First(&token, "id = ?", id).Error
	return &token, HandleNotFound(err, ErrTokenNotFound)
}

func UpdateTokenStatus(id int, status int) (err error) {
	token := Token{Id: id}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Model(&token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ?", id).
		Updates(
			map[string]interface{}{
				"status": status,
			},
		)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateTokenStatusAndAccessedAt(id int, status int) (err error) {
	token := Token{Id: id}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Model(&token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ?", id).Updates(
		map[string]interface{}{
			"status":      status,
			"accessed_at": time.Now(),
		},
	)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateGroupTokenStatusAndAccessedAt(group string, id int, status int) (err error) {
	token := Token{}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Model(&token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ? and group_id = ?", id, group).
		Updates(
			map[string]interface{}{
				"status":      status,
				"accessed_at": time.Now(),
			},
		)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateGroupTokenStatus(group string, id int, status int) (err error) {
	token := Token{}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Model(&token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ? and group_id = ?", id, group).
		Updates(
			map[string]interface{}{
				"status": status,
			},
		)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func DeleteTokenByIdAndGroupId(id int, groupId string) (err error) {
	if id == 0 || groupId == "" {
		return errors.New("id 或 group 为空！")
	}
	token := Token{Id: id, GroupId: groupId}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where(token).
		Delete(&token)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func DeleteTokenById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	token := Token{Id: id}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where(token).
		Delete(&token)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateToken(token *Token) (err error) {
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.Omit("created_at", "status", "key", "group_id", "used_amount", "request_count").Save(token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errors.New("token remark already exists in this group")
		}
	}
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateTokenUsedAmount(id int, amount float64, requestCount int) (err error) {
	token := &Token{Id: id}
	defer func() {
		if err == nil && token.Quota > 0 {
			_ = CacheUpdateTokenUsedAmount(token.Key, token.UsedAmount)
		}
	}()
	result := DB.
		Model(token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
				{Name: "quota"},
				{Name: "used_amount"},
			},
		}).
		Where("id = ?", id).
		Updates(
			map[string]interface{}{
				"used_amount":   gorm.Expr("used_amount + ?", amount),
				"request_count": gorm.Expr("request_count + ?", requestCount),
				"accessed_at":   time.Now(),
			},
		)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateTokenRemark(id int, remark string) (err error) {
	token := &Token{Id: id}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Model(token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ?", id).
		Update("remark", remark)
	if result.Error != nil && errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return errors.New("token remark already exists in this group")
	}
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateGroupTokenRemark(group string, id int, remark string) (err error) {
	token := &Token{Id: id, GroupId: group}
	defer func() {
		if err == nil {
			_ = CacheDeleteToken(token.Key)
		}
	}()
	result := DB.
		Model(token).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "key"},
			},
		}).
		Where("id = ? and group_id = ?", id, group).
		Update("remark", remark)
	if result.Error != nil && errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return errors.New("token remark already exists in this group")
	}
	return HandleUpdateResult(result, ErrTokenNotFound)
}
