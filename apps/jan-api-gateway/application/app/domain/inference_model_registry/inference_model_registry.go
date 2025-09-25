package inferencemodelregistry

import (
	"context"
	"encoding/base64"
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
		cacheExpiry:      cache.ModelsCacheTTL,
	}
}

func (r *InferenceModelRegistry) ListModels(ctx context.Context) []inferencemodel.Model {
	var models []inferencemodel.Model

	// Try to get from cache first
	err := r.cache.GetWithFallback(ctx, cache.ModelsCacheKey, &models, func() (any, error) {
		// Cache miss, return current models
		return r.models, nil
	}, r.cacheExpiry)

	if err != nil {
		// If cache fails, return current models
		return r.models
	}

	return models
}

// modelsEqual compares two slices of models for equality
func modelsEqual(models1, models2 []inferencemodel.Model) bool {
	if len(models1) != len(models2) {
		return false
	}

	// Create maps for efficient comparison
	map1 := make(map[string]inferencemodel.Model)
	map2 := make(map[string]inferencemodel.Model)

	for _, model := range models1 {
		map1[model.ID] = model
	}
	for _, model := range models2 {
		map2[model.ID] = model
	}

	// Compare the maps
	for id, model1 := range map1 {
		model2, exists := map2[id]
		if !exists || model1.Object != model2.Object || model1.OwnedBy != model2.OwnedBy {
			return false
		}
	}

	return true
}

// hasModelsChanged checks if the models for a service have changed
func (r *InferenceModelRegistry) hasModelsChanged(serviceName string, newModels []inferencemodel.Model) bool {
	existingModels, exists := r.endpointToModels[serviceName]
	if !exists {
		// Service doesn't exist, so it's a change
		return len(newModels) > 0
	}

	// Convert existing model IDs to full model objects
	existingModelObjects := make([]inferencemodel.Model, 0, len(existingModels))
	for _, modelID := range existingModels {
		if model, exists := r.modelsDetail[modelID]; exists {
			existingModelObjects = append(existingModelObjects, model)
		}
	}

	return !modelsEqual(existingModelObjects, newModels)
}

func (r *InferenceModelRegistry) AddModels(ctx context.Context, serviceName string, models []inferencemodel.Model) {
	// Check if models have actually changed to avoid unnecessary cache operations
	if !r.hasModelsChanged(serviceName, models) {
		return // No changes, skip cache update
	}

	r.endpointToModels[serviceName] = functional.Map(models, func(model inferencemodel.Model) string {
		r.modelsDetail[model.ID] = model
		return model.ID
	})
	r.rebuild()

	// Invalidate cache after adding models
	r.invalidateCache(ctx)

	// Populate cache with new registry data
	r.populateCache(ctx)
}

func (r *InferenceModelRegistry) RemoveServiceModels(ctx context.Context, serviceName string) {
	// Check if service actually exists
	if _, exists := r.endpointToModels[serviceName]; !exists {
		return // Service doesn't exist, no changes needed
	}

	delete(r.endpointToModels, serviceName)
	r.rebuild()

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
	// Clear endpoint models cache (pattern deletion uses UNLINK internally)
	pattern := cache.RegistryEndpointModelsKey + ":*"
	r.cache.DeletePattern(ctx, pattern)

	// Clear model endpoints cache (potentially large mapping data)
	r.cache.Unlink(ctx, cache.RegistryModelEndpointsKey)

	// Clear the cached models list (used by both registry and inference provider)
	r.cache.Unlink(ctx, cache.ModelsCacheKey)
}

// populateCache populates cache with current registry data
func (r *InferenceModelRegistry) populateCache(ctx context.Context) {
	// Populate model endpoints cache
	if err := r.cache.Set(ctx, cache.RegistryModelEndpointsKey, r.modelToEndpoints, r.cacheExpiry); err != nil {
		// Log error but continue
	}

	// Populate models list cache (used by both registry and inference provider)
	if err := r.cache.Set(ctx, cache.ModelsCacheKey, r.models, r.cacheExpiry); err != nil {
		// Log error but continue
	}
}

// ClearAllCache clears all registry cache entries (public method)
func (r *InferenceModelRegistry) ClearAllCache(ctx context.Context) error {
	r.invalidateCache(ctx)
	return nil
}
