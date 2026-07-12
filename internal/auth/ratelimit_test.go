package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLoginRateLimiterBlocksRepeatedFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewLoginRateLimiter(2, time.Minute, 100)

	router := gin.New()
	router.POST("/login", limiter.Middleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusUnauthorized)
	})

	for i := 0; i < 2; i++ {
		req := loginRequest(t, "user", "wrong")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	}

	req := loginRequest(t, "user", "wrong")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusTooManyRequests, rr.Code)
	require.Equal(t, rateLimitErrorMessage, decodeError(t, rr))
	require.NotEmpty(t, rr.Header().Get("Retry-After"))
}

func TestLoginRateLimiterSuccessfulLoginDoesNotConsumeBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewLoginRateLimiter(2, time.Minute, 100)

	router := gin.New()
	router.POST("/login", limiter.Middleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		req := loginRequest(t, "user", "correct")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestLoginRateLimiterSuccessClearsUsernameIPBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewLoginRateLimiter(10, time.Minute, 100)

	router := gin.New()
	router.POST("/login", limiter.Middleware(), func(ctx *gin.Context) {
		var body loginBody
		if err := ctx.ShouldBindJSON(&body); err != nil {
			ctx.Status(http.StatusUnauthorized)
			return
		}
		if body.Password == "correct" {
			ctx.Status(http.StatusOK)
			return
		}
		ctx.Status(http.StatusUnauthorized)
	})

	for i := 0; i < 2; i++ {
		req := loginRequest(t, "user", "wrong")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	}

	userKey := loginUsernameIPKey("user", "127.0.0.1")
	require.Len(t, limiter.tracker.entries[userKey].attempts, 2)

	success := loginRequest(t, "user", "correct")
	successRR := httptest.NewRecorder()
	router.ServeHTTP(successRR, success)
	require.Equal(t, http.StatusOK, successRR.Code)

	_, hasUserKey := limiter.tracker.entries[userKey]
	require.False(t, hasUserKey)
}

func TestFailureTrackerClear(t *testing.T) {
	tracker := newFailureTracker(2, time.Minute, 100)
	key := "test-key"

	tracker.recordFailure(key)
	tracker.recordFailure(key)

	blocked, _ := tracker.isBlocked(key)
	require.True(t, blocked)

	tracker.clear(key)

	blocked, _ = tracker.isBlocked(key)
	require.False(t, blocked)
}

func TestLoginRateLimiterDifferentUsernamesShareIPBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewLoginRateLimiter(3, time.Minute, 100)

	router := gin.New()
	router.POST("/login", limiter.Middleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusUnauthorized)
	})

	for _, username := range []string{"alice", "bob", "carol"} {
		req := loginRequest(t, username, "wrong")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	}

	req := loginRequest(t, "dave", "wrong")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestLoginRateLimiterWindowExpiryRestoresAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginRateLimiter(2, time.Minute, 100)
	limiter.tracker.now = func() time.Time { return now }

	router := gin.New()
	router.POST("/login", limiter.Middleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusUnauthorized)
	})

	for i := 0; i < 2; i++ {
		req := loginRequest(t, "user", "wrong")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	}

	blocked := loginRequest(t, "user", "wrong")
	blockedRR := httptest.NewRecorder()
	router.ServeHTTP(blockedRR, blocked)
	require.Equal(t, http.StatusTooManyRequests, blockedRR.Code)

	now = now.Add(2 * time.Minute)

	allowed := loginRequest(t, "user", "wrong")
	allowedRR := httptest.NewRecorder()
	router.ServeHTTP(allowedRR, allowed)
	require.Equal(t, http.StatusUnauthorized, allowedRR.Code)
}

func TestLoginRateLimiterMaxKeysEviction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewLoginRateLimiter(10, time.Minute, 2)

	router := gin.New()
	router.POST("/login", limiter.Middleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusUnauthorized)
	})

	for _, username := range []string{"one", "two", "three"} {
		req := loginRequest(t, username, "wrong")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
	}

	require.LessOrEqual(t, len(limiter.tracker.entries), 2)
}

func TestLoginRateLimiterConcurrentAttempts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewLoginRateLimiter(100, time.Minute, 1000)

	router := gin.New()
	router.POST("/login", limiter.Middleware(), func(ctx *gin.Context) {
		ctx.Status(http.StatusUnauthorized)
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := loginRequest(t, "user"+string(rune('a'+idx%26)), "wrong")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			require.Contains(t, []int{http.StatusUnauthorized, http.StatusTooManyRequests}, rr.Code)
		}(i)
	}
	wg.Wait()
}

func loginRequest(t *testing.T, username, password string) *http.Request {
	t.Helper()

	body, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	return req
}

func decodeError(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()

	var payload struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))
	return payload.Error
}

func TestLoginUsernameIPKeyUsesNormalizedUsername(t *testing.T) {
	ip := "127.0.0.1"
	require.Equal(t, loginUsernameIPKey(" User ", ip), loginUsernameIPKey(strings.ToLower("User"), ip))
}
