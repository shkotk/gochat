// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/shkotk/gochat/server/config"
	"github.com/shkotk/gochat/server/controllers"
	"github.com/shkotk/gochat/server/core"
	"github.com/shkotk/gochat/server/repositories"
	"github.com/shkotk/gochat/server/services"
)

func InitializeRouter(cfg config.Config) *gin.Engine {
	wire.Build(
		setupLogger,
		setupDB,
		services.NewJWTManager,
		repositories.NewUserRepository,
		core.NewChatManager,
		controllers.NewUserController,
		controllers.NewChatController,
		setupRouter,
	)

	return &gin.Engine{}
}
