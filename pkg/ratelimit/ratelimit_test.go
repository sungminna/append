package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name           string
		requestsPerSec int
		requests       int
		expectedPasses int
		duration       time.Duration
	}{
		{
			name:           "should allow requests within limit",
			requestsPerSec: 10,
			requests:       10,
			expectedPasses: 10,
			duration:       1 * time.Second,
		},
		{
			name:           "should block requests exceeding limit",
			requestsPerSec: 5,
			requests:       10,
			expectedPasses: 5,
			duration:       500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.requestsPerSec)

			passed := 0
			for i := 0; i < tt.requests; i++ {
				if limiter.Allow() {
					passed++
				}
			}

			assert.LessOrEqual(t, passed, tt.expectedPasses)
		})
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	limiter := NewRateLimiter(10) // 10 requests per second

	ctx := context.Background()
	start := time.Now()

	// First request should pass immediately
	err := limiter.Wait(ctx)
	assert.NoError(t, err)

	// Fill up the bucket
	for i := 0; i < 9; i++ {
		err := limiter.Wait(ctx)
		assert.NoError(t, err)
	}

	elapsed := time.Since(start)
	assert.Less(t, elapsed, 200*time.Millisecond)
}

func TestRateLimiter_WaitWithContext(t *testing.T) {
	limiter := NewRateLimiter(1) // 1 request per second

	// Fill the bucket
	limiter.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	assert.Error(t, err)
	// The error should be related to context deadline
	assert.Contains(t, err.Error(), "context deadline")
}

func TestRateLimiter_Concurrent(t *testing.T) {
	limiter := NewRateLimiter(100) // 100 requests per second

	var wg sync.WaitGroup
	passed := 0
	var mu sync.Mutex

	// Simulate 200 concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow() {
				mu.Lock()
				passed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow approximately 100 requests in a short burst
	assert.LessOrEqual(t, passed, 150) // Some tolerance for timing
	assert.GreaterOrEqual(t, passed, 50)
}

func TestMultiRateLimiter(t *testing.T) {
	quotationLimiter := NewRateLimiter(30) // Upbit Quotation API: 30 req/sec
	exchangeLimiter := NewRateLimiter(8)   // Upbit Exchange API: 8 req/sec

	multi := NewMultiRateLimiter(map[string]*RateLimiter{
		"quotation": quotationLimiter,
		"exchange":  exchangeLimiter,
	})

	// Test quotation limiter
	assert.True(t, multi.Allow("quotation"))

	// Test exchange limiter
	assert.True(t, multi.Allow("exchange"))

	// Test non-existent limiter
	assert.False(t, multi.Allow("nonexistent"))
}
