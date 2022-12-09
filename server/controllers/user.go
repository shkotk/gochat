package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shkotk/gochat/common/apimodels/requests"
	"github.com/shkotk/gochat/common/apimodels/responses"
	"github.com/shkotk/gochat/server/models"
	"github.com/shkotk/gochat/server/repositories"
	"github.com/shkotk/gochat/server/services"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	logger         *logrus.Logger
	userRepository *repositories.UserRepository
	jwtManager     *services.JWTManager
}

func NewUserController(
	logger *logrus.Logger,
	userRepository *repositories.UserRepository,
	jwtManager *services.JWTManager,
) *UserController {
	return &UserController{logger, userRepository, jwtManager}
}

type existsRequest struct {
	Username string `uri:"username" binding:"required,username"`
}

func (c *UserController) Exists(ctx *gin.Context) {
	var request existsRequest
	if err := ctx.ShouldBindUri(&request); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	exists, err := c.userRepository.Exists(request.Username)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, responses.Exists{Exists: exists})
}

func (c *UserController) Register(ctx *gin.Context) {
	var request requests.Auth
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	passwordHash, err :=
		bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}
	user := models.User{
		Username:     request.Username,
		PasswordHash: string(passwordHash),
	}
	if err := c.userRepository.Create(user); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *UserController) GetToken(ctx *gin.Context) {
	var request requests.Auth
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, responses.Error{Error: err.Error()})
		return
	}

	user, err := c.userRepository.Get(request.Username)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	if user == nil {
		ctx.JSON(http.StatusNotFound, responses.Error{
			Error: fmt.Sprintf("User '%v' does not exist", request.Username),
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	if err != nil {
		ctx.Error(err)

		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			ctx.JSON(http.StatusUnauthorized, responses.Error{Error: err.Error()})
			return
		}

		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	token, err := c.jwtManager.IssueToken(request.Username)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, responses.Token{Token: token})
}

func (c *UserController) RefreshToken(ctx *gin.Context) {
	token, claims, err := c.jwtManager.ParseToken(ctx)
	if err != nil || !token.Valid {
		ctx.Error(err)
		ctx.JSON(http.StatusUnauthorized, responses.Error{Error: err.Error()})
		return
	}

	refreshedTokenString, err := c.jwtManager.IssueToken(claims.Username)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, responses.Token{Token: refreshedTokenString})
}
