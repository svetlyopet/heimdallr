package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/http/middleware"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

func TestLogMiddlewareWritesRequestLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	appLogger := logger.New(logger.Config{
		Format: logger.FormatText,
		Level:  slog.LevelInfo,
		Output: &output,
	})

	router := gin.New()
	router.Use(middleware.Log(appLogger))
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, output.String(), "http_request")
	require.Contains(t, output.String(), "/ping")
}

func TestRecoverMiddlewareHandlesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	appLogger := logger.New(logger.Config{
		Format: logger.FormatText,
		Level:  slog.LevelError,
		Output: &output,
	})

	router := gin.New()
	router.Use(middleware.Recover(appLogger))
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), http.StatusText(http.StatusInternalServerError))
	require.Contains(t, output.String(), "panic recovered")
}
