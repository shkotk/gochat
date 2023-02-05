package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/shkotk/gochat/common/validation"
	"github.com/shkotk/gochat/server/config"
	"github.com/shkotk/gochat/server/controllers"
	"github.com/shkotk/gochat/server/middleware"
	"github.com/shkotk/gochat/server/models"
	"github.com/shkotk/gochat/server/services"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load(
		".env.local", ".env",
		"server/.env.local", "server/.env", // if running in workspace mode
	)

	// Register custom validators // TODO move validator configuration to common
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("name", validation.IsValidName)
	}

	err := InitializeRouter(cfg).RunTLS(
		fmt.Sprintf(":%d", cfg.Port),
		cfg.TLS.CertPath,
		cfg.TLS.KeyPath)
	if err != nil {
		log.Fatalf("error running roter: %s", err)
	}
}

func setupLogger(cfg config.Config) *logrus.Logger {
	logger := logrus.New()
	if !cfg.Debug {
		logger.SetOutput(os.Stdout)
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.WithError(err).Fatal("Can't parse log level")
	}
	logger.SetLevel(logLevel)

	return logger
}

func setupDB(cfg config.Config, logger *logrus.Logger) *gorm.DB {
	logLevel := gormlogger.Warn
	logParameterizedQueries := true
	if cfg.Debug {
		logLevel = gormlogger.Info
		logParameterizedQueries = false
	}
	db, err := gorm.Open(postgres.Open(cfg.PGConnString), &gorm.Config{
		Logger: gormlogger.New(
			logger.WithField("component", "gorm"),
			gormlogger.Config{
				LogLevel:             logLevel,
				ParameterizedQueries: logParameterizedQueries,
			}),
	})
	if err != nil {
		logger.WithError(err).Fatal("Can't connect to DB")
	}

	err = db.AutoMigrate(models.User{}) // TODO add migrations?
	if err != nil {
		logger.WithError(err).Fatal("Can't apply automatic migration")
	}

	return db
}

func setupRouter(
	cfg config.Config,
	logger *logrus.Logger,
	jwtManager *services.JWTManager,
	userController *controllers.UserController,
	chatController *controllers.ChatController,
) *gin.Engine {
	if !cfg.Debug {
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

	return router
}
