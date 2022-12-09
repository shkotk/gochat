package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return gin.RecoveryWithWriter(&logrusRecoveryWriter{logger})
}

type logrusRecoveryWriter struct {
	logger *logrus.Logger
}

func (w *logrusRecoveryWriter) Write(p []byte) (int, error) {
	w.logger.WithFields(logrus.Fields{
		"component": "gin",
		"action":    "recovery",
	}).Error(string(p))
	return len(p), nil
}
