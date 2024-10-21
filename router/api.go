package router

import (
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/models", middleware.AdminAuth, controller.DashboardListModels)

		userRoute := apiRouter.Group("/user")
		{
			adminRoute := userRoute.Group("/")
			adminRoute.Use(middleware.AdminAuth)
			{
				adminRoute.GET("/", controller.GetAllGroups)
				adminRoute.GET("/search", controller.SearchGroups)
				adminRoute.GET("/:id", controller.GetGroup)
				adminRoute.POST("/", controller.CreateGroup)
				adminRoute.DELETE("/:id", controller.DeleteGroup)
			}
		}
		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.AdminAuth)
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
		}
		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AdminAuth)
		{
			channelRoute.GET("/", controller.GetAllChannels)
			channelRoute.GET("/search", controller.SearchChannels)
			channelRoute.GET("/models", controller.ListAllModels)
			channelRoute.GET("/:id", controller.GetChannel)
			channelRoute.GET("/test", controller.TestChannels)
			channelRoute.GET("/test/:id", controller.TestChannel)
			channelRoute.GET("/update_balance", controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", controller.UpdateChannelBalance)
			channelRoute.POST("/", controller.AddChannel)
			channelRoute.PUT("/", controller.UpdateChannel)
			channelRoute.DELETE("/:id", controller.DeleteChannel)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.AdminAuth)
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.Use(middleware.AdminAuth)
		{
			logRoute.GET("/", controller.GetAllLogs)
			logRoute.DELETE("/", controller.DeleteHistoryLogs)
			logRoute.GET("/stat", controller.GetLogsStat)
			logRoute.GET("/search", controller.SearchAllLogs)
		}
	}
}
