package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
)

const apiRateLimitErrorMessage = "too many requests"

type APIRateLimitConfig struct {
	IPRate      float64
	IPBurst     float64
	UserRate    float64
	UserBurst   float64
	MaxKeys     int
	StaleKeyTTL time.Duration
}

type tokenBucket struct {
	tokens    float64
	lastCheck time.Time
}

type bucketStore struct {
	rate     float64
	burst    float64
	maxKeys  int
	staleTTL time.Duration
	buckets  map[string]tokenBucket
	lastSeen map[string]time.Time
	mu       sync.Mutex
	now      func() time.Time
}

func newBucketStore(rate, burst float64, maxKeys int, staleTTL time.Duration) *bucketStore {
	if rate <= 0 {
		rate = 20
	}
	if burst <= 0 {
		burst = 40
	}
	if maxKeys <= 0 {
		maxKeys = 10_000
	}
	if staleTTL <= 0 {
		staleTTL = time.Hour
	}

	return &bucketStore{
		rate:     rate,
		burst:    burst,
		maxKeys:  maxKeys,
		staleTTL: staleTTL,
		buckets:  map[string]tokenBucket{},
		lastSeen: map[string]time.Time{},
		now:      time.Now,
	}
}

func (s *bucketStore) cleanupStaleLocked(now time.Time) {
	cutoff := now.Add(-s.staleTTL)
	for key, seen := range s.lastSeen {
		if seen.Before(cutoff) {
			delete(s.lastSeen, key)
			delete(s.buckets, key)
		}
	}
}

func (s *bucketStore) evictOldestLocked() {
	var (
		oldestKey string
		oldest    time.Time
		found     bool
	)

	for key, seen := range s.lastSeen {
		if !found || seen.Before(oldest) {
			oldestKey = key
			oldest = seen
			found = true
		}
	}

	if found {
		delete(s.lastSeen, oldestKey)
		delete(s.buckets, oldestKey)
	}
}

func (s *bucketStore) allow(key string) (bool, time.Duration) {
	if key == "" {
		return true, 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	s.cleanupStaleLocked(now)

	bucket, ok := s.buckets[key]
	if !ok {
		if len(s.buckets) >= s.maxKeys {
			s.evictOldestLocked()
		}
		bucket = tokenBucket{tokens: s.burst, lastCheck: now}
	}

	elapsed := now.Sub(bucket.lastCheck).Seconds()
	bucket.tokens += elapsed * s.rate
	if bucket.tokens > s.burst {
		bucket.tokens = s.burst
	}
	bucket.lastCheck = now

	if bucket.tokens < 1 {
		deficit := 1 - bucket.tokens
		retrySeconds := int(deficit/s.rate) + 1
		if retrySeconds < 1 {
			retrySeconds = 1
		}

		s.buckets[key] = bucket
		s.lastSeen[key] = now
		return false, time.Duration(retrySeconds) * time.Second
	}

	bucket.tokens--
	s.buckets[key] = bucket
	s.lastSeen[key] = now
	return true, 0
}

type APIRateLimiter struct {
	ipStore   *bucketStore
	userStore *bucketStore
}

func NewAPIRateLimiter(cfg APIRateLimitConfig) *APIRateLimiter {
	return &APIRateLimiter{
		ipStore: newBucketStore(
			cfg.IPRate,
			cfg.IPBurst,
			cfg.MaxKeys,
			cfg.StaleKeyTTL,
		),
		userStore: newBucketStore(
			cfg.UserRate,
			cfg.UserBurst,
			cfg.MaxKeys,
			cfg.StaleKeyTTL,
		),
	}
}

func abortAPIRateLimited(ctx *gin.Context, retryAfter time.Duration, limit float64) {
	if retryAfter < time.Second {
		retryAfter = time.Second
	}
	ctx.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	ctx.Header("X-RateLimit-Limit", strconv.Itoa(int(limit)))
	ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": apiRateLimitErrorMessage})
}

func (l *APIRateLimiter) PreAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == "/api/health" {
			ctx.Next()
			return
		}

		allowed, retryAfter := l.ipStore.allow("ip:" + ctx.ClientIP())
		if !allowed {
			abortAPIRateLimited(ctx, retryAfter, l.ipStore.burst)
			return
		}

		ctx.Next()
	}
}

func (l *APIRateLimiter) PostAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == "/api/health" {
			ctx.Next()
			return
		}

		value, exists := ctx.Get("auth.user")
		if !exists {
			ctx.Next()
			return
		}

		user, ok := value.(authapi.AuthUser)
		if !ok || user.Id == "" {
			ctx.Next()
			return
		}

		allowed, retryAfter := l.userStore.allow("user:" + user.Id)
		if !allowed {
			abortAPIRateLimited(ctx, retryAfter, l.userStore.burst)
			return
		}

		ctx.Next()
	}
}
