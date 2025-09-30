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

	// UserByPublicIDKey is the cache key template for user lookups by public ID.
	UserByPublicIDKey = CacheVersion + ":user:public_id:%s"
)
