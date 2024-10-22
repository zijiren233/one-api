package router

import (
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api", middleware.AdminAuth)
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/models", controller.DashboardListModels)

		groupsRoute := apiRouter.Group("/groups")
		{
			groupsRoute.GET("/", controller.GetGroups)
			groupsRoute.GET("/search", controller.SearchGroups)
		}
		groupRoute := apiRouter.Group("/group")
		{
			groupRoute.GET("/:id", controller.GetGroup)
			groupRoute.POST("/", controller.CreateGroup)
			groupRoute.DELETE("/:id", controller.DeleteGroup)
		}
		optionRoute := apiRouter.Group("/option")
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
		}
		channelsRoute := apiRouter.Group("/channels")
		{
			channelsRoute.GET("/", controller.GetChannels)
			channelsRoute.GET("/search", controller.SearchChannels)
			channelsRoute.GET("/test", controller.TestChannels)
			channelsRoute.GET("/update_balance", controller.UpdateAllChannelsBalance)
		}
		channelRoute := apiRouter.Group("/channel")
		{
			channelRoute.GET("/:id", controller.GetChannel)
			channelRoute.POST("/", controller.AddChannel)
			channelRoute.PUT("/", controller.UpdateChannel)
			channelRoute.DELETE("/:id", controller.DeleteChannel)
			channelRoute.GET("/test/:id", controller.TestChannel)
			channelRoute.GET("/update_balance/:id", controller.UpdateChannelBalance)
		}
		tokensRoute := apiRouter.Group("/tokens")
		{
			tokensRoute.GET("/", controller.GetTokens)
			tokensRoute.GET("/:id", controller.GetToken)
			tokensRoute.PUT("/:id", controller.UpdateToken)
			tokensRoute.POST("/:id", controller.UpdateTokenStatus)
			tokensRoute.DELETE("/:id", controller.DeleteToken)
			tokensRoute.GET("/search", controller.SearchTokens)
		}
		tokenRoute := apiRouter.Group("/token")
		{
			tokenRoute.GET("/search/:group", controller.SearchGroupTokens)
			tokenRoute.GET("/group/:group", controller.GetGroupTokens)
			tokenRoute.GET("/group/:group/:id", controller.GetGroupToken)
			tokenRoute.POST("/group/:group", controller.AddToken)
			tokenRoute.PUT("/group/:group/:id", controller.UpdateGroupToken)
			tokenRoute.POST("/group/:group/:id", controller.UpdateGroupTokenStatus)
			tokenRoute.DELETE("/group/:group/:id", controller.DeleteGroupToken)
		}
		logsRoute := apiRouter.Group("/logs")
		{
			logsRoute.GET("/", controller.GetLogs)
			logsRoute.GET("/group/:group", controller.GetGroupLogs)
			logsRoute.DELETE("/", controller.DeleteHistoryLogs)
			logsRoute.GET("/stat", controller.GetLogsStat)
			logsRoute.GET("/search", controller.SearchLogs)
			logsRoute.GET("/search/:group", controller.SearchGroupLogs)
		}
	}
}
