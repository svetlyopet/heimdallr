package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func Log(appLogger *logger.Logger) gin.HandlerFunc {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		request := c.Request

		attrs := []slog.Attr{
			slog.String("method", request.Method),
			slog.String("path", request.URL.Path),
			slog.String("query", request.URL.RawQuery),
			slog.Int("status", statusCode),
			slog.Int("body_size", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", request.UserAgent()),
			slog.Duration("latency", latency),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("gin_errors", c.Errors.String()))
		}

		switch {
		case statusCode >= http.StatusInternalServerError:
			appLogger.Error(request.Context(), "http_request", attrs...)
		case statusCode >= http.StatusBadRequest:
			appLogger.Warn(request.Context(), "http_request", attrs...)
		default:
			appLogger.Info(request.Context(), "http_request", attrs...)
		}
	}
}
