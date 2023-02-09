// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/shkotk/gochat/server/config"
	"github.com/shkotk/gochat/server/controllers"
	"github.com/shkotk/gochat/server/repositories"
	"github.com/shkotk/gochat/server/services"
)

// Injectors from wire.go:

func InitializeRouter(cfg config.Config) *gin.Engine {
	logger := setupLogger(cfg)
	jwtManager := services.NewJWTManager(cfg)
	db := setupDB(cfg, logger)
	userRepository := repositories.NewUserRepository(logger, db)
	userController := controllers.NewUserController(logger, userRepository, jwtManager)
	eventPreProcessor := services.NewEventPreProcessor()
	chatManager := services.NewChatManager(logger, eventPreProcessor)
	chatController := controllers.NewChatController(logger, jwtManager, chatManager)
	engine := setupRouter(cfg, logger, jwtManager, userController, chatController)
	return engine
}
