package middleware

import (
	"strconv"
	"sync"
	"time"

	"iforge/iforge/internal/contextutil"

	"github.com/gofiber/fiber/v2"
)

// slidingWindowLimiter is an in-memory sliding-window rate limiter.
//
// Pattern reused from password_reset_handler.resetRateLimit (sync.Mutex +
// map[key][]time.Time), extended with a background goroutine that periodically
// evicts stale entries to bound memory growth.
//
// Single-process only. For multi-instance deployments, replace with a Redis
// backend (project currently has no Redis dependency).
type slidingWindowLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newSlidingWindowLimiter() *slidingWindowLimiter {
	l := &slidingWindowLimiter{attempts: make(map[string][]time.Time)}
	// Background cleanup every 5 minutes evicts keys with no recent activity,
	// bounding memory usage for high-cardinality keys (per-user, per-IP).
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			l.cleanup(time.Now())
		}
	}()
	return l
}

// cleanup removes all attempt timestamps older than the retention horizon.
// Called periodically to bound memory usage.
func (l *slidingWindowLimiter) cleanup(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// 1-hour retention horizon (longer than any configured window).
	cutoff := now.Add(-time.Hour)
	for key, ts := range l.attempts {
		var valid []time.Time
		for _, t := range ts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(l.attempts, key)
		} else {
			l.attempts[key] = valid
		}
	}
}

// allow returns true if the request should be permitted under the limit.
// Records the attempt either way (so a rejected request still counts toward
// the window, preventing retry storms from bypassing the limit).
func (l *slidingWindowLimiter) allow(key string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// Filter to in-window attempts.
	attempts := l.attempts[key]
	var valid []time.Time
	for _, t := range attempts {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= max {
		l.attempts[key] = valid
		return false
	}

	l.attempts[key] = append(valid, now)
	return true
}

// rateLimitConfig bundles limit parameters for a named limiter.
type rateLimitConfig struct {
	limiter *slidingWindowLimiter
	max     int
	window  time.Duration
}

var (
	// strictLimiter: 5 req/min by IP. Used on auth endpoints to defend
	// against brute-force login/registration/password-reset attacks.
	strictLimiter = &rateLimitConfig{
		limiter: newSlidingWindowLimiter(),
		max:     5,
		window:  time.Minute,
	}
	// generalLimiter: 100 req/min by user (or by IP if unauthenticated).
	// Mounted on all /api/v1/* routes to prevent any single user from
	// monopolizing server resources.
	generalLimiter = &rateLimitConfig{
		limiter: newSlidingWindowLimiter(),
		max:     500,
		window:  time.Minute,
	}
)

// StrictRateLimit applies a strict per-IP rate limit (5 req/min).
// Use on authentication endpoints: login, register, password reset.
//
// On limit exceeded: 429 Too Many Requests with Retry-After header.
func StrictRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := "ip:" + c.IP()
		if !strictLimiter.limiter.allow(key, strictLimiter.max, strictLimiter.window) {
			c.Set("X-RateLimit-Limit", strconv.Itoa(strictLimiter.max))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", strconv.Itoa(int(strictLimiter.window.Seconds())))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests, please try again later",
			})
		}
		return c.Next()
	}
}

// GeneralRateLimit applies a per-user (or per-IP if unauthenticated) rate
// limit (100 req/min). Mount on /api/v1/* after AuthMiddleware so the user
// context is available for the key.
//
// On limit exceeded: 429 Too Many Requests with Retry-After header.
func GeneralRateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var key string
		if user := contextutil.GetUserFromContext(c); user != nil {
			key = "user:" + user.UserName
		} else {
			key = "ip:" + c.IP()
		}
		if !generalLimiter.limiter.allow(key, generalLimiter.max, generalLimiter.window) {
			c.Set("X-RateLimit-Limit", strconv.Itoa(generalLimiter.max))
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", strconv.Itoa(int(generalLimiter.window.Seconds())))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests, please try again later",
			})
		}
		return c.Next()
	}
}
