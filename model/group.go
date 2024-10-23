package model

import (
	"errors"
	"time"

	json "github.com/json-iterator/go"

	"github.com/songquanpeng/one-api/common"
	"gorm.io/gorm"
)

const (
	ErrGroupNotFound = "group"
)

const (
	GroupStatusEnabled  = 1 // don't use 0, 0 is the default value!
	GroupStatusDisabled = 2 // also don't use 0
	GroupStatusDeleted  = 3
)

type Group struct {
	Id           string    `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	AccessedAt   time.Time `json:"accessed_at"`
	Status       int       `gorm:"type:int;default:1" json:"status"` // enabled, disabled
	UsedAmount   float64   `gorm:"bigint" json:"used_amount"`        // used amount
	QPM          int64     `gorm:"bigint" json:"qpm"`                // queries per minute
	RequestCount int       `gorm:"type:int" json:"request_count"`    // request number
	Tokens       []*Token  `gorm:"foreignKey:GroupId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Logs         []*Log    `gorm:"foreignKey:GroupId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (g *Group) MarshalJSON() ([]byte, error) {
	type Alias Group
	return json.Marshal(&struct {
		Alias
		CreatedAt  int64 `json:"created_at"`
		AccessedAt int64 `json:"accessed_at"`
	}{
		Alias:      (Alias)(*g),
		CreatedAt:  g.CreatedAt.UnixMilli(),
		AccessedAt: g.AccessedAt.UnixMilli(),
	})
}

func GetGroups(startIdx int, num int, order string, onlyDisabled bool) (groups []*Group, total int64, err error) {
	tx := DB.Model(&Group{})
	if onlyDisabled {
		tx = tx.Where("status = ?", GroupStatusDisabled)
	}

	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total <= 0 {
		return nil, 0, nil
	}

	switch order {
	case "quota":
		tx = tx.Order("quota desc")
	case "used_quota":
		tx = tx.Order("used_quota desc")
	case "request_count":
		tx = tx.Order("request_count desc")
	default:
		tx = tx.Order("id desc")
	}

	err = tx.Limit(num).Offset(startIdx).Find(&groups).Error
	return groups, total, err
}

func GetGroupById(id string) (*Group, error) {
	if id == "" {
		return nil, errors.New("id 为空！")
	}
	group := Group{Id: id}
	err := DB.First(&group, "id = ?", id).Error
	return &group, HandleNotFound(err, ErrGroupNotFound)
}

func DeleteGroupById(id string) (err error) {
	if id == "" {
		return errors.New("id 为空！")
	}
	defer func() {
		if err == nil {
			_ = CacheDeleteGroup(id)
		}
	}()
	result := DB.
		Delete(&Group{
			Id: id,
		})
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupUsedAmountAndRequestCount(id string, amount float64, count int) error {
	result := DB.Model(&Group{}).Where("id = ?", id).Updates(map[string]interface{}{
		"used_amount":   gorm.Expr("used_amount + ?", amount),
		"request_count": gorm.Expr("request_count + ?", count),
		"accessed_at":   time.Now(),
	})
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupUsedAmount(id string, amount float64) error {
	result := DB.Model(&Group{}).Where("id = ?", id).Updates(map[string]interface{}{
		"used_amount": gorm.Expr("used_amount + ?", amount),
		"accessed_at": time.Now(),
	})
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupRequestCount(id string, count int) error {
	result := DB.Model(&Group{}).Where("id = ?", id).Updates(map[string]interface{}{
		"request_count": gorm.Expr("request_count + ?", count),
		"accessed_at":   time.Now(),
	})
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupQPM(id string, qpm int64) (err error) {
	defer func() {
		if err == nil {
			_ = CacheDeleteGroup(id)
		}
	}()
	result := DB.Model(&Group{}).Where("id = ?", id).UpdateColumn("qpm", gorm.Expr("qpm = ?", qpm))
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupStatus(id string, status int) (err error) {
	defer func() {
		if err == nil {
			_ = CacheDeleteGroup(id)
		}
	}()
	result := DB.Model(&Group{}).Where("id = ?", id).UpdateColumn("status", gorm.Expr("status = ?", status))
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func SearchGroup(keyword string, startIdx int, num int, onlyDisabled bool) (groups []*Group, total int64, err error) {
	tx := DB.Model(&Group{})
	if onlyDisabled {
		tx = tx.Where("status = ?", GroupStatusDisabled)
	}
	if common.UsingPostgreSQL {
		tx = tx.Where("id ILIKE ?", "%"+keyword+"%")
	} else {
		tx = tx.Where("id LIKE ?", "%"+keyword+"%")
	}
	err = tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if total <= 0 {
		return nil, 0, nil
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&groups).Error
	return groups, total, err
}

func CreateGroup(group *Group) error {
	return DB.Create(group).Error
}
