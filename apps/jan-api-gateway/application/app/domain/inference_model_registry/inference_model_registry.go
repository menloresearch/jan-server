package inferencemodelregistry

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

type InferenceModelRegistry struct {
	endpointToModels map[string][]string
	modelToEndpoints map[string][]string
	modelsDetail     map[string]inferencemodel.Model
	models           []inferencemodel.Model
	cache            cache.CacheService
	cacheExpiry      time.Duration
}

// sanitizeKeyPart encodes dynamic key parts to be Redis-key safe
func sanitizeKeyPart(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

// NewInferenceModelRegistry creates a new registry instance with cache service
func NewInferenceModelRegistry(cacheService cache.CacheService) *InferenceModelRegistry {
	return &InferenceModelRegistry{
		endpointToModels: make(map[string][]string),
		modelToEndpoints: make(map[string][]string),
		modelsDetail:     make(map[string]inferencemodel.Model),
		models:           make([]inferencemodel.Model, 0),
		cache:            cacheService,
		cacheExpiry:      cache.RegistryCacheTTL,
	}
}

func (r *InferenceModelRegistry) ListModels(ctx context.Context) []inferencemodel.Model {
	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] ListModels called, in-memory models count: %d", len(r.models)))

	var models []inferencemodel.Model

	// Try to get from cache first
	err := r.cache.GetWithFallback(ctx, cache.RegistryCacheKey, &models, func() (any, error) {
		// Cache miss, return current models
		logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] ListModels cache miss, returning %d in-memory models", len(r.models)))
		return r.models, nil
	}, r.cacheExpiry)

	if err != nil {
		// If cache fails, return current models
		logger.GetLogger().Error(fmt.Sprintf("[REGISTRY] ListModels cache error: %v, returning %d in-memory models", err, len(r.models)))
		return r.models
	}

	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] ListModels returning %d models", len(models)))
	return models
}

func (r *InferenceModelRegistry) AddModels(ctx context.Context, serviceName string, models []inferencemodel.Model) {
	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] AddModels called for service=%s with %d models", serviceName, len(models)))

	r.endpointToModels[serviceName] = functional.Map(models, func(model inferencemodel.Model) string {
		r.modelsDetail[model.ID] = model
		return model.ID
	})
	r.rebuild()

	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] AddModels rebuilt registry: %d total models, %d endpoints", len(r.models), len(r.endpointToModels)))

	// Invalidate cache after adding models
	r.invalidateCache(ctx)
}

func (r *InferenceModelRegistry) RemoveServiceModels(ctx context.Context, serviceName string) {
	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] RemoveServiceModels called for service=%s", serviceName))

	modelsCount := 0
	if models, exists := r.endpointToModels[serviceName]; exists {
		modelsCount = len(models)
	}

	delete(r.endpointToModels, serviceName)
	r.rebuild()

	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] RemoveServiceModels removed %d models for service=%s, remaining: %d total models, %d endpoints", modelsCount, serviceName, len(r.models), len(r.endpointToModels)))

	// Invalidate cache after removing models
	r.invalidateCache(ctx)
}

func (r *InferenceModelRegistry) GetEndpointToModels(ctx context.Context, serviceName string) ([]string, bool) {
	// Try to get from cache first
	var models []string
	cacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)

	err := r.cache.GetWithFallback(ctx, cacheKey, &models, func() (any, error) {
		// Cache miss, return from memory
		models := r.endpointToModels[serviceName]
		return models, nil
	}, r.cacheExpiry)

	if err != nil {
		// If cache fails, return from memory
		models, ok := r.endpointToModels[serviceName]
		return models, ok
	}

	return models, len(models) > 0
}

func (r *InferenceModelRegistry) GetModelToEndpoints(ctx context.Context) map[string][]string {
	// Try to get from cache first
	var modelToEndpoints map[string][]string

	err := r.cache.GetWithFallback(ctx, cache.RegistryModelEndpointsKey, &modelToEndpoints, func() (any, error) {
		// Cache miss, return from memory
		return r.modelToEndpoints, nil
	}, r.cacheExpiry)

	if err != nil {
		// If cache fails, return from memory
		return r.modelToEndpoints
	}

	return modelToEndpoints
}

func (r *InferenceModelRegistry) rebuild() {
	newModelToEndpoints := make(map[string][]string)
	newModels := make([]inferencemodel.Model, 0)
	for endpoint, models := range r.endpointToModels {
		for _, model := range models {
			newModelToEndpoints[model] = append(newModelToEndpoints[model], endpoint)
		}
	}
	r.modelToEndpoints = newModelToEndpoints

	for key := range r.modelToEndpoints {
		newModels = append(newModels, r.modelsDetail[key])
	}
	r.models = newModels
}

// invalidateCache clears all registry-related cache entries
func (r *InferenceModelRegistry) invalidateCache(ctx context.Context) {
	logger.GetLogger().Debug("[REGISTRY] invalidateCache called - clearing all cache entries")

	// Clear main registry cache (potentially large data structure with many models)
	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] UNLINK (async) cache key: %s", cache.RegistryCacheKey))
	r.cache.Unlink(ctx, cache.RegistryCacheKey)

	// Clear endpoint models cache (pattern deletion uses UNLINK internally)
	pattern := cache.RegistryEndpointModelsKey + ":*"
	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] DeletePattern (uses UNLINK internally) pattern: %s", pattern))
	r.cache.DeletePattern(ctx, pattern)

	// Clear model endpoints cache (potentially large mapping data)
	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] UNLINK (async) cache key: %s", cache.RegistryModelEndpointsKey))
	r.cache.Unlink(ctx, cache.RegistryModelEndpointsKey)

	// Also invalidate the cached models list used by inference provider (large list)
	logger.GetLogger().Debug(fmt.Sprintf("[REGISTRY] UNLINK (async) cache key: %s", cache.ModelsCacheKey))
	r.cache.Unlink(ctx, cache.ModelsCacheKey)

	logger.GetLogger().Debug("[REGISTRY] invalidateCache completed - all cache entries cleared using UNLINK (async)")
}

// ClearAllCache clears all registry cache entries (public method)
func (r *InferenceModelRegistry) ClearAllCache(ctx context.Context) error {
	logger.GetLogger().Debug("[REGISTRY] ClearAllCache called (public method)")
	r.invalidateCache(ctx)
	logger.GetLogger().Debug("[REGISTRY] ClearAllCache completed")
	return nil
}
