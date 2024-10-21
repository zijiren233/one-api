package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/model"

	"github.com/gin-gonic/gin"
)

func GetAllGroups(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage <= 0 {
		perPage = 10
	}

	order := c.DefaultQuery("order", "")
	groups, err := model.GetAllGroups(p*perPage, perPage, order)
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
		"data":    groups,
	})
}

func SearchGroups(c *gin.Context) {
	keyword := c.Query("keyword")
	groups, err := model.SearchGroup(keyword)
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
		"data":    groups,
	})
	return
}

func GetGroup(c *gin.Context) {
	id := c.Param("id")
	group, err := model.GetGroupById(id)
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
		"data":    group,
	})
	return
}

func GetGroupDashboard(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()
	startOfDay := now.Truncate(24*time.Hour).AddDate(0, 0, -6).Unix()
	endOfDay := now.Truncate(24 * time.Hour).Add(24*time.Hour - time.Second).Unix()

	dashboards, err := model.SearchLogsByDayAndModel(id, time.Unix(startOfDay, 0), time.Unix(endOfDay, 0))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无法获取统计信息",
			"data":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dashboards,
	})
	return
}

type UpdateGroupQPMRequest struct {
	Id  string `json:"id"`
	QPM int64  `json:"qpm"`
}

func UpdateGroupQPM(c *gin.Context) {
	req := UpdateGroupQPMRequest{}
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil || req.Id == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	err = model.UpdateGroupQPM(req.Id, req.QPM)
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

type UpdateGroupStatusRequest struct {
	Id     string `json:"id"`
	Status int    `json:"status"`
}

func UpdateGroupStatus(c *gin.Context) {
	req := UpdateGroupStatusRequest{}
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil || req.Id == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	err = model.UpdateGroupStatus(req.Id, req.Status)
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

func DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	err := model.DeleteGroupById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
		})
		return
	}
}

func CreateGroup(c *gin.Context) {
	var group model.Group
	err := json.NewDecoder(c.Request.Body).Decode(&group)
	if err != nil || group.Id == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	if err := common.Validate.Struct(&group); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "输入不合法 " + err.Error(),
		})
		return
	}
	if err := model.CreateGroup(&group); err != nil {
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
