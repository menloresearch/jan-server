package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

// RedisCacheService provides caching functionality using Redis
type RedisCacheService struct {
	client *redis.Client
	rs     *redsync.Redsync
}

// NewRedisCacheService creates a new Redis cache service
func NewRedisCacheService() CacheService {
	// Parse Redis URL and options
	redisURL := environment_variables.EnvironmentVariables.REDIS_URL
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.GetLogger().Error(fmt.Sprintf("Failed to parse Redis URL: %v", err))
		// Fallback to default configuration
		opts = &redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}
	}

	// Override with environment variables if provided
	if environment_variables.EnvironmentVariables.REDIS_PASSWORD != "" {
		opts.Password = environment_variables.EnvironmentVariables.REDIS_PASSWORD
	}
	if environment_variables.EnvironmentVariables.REDIS_DB != "" {
		if db, err := strconv.Atoi(environment_variables.EnvironmentVariables.REDIS_DB); err == nil {
			opts.DB = db
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.GetLogger().Error(fmt.Sprintf("Failed to connect to Redis: %v", err))
		// Return a no-op implementation for graceful degradation
		return &NoOpCacheService{}
	} else {
		logger.GetLogger().Info("Successfully connected to Redis")
	}

	// Create redsync instance
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)

	return &RedisCacheService{
		client: client,
		rs:     rs,
	}
}

// Set stores a string value in Redis with an expiration time
func (r *RedisCacheService) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a string value from Redis
func (r *RedisCacheService) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get value: %w", err)
	}

	return val, nil
}

// GetWithFallback retrieves a string value from Redis, or executes fallback function if not found
func (r *RedisCacheService) GetWithFallback(ctx context.Context, key string, fallback func() (string, error), expiration time.Duration) (string, error) {
	// Try to get from cache first
	result, err := r.Get(ctx, key)
	if err == nil {
		return result, nil // Found in cache
	}

	// Cache miss, execute fallback
	result, err = fallback()
	if err != nil {
		return "", fmt.Errorf("fallback function failed: %w", err)
	}

	// Store in cache for future requests
	if err := r.Set(ctx, key, result, expiration); err != nil {
		logger.GetLogger().Error(fmt.Sprintf("Failed to cache value: %v", err))
		// Don't return error, just log it
	}

	return result, nil
}

// Delete removes a key from Redis synchronously (blocking)
func (r *RedisCacheService) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Unlink removes a key from Redis asynchronously (non-blocking)
func (r *RedisCacheService) Unlink(ctx context.Context, key string) error {
	return r.client.Unlink(ctx, key).Err()
}

// DeletePattern removes all keys matching a pattern
func (r *RedisCacheService) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}
		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			for _, k := range keys {
				pipe.Unlink(ctx, k)
			}
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("failed to unlink keys: %w", err)
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return nil
}

// Exists checks if a key exists in Redis
func (r *RedisCacheService) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return result > 0, nil
}

// Close closes the Redis connection
func (r *RedisCacheService) Close() error {
	return r.client.Close()
}

// HealthCheck verifies Redis connectivity
func (r *RedisCacheService) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// NewMutex creates a new distributed mutex using go-redsync
func (r *RedisCacheService) NewMutex(name string, options ...redsync.Option) *redsync.Mutex {
	return r.rs.NewMutex(name, options...)
}

// WithLock executes a function with a distributed lock using go-redsync
func WithLock(cache CacheService, lockName string, fn func() error, ttl time.Duration) error {
	mutex := cache.NewMutex(lockName, redsync.WithExpiry(ttl))

	// Acquire lock
	if err := mutex.Lock(); err != nil {
		return err
	}

	// Ensure lock is released
	defer func() {
		if _, err := mutex.Unlock(); err != nil {
			// swallow unlock error
		}
	}()

	// Execute the function
	return fn()
}
