package cache

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v4"
)

// CacheService defines the interface for cache operations
type CacheService interface {
	// Set stores a string value in cache with an expiration time
	Set(ctx context.Context, key string, value string, expiration time.Duration) error

	// Get retrieves a string value from cache
	Get(ctx context.Context, key string) (string, error)

	// GetWithFallback retrieves a string value from cache, or executes fallback function if not found
	GetWithFallback(ctx context.Context, key string, fallback func() (string, error), expiration time.Duration) (string, error)

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

	// HealthCheck verifies cache connectivity
	HealthCheck(ctx context.Context) error

	// Redlock distributed locking functions
	NewMutex(name string, options ...redsync.Option) *redsync.Mutex
}
