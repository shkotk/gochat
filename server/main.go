package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/shkotk/gochat/common/validation"
	"github.com/shkotk/gochat/server/controllers"
	"github.com/shkotk/gochat/server/core"
	"github.com/shkotk/gochat/server/middleware"
	"github.com/shkotk/gochat/server/models"
	"github.com/shkotk/gochat/server/repositories"
	"github.com/shkotk/gochat/server/services"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// TODO split to improve readability and testability
func main() {
	// Load environment variables
	godotenv.Load(".env.local")
	godotenv.Load(".env")
	// Or if server is run in workspace mode
	godotenv.Load("server/.env.local")
	godotenv.Load("server/.env")

	debug := os.Getenv("DEBUG") == "1"

	// Setup logger
	logger := logrus.New()
	if !debug {
		logger.SetOutput(os.Stdout)
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logger.WithError(err).Fatal("Can't parse log level")
	}
	logger.SetLevel(logLevel)

	// Setup ORM
	gormLogLevel := gormlogger.Warn
	logParameterizedQueries := true
	if debug {
		gormLogLevel = gormlogger.Info
		logParameterizedQueries = false
	}
	connString := readRequiredConfig("PG_CONNECTION_STRING", logger)
	db, err := gorm.Open(postgres.Open(connString), &gorm.Config{
		Logger: gormlogger.New(
			logger.WithField("component", "gorm"),
			gormlogger.Config{
				LogLevel:             gormLogLevel,
				ParameterizedQueries: logParameterizedQueries,
			}),
	})
	if err != nil {
		logger.WithError(err).Fatal("Can't connect to DB")
	}

	db.AutoMigrate(models.User{}) // TODO add migrations?

	// Register custom validators
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("name", validation.IsValidName)
	}

	// Setup services
	// TODO use wire?
	jwtKey := readRequiredConfig("JWT_KEY", logger)
	jwtExpirationStr := readRequiredConfig("JWT_EXPIRATION", logger)
	jwtExpiration, err := time.ParseDuration(jwtExpirationStr)
	if err != nil {
		logger.WithError(err).Fatalf(
			"Can't parse Duration from 'JWT_EXPIRATION' config value '%v'",
			jwtExpirationStr)
	}
	jwtManager := services.NewJWTManager(jwtKey, jwtExpiration)
	userRepository := repositories.NewUserRepository(logger, db)
	chatManager := core.NewChatManager(logger)

	userController := controllers.NewUserController(logger, userRepository, jwtManager)
	chatController := controllers.NewChatController(logger, jwtManager, chatManager)

	// Setup router
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.SetTrustedProxies(nil)
	router.Use(middleware.Logger(logger), middleware.Recovery(logger))
	jwtRouterGroup := router.Group("", middleware.JWT(jwtManager))

	router.GET("/user/exists/:username", userController.Exists)
	router.POST("/user/register", userController.Register)
	router.GET("/token/get", userController.GetToken)
	jwtRouterGroup.GET("/token/refresh", userController.RefreshToken)

	jwtRouterGroup.POST("/chat/create/:chatName", chatController.Create)
	jwtRouterGroup.GET("/chat/list", chatController.List)
	jwtRouterGroup.GET("/chat/join/:chatName", chatController.Join)

	// Start router
	port := readRequiredConfig("PORT", logger)
	certPath := readRequiredConfig("SSL_CERT_PATH", logger)
	keyPath := readRequiredConfig("SSL_KEY_PATH", logger)
	router.RunTLS(":"+port, certPath, keyPath)
}

func readRequiredConfig(configKey string, logger *logrus.Logger) string {
	configValue := os.Getenv(configKey)
	if configValue == "" {
		logger.Fatalf("'%v' config is required, but was empty or missing", configKey)
	}

	return configValue
}
