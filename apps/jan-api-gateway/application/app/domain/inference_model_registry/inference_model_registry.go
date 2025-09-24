package inferencemodelregistry

import (
	"context"
	"time"

	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type InferenceModelRegistry struct {
	endpointToModels map[string][]string
	modelToEndpoints map[string][]string
	modelsDetail     map[string]inferencemodel.Model
	models           []inferencemodel.Model
	cache            *cache.RedisCacheService
	cacheExpiry      time.Duration
}

// NewInferenceModelRegistry creates a new registry instance with Redis caching
func NewInferenceModelRegistry(cacheService *cache.RedisCacheService) *InferenceModelRegistry {
	return &InferenceModelRegistry{
		endpointToModels: make(map[string][]string),
		modelToEndpoints: make(map[string][]string),
		modelsDetail:     make(map[string]inferencemodel.Model),
		models:           make([]inferencemodel.Model, 0),
		cache:            cacheService,
		cacheExpiry:      1 * time.Minute, // Cache registry data for 1 minute
	}
}

func (r *InferenceModelRegistry) ListModels(ctx context.Context) []inferencemodel.Model {
	var models []inferencemodel.Model

	// Try to get from cache first
	err := r.cache.GetWithFallback(ctx, cache.RegistryCacheKey, &models, func() (any, error) {
		// Cache miss, return current models
		return r.models, nil
	}, r.cacheExpiry)

	if err != nil {
		// If cache fails, return current models
		return r.models
	}

	return models
}

func (r *InferenceModelRegistry) AddModels(ctx context.Context, serviceName string, models []inferencemodel.Model) {
	r.endpointToModels[serviceName] = functional.Map(models, func(model inferencemodel.Model) string {
		r.modelsDetail[model.ID] = model
		return model.ID
	})
	r.rebuild()

	// Invalidate cache after adding models
	r.invalidateCache(ctx)
}

func (r *InferenceModelRegistry) RemoveServiceModels(ctx context.Context, serviceName string) {
	delete(r.endpointToModels, serviceName)
	r.rebuild()

	// Invalidate cache after removing models
	r.invalidateCache(ctx)
}

func (r *InferenceModelRegistry) GetEndpointToModels(ctx context.Context, serviceName string) ([]string, bool) {
	// Try to get from cache first
	var models []string
	cacheKey := cache.RegistryEndpointModelsKey + ":" + serviceName

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

	models, ok := r.endpointToModels[serviceName]
	return models, ok
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

	return r.modelToEndpoints
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
	// Clear main registry cache
	r.cache.Delete(ctx, cache.RegistryCacheKey)

	// Clear endpoint models cache
	r.cache.DeletePattern(ctx, cache.RegistryEndpointModelsKey+":*")

	// Clear model endpoints cache
	r.cache.Delete(ctx, cache.RegistryModelEndpointsKey)
}

// ClearAllCache clears all registry cache entries (public method)
func (r *InferenceModelRegistry) ClearAllCache(ctx context.Context) error {
	r.invalidateCache(ctx)
	return nil
}
