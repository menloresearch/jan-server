package inferencemodelregistry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

type InferenceModelRegistry struct {
	cache     *cache.RedisCacheService
	janClient *janinference.JanInferenceClient
}

const (
	// Consistent timeout for all Jan client operations
	janClientTimeout = 20 * time.Second
	ModelsCacheTTL   = 10 * time.Minute
)

// sanitizeKeyPart encodes dynamic key parts to be Redis-key safe
func sanitizeKeyPart(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

// NewInferenceModelRegistry creates a new registry instance with cache service
func NewInferenceModelRegistry(cacheService *cache.RedisCacheService, janClient *janinference.JanInferenceClient) *InferenceModelRegistry {
	return &InferenceModelRegistry{
		cache:     cacheService,
		janClient: janClient,
	}
}

func (r *InferenceModelRegistry) ListModels(ctx context.Context) []inferencemodel.Model {
	var models []inferencemodel.Model

	// Try to get from cache first
	cachedModelsJSON, err := r.cache.Get(ctx, cache.ModelsCacheKey)
	if err == nil && cachedModelsJSON != "" {
		if jsonErr := json.Unmarshal([]byte(cachedModelsJSON), &models); jsonErr == nil {
			return models
		}
	}

	// Cache miss - rebuild from JanInferenceClient
	models = r.rebuildModelsFromJanClient(ctx)
	return models
}

// hasModelsChanged checks if the models for a service have changed compared to cached data
func (r *InferenceModelRegistry) hasModelsChanged(ctx context.Context, serviceName string, newModels []inferencemodel.Model) bool {
	// Compare by model IDs only to avoid relying on per-model detail cache
	cacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)
	cachedIDsJSON, err := r.cache.Get(ctx, cacheKey)
	if err != nil {
		// Cache miss or error - treat as changed so we populate
		return true
	}

	var cachedIDs []string
	if jsonErr := json.Unmarshal([]byte(cachedIDsJSON), &cachedIDs); jsonErr != nil {
		return true
	}

	if len(cachedIDs) != len(newModels) {
		return true
	}

	newIDs := functional.Map(newModels, func(model inferencemodel.Model) string { return model.ID })
	idSet := make(map[string]struct{}, len(cachedIDs))
	for _, id := range cachedIDs {
		idSet[id] = struct{}{}
	}
	for _, id := range newIDs {
		if _, ok := idSet[id]; !ok {
			return true
		}
	}
	return false
}

func (r *InferenceModelRegistry) SetModels(ctx context.Context, serviceName string, models []inferencemodel.Model) error {
	if strings.TrimSpace(serviceName) == "" {
		return errors.New("service name cannot be empty")
	}

	if !r.hasModelsChanged(ctx, serviceName, models) {
		return nil
	}

	// Clear all existing cache
	r.cache.Unlink(ctx, cache.RegistryModelEndpointsKey)
	r.cache.Unlink(ctx, cache.ModelsCacheKey)

	// Clear pattern-based entries
	pattern := cache.RegistryEndpointModelsKey + ":*"
	r.cache.DeletePattern(ctx, pattern)

	// Add back all models
	serviceCacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)
	modelIDs := functional.Map(models, func(m inferencemodel.Model) string { return m.ID })

	// Convert to JSON strings for cache storage
	modelIDsJSON, err := json.Marshal(modelIDs)
	if err != nil {
		return err
	}
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return err
	}

	if err := r.cache.Set(ctx, serviceCacheKey, string(modelIDsJSON), ModelsCacheTTL); err != nil {
		return err
	}
	if err := r.cache.Set(ctx, cache.ModelsCacheKey, string(modelsJSON), ModelsCacheTTL); err != nil {
		return err
	}

	// Rebuild reverse mapping
	return r.rebuildModelToEndpointsMapping(ctx)
}

func (r *InferenceModelRegistry) RemoveServiceModels(ctx context.Context, serviceName string) error {
	if strings.TrimSpace(serviceName) == "" {
		return errors.New("service name cannot be empty")
	}

	serviceCacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)

	// 1) Read BEFORE deleting
	serviceModelIDsJSON, err := r.cache.Get(ctx, serviceCacheKey)
	if err != nil {
		// nothing to do
		return nil
	}

	var serviceModelIDs []string
	if jsonErr := json.Unmarshal([]byte(serviceModelIDsJSON), &serviceModelIDs); jsonErr != nil {
		return nil
	}
	serviceModelSet := make(map[string]struct{}, len(serviceModelIDs))
	for _, id := range serviceModelIDs {
		serviceModelSet[id] = struct{}{}
	}

	// 2) Delete mapping
	if err := r.cache.Unlink(ctx, serviceCacheKey); err != nil {
		return err
	}

	// 3) Remove those models from the global list
	existingJSON, _ := r.cache.Get(ctx, cache.ModelsCacheKey)
	var existing []inferencemodel.Model
	if existingJSON != "" {
		json.Unmarshal([]byte(existingJSON), &existing)
	}

	var filtered []inferencemodel.Model
	for _, m := range existing {
		if _, ok := serviceModelSet[m.ID]; !ok {
			filtered = append(filtered, m)
		}
	}

	filteredJSON, err := json.Marshal(filtered)
	if err != nil {
		return err
	}
	if err := r.cache.Set(ctx, cache.ModelsCacheKey, string(filteredJSON), ModelsCacheTTL); err != nil {
		return err
	}

	// 4) Rebuild reverse mapping
	return r.rebuildModelToEndpointsMapping(ctx)
}

func (r *InferenceModelRegistry) GetEndpointToModels(ctx context.Context, serviceName string) ([]string, bool) {
	// Try to get from cache first
	cacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)
	modelsJSON, err := r.cache.Get(ctx, cacheKey)
	if err != nil {
		// Cache miss - this service has no models yet
		// Return empty result and don't populate cache
		return nil, false
	}

	var models []string
	if jsonErr := json.Unmarshal([]byte(modelsJSON), &models); jsonErr != nil {
		return nil, false
	}

	return models, len(models) > 0
}

func (r *InferenceModelRegistry) GetModelToEndpoints(ctx context.Context) map[string][]string {
	// Try to get from cache first
	modelToEndpointsJSON, err := r.cache.Get(ctx, cache.RegistryModelEndpointsKey)
	if err != nil {
		// Cache miss - rebuild from JanInferenceClient
		r.rebuildModelsFromJanClient(ctx)

		// Try to get again after rebuild
		modelToEndpointsJSON, err = r.cache.Get(ctx, cache.RegistryModelEndpointsKey)
		if err != nil {
			return make(map[string][]string)
		}
	}

	var modelToEndpoints map[string][]string
	if jsonErr := json.Unmarshal([]byte(modelToEndpointsJSON), &modelToEndpoints); jsonErr != nil {
		return make(map[string][]string)
	}

	return modelToEndpoints
}

// rebuildModelsFromJanClient fetches models from JanInferenceClient and rebuilds cache
func (r *InferenceModelRegistry) rebuildModelsFromJanClient(ctx context.Context) []inferencemodel.Model {
	if r.janClient == nil {
		return []inferencemodel.Model{}
	}

	// Apply consistent timeout for Jan client operations
	timeoutCtx, cancel := context.WithTimeout(ctx, janClientTimeout)
	defer cancel()

	janModelResp, err := r.janClient.GetModels(timeoutCtx)
	if err != nil {
		return []inferencemodel.Model{}
	}

	models := make([]inferencemodel.Model, 0)
	for _, model := range janModelResp.Data {
		models = append(models, inferencemodel.Model{
			ID:      model.ID,
			Object:  model.Object,
			Created: model.Created,
			OwnedBy: model.OwnedBy,
		})
	}

	// Store models in cache
	if len(models) > 0 {
		modelsJSON, _ := json.Marshal(models)
		r.cache.Set(ctx, cache.ModelsCacheKey, string(modelsJSON), ModelsCacheTTL)

		// Store service models mapping
		serviceCacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(r.janClient.BaseURL)
		modelIDs := functional.Map(models, func(model inferencemodel.Model) string {
			return model.ID
		})
		modelIDsJSON, _ := json.Marshal(modelIDs)
		r.cache.Set(ctx, serviceCacheKey, string(modelIDsJSON), ModelsCacheTTL)

		// Build model-to-endpoints mapping
		modelToEndpoints := make(map[string][]string)
		for _, model := range models {
			modelToEndpoints[model.ID] = append(modelToEndpoints[model.ID], r.janClient.BaseURL)
		}
		modelToEndpointsJSON, _ := json.Marshal(modelToEndpoints)
		r.cache.Set(ctx, cache.RegistryModelEndpointsKey, string(modelToEndpointsJSON), ModelsCacheTTL)
	}

	return models
}

// rebuildModelToEndpointsMapping rebuilds the model-to-endpoints mapping from all service mappings
func (r *InferenceModelRegistry) rebuildModelToEndpointsMapping(ctx context.Context) error {
	modelToEndpoints := make(map[string][]string)

	// This is a simplified implementation - in production you'd scan all service keys
	// For now, we'll just rebuild from known models
	allModelsJSON, err := r.cache.Get(ctx, cache.ModelsCacheKey)
	if err != nil {
		return err
	}

	var allModels []inferencemodel.Model
	if jsonErr := json.Unmarshal([]byte(allModelsJSON), &allModels); jsonErr != nil {
		return jsonErr
	}

	// For each model, find which services have it (this is not optimal but works)
	for _, model := range allModels {
		// You could optimize this by scanning service keys pattern
		if r.janClient != nil {
			modelToEndpoints[model.ID] = append(modelToEndpoints[model.ID], r.janClient.BaseURL)
		}
	}

	modelToEndpointsJSON, err := json.Marshal(modelToEndpoints)
	if err != nil {
		return err
	}
	return r.cache.Set(ctx, cache.RegistryModelEndpointsKey, string(modelToEndpointsJSON), ModelsCacheTTL)
}

// CheckInferenceModels checks and updates models from JanInferenceClient (moved from cron service)
func (r *InferenceModelRegistry) CheckInferenceModels(ctx context.Context) {
	if r.janClient == nil {
		return
	}

	// Apply consistent timeout for Jan client operations
	timeoutCtx, cancel := context.WithTimeout(ctx, janClientTimeout)
	defer cancel()

	janModelResp, err := r.janClient.GetModels(timeoutCtx)
	if err != nil {
		_ = r.RemoveServiceModels(ctx, r.janClient.BaseURL) // Ignore error in cron context
	} else {
		models := make([]inferencemodel.Model, 0)
		for _, model := range janModelResp.Data {
			models = append(models, inferencemodel.Model{
				ID:      model.ID,
				Object:  model.Object,
				Created: model.Created,
				OwnedBy: model.OwnedBy,
			})
		}

		// Clean and add new models (no merging or change checking)
		_ = r.SetModels(ctx, r.janClient.BaseURL, models) // Ignore error in cron context
	}
}
