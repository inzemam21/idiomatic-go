package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.WithFields(logrus.Fields{
			"method":  method,
			"path":    path,
			"status":  status,
			"latency": latency,
			"ip":      c.ClientIP(),
		}).Info("request processed")
	}
}
