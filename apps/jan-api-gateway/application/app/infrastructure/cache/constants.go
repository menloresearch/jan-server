package cache

const (
	CacheVersion                      = "v1"
	ModelsCacheKey                    = CacheVersion + ":models:list"
	JanModelsCacheKey                 = CacheVersion + ":models:jan"
	OrganizationModelsCacheKeyPattern = CacheVersion + ":models:organization:%d"
	ProjectModelsCacheKeyPattern      = CacheVersion + ":models:project:%d"
	RegistryEndpointModelsKey         = CacheVersion + ":registry:endpoint_models"
	RegistryModelEndpointsKey         = CacheVersion + ":registry:model_endpoints"
	UserByPublicIDKey                 = CacheVersion + ":user:public_id:%s"
	RegistryLockKey                   = CacheVersion + ":registry:lock"
	UserLockKey                       = CacheVersion + ":user:lock:%s"
)
