package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// NoOpCacheService provides a no-operation cache service for graceful degradation
type NoOpCacheService struct{}

// Set is a no-op implementation
func (n *NoOpCacheService) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return nil
}

// Get always returns "key not found" error
func (n *NoOpCacheService) Get(ctx context.Context, key string, dest any) error {
	return fmt.Errorf("key not found: %s", key)
}

// GetWithFallback always executes the fallback function
func (n *NoOpCacheService) GetWithFallback(ctx context.Context, key string, dest any, fallback func() (any, error), expiration time.Duration) error {
	value, err := fallback()
	if err != nil {
		return fmt.Errorf("fallback function failed: %w", err)
	}

	// Copy the value to dest
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal fallback value: %w", err)
	}

	return json.Unmarshal(jsonValue, dest)
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
