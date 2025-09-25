package cache

import (
	"context"
	"time"
)

// CacheService defines the interface for cache operations
type CacheService interface {
	// Set stores a value in cache with an expiration time
	Set(ctx context.Context, key string, value any, expiration time.Duration) error

	// Get retrieves a value from cache
	Get(ctx context.Context, key string, dest any) error

	// GetWithFallback retrieves a value from cache, or executes fallback function if not found
	GetWithFallback(ctx context.Context, key string, dest any, fallback func() (any, error), expiration time.Duration) error

	// Delete removes a key from cache synchronously (blocking)
	Delete(ctx context.Context, key string) error

	// Unlink removes a key from cache asynchronously (non-blocking)
	Unlink(ctx context.Context, key string) error

	// DeletePattern removes all keys matching a pattern
	DeletePattern(ctx context.Context, pattern string) error

	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)

	// Close closes the cache connection
	Close() error
}
