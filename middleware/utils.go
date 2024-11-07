package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
)

func abortWithMessage(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": helper.MessageWithRequestId(message, c.GetString(helper.RequestIdKey)),
			"type":    "aiproxy_error",
		},
	})
	c.Abort()
	logger.Error(c.Request.Context(), message)
}

func getRequestModel(c *gin.Context) (string, error) {
	var modelRequest ModelRequest
	err := common.UnmarshalBodyReusable(c, &modelRequest)
	if err != nil {
		return "", fmt.Errorf("common.UnmarshalBodyReusable failed: %w", err)
	}

	if modelRequest.Model != "" {
		return modelRequest.Model, nil
	}

	path := c.Request.URL.Path
	switch {
	case strings.HasPrefix(path, "/v1/moderations"):
		modelRequest.Model = "text-moderation-stable"
	case strings.HasSuffix(path, "embeddings"):
		modelRequest.Model = c.Param("model")
	case strings.HasPrefix(path, "/v1/images/generations"):
		modelRequest.Model = "dall-e-2"
	case strings.HasPrefix(path, "/v1/audio/transcriptions"), strings.HasPrefix(path, "/v1/audio/translations"):
		modelRequest.Model = "whisper-1"
	}

	return modelRequest.Model, nil
}
