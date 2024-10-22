package model

import (
	"encoding/json"
	"errors"
	"time"

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
	UpdatedAt    time.Time `json:"updated_at"`
	Status       int       `gorm:"type:int;default:1" json:"status"` // enabled, disabled
	UsedQuota    int64     `gorm:"bigint" json:"used_quota"`         // used quota
	QPM          int64     `gorm:"bigint" json:"qpm"`                // queries per minute
	RequestCount int       `gorm:"type:int" json:"request_count"`    // request number
	Tokens       []*Token  `gorm:"foreignKey:Group;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Logs         []*Log    `gorm:"foreignKey:Group;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (g *Group) MarshalJSON() ([]byte, error) {
	type Alias Group
	return json.Marshal(&struct {
		Alias
		CreatedAt int64 `json:"created_at"`
		UpdatedAt int64 `json:"updated_at"`
	}{
		Alias:     (Alias)(*g),
		CreatedAt: g.CreatedAt.UnixMilli(),
		UpdatedAt: g.UpdatedAt.UnixMilli(),
	})
}

func GetAllGroups(startIdx int, num int, order string) (groups []*Group, err error) {
	query := DB.Limit(num).Offset(startIdx).Where("status != ?", GroupStatusDeleted)

	switch order {
	case "quota":
		query = query.Order("quota desc")
	case "used_quota":
		query = query.Order("used_quota desc")
	case "request_count":
		query = query.Order("request_count desc")
	default:
		query = query.Order("id desc")
	}

	err = query.Find(&groups).Error
	return groups, err
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
	result := DB.Delete(&Group{
		Id: id,
	})
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func GetGroupQuota(id string) (int64, error) {
	var quota int64
	err := DB.Model(&Group{}).Where("id = ?", id).Select("quota").Find(&quota).Error
	return quota, HandleNotFound(err, ErrGroupNotFound)
}

func GetGroupUsedQuota(id string) (int64, error) {
	var quota int64
	err := DB.Model(&Group{}).Where("id = ?", id).Select("used_quota").Find(&quota).Error
	return quota, HandleNotFound(err, ErrGroupNotFound)
}

func UpdateGroupUsedQuotaAndRequestCount(id string, quota int64, count int) error {
	result := DB.Model(&Group{}).Where("id = ?", id).Updates(map[string]interface{}{
		"used_quota":    gorm.Expr("used_quota + ?", quota),
		"request_count": gorm.Expr("request_count + ?", count),
	})
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupUsedQuota(id string, quota int64) error {
	result := DB.Model(&Group{}).Where("id = ?", id).UpdateColumn("used_quota", gorm.Expr("used_quota + ?", quota))
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupRequestCount(id string, count int) error {
	result := DB.Model(&Group{}).Where("id = ?", id).UpdateColumn("request_count", gorm.Expr("request_count + ?", count))
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupQPM(id string, qpm int64) error {
	result := DB.Model(&Group{}).Where("id = ?", id).UpdateColumn("qpm", gorm.Expr("qpm = ?", qpm))
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func UpdateGroupStatus(id string, status int) error {
	result := DB.Model(&Group{}).Where("id = ?", id).UpdateColumn("status", gorm.Expr("status = ?", status))
	return HandleUpdateResult(result, ErrGroupNotFound)
}

func IsGroupEnabled(id string) (bool, error) {
	if id == "" {
		return false, errors.New("group id is empty")
	}
	var group Group
	err := DB.Where("id = ?", id).Select("status").Find(&group).Error
	if err != nil {
		return false, err
	}
	return group.Status == GroupStatusEnabled, nil
}

func SearchGroup(keyword string, startIdx int, num int) (groups []*Group, total int64, err error) {
	tx := DB.Model(&Group{})
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