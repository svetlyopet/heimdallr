package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type LoginRateLimiter struct {
	maxAttempts int
	window      time.Duration
	attempts    map[string][]time.Time
	mu          sync.Mutex
}

func NewLoginRateLimiter(maxAttempts int, window time.Duration) *LoginRateLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}

	if window <= 0 {
		window = 15 * time.Minute
	}

	return &LoginRateLimiter{
		maxAttempts: maxAttempts,
		window:      window,
		attempts:    map[string][]time.Time{},
	}
}

func (l *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.ClientIP()
		now := time.Now()

		l.mu.Lock()
		times := l.attempts[key]
		cutoff := now.Add(-l.window)
		filtered := times[:0]
		for _, attempt := range times {
			if attempt.After(cutoff) {
				filtered = append(filtered, attempt)
			}
		}

		if len(filtered) >= l.maxAttempts {
			l.attempts[key] = filtered
			l.mu.Unlock()
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts"})
			return
		}

		filtered = append(filtered, now)
		l.attempts[key] = filtered
		l.mu.Unlock()

		ctx.Next()
	}
}
