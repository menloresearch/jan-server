package cache

import "time"

// Cache key constants
const (
	// ModelsCacheKey is the cache key for the models list
	ModelsCacheKey = "v1:models:list"

	// RegistryEndpointModelsKey is the cache key for endpoint to models mapping
	RegistryEndpointModelsKey = "v1:registry:endpoint_models"

	// RegistryModelEndpointsKey is the cache key for model to endpoints mapping
	RegistryModelEndpointsKey = "v1:registry:model_endpoints"
)

// Cache TTL constants
const (
	// ModelsCacheTTL is the TTL for cached models list
	ModelsCacheTTL = 10 * time.Minute
)
