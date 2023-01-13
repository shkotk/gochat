package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shkotk/gochat/common/apimodels/responses"
	"github.com/shkotk/gochat/server/services"
)

const UserClaimsKey = "USER_CLAIMS"

func JWT(manager *services.JWTManager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, claims, err := manager.ParseToken(ctx)
		if err != nil || !token.Valid {
			ctx.Error(err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, responses.Error{Error: err.Error()})
			return
		}

		ctx.Set(UserClaimsKey, claims)

		ctx.Next()
	}
}
