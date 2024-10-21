package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/network"
	"github.com/songquanpeng/one-api/model"
)

func AdminAuth(c *gin.Context) {
	accessToken := c.Request.Header.Get("Authorization")
	if accessToken == "" || accessToken != config.AdminKey {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "无权进行此操作，未登录且未提供 access token",
		})
		c.Abort()
		return
	}
	c.Next()
}

func TokenAuth(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Request.Header.Get("Authorization")
	key = strings.TrimPrefix(
		strings.TrimPrefix(key, "Bearer "),
		"sk-",
	)
	parts := strings.Split(key, "-")
	key = parts[0]
	token, err := model.ValidateAndGetToken(key)
	if err != nil {
		abortWithMessage(c, http.StatusUnauthorized, err.Error())
		return
	}
	if token.Subnet != "" {
		if !network.IsIpInSubnets(ctx, c.ClientIP(), token.Subnet) {
			abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("该令牌只能在指定网段使用：%s，当前 ip：%s", token.Subnet, c.ClientIP()))
			return
		}
	}
	groupEnabled, err := model.CacheIsGroupEnabled(ctx, token.Group)
	if err != nil {
		abortWithMessage(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !groupEnabled {
		abortWithMessage(c, http.StatusForbidden, "用户组已被禁用")
		return
	}
	requestModel, err := getRequestModel(c)
	if err != nil && shouldCheckModel(c) {
		abortWithMessage(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Set(ctxkey.RequestModel, requestModel)
	if token.Models != "" {
		c.Set(ctxkey.AvailableModels, token.Models)
		if requestModel != "" && !isModelInList(requestModel, token.Models) {
			abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("该令牌无权使用模型：%s", requestModel))
			return
		}
	}
	c.Set(ctxkey.Group, token.Group)
	c.Set(ctxkey.TokenId, token.Id)
	c.Set(ctxkey.TokenName, token.Name)
	if len(parts) > 1 {
		// c.Set(ctxkey.SpecificChannelId, parts[1])
	}

	// set channel id for proxy relay
	if channelId := c.Param("channelid"); channelId != "" {
		c.Set(ctxkey.SpecificChannelId, channelId)
	}

	c.Next()
}

func shouldCheckModel(c *gin.Context) bool {
	if strings.HasPrefix(c.Request.URL.Path, "/v1/completions") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/chat/completions") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/images") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/audio") {
		return true
	}
	return false
}
