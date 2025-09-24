package cache

// Redis cache key constants
const (
	// ModelsCacheKey is the cache key for the models list
	ModelsCacheKey = "models:list"

	// RegistryCacheKey is the cache key for the registry data
	RegistryCacheKey = "registry:data"

	// RegistryEndpointModelsKey is the cache key for endpoint to models mapping
	RegistryEndpointModelsKey = "registry:endpoint_models"

	// RegistryModelEndpointsKey is the cache key for model to endpoints mapping
	RegistryModelEndpointsKey = "registry:model_endpoints"
)
