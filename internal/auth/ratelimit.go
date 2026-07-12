package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const rateLimitErrorMessage = "too many requests"

type failureEntry struct {
	attempts []time.Time
}

type failureTracker struct {
	maxAttempts int
	window      time.Duration
	maxKeys     int
	entries     map[string]failureEntry
	mu          sync.Mutex
	now         func() time.Time
}

func newFailureTracker(maxAttempts int, window time.Duration, maxKeys int) *failureTracker {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if maxKeys <= 0 {
		maxKeys = 10_000
	}

	return &failureTracker{
		maxAttempts: maxAttempts,
		window:      window,
		maxKeys:     maxKeys,
		entries:     map[string]failureEntry{},
		now:         time.Now,
	}
}

func (t *failureTracker) pruneAttempts(attempts []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-t.window)
	filtered := attempts[:0]
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func (t *failureTracker) retryAfter(attempts []time.Time, now time.Time) time.Duration {
	if len(attempts) == 0 {
		return 0
	}

	expiry := attempts[0].Add(t.window)
	if !expiry.After(now) {
		return 0
	}

	seconds := int(expiry.Sub(now).Seconds()) + 1
	if seconds < 1 {
		seconds = 1
	}

	return time.Duration(seconds) * time.Second
}

func (t *failureTracker) lazyCleanupLocked(now time.Time) {
	for key, entry := range t.entries {
		pruned := t.pruneAttempts(entry.attempts, now)
		if len(pruned) == 0 {
			delete(t.entries, key)
			continue
		}
		entry.attempts = pruned
		t.entries[key] = entry
	}
}

func (t *failureTracker) evictOldestLocked(now time.Time) {
	var (
		oldestKey string
		oldestExp time.Time
		found     bool
	)

	for key, entry := range t.entries {
		pruned := t.pruneAttempts(entry.attempts, now)
		if len(pruned) == 0 {
			continue
		}

		expiry := pruned[0].Add(t.window)
		if !found || expiry.Before(oldestExp) {
			oldestKey = key
			oldestExp = expiry
			found = true
		}
	}

	if found {
		delete(t.entries, oldestKey)
	}
}

func (t *failureTracker) ensureCapacityLocked() {
	if len(t.entries) < t.maxKeys {
		return
	}
	t.evictOldestLocked(t.now())
}

func (t *failureTracker) isBlocked(key string) (bool, time.Duration) {
	if key == "" {
		return false, 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	t.lazyCleanupLocked(now)

	entry, ok := t.entries[key]
	if !ok {
		return false, 0
	}

	attempts := t.pruneAttempts(entry.attempts, now)
	if len(attempts) == 0 {
		delete(t.entries, key)
		return false, 0
	}

	entry.attempts = attempts
	t.entries[key] = entry

	if len(attempts) >= t.maxAttempts {
		return true, t.retryAfter(attempts, now)
	}

	return false, 0
}

func (t *failureTracker) recordFailure(key string) {
	if key == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	t.lazyCleanupLocked(now)

	if _, ok := t.entries[key]; !ok {
		t.ensureCapacityLocked()
	}

	entry := t.entries[key]
	attempts := t.pruneAttempts(entry.attempts, now)
	attempts = append(attempts, now)
	t.entries[key] = failureEntry{attempts: attempts}
}

func (t *failureTracker) clear(key string) {
	if key == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, key)
}

type LoginRateLimiter struct {
	tracker *failureTracker
}

func NewLoginRateLimiter(maxAttempts int, window time.Duration, maxKeys int) *LoginRateLimiter {
	return &LoginRateLimiter{
		tracker: newFailureTracker(maxAttempts, window, maxKeys),
	}
}

func normalizeLoginUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func loginUsernameIPKey(username, clientIP string) string {
	normalized := normalizeLoginUsername(username)
	sum := sha256.Sum256([]byte(normalized + clientIP))
	return hex.EncodeToString(sum[:])
}

func loginIPKey(clientIP string) string {
	return "ip:" + clientIP
}

func abortRateLimited(ctx *gin.Context, retryAfter time.Duration) {
	if retryAfter < time.Second {
		retryAfter = time.Second
	}
	ctx.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": rateLimitErrorMessage})
}

func (l *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := ctx.ClientIP()
		username := extractLoginUsername(ctx)

		if blocked, retryAfter := l.tracker.isBlocked(loginIPKey(clientIP)); blocked {
			abortRateLimited(ctx, retryAfter)
			return
		}

		if username != "" {
			if blocked, retryAfter := l.tracker.isBlocked(loginUsernameIPKey(username, clientIP)); blocked {
				abortRateLimited(ctx, retryAfter)
				return
			}
		}

		ctx.Next()

		switch ctx.Writer.Status() {
		case http.StatusOK:
			if username != "" {
				l.tracker.clear(loginUsernameIPKey(username, clientIP))
			}
		case http.StatusUnauthorized:
			l.tracker.recordFailure(loginIPKey(clientIP))
			if username != "" {
				l.tracker.recordFailure(loginUsernameIPKey(username, clientIP))
			}
		}
	}
}

type loginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func extractLoginUsername(ctx *gin.Context) string {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return ""
	}

	ctx.Request.Body = io.NopCloser(bytes.NewReader(body))

	var parsed loginBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}

	return normalizeLoginUsername(parsed.Username)
}
