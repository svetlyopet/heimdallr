package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func Recover(appLogger *logger.Logger) gin.HandlerFunc {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				appLogger.Error(
					c.Request.Context(),
					"panic recovered",
					slog.Any("panic", recovered),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.String("client_ip", c.ClientIP()),
					slog.String("stack_trace", string(debug.Stack())),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": http.StatusText(http.StatusInternalServerError),
				})
			}
		}()

		c.Next()
	}
}
