package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type RedisCacheService struct {
	client redis.UniversalClient
	rs     *redsync.Redsync
}

func NewRedisCacheService() *RedisCacheService {
	redisURL := environment_variables.EnvironmentVariables.REDIS_URL
	if redisURL == "" {
		panic("REDIS_URL environment variable must be set")
	}

	opts, err := buildUniversalOptions(redisURL)
	if err != nil {
		panic(fmt.Sprintf("failed to parse Redis URL: %v", err))
	}

	if pwd := environment_variables.EnvironmentVariables.REDIS_PASSWORD; pwd != "" {
		opts.Password = pwd
	}

	if dbVal := environment_variables.EnvironmentVariables.REDIS_DB; dbVal != 0 {
		opts.DB = dbVal
	}

	if len(opts.Addrs) > 1 && opts.DB != 0 {
		logger.GetLogger().Warn("Ignoring non-zero REDIS_DB when using Redis Cluster configuration")
		opts.DB = 0
	}

	client := redis.NewUniversalClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("failed to connect to Redis: %v", err))
	}

	logger.GetLogger().Info("Successfully connected to Redis")

	rs := redsync.New(goredis.NewPool(client))

	return &RedisCacheService{
		client: client,
		rs:     rs,
	}
}

func buildUniversalOptions(raw string) (*redis.UniversalOptions, error) {
	parts := strings.Split(raw, ",")
	opts := &redis.UniversalOptions{}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "://") {
			parsed, err := redis.ParseURL(part)
			if err != nil {
				return nil, err
			}

			opts.Addrs = append(opts.Addrs, parsed.Addr)

			if opts.Username == "" {
				opts.Username = parsed.Username
			}

			if opts.Password == "" {
				opts.Password = parsed.Password
			}

			if opts.DB == 0 {
				opts.DB = parsed.DB
			}

			if opts.TLSConfig == nil {
				opts.TLSConfig = parsed.TLSConfig
			}

			if opts.ReadTimeout == 0 {
				opts.ReadTimeout = parsed.ReadTimeout
			}

			if opts.WriteTimeout == 0 {
				opts.WriteTimeout = parsed.WriteTimeout
			}

			if opts.DialTimeout == 0 {
				opts.DialTimeout = parsed.DialTimeout
			}

			if opts.PoolSize == 0 {
				opts.PoolSize = parsed.PoolSize
			}

			if opts.MinIdleConns == 0 {
				opts.MinIdleConns = parsed.MinIdleConns
			}
		} else {
			opts.Addrs = append(opts.Addrs, part)
		}
	}

	if len(opts.Addrs) == 0 {
		return nil, fmt.Errorf("no Redis addresses provided")
	}

	return opts, nil
}

func (r *RedisCacheService) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

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

func (r *RedisCacheService) GetWithFallback(ctx context.Context, key string, fallback func() (string, error), expiration time.Duration) (string, error) {
	result, err := r.Get(ctx, key)
	if err == nil {
		return result, nil
	}

	result, err = fallback()
	if err != nil {
		return "", fmt.Errorf("fallback function failed: %w", err)
	}

	if err := r.Set(ctx, key, result, expiration); err != nil {
		logger.GetLogger().Error(fmt.Sprintf("Failed to cache value: %v", err))
	}

	return result, nil
}

func (r *RedisCacheService) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCacheService) Unlink(ctx context.Context, key string) error {
	return r.client.Unlink(ctx, key).Err()
}

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

func (r *RedisCacheService) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return result > 0, nil
}

func (r *RedisCacheService) Close() error {
	return r.client.Close()
}

func (r *RedisCacheService) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisCacheService) NewMutex(name string, options ...redsync.Option) *redsync.Mutex {
	return r.rs.NewMutex(name, options...)
}

func WithLock(cache RedisCacheService, lockName string, fn func() error, ttl time.Duration) error {
	mutex := cache.NewMutex(lockName, redsync.WithExpiry(ttl))

	if err := mutex.Lock(); err != nil {
		return err
	}

	defer func() {
		if _, err := mutex.Unlock(); err != nil {
		}
	}()

	return fn()
}
