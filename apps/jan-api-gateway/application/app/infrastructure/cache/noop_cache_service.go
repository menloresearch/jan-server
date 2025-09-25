package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
)

// NoOpCacheService provides a no-operation cache service for graceful degradation
type NoOpCacheService struct{}

// Set is a no-op implementation
func (n *NoOpCacheService) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return nil
}

// Get always returns "key not found" error
func (n *NoOpCacheService) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("key not found: %s", key)
}

// GetWithFallback always executes the fallback function
func (n *NoOpCacheService) GetWithFallback(ctx context.Context, key string, fallback func() (string, error), expiration time.Duration) (string, error) {
	return fallback()
}

// Delete is a no-op implementation
func (n *NoOpCacheService) Delete(ctx context.Context, key string) error {
	return nil
}

// Unlink is a no-op implementation
func (n *NoOpCacheService) Unlink(ctx context.Context, key string) error {
	return nil
}

// DeletePattern is a no-op implementation
func (n *NoOpCacheService) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

// Exists always returns false
func (n *NoOpCacheService) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// Close is a no-op implementation
func (n *NoOpCacheService) Close() error {
	return nil
}

// HealthCheck always returns nil (healthy)
func (n *NoOpCacheService) HealthCheck(ctx context.Context) error {
	return nil
}

// NewMutex is a no-op implementation (always succeeds)
func (n *NoOpCacheService) NewMutex(name string, options ...redsync.Option) *redsync.Mutex {
	// Return a no-op mutex that always succeeds
	return &redsync.Mutex{}
}
