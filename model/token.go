package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
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
	Id             int       `gorm:"primaryKey" json:"id"`
	Group          string    `gorm:"index" json:"group"`
	Key            string    `json:"key" gorm:"type:char(48);uniqueIndex"`
	Status         int       `json:"status" gorm:"default:1"`
	Name           string    `json:"name" gorm:"index" `
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	AccessedAt     time.Time `json:"accessed_at"`
	ExpiredAt      time.Time `json:"expired_at"`
	RemainQuota    int64     `json:"remain_quota" gorm:"bigint;default:0"`
	UnlimitedQuota bool      `json:"unlimited_quota" gorm:"default:false"`
	UsedQuota      int64     `json:"used_quota" gorm:"bigint;default:0"` // used quota
	Models         string    `json:"models" gorm:"type:text"`            // allowed models
	Subnet         string    `json:"subnet" gorm:"default:''"`           // allowed subnet
}

func InsertToken(token *Token) error {
	return DB.Create(token).Error
}

func GetAllGroupTokens(group string, startIdx int, num int, order string) (tokens []*Token, total int64, err error) {
	tx := DB.Model(&Token{}).Where("group = ?", group)

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}
	switch order {
	case "remain_quota":
		tx = tx.Order("unlimited_quota desc, remain_quota desc")
	case "used_quota":
		tx = tx.Order("used_quota desc")
	default:
		tx = tx.Order("id desc")
	}
	err = tx.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, total, err
}

func SearchGroupTokens(group string, keyword string, startIdx int, num int, order string) (tokens []*Token, total int64, err error) {
	tx := DB.Model(&Token{})
	if common.UsingPostgreSQL {
		tx = tx.Where("group ILIKE ?", "%"+group+"%")
		tx = tx.Where("name ILIKE ?", "%"+keyword+"%")
	} else {
		tx = tx.Where("group LIKE ?", "%"+group+"%")
		tx = tx.Where("name LIKE ?", "%"+keyword+"%")
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	switch order {
	case "remain_quota":
		tx = tx.Order("unlimited_quota desc, remain_quota desc")
	case "used_quota":
		tx = tx.Order("used_quota desc")
	default:
		tx = tx.Order("id desc")
	}
	err = tx.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, total, err
}

func ValidateAndGetToken(key string) (token *Token, err error) {
	if key == "" {
		return nil, errors.New("未提供令牌")
	}
	token, err = CacheGetTokenByKey(key)
	if err != nil {
		logger.SysError("CacheGetTokenByKey failed: " + err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无效的令牌")
		}
		return nil, errors.New("令牌验证失败")
	}
	if token.Status == TokenStatusExhausted {
		return nil, fmt.Errorf("令牌 %s（#%d）额度已用尽", token.Name, token.Id)
	} else if token.Status == TokenStatusExpired {
		return nil, errors.New("该令牌已过期")
	}
	if token.Status != TokenStatusEnabled {
		return nil, errors.New("该令牌状态不可用")
	}
	if !token.ExpiredAt.IsZero() && token.ExpiredAt.Before(time.Now()) {
		if !common.RedisEnabled {
			token.Status = TokenStatusExpired
			err := token.UpdateStatusAndAccessedAt()
			if err != nil {
				logger.SysError("failed to update token status" + err.Error())
			}
		}
		return nil, errors.New("该令牌已过期")
	}
	if !token.UnlimitedQuota && token.RemainQuota <= 0 {
		if !common.RedisEnabled {
			// in this case, we can make sure the token is exhausted
			token.Status = TokenStatusExhausted
			err := token.UpdateStatusAndAccessedAt()
			if err != nil {
				logger.SysError("failed to update token status" + err.Error())
			}
		}
		return nil, errors.New("该令牌额度已用尽")
	}
	return token, nil
}

func GetTokenByIdAndGroupId(id int, group string) (*Token, error) {
	if id == 0 || group == "" {
		return nil, errors.New("id 或 group 为空！")
	}
	token := Token{Id: id, Group: group}
	err := DB.First(&token, "id = ? and group = ?", id, group).Error
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

func (t *Token) UpdateStatusAndAccessedAt() error {
	result := DB.Model(t).Updates(
		map[string]interface{}{
			"status":      t.Status,
			"accessed_at": time.Now(),
		},
	)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func DeleteTokenByIdAndGroupId(id int, groupId string) (err error) {
	if id == 0 || groupId == "" {
		return errors.New("id 或 group 为空！")
	}
	token := Token{Id: id, Group: groupId}
	result := DB.Where(token).Delete(&token)
	return HandleUpdateResult(result, ErrTokenNotFound)
}

func UpdateToken(token *Token) error {
	return DB.Save(token).Error
}

func UpdateTokenUsedQuota(id int, quota int64) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"remain_quota": gorm.Expr("remain_quota - ?", quota),
			"used_quota":   gorm.Expr("used_quota + ?", quota),
			"accessed_at":  time.Now(),
		},
	).Error
	return err
}

func PreConsumeTokenQuota(tokenId int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	if !token.UnlimitedQuota && token.RemainQuota < quota {
		return errors.New("令牌额度不足")
	}
	userQuota, err := GetGroupQuota(token.Group)
	if err != nil {
		return err
	}
	if userQuota < quota {
		return errors.New("用户额度不足")
	}
	quotaTooLow := userQuota >= config.QuotaRemindThreshold && userQuota-quota < config.QuotaRemindThreshold
	noMoreQuota := userQuota-quota <= 0
	if quotaTooLow || noMoreQuota {
		// go func() {
		// 	email, err := GetUserEmail(token.UserId)
		// 	if err != nil {
		// 		logger.SysError("failed to fetch user email: " + err.Error())
		// 	}
		// 	prompt := "您的额度即将用尽"
		// 	if noMoreQuota {
		// 		prompt = "您的额度已用尽"
		// 	}
		// 	if email != "" {
		// 		topUpLink := fmt.Sprintf("%s/topup", config.ServerAddress)
		// 		err = message.SendEmail(prompt, email,
		// 			fmt.Sprintf("%s，当前剩余额度为 %d，为了不影响您的使用，请及时充值。<br/>充值链接：<a href='%s'>%s</a>", prompt, userQuota, topUpLink, topUpLink))
		// 		if err != nil {
		// 			logger.SysError("failed to send email" + err.Error())
		// 		}
		// 	}
		// }()
	}
	if token.UnlimitedQuota {
		return nil
	}
	err = UpdateGroupUsedQuota(token.Group, -quota)
	return err
}

func PostConsumeTokenQuota(tokenId int, quota int64) (err error) {
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	if token.UnlimitedQuota {
		return nil
	}
	err = UpdateGroupUsedQuota(token.Group, -quota)
	return err
}
