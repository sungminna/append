package ratelimit

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter wraps golang.org/x/time/rate.Limiter for API rate limiting
type RateLimiter struct {
	limiter *rate.Limiter
	mu      sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the specified requests per second
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond),
	}
}

// Allow checks if a request can proceed without blocking
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// Wait blocks until the rate limiter allows a request or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// MultiRateLimiter manages multiple rate limiters for different API categories
type MultiRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
}

// NewMultiRateLimiter creates a new multi-rate limiter
func NewMultiRateLimiter(limiters map[string]*RateLimiter) *MultiRateLimiter {
	return &MultiRateLimiter{
		limiters: limiters,
	}
}

// Allow checks if a request can proceed for the specified category
func (mrl *MultiRateLimiter) Allow(category string) bool {
	mrl.mu.RLock()
	limiter, exists := mrl.limiters[category]
	mrl.mu.RUnlock()

	if !exists {
		return false
	}

	return limiter.Allow()
}

// Wait blocks until the rate limiter allows a request for the specified category
func (mrl *MultiRateLimiter) Wait(ctx context.Context, category string) error {
	mrl.mu.RLock()
	limiter, exists := mrl.limiters[category]
	mrl.mu.RUnlock()

	if !exists {
		return ErrCategoryNotFound
	}

	return limiter.Wait(ctx)
}

// Add adds a new rate limiter for a category
func (mrl *MultiRateLimiter) Add(category string, limiter *RateLimiter) {
	mrl.mu.Lock()
	defer mrl.mu.Unlock()
	mrl.limiters[category] = limiter
}

// ErrCategoryNotFound is returned when the specified rate limiter category doesn't exist
var ErrCategoryNotFound = &RateLimitError{message: "rate limiter category not found"}

// RateLimitError represents a rate limiting error
type RateLimitError struct {
	message string
}

func (e *RateLimitError) Error() string {
	return e.message
}
