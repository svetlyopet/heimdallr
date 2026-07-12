package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/http/middleware"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/requestlimits"
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

func TestRequestLimitsBoundsBodyAndPropagatesOutputLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RequestLimits(middleware.RequestLimitsConfig{
		MaxRequestBodyBytes:   4,
		MaxDecodedOutputBytes: 3,
		MaxPaginationLimit:    100,
	}))
	router.POST("/", func(ctx *gin.Context) {
		require.EqualValues(t, 3, requestlimits.MaxDecodedOutputBytes(ctx.Request.Context()))
		if _, err := io.ReadAll(ctx.Request.Body); err != nil {
			var maxBytesErr *http.MaxBytesError
			require.True(t, errors.As(err, &maxBytesErr))
			ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		ctx.Status(http.StatusNoContent)
	})

	exact := httptest.NewRecorder()
	router.ServeHTTP(exact, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("1234")))
	require.Equal(t, http.StatusNoContent, exact.Code)

	oversized := httptest.NewRecorder()
	router.ServeHTTP(oversized, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345")))
	require.Equal(t, http.StatusRequestEntityTooLarge, oversized.Code)
}

func TestRequestLimitsBoundsPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RequestLimits(middleware.RequestLimitsConfig{MaxPaginationLimit: 100}))
	router.GET("/", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/?limit=100", nil))
	require.Equal(t, http.StatusOK, allowed.Code)

	rejected := httptest.NewRecorder()
	router.ServeHTTP(rejected, httptest.NewRequest(http.MethodGet, "/?limit=101", nil))
	require.Equal(t, http.StatusBadRequest, rejected.Code)

	overflow := httptest.NewRecorder()
	router.ServeHTTP(overflow, httptest.NewRequest(http.MethodGet, "/?page=9223372036854775807&limit=2", nil))
	require.Equal(t, http.StatusBadRequest, overflow.Code)
}
