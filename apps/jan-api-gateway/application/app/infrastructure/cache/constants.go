package cache

import "time"

// Cache key constants
const (
	// CacheVersion is the API version prefix for cache keys
	CacheVersion = "v1"

	// ModelsCacheKey is the cache key for the models list
	ModelsCacheKey = CacheVersion + ":models:list"

	// RegistryEndpointModelsKey is the cache key for endpoint to models mapping
	RegistryEndpointModelsKey = CacheVersion + ":registry:endpoint_models"

	// RegistryModelEndpointsKey is the cache key for model to endpoints mapping
	RegistryModelEndpointsKey = CacheVersion + ":registry:model_endpoints"

	// UserByPublicIDKey is the cache key template for user lookups by public ID
	UserByPublicIDKey = CacheVersion + ":user:public_id:%s"

	// RegistryLockKey is the cache key for distributed lock on model registry updates
	RegistryLockKey = CacheVersion + ":registry:lock"

	// UserLockKey is the cache key template for distributed lock on user updates
	UserLockKey = CacheVersion + ":user:lock:%s"
)

// Cache TTL constants
const (
	// ModelsCacheTTL is the TTL for cached models list
	ModelsCacheTTL = 10 * time.Minute

	// UserCacheTTL is the TTL for cached user lookups
	UserCacheTTL = 15 * time.Minute

	// RegistryLockTTL is the TTL for distributed lock (short duration)
	RegistryLockTTL = 30 * time.Second

	// UserLockTTL is the TTL for user-specific distributed lock (short duration)
	UserLockTTL = 10 * time.Second
)
