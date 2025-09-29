package inferencemodelregistry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"menlo.ai/jan-api-gateway/app/utils/logger"

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
	log := logger.GetLogger()
	log.Info("ListModels: retrieving models from cache")

	var models []inferencemodel.Model

	// Try to get from cache first
	cachedModelsJSON, err := r.cache.Get(ctx, cache.ModelsCacheKey)
	if err == nil && cachedModelsJSON != "" {
		if jsonErr := json.Unmarshal([]byte(cachedModelsJSON), &models); jsonErr == nil {
			log.Infof("ListModels: returning %d models from cache", len(models))
			return models
		}
		log.Infof("ListModels: failed to unmarshal cached models: %v", err)
	} else {
		if err != nil {
			log.Infof("ListModels: cache lookup failed: %v", err)
		} else {
			log.Info("ListModels: cache hit returned empty payload")
		}
	}

	log.Info("ListModels: rebuilding models from Jan inference client")
	models = r.rebuildModelsFromJanClient(ctx)
	log.Infof("ListModels: returning %d models after rebuild", len(models))
	return models
}

// hasModelsChanged checks if the models for a service have changed compared to cached data
func (r *InferenceModelRegistry) hasModelsChanged(ctx context.Context, serviceName string, newModels []inferencemodel.Model) bool {
	log := logger.GetLogger()
	log.Infof("hasModelsChanged: evaluating %d incoming models for service %s", len(newModels), serviceName)

	// Compare by model IDs only to avoid relying on per-model detail cache
	cacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)
	cachedIDsJSON, err := r.cache.Get(ctx, cacheKey)
	if err != nil {
		log.Infof("hasModelsChanged: cache lookup failed for %s: %v", serviceName, err)
		// Cache miss or error - treat as changed so we populate
		return true
	}

	var cachedIDs []string
	if jsonErr := json.Unmarshal([]byte(cachedIDsJSON), &cachedIDs); jsonErr != nil {
		log.Infof("hasModelsChanged: failed to unmarshal cached IDs for %s: %v", serviceName, jsonErr)
		return true
	}

	if len(cachedIDs) != len(newModels) {
		log.Infof("hasModelsChanged: model count changed for %s (cached=%d incoming=%d)", serviceName, len(cachedIDs), len(newModels))
		return true
	}

	newIDs := functional.Map(newModels, func(model inferencemodel.Model) string { return model.ID })
	idSet := make(map[string]struct{}, len(cachedIDs))
	for _, id := range cachedIDs {
		idSet[id] = struct{}{}
	}
	for _, id := range newIDs {
		if _, ok := idSet[id]; !ok {
			log.Infof("hasModelsChanged: detected new model %s for service %s", id, serviceName)
			return true
		}
	}

	log.Infof("hasModelsChanged: no change detected for service %s", serviceName)
	return false
}

func (r *InferenceModelRegistry) SetModels(ctx context.Context, serviceName string, models []inferencemodel.Model) error {
	log := logger.GetLogger()
	log.Infof("SetModels: updating %d models for service %s", len(models), serviceName)

	if strings.TrimSpace(serviceName) == "" {
		log.Info("SetModels: received empty service name")
		return errors.New("service name cannot be empty")
	}

	if !r.hasModelsChanged(ctx, serviceName, models) {
		log.Infof("SetModels: no changes detected for service %s", serviceName)
		return nil
	}

	log.Info("SetModels: clearing cached model mappings")
	r.cache.Unlink(ctx, cache.RegistryModelEndpointsKey)
	r.cache.Unlink(ctx, cache.ModelsCacheKey)

	pattern := cache.RegistryEndpointModelsKey + ":*"
	r.cache.DeletePattern(ctx, pattern)

	serviceCacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)
	modelIDs := functional.Map(models, func(m inferencemodel.Model) string { return m.ID })

	modelIDsJSON, err := json.Marshal(modelIDs)
	if err != nil {
		log.Infof("SetModels: failed to marshal model IDs for %s: %v", serviceName, err)
		return err
	}
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		log.Infof("SetModels: failed to marshal models for %s: %v", serviceName, err)
		return err
	}

	if err := r.cache.Set(ctx, serviceCacheKey, string(modelIDsJSON), ModelsCacheTTL); err != nil {
		log.Infof("SetModels: failed to write service mapping for %s: %v", serviceName, err)
		return err
	}
	if err := r.cache.Set(ctx, cache.ModelsCacheKey, string(modelsJSON), ModelsCacheTTL); err != nil {
		log.Infof("SetModels: failed to write global models cache for %s: %v", serviceName, err)
		return err
	}

	log.Info("SetModels: rebuilding model to endpoint mapping")
	if err := r.rebuildModelToEndpointsMapping(ctx); err != nil {
		log.Infof("SetModels: failed to rebuild model to endpoints mapping for %s: %v", serviceName, err)
		return err
	}

	log.Infof("SetModels: completed update for service %s", serviceName)
	return nil
}

func (r *InferenceModelRegistry) RemoveServiceModels(ctx context.Context, serviceName string) error {
	log := logger.GetLogger()
	log.Infof("RemoveServiceModels: removing models for service %s", serviceName)

	if strings.TrimSpace(serviceName) == "" {
		log.Info("RemoveServiceModels: received empty service name")
		return errors.New("service name cannot be empty")
	}

	serviceCacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)

	serviceModelIDsJSON, err := r.cache.Get(ctx, serviceCacheKey)
	if err != nil {
		log.Infof("RemoveServiceModels: no cached models found for %s", serviceName)
		return nil
	}

	var serviceModelIDs []string
	if jsonErr := json.Unmarshal([]byte(serviceModelIDsJSON), &serviceModelIDs); jsonErr != nil {
		log.Infof("RemoveServiceModels: failed to parse cached ids for %s: %v", serviceName, jsonErr)
		return nil
	}
	serviceModelSet := make(map[string]struct{}, len(serviceModelIDs))
	for _, id := range serviceModelIDs {
		serviceModelSet[id] = struct{}{}
	}
	log.Infof("RemoveServiceModels: removing %d models for %s", len(serviceModelSet), serviceName)

	if err := r.cache.Unlink(ctx, serviceCacheKey); err != nil {
		log.Infof("RemoveServiceModels: failed to unlink service mapping for %s: %v", serviceName, err)
		return err
	}

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
		log.Infof("RemoveServiceModels: failed to marshal filtered models for %s: %v", serviceName, err)
		return err
	}
	if err := r.cache.Set(ctx, cache.ModelsCacheKey, string(filteredJSON), ModelsCacheTTL); err != nil {
		log.Infof("RemoveServiceModels: failed to update global models cache for %s: %v", serviceName, err)
		return err
	}

	log.Info("RemoveServiceModels: rebuilding model to endpoint mapping")
	if err := r.rebuildModelToEndpointsMapping(ctx); err != nil {
		log.Infof("RemoveServiceModels: failed to rebuild model mapping for %s: %v", serviceName, err)
		return err
	}

	log.Infof("RemoveServiceModels: completed removal for service %s", serviceName)
	return nil
}

func (r *InferenceModelRegistry) GetEndpointToModels(ctx context.Context, serviceName string) ([]string, bool) {
	log := logger.GetLogger()
	log.Infof("GetEndpointToModels: fetching models for service %s", serviceName)

	cacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(serviceName)
	modelsJSON, err := r.cache.Get(ctx, cacheKey)
	if err != nil {
		log.Infof("GetEndpointToModels: cache miss for service %s: %v", serviceName, err)
		return nil, false
	}

	var models []string
	if jsonErr := json.Unmarshal([]byte(modelsJSON), &models); jsonErr != nil {
		log.Infof("GetEndpointToModels: failed to decode cache for service %s: %v", serviceName, jsonErr)
		return nil, false
	}

	log.Infof("GetEndpointToModels: returning %d models for service %s", len(models), serviceName)
	return models, len(models) > 0
}

func (r *InferenceModelRegistry) GetModelToEndpoints(ctx context.Context) map[string][]string {
	log := logger.GetLogger()
	log.Info("GetModelToEndpoints: fetching model to endpoint mapping")

	modelToEndpointsJSON, err := r.cache.Get(ctx, cache.RegistryModelEndpointsKey)
	if err != nil {
		log.Infof("GetModelToEndpoints: cache miss, rebuilding from Jan inference client: %v", err)
		r.rebuildModelsFromJanClient(ctx)

		modelToEndpointsJSON, err = r.cache.Get(ctx, cache.RegistryModelEndpointsKey)
		if err != nil {
			log.Infof("GetModelToEndpoints: still missing after rebuild: %v", err)
			return make(map[string][]string)
		}
	}

	var modelToEndpoints map[string][]string
	if jsonErr := json.Unmarshal([]byte(modelToEndpointsJSON), &modelToEndpoints); jsonErr != nil {
		log.Infof("GetModelToEndpoints: failed to decode cache: %v", jsonErr)
		return make(map[string][]string)
	}

	for id, endpoints := range modelToEndpoints {
		log.Infof("GetModelToEndpoints: model %s available on %d endpoints", id, len(endpoints))
	}

	return modelToEndpoints
}

// rebuildModelsFromJanClient fetches models from JanInferenceClient and rebuilds cache
func (r *InferenceModelRegistry) rebuildModelsFromJanClient(ctx context.Context) []inferencemodel.Model {
	log := logger.GetLogger()
	log.Info("rebuildModelsFromJanClient: rebuilding models from Jan inference client")

	if r.janClient == nil {
		log.Info("rebuildModelsFromJanClient: Jan inference client not configured")
		return []inferencemodel.Model{}
	}

	// Apply consistent timeout for Jan client operations
	timeoutCtx, cancel := context.WithTimeout(ctx, janClientTimeout)
	defer cancel()

	janModelResp, err := r.janClient.GetModels(timeoutCtx)
	if err != nil {
		log.Infof("rebuildModelsFromJanClient: failed to fetch models from %s: %v", r.janClient.BaseURL, err)
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

	log.Infof("rebuildModelsFromJanClient: fetched %d models from %s", len(models), r.janClient.BaseURL)

	if len(models) > 0 {
		modelsJSON, err := json.Marshal(models)
		if err == nil {
			r.cache.Set(ctx, cache.ModelsCacheKey, string(modelsJSON), ModelsCacheTTL)
			log.Infof("rebuildModelsFromJanClient: cached %d models under global key", len(models))
		} else {
			log.Infof("rebuildModelsFromJanClient: failed to marshal models for caching: %v", err)
		}

		serviceCacheKey := cache.RegistryEndpointModelsKey + ":" + sanitizeKeyPart(r.janClient.BaseURL)
		modelIDs := functional.Map(models, func(model inferencemodel.Model) string {
			return model.ID
		})
		modelIDsJSON, err := json.Marshal(modelIDs)
		if err == nil {
			r.cache.Set(ctx, serviceCacheKey, string(modelIDsJSON), ModelsCacheTTL)
			log.Infof("rebuildModelsFromJanClient: cached %d model ids for service %s", len(modelIDs), r.janClient.BaseURL)
		} else {
			log.Infof("rebuildModelsFromJanClient: failed to marshal model ids for service %s: %v", r.janClient.BaseURL, err)
		}

		modelToEndpoints := make(map[string][]string)
		for _, model := range models {
			modelToEndpoints[model.ID] = append(modelToEndpoints[model.ID], r.janClient.BaseURL)
		}
		modelToEndpointsJSON, err := json.Marshal(modelToEndpoints)
		if err == nil {
			r.cache.Set(ctx, cache.RegistryModelEndpointsKey, string(modelToEndpointsJSON), ModelsCacheTTL)
			log.Info("rebuildModelsFromJanClient: cached model to endpoint mapping")
		} else {
			log.Infof("rebuildModelsFromJanClient: failed to marshal model to endpoint mapping: %v", err)
		}
	}

	return models
}

// rebuildModelToEndpointsMapping rebuilds the model-to-endpoints mapping from all service mappings
func (r *InferenceModelRegistry) rebuildModelToEndpointsMapping(ctx context.Context) error {
	log := logger.GetLogger()
	log.Info("rebuildModelToEndpointsMapping: rebuilding model to endpoint mapping")

	modelToEndpoints := make(map[string][]string)

	allModelsJSON, err := r.cache.Get(ctx, cache.ModelsCacheKey)
	if err != nil {
		log.Infof("rebuildModelToEndpointsMapping: failed to read global models: %v", err)
		return err
	}

	var allModels []inferencemodel.Model
	if jsonErr := json.Unmarshal([]byte(allModelsJSON), &allModels); jsonErr != nil {
		log.Infof("rebuildModelToEndpointsMapping: failed to decode models: %v", jsonErr)
		return jsonErr
	}

	for _, model := range allModels {
		if r.janClient != nil {
			modelToEndpoints[model.ID] = append(modelToEndpoints[model.ID], r.janClient.BaseURL)
		}
	}

	log.Infof("rebuildModelToEndpointsMapping: prepared mapping for %d models", len(modelToEndpoints))

	modelToEndpointsJSON, err := json.Marshal(modelToEndpoints)
	if err != nil {
		log.Infof("rebuildModelToEndpointsMapping: failed to marshal mapping: %v", err)
		return err
	}

	if err := r.cache.Set(ctx, cache.RegistryModelEndpointsKey, string(modelToEndpointsJSON), ModelsCacheTTL); err != nil {
		log.Infof("rebuildModelToEndpointsMapping: failed to cache mapping: %v", err)
		return err
	}

	log.Info("rebuildModelToEndpointsMapping: mapping cached successfully")
	return nil
}

// CheckInferenceModels checks and updates models from JanInferenceClient (moved from cron service)
func (r *InferenceModelRegistry) CheckInferenceModels(ctx context.Context) {
	log := logger.GetLogger()

	if r.janClient == nil {
		log.Info("CheckInferenceModels skipped: Jan inference client not configured")
		return
	}

	log.Infof("CheckInferenceModels: fetching models from %s", r.janClient.BaseURL)

	// Apply consistent timeout for Jan client operations
	timeoutCtx, cancel := context.WithTimeout(ctx, janClientTimeout)
	defer cancel()

	janModelResp, err := r.janClient.GetModels(timeoutCtx)
	if err != nil {
		log.Infof("CheckInferenceModels: failed to fetch models from %s: %v", r.janClient.BaseURL, err)
		_ = r.RemoveServiceModels(ctx, r.janClient.BaseURL) // Ignore error in cron context
		return
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

	log.Infof("CheckInferenceModels: received %d models from %s", len(models), r.janClient.BaseURL)

	if err := r.SetModels(ctx, r.janClient.BaseURL, models); err != nil {
		log.Infof("CheckInferenceModels: failed to cache models for %s: %v", r.janClient.BaseURL, err)
		return
	}

	log.Infof("CheckInferenceModels: cached %d models for %s", len(models), r.janClient.BaseURL)
}
