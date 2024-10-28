package main

import (
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/balance"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/router"
)

func main() {
	common.Init()
	logger.SetupLogger()
	logger.SysLogf("One API %s started", common.Version)

	sealosJwtKey := os.Getenv("SEALOS_JWT_KEY")
	if sealosJwtKey == "" {
		logger.SysLog("SEALOS_JWT_KEY is not set, balance will not be enabled")
	} else {
		logger.SysLog("SEALOS_JWT_KEY is set, balance will be enabled")
		err := balance.InitSealos(sealosJwtKey, os.Getenv("SEALOS_ACCOUNT_URL"))
		if err != nil {
			logger.FatalLog("failed to initialize sealos balance: " + err.Error())
		}
	}

	if os.Getenv("GIN_MODE") != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	if config.DebugEnabled {
		logger.SysLog("running in debug mode")
	}

	// Initialize SQL Database
	model.InitDB()
	model.InitLogDB()

	defer func() {
		err := model.CloseDB()
		if err != nil {
			logger.FatalLog("failed to close database: " + err.Error())
		}
	}()

	// Initialize Redis
	err := common.InitRedisClient()
	if err != nil {
		logger.FatalLog("failed to initialize Redis: " + err.Error())
	}

	// Initialize options
	model.InitOptionMap()
	model.InitChannelCache()
	go model.SyncOptions(time.Second * 5)
	go model.SyncChannelCache(time.Second * 5)
	if os.Getenv("CHANNEL_TEST_FREQUENCY") != "" {
		frequency, err := strconv.Atoi(os.Getenv("CHANNEL_TEST_FREQUENCY"))
		if err != nil {
			logger.FatalLog("failed to parse CHANNEL_TEST_FREQUENCY: " + err.Error())
		}
		go controller.AutomaticallyTestChannels(frequency)
	}
	if config.EnableMetric {
		logger.SysLog("metric enabled, will disable channel if too much request failed")
	}
	openai.InitTokenEncoders()
	client.Init()

	// Initialize HTTP server
	server := gin.New()
	server.Use(gin.Recovery())
	server.Use(middleware.RequestId())
	middleware.SetUpLogger(server)
	// Initialize session store

	router.SetRouter(server)
	port := os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	logger.SysLogf("server started on http://localhost:%s", port)
	err = server.Run(":" + port)
	if err != nil {
		logger.FatalLog("failed to start HTTP server: " + err.Error())
	}
}
