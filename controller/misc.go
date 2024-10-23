package controller

import (
	"net/http"

	"github.com/songquanpeng/one-api/common"

	"github.com/gin-gonic/gin"
)

func GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"version":    common.Version,
			"start_time": common.StartTime,
		},
	})
	return
}
