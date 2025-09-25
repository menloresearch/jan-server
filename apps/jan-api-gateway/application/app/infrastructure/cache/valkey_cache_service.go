package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/valkey-io/valkey-go"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

// ValkeyCacheService provides caching functionality using Valkey
type ValkeyCacheService struct {
	client valkey.Client
}

// parseValkeyURL parses a Valkey URL and returns address, password, database, and error
func parseValkeyURL(valkeyURL string) (address, password string, database int, err error) {
	// Default values
	database = -1 // -1 means no database specified

	// Handle plain address without protocol
	if !strings.Contains(valkeyURL, "://") {
		return valkeyURL, "", -1, nil
	}

	// Parse the URL
	u, err := url.Parse(valkeyURL)
	if err != nil {
		return "", "", -1, fmt.Errorf("invalid URL format: %w", err)
	}

	// Extract host and port
	address = u.Host
	if address == "" {
		return "", "", -1, fmt.Errorf("no host specified in URL")
	}

	// Extract password
	if u.User != nil {
		password, _ = u.User.Password()
	}

	// Extract database from path
	if u.Path != "" && u.Path != "/" {
		// Remove leading slash and parse as database number
		dbStr := strings.TrimPrefix(u.Path, "/")
		if dbStr != "" {
			if db, parseErr := strconv.Atoi(dbStr); parseErr == nil {
				database = db
			}
		}
	}

	return address, password, database, nil
}

// NewValkeyCacheService creates a new Valkey cache service
func NewValkeyCacheService() CacheService {
	// Parse Valkey URL and options
	valkeyURL := environment_variables.EnvironmentVariables.CACHE_URL
	if valkeyURL == "" {
		valkeyURL = "valkey://localhost:6379"
	}

	// Parse the URL to extract the address
	address, password, db, err := parseValkeyURL(valkeyURL)
	if err != nil {
		// Return a no-op implementation for graceful degradation
		return &NoOpCacheService{}
	}

	opts := valkey.ClientOption{
		InitAddress: []string{address},
	}

	// Set password from URL if present
	if password != "" {
		opts.Password = password
	}

	// Set database from URL if present
	if db != -1 {
		opts.SelectDB = db
	}

	// Override with environment variables if provided
	if environment_variables.EnvironmentVariables.CACHE_PASSWORD != "" {
		opts.Password = environment_variables.EnvironmentVariables.CACHE_PASSWORD
	}
	if environment_variables.EnvironmentVariables.CACHE_DB != "" {
		if db, err := strconv.Atoi(environment_variables.EnvironmentVariables.CACHE_DB); err == nil {
			opts.SelectDB = db
		}
	}

	client, err := valkey.NewClient(opts)
	if err != nil {
		// Return a no-op implementation for graceful degradation
		return &NoOpCacheService{}
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		// Return a no-op implementation for graceful degradation
		return &NoOpCacheService{}
	}

	return &ValkeyCacheService{
		client: client,
	}
}

// Set stores a value in Valkey with an expiration time
func (v *ValkeyCacheService) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = v.client.Do(ctx, v.client.B().Set().Key(key).Value(string(jsonValue)).ExSeconds(int64(expiration.Seconds())).Build()).Error()
	return err
}

// Get retrieves a value from Valkey
func (v *ValkeyCacheService) Get(ctx context.Context, key string, dest any) error {
	result := v.client.Do(ctx, v.client.B().Get().Key(key).Build())
	if result.Error() != nil {
		if result.Error().Error() == "redis: nil" || result.Error().Error() == "valkey nil message" {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get value: %w", result.Error())
	}

	val, err := result.ToString()
	if err != nil {
		return fmt.Errorf("failed to convert result to string: %w", err)
	}

	err = json.Unmarshal([]byte(val), dest)
	return err
}

// GetWithFallback retrieves a value from Valkey, or executes fallback function if not found
func (v *ValkeyCacheService) GetWithFallback(ctx context.Context, key string, dest any, fallback func() (any, error), expiration time.Duration) error {
	// Try to get from cache first
	err := v.Get(ctx, key, dest)
	if err == nil {
		return nil // Found in cache
	}

	// Cache miss, execute fallback
	value, err := fallback()
	if err != nil {
		return fmt.Errorf("fallback function failed: %w", err)
	}

	// Store in cache for future requests
	if err := v.Set(ctx, key, value, expiration); err != nil {
		// Don't return error, just log it
	}

	// Copy the value to dest
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal fallback value: %w", err)
	}

	return json.Unmarshal(jsonValue, dest)
}

// Delete removes a key from Valkey synchronously (blocking)
func (v *ValkeyCacheService) Delete(ctx context.Context, key string) error {
	err := v.client.Do(ctx, v.client.B().Del().Key(key).Build()).Error()
	return err
}

// Unlink removes a key from Valkey asynchronously (non-blocking)
func (v *ValkeyCacheService) Unlink(ctx context.Context, key string) error {
	err := v.client.Do(ctx, v.client.B().Unlink().Key(key).Build()).Error()
	return err
}

// DeletePattern removes all keys matching a pattern
func (v *ValkeyCacheService) DeletePattern(ctx context.Context, pattern string) error {
	// For now, implement a simple version that doesn't use SCAN
	// This can be enhanced later with proper SCAN implementation
	// Valkey supports the same commands as Redis, so we can use KEYS for small datasets
	// Note: KEYS should be avoided in production for large datasets

	// Get all keys matching the pattern
	result := v.client.Do(ctx, v.client.B().Keys().Pattern(pattern).Build())
	if result.Error() != nil {
		return fmt.Errorf("failed to get keys: %w", result.Error())
	}

	keys, err := result.AsStrSlice()
	if err != nil {
		return fmt.Errorf("failed to parse keys: %w", err)
	}

	if len(keys) > 0 {
		if err := v.client.Do(ctx, v.client.B().Unlink().Key(keys...).Build()).Error(); err != nil {
			return fmt.Errorf("failed to unlink keys: %w", err)
		}
	}

	return nil
}

// Exists checks if a key exists in Valkey
func (v *ValkeyCacheService) Exists(ctx context.Context, key string) (bool, error) {
	result := v.client.Do(ctx, v.client.B().Exists().Key(key).Build())
	if result.Error() != nil {
		return false, fmt.Errorf("failed to check key existence: %w", result.Error())
	}

	count, err := result.AsInt64()
	if err != nil {
		return false, fmt.Errorf("failed to parse exists result: %w", err)
	}

	return count > 0, nil
}

// Close closes the Valkey connection
func (v *ValkeyCacheService) Close() error {
	v.client.Close()
	return nil
}
