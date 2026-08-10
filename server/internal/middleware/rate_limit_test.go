package middleware

import (
	"os"
	"testing"
	"time"
)

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	limiter := &slidingWindowLimiter{
		attempts: make(map[string][]time.Time),
	}

	key := "test-key"
	max := 3
	window := time.Minute

	// First 3 requests should be allowed
	for i := 0; i < max; i++ {
		if !limiter.allow(key, max, window) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be rejected
	if limiter.allow(key, max, window) {
		t.Error("4th request should be rejected")
	}

	// Verify attempts are recorded
	if len(limiter.attempts[key]) != max {
		t.Errorf("Expected %d attempts, got %d", max, len(limiter.attempts[key]))
	}
}

func TestSlidingWindowLimiter_WindowExpiry(t *testing.T) {
	limiter := &slidingWindowLimiter{
		attempts: make(map[string][]time.Time),
	}

	key := "test-key"
	max := 2
	window := 50 * time.Millisecond // Short window for testing

	// Fill the window
	limiter.allow(key, max, window)
	limiter.allow(key, max, window)

	// Should be rejected now
	if limiter.allow(key, max, window) {
		t.Error("Should be rejected when window is full")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !limiter.allow(key, max, window) {
		t.Error("Should be allowed after window expires")
	}
}

func TestSlidingWindowLimiter_Cleanup(t *testing.T) {
	limiter := &slidingWindowLimiter{
		attempts: make(map[string][]time.Time),
	}

	key1 := "key1"
	key2 := "key2"

	// Add old attempts (older than 1 hour)
	oldTime := time.Now().Add(-2 * time.Hour)
	limiter.attempts[key1] = []time.Time{oldTime}
	limiter.attempts[key2] = []time.Time{oldTime, time.Now()} // Mix of old and new

	// Run cleanup
	limiter.cleanup(time.Now())

	// key1 should be completely removed (all attempts old)
	if _, exists := limiter.attempts[key1]; exists {
		t.Error("key1 should be removed after cleanup")
	}

	// key2 should still exist with only the recent attempt
	if attempts, exists := limiter.attempts[key2]; !exists {
		t.Error("key2 should still exist")
	} else if len(attempts) != 1 {
		t.Errorf("key2 should have 1 attempt, got %d", len(attempts))
	}
}

func TestSlidingWindowLimiter_DifferentKeys(t *testing.T) {
	limiter := &slidingWindowLimiter{
		attempts: make(map[string][]time.Time),
	}

	key1 := "user:alice"
	key2 := "user:bob"
	max := 2
	window := time.Minute

	// Fill key1's window
	limiter.allow(key1, max, window)
	limiter.allow(key1, max, window)

	// key1 should be rejected
	if limiter.allow(key1, max, window) {
		t.Error("key1 should be rejected")
	}

	// key2 should still be allowed (independent limit)
	if !limiter.allow(key2, max, window) {
		t.Error("key2 should be allowed (independent limit)")
	}
}

func TestGetAllowedOrigins_Default(t *testing.T) {
	// Save and restore env
	origEnv := os.Getenv("IFORGE_CORS_ORIGINS")
	defer os.Setenv("IFORGE_CORS_ORIGINS", origEnv)

	os.Unsetenv("IFORGE_CORS_ORIGINS")

	origins := getAllowedOrigins()

	// Should have default localhost origins
	expected := []string{
		"http://localhost:3001",
		"http://localhost:3000",
		"http://localhost:8081",
	}

	for _, e := range expected {
		if !origins[e] {
			t.Errorf("Expected origin %s not found", e)
		}
	}
}

func TestGetAllowedOrigins_Custom(t *testing.T) {
	// Save and restore env
	origEnv := os.Getenv("IFORGE_CORS_ORIGINS")
	defer os.Setenv("IFORGE_CORS_ORIGINS", origEnv)

	os.Setenv("IFORGE_CORS_ORIGINS", "https://example.com,https://api.example.com")

	origins := getAllowedOrigins()

	if !origins["https://example.com"] {
		t.Error("Expected https://example.com to be allowed")
	}
	if !origins["https://api.example.com"] {
		t.Error("Expected https://api.example.com to be allowed")
	}
}

func TestGetAllowedOrigins_Wildcard(t *testing.T) {
	// Save and restore env
	origEnv := os.Getenv("IFORGE_CORS_ORIGINS")
	defer os.Setenv("IFORGE_CORS_ORIGINS", origEnv)

	os.Setenv("IFORGE_CORS_ORIGINS", "*")

	origins := getAllowedOrigins()

	if !origins["*"] {
		t.Error("Expected wildcard origin to be set")
	}
}

func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  map[string]bool
		expected bool
	}{
		{
			name:     "allowed origin",
			origin:   "http://localhost:3001",
			allowed:  map[string]bool{"http://localhost:3001": true},
			expected: true,
		},
		{
			name:     "not allowed origin",
			origin:   "http://evil.com",
			allowed:  map[string]bool{"http://localhost:3001": true},
			expected: false,
		},
		{
			name:     "wildcard allows all",
			origin:   "http://anything.com",
			allowed:  map[string]bool{"*": true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAllowedOrigin(tt.origin, tt.allowed)
			if result != tt.expected {
				t.Errorf("isAllowedOrigin(%s) = %v, want %v", tt.origin, result, tt.expected)
			}
		})
	}
}
