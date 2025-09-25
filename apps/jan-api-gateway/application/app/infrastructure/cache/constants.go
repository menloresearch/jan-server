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
)

// Cache TTL constants
const (
	// ModelsCacheTTL is the TTL for cached models list
	ModelsCacheTTL = 10 * time.Minute
)
