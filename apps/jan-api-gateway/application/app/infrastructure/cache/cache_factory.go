package cache

import (
	"strings"

	"menlo.ai/jan-api-gateway/config/environment_variables"
)

// NewCacheService creates a cache service based on configuration
func NewCacheService() CacheService {
	cacheType := strings.ToLower(environment_variables.EnvironmentVariables.CACHE_TYPE)

	// Default to Valkey if no cache type is specified
	if cacheType == "" {
		cacheType = "valkey"
	}

	switch cacheType {
	case "redis":
		return NewRedisCacheService()
	case "valkey":
		return NewValkeyCacheService()
	default:
		// Fallback to Valkey for unknown types
		return NewValkeyCacheService()
	}
}
