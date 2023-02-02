package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shkotk/gochat/common/apimodels/responses"
	"github.com/shkotk/gochat/server/core"
	"github.com/shkotk/gochat/server/middleware"
	"github.com/shkotk/gochat/server/services"
	"github.com/sirupsen/logrus"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type ChatController struct {
	logger      *logrus.Logger
	jwtManager  *services.JWTManager
	chatManager *core.ChatManager
}

func NewChatController(
	logger *logrus.Logger,
	jwtManager *services.JWTManager,
	chatManager *core.ChatManager,
) *ChatController {
	return &ChatController{logger, jwtManager, chatManager}
}

type createRequest struct {
	ChatName string `uri:"chatName" binding:"required,name"`
}

func (c *ChatController) Create(ctx *gin.Context) {
	var request createRequest
	if err := ctx.ShouldBindUri(&request); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	if err := c.chatManager.Create(request.ChatName); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()}) // TODO may be 500
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *ChatController) List(ctx *gin.Context) {
	chats, err := c.chatManager.List()
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, responses.Chats{Chats: chats})
}

type joinRequest struct {
	ChatName string `uri:"chatName" binding:"required,name"`
}

func (c *ChatController) Join(ctx *gin.Context) {
	claims := ctx.MustGet(middleware.UserClaimsKey).(services.UserClaims)

	var request joinRequest
	if err := ctx.ShouldBindUri(&request); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	conn, err := websocketUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to upgrade connection.")
		ctx.Error(err)
		if !ctx.Writer.Written() {
			ctx.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		}
		return
	}

	err = c.chatManager.Join(claims.Username, conn, request.ChatName)
	if err != nil {
		c.logger.WithError(err).Warnf(
			"Failed to add user '%s' to chat '%s'.", claims.Username, request.ChatName)
		ctx.Error(err)

		if err = conn.Close(); err != nil {
			c.logger.WithError(err).Warnf(
				"Failed to close connection with user '%s'.", request.ChatName)
			ctx.Error(err)
		}
	}
}
