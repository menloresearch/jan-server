package cache

const (
	// CacheVersion is the API version prefix for cache keys.
	CacheVersion = "v1"

	// ModelsCacheKey is the cache key for the aggregated models list.
	ModelsCacheKey = CacheVersion + ":models:list"

	// JanModelsCacheKey stores the cached model list for the built-in Jan provider.
	JanModelsCacheKey = CacheVersion + ":models:jan"

	// OrganizationModelsCacheKeyPattern formats cache keys for organization-scoped model lists.
	OrganizationModelsCacheKeyPattern = CacheVersion + ":models:organization:%d"

	// ProjectModelsCacheKeyPattern formats cache keys for project-scoped model lists.
	ProjectModelsCacheKeyPattern = CacheVersion + ":models:project:%d"

	// RegistryEndpointModelsKey maps registry endpoints to their advertised models.
	RegistryEndpointModelsKey = CacheVersion + ":registry:endpoint_models"

	// RegistryModelEndpointsKey maps models to the registry endpoints that expose them.
	RegistryModelEndpointsKey = CacheVersion + ":registry:model_endpoints"

	// UserByPublicIDKey is the cache key template for user lookups by public ID.
	UserByPublicIDKey = CacheVersion + ":user:public_id:%s"

	// RegistryLockKey is the cache key used to coordinate registry synchronization.
	RegistryLockKey = CacheVersion + ":registry:lock"

	// UserLockKey is the cache key used to coordinate user-related critical sections.
	UserLockKey = CacheVersion + ":user:lock:%s"
)
