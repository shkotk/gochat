package middleware

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		method := ctx.Request.Method
		path := ctx.Request.URL.Path
		statusCode := ctx.Writer.Status()
		elapsed := time.Since(start)

		entry := logger.WithFields(logrus.Fields{
			"method":     method,
			"path":       path,
			"statusCode": statusCode,
			"elapsed":    elapsed.Milliseconds(),
		})

		entry = entry.WithError(errors.New(
			strings.Join(ctx.Errors.Errors(), "; ")))

		msg := fmt.Sprintf("%v %v %v (%s)", method, path, statusCode, elapsed)
		switch {
		case statusCode >= 500:
			entry.Error(msg)
		case statusCode >= 400:
			entry.Warning(msg)
		default:
			entry.Info(msg)
		}
	}
}
