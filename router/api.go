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
		tokenRoute := apiRouter.Group("/token")
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
		}
		logRoute := apiRouter.Group("/log")
		{
			logRoute.GET("/", controller.GetAllLogs)
			logRoute.DELETE("/", controller.DeleteHistoryLogs)
			logRoute.GET("/stat", controller.GetLogsStat)
			logRoute.GET("/search", controller.SearchAllLogs)
		}
	}
}
