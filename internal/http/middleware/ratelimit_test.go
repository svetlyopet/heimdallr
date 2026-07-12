package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
)

func TestAPIRateLimiterPreAuthBlocksIPFlood(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewAPIRateLimiter(APIRateLimitConfig{
		IPRate:      1,
		IPBurst:     1,
		UserRate:    100,
		UserBurst:   100,
		MaxKeys:     10,
		StaleKeyTTL: time.Minute,
	})

	router := gin.New()
	router.Use(limiter.PreAuthMiddleware())
	router.GET("/api/v1/resource", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))
	require.Equal(t, http.StatusOK, first.Code)

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	require.Equal(t, apiRateLimitErrorMessage, decodeAPIError(t, second))
	require.NotEmpty(t, second.Header().Get("Retry-After"))
	require.Equal(t, "1", second.Header().Get("X-RateLimit-Limit"))
}

func TestAPIRateLimiterSkipsHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewAPIRateLimiter(APIRateLimitConfig{
		IPRate:      1,
		IPBurst:     1,
		UserRate:    1,
		UserBurst:   1,
		MaxKeys:     10,
		StaleKeyTTL: time.Minute,
	})

	router := gin.New()
	router.Use(limiter.PreAuthMiddleware())
	router.GET("/api/health", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/health", nil))
		require.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestAPIRateLimiterPostAuthIsolatesUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewAPIRateLimiter(APIRateLimitConfig{
		IPRate:      100,
		IPBurst:     100,
		UserRate:    1,
		UserBurst:   1,
		MaxKeys:     10,
		StaleKeyTTL: time.Minute,
	})

	router := gin.New()
	router.Use(setUserMiddleware("user-a"))
	router.Use(limiter.PostAuthMiddleware())
	router.GET("/api/v1/resource", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))
	require.Equal(t, http.StatusOK, first.Code)

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))
	require.Equal(t, http.StatusTooManyRequests, second.Code)

	routerOtherUser := gin.New()
	routerOtherUser.Use(setUserMiddleware("user-b"))
	routerOtherUser.Use(limiter.PostAuthMiddleware())
	routerOtherUser.GET("/api/v1/resource", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	other := httptest.NewRecorder()
	routerOtherUser.ServeHTTP(other, httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))
	require.Equal(t, http.StatusOK, other.Code)
}

func TestAPIRateLimiterTokenBucketRefillsWithFakeClock(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	store := newBucketStore(1, 1, 10, time.Minute)
	store.now = func() time.Time { return now }

	allowed, _ := store.allow("ip:test")
	require.True(t, allowed)

	allowed, retryAfter := store.allow("ip:test")
	require.False(t, allowed)
	require.GreaterOrEqual(t, retryAfter, time.Second)

	now = now.Add(2 * time.Second)

	allowed, _ = store.allow("ip:test")
	require.True(t, allowed)
}

func TestAPIRateLimiterMaxKeysBounded(t *testing.T) {
	store := newBucketStore(10, 10, 2, time.Minute)

	for _, key := range []string{"a", "b", "c"} {
		allowed, _ := store.allow(key)
		require.True(t, allowed)
		time.Sleep(time.Millisecond)
	}

	require.LessOrEqual(t, len(store.buckets), 2)
}

func TestAPIRateLimiterConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewAPIRateLimiter(APIRateLimitConfig{
		IPRate:      100,
		IPBurst:     100,
		MaxKeys:     100,
		StaleKeyTTL: time.Minute,
	})

	router := gin.New()
	router.Use(limiter.PreAuthMiddleware())
	router.GET("/api/v1/resource", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil))
			require.Contains(t, []int{http.StatusOK, http.StatusTooManyRequests}, rr.Code)
		}()
	}
	wg.Wait()
}

func setUserMiddleware(userID string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("auth.user", authapi.AuthUser{Id: userID, Username: userID})
		ctx.Next()
	}
}

func decodeAPIError(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()

	var payload struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))
	return payload.Error
}
