// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/shkotk/gochat/server/config"
)

func InitializeRouter(cfg config.Config) *gin.Engine {
	wire.Build(servicesSet)
	return &gin.Engine{}
}
