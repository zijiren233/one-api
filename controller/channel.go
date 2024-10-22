package controller

import (
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/model"
)

func GetChannels(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage <= 0 {
		perPage = 10
	}
	channels, total, err := model.GetChannels(p*perPage, perPage, false, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"channels": channels,
			"total":    total,
		},
	})
	return
}

func SearchChannels(c *gin.Context) {
	keyword := c.Query("keyword")
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage <= 0 {
		perPage = 10
	}
	channels, total, err := model.SearchChannels(keyword, p*perPage, perPage, false, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"channels": channels,
			"total":    total,
		},
	})
	return
}

func GetChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel, err := model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channel,
	})
	return
}

type AddChannelRequest struct {
	Type         int                 `json:"type"`
	Name         string              `json:"name"`
	Key          string              `json:"key"`
	BaseURL      string              `json:"base_url"`
	Other        string              `json:"other"`
	Models       []string            `json:"models"`
	ModelMapping map[string]string   `json:"model_mapping"`
	Priority     int32               `json:"priority"`
	Config       model.ChannelConfig `json:"config"`
}

func (r *AddChannelRequest) ToChannel() *model.Channel {
	return &model.Channel{
		Type:         r.Type,
		Name:         r.Name,
		Key:          r.Key,
		BaseURL:      r.BaseURL,
		Other:        r.Other,
		Models:       slices.Clone(r.Models),
		ModelMapping: maps.Clone(r.ModelMapping),
		Config:       r.Config,
		Priority:     r.Priority,
	}
}

func AddChannel(c *gin.Context) {
	channel := AddChannelRequest{}
	err := c.ShouldBindJSON(&channel)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	keys := strings.Split(channel.Key, "\n")
	channels := make([]*model.Channel, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		localChannel := channel
		localChannel.Key = key
		channels = append(channels, localChannel.ToChannel())
	}
	err = model.BatchInsertChannels(channels)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteChannelById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

type UpdateChannelRequest struct {
	AddChannelRequest
	Id int `json:"id"`
}

func (r *UpdateChannelRequest) ToChannel() *model.Channel {
	c := r.AddChannelRequest.ToChannel()
	c.Id = r.Id
	return c
}

func UpdateChannel(c *gin.Context) {
	channel := UpdateChannelRequest{}
	err := c.ShouldBindJSON(&channel)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	ch := channel.ToChannel()
	err = model.UpdateChannel(ch)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": UpdateChannelRequest{
			Id: ch.Id,
			AddChannelRequest: AddChannelRequest{
				Type:         ch.Type,
				Name:         ch.Name,
				Key:          ch.Key,
				BaseURL:      ch.BaseURL,
				Other:        ch.Other,
				Models:       ch.Models,
				ModelMapping: ch.ModelMapping,
				Config:       ch.Config,
				Priority:     ch.Priority,
			},
		},
	})
	return
}
