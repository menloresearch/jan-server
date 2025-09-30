package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	inference "menlo.ai/jan-api-gateway/app/domain/inference"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	modelproviderservice "menlo.ai/jan-api-gateway/app/domain/modelprovider/service"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/gemini"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/openrouter"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

const aggregatedModelsCacheTTL = 10 * time.Minute

type MultiProviderInference struct {
	janProvider       *JanProvider
	providerService   *modelproviderservice.ModelProviderService
	cache             *cache.RedisCacheService
	openRouterClient  *openrouter.Client
	mu                sync.RWMutex
	geminiClient      *gemini.Client
	organizationCache map[string]providerInstance
}

func NewMultiProviderInference(jan *JanProvider, providerService *modelproviderservice.ModelProviderService, cacheService *cache.RedisCacheService, openRouterClient *openrouter.Client, geminiClient *gemini.Client) *MultiProviderInference {
	return &MultiProviderInference{
		janProvider:       jan,
		providerService:   providerService,
		cache:             cacheService,
		openRouterClient:  openRouterClient,
		geminiClient:      geminiClient,
		organizationCache: make(map[string]providerInstance),
	}
}

func (m *MultiProviderInference) CreateCompletion(ctx context.Context, selection inference.ProviderSelection, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	provider, err := m.resolveProvider(ctx, selection)
	if err != nil {
		return nil, err
	}

	logger.GetLogger().WithFields(logrus.Fields{
		"provider_id":     provider.ID(),
		"provider_type":   provider.Type(),
		"provider_vendor": provider.Vendor(),
	}).Info("multi-provider inference: CreateCompletion")
	return provider.CreateCompletion(ctx, request)
}

func (m *MultiProviderInference) CreateCompletionStream(ctx context.Context, selection inference.ProviderSelection, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	provider, err := m.resolveProvider(ctx, selection)
	if err != nil {
		return nil, err
	}
	return provider.CreateCompletionStream(ctx, request)
}

func (m *MultiProviderInference) GetModels(ctx context.Context, selection inference.ProviderSelection) (*inference.ModelsResponse, error) {
	if strings.TrimSpace(selection.ProviderID) != "" {
		provider, err := m.resolveProvider(ctx, selection)
		if err != nil {
			return nil, err
		}
		return provider.GetModels(ctx)
	}

	models, err := m.getAggregatedModels(ctx, selection)
	if err != nil {
		return nil, err
	}

	return &inference.ModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}

func (m *MultiProviderInference) ValidateModel(ctx context.Context, selection inference.ProviderSelection, model string) error {
	provider, err := m.resolveProvider(ctx, selection)
	if err != nil {
		return err
	}
	return provider.ValidateModel(ctx, model)
}

func (m *MultiProviderInference) ListProviders(ctx context.Context, filter inference.ProviderSummaryFilter) ([]inference.ProviderSummary, error) {
	summaries := make([]inference.ProviderSummary, 0)
	seen := make(map[string]struct{})

	if shouldIncludeJan(filter) {
		summaries = append(summaries, inference.ProviderSummary{
			ProviderID: JanDefaultProviderID,
			Name:       m.janProvider.Name(),
			Type:       m.janProvider.Type(),
			Vendor:     m.janProvider.Vendor(),
			Active:     m.janProvider.Active(),
		})
		seen[JanDefaultProviderID] = struct{}{}
	}

	providerFilter := modelprovider.ProviderFilter{}
	if filter.OrganizationID != nil {
		providerFilter.OrganizationID = filter.OrganizationID
	}
	if filter.Type != nil {
		providerFilter.Type = filter.Type
	}
	if filter.Vendor != nil {
		providerFilter.Vendor = filter.Vendor
	}
	if filter.Active != nil {
		providerFilter.Active = filter.Active
	}

	providers, err := m.providerService.List(ctx, providerFilter, nil)
	if err != nil {
		return nil, err
	}

	allowedProjectIDs := make(map[uint]struct{})
	if filter.ProjectID != nil {
		allowedProjectIDs[*filter.ProjectID] = struct{}{}
	}
	if filter.ProjectIDs != nil {
		for _, pid := range *filter.ProjectIDs {
			allowedProjectIDs[pid] = struct{}{}
		}
	}

	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if provider.ProjectID != nil {
			if len(allowedProjectIDs) == 0 {
				continue
			}
			if _, ok := allowedProjectIDs[*provider.ProjectID]; !ok {
				continue
			}
		} else if filter.OrganizationID != nil {
			if provider.OrganizationID == nil || *provider.OrganizationID != *filter.OrganizationID {
				continue
			}
		}
		if _, exists := seen[provider.PublicID]; exists {
			continue
		}
		summaries = append(summaries, inference.ProviderSummary{
			ProviderID: provider.PublicID,
			Name:       provider.Name,
			Type:       provider.Type,
			Vendor:     provider.Vendor,
			APIKeyHint: provider.APIKeyHint,
			Active:     provider.Active,
		})
		seen[provider.PublicID] = struct{}{}
		if provider.Active {
			_ = m.ensureCachedProvider(ctx, provider.PublicID)
		}
	}

	return summaries, nil
}

func shouldIncludeJan(filter inference.ProviderSummaryFilter) bool {
	if filter.Type != nil && *filter.Type != modelprovider.ProviderTypeJan {
		return false
	}
	if filter.Vendor != nil && *filter.Vendor != modelprovider.ProviderVendorJan {
		return false
	}
	return true
}

func (m *MultiProviderInference) resolveProvider(ctx context.Context, selection inference.ProviderSelection) (providerInstance, error) {
	providerID := strings.TrimSpace(selection.ProviderID)
	if providerID != "" {
		if providerID == JanDefaultProviderID {
			return m.janProvider, nil
		}

		if provider := m.getCachedProvider(providerID); provider != nil {
			return provider, nil
		}

		if err := m.ensureCachedProvider(ctx, providerID); err != nil {
			return nil, err
		}

		if provider := m.getCachedProvider(providerID); provider != nil {
			return provider, nil
		}
		return nil, fmt.Errorf("provider %s not found", providerID)
	}

	modelID := strings.TrimSpace(selection.Model)
	if modelID != "" {
		provider, err := m.findProviderByModel(ctx, selection, modelID)
		if err != nil {
			return nil, err
		}
		return provider, nil
	}

	provider, err := m.selectScopedProvider(ctx, selection)
	if err != nil {
		return nil, err
	}
	if provider != nil {
		return provider, nil
	}

	return m.janProvider, nil
}

func (m *MultiProviderInference) getCachedProvider(providerID string) providerInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.organizationCache[providerID]
}

func (m *MultiProviderInference) ensureCachedProvider(ctx context.Context, providerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.organizationCache[providerID]; exists {
		return nil
	}
	provider, apiKey, err := m.providerService.GetByPublicIDWithKey(ctx, providerID)
	if err != nil {
		return err
	}
	if !provider.Active {
		return fmt.Errorf("provider %s is disabled", providerID)
	}
	instance := NewOrganizationProvider(provider, apiKey, m.cache, m.openRouterClient, m.geminiClient)
	m.organizationCache[providerID] = instance
	return nil
}

func (m *MultiProviderInference) findProviderByModel(ctx context.Context, selection inference.ProviderSelection, modelID string) (providerInstance, error) {
	projectIDs := uniqueSortedProjectIDs(selection)
	for _, projectID := range projectIDs {
		models, err := m.loadProjectModels(ctx, projectID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: failed to load project models for project %d: %v", projectID, err)
			continue
		}
		for _, model := range models {
			if model.ID == modelID {
				return m.getProviderInstance(ctx, model.ProviderID)
			}
		}
	}

	if selection.OrganizationID != nil {
		models, err := m.loadOrganizationModels(ctx, *selection.OrganizationID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: failed to load organization models for org %d: %v", *selection.OrganizationID, err)
		} else {
			for _, model := range models {
				if model.ID == modelID {
					return m.getProviderInstance(ctx, model.ProviderID)
				}
			}
		}
	}

	models, err := m.loadJanModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model %s: %w", modelID, err)
	}
	for _, model := range models {
		if model.ID == modelID {
			return m.janProvider, nil
		}
	}

	return nil, fmt.Errorf("model %s not found", modelID)
}

func (m *MultiProviderInference) getAggregatedModels(ctx context.Context, selection inference.ProviderSelection) ([]inference.InferenceProviderModel, error) {
	janModels, janErr := m.loadJanModels(ctx)
	if janErr != nil {
		logger.GetLogger().Warnf("multi-provider inference: failed to load Jan models from cache: %v", janErr)
	}

	var organizationModels []inference.InferenceProviderModel
	if selection.OrganizationID != nil {
		var err error
		organizationModels, err = m.loadOrganizationModels(ctx, *selection.OrganizationID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: failed to load organization models for org %d: %v", *selection.OrganizationID, err)
		}
	}

	projectModels := make([][]inference.InferenceProviderModel, 0)
	for _, projectID := range uniqueSortedProjectIDs(selection) {
		models, err := m.loadProjectModels(ctx, projectID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: failed to load project models for project %d: %v", projectID, err)
			continue
		}
		if len(models) > 0 {
			projectModels = append(projectModels, models)
		}
	}

	result := make([]inference.InferenceProviderModel, 0, len(janModels)+len(organizationModels))
	seen := make(map[string]struct{})

	for _, models := range projectModels {
		for _, model := range models {
			if _, exists := seen[model.ID]; exists {
				continue
			}
			seen[model.ID] = struct{}{}
			result = append(result, model)
		}
	}

	for _, model := range organizationModels {
		if _, exists := seen[model.ID]; exists {
			continue
		}
		seen[model.ID] = struct{}{}
		result = append(result, model)
	}

	for _, model := range janModels {
		if _, exists := seen[model.ID]; exists {
			continue
		}
		seen[model.ID] = struct{}{}
		result = append(result, model)
	}

	if len(result) == 0 && janErr != nil {
		return nil, janErr
	}

	return result, nil
}

func (m *MultiProviderInference) loadJanModels(ctx context.Context) ([]inference.InferenceProviderModel, error) {
	cached, err := m.cache.Get(ctx, cache.JanModelsCacheKey)
	if err == nil && cached != "" {
		var models []inference.InferenceProviderModel
		decodeErr := json.Unmarshal([]byte(cached), &models)
		if decodeErr == nil {
			return models, nil
		}
		logger.GetLogger().Warnf("multi-provider inference: unable to unmarshal cached Jan models: %v", decodeErr)
	}

	resp, err := m.janProvider.GetModels(ctx)
	if err != nil {
		return nil, err
	}
	models := resp.Data
	m.storeModelsInCache(ctx, cache.JanModelsCacheKey, models)
	return models, nil
}

func (m *MultiProviderInference) loadOrganizationModels(ctx context.Context, organizationID uint) ([]inference.InferenceProviderModel, error) {
	cacheKey := fmt.Sprintf(cache.OrganizationModelsCacheKeyPattern, organizationID)
	cached, err := m.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var models []inference.InferenceProviderModel
		decodeErr := json.Unmarshal([]byte(cached), &models)
		if decodeErr == nil {
			return models, nil
		}
		logger.GetLogger().Warnf("multi-provider inference: unable to unmarshal cached organization models for org %d: %v", organizationID, decodeErr)
	}

	models, err := m.rebuildOrganizationModels(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	m.storeModelsInCache(ctx, cacheKey, models)
	return models, nil
}

func (m *MultiProviderInference) loadProjectModels(ctx context.Context, projectID uint) ([]inference.InferenceProviderModel, error) {
	cacheKey := fmt.Sprintf(cache.ProjectModelsCacheKeyPattern, projectID)
	cached, err := m.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var models []inference.InferenceProviderModel
		decodeErr := json.Unmarshal([]byte(cached), &models)
		if decodeErr == nil {
			return models, nil
		}
		logger.GetLogger().Warnf("multi-provider inference: unable to unmarshal cached project models for project %d: %v", projectID, decodeErr)
	}

	models, err := m.rebuildProjectModels(ctx, projectID)
	if err != nil {
		return nil, err
	}
	m.storeModelsInCache(ctx, cacheKey, models)
	return models, nil
}

func (m *MultiProviderInference) rebuildOrganizationModels(ctx context.Context, organizationID uint) ([]inference.InferenceProviderModel, error) {
	active := true
	filter := modelprovider.ProviderFilter{
		OrganizationID: &organizationID,
		Active:         &active,
	}

	providers, err := m.providerService.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}

	models := make([]inference.InferenceProviderModel, 0)
	for _, provider := range providers {
		if provider == nil || !provider.Active || provider.ProjectID != nil {
			continue
		}
		providerModels, err := m.getProviderModels(ctx, provider.PublicID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: failed to load models for provider %s: %v", provider.PublicID, err)
			continue
		}
		models = append(models, providerModels...)
	}

	return models, nil
}

func (m *MultiProviderInference) rebuildProjectModels(ctx context.Context, projectID uint) ([]inference.InferenceProviderModel, error) {
	active := true
	filter := modelprovider.ProviderFilter{
		ProjectID: &projectID,
		Active:    &active,
	}

	providers, err := m.providerService.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}

	models := make([]inference.InferenceProviderModel, 0)
	for _, provider := range providers {
		if provider == nil || !provider.Active {
			continue
		}
		providerModels, err := m.getProviderModels(ctx, provider.PublicID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: failed to load models for provider %s: %v", provider.PublicID, err)
			continue
		}
		models = append(models, providerModels...)
	}

	return models, nil
}

func (m *MultiProviderInference) selectScopedProvider(ctx context.Context, selection inference.ProviderSelection) (providerInstance, error) {
	active := true

	var providerType *modelprovider.ProviderType
	if selection.ProviderType != "" {
		typ := selection.ProviderType
		providerType = &typ
	}

	var vendor *modelprovider.ProviderVendor
	if selection.Vendor != "" {
		v := selection.Vendor
		vendor = &v
	}

	projectIDs := uniqueSortedProjectIDs(selection)
	if len(projectIDs) > 0 {
		ids := make([]uint, len(projectIDs))
		copy(ids, projectIDs)
		filter := modelprovider.ProviderFilter{
			ProjectIDs: &ids,
			Active:     &active,
		}
		if providerType != nil {
			filter.Type = providerType
		}
		if vendor != nil {
			filter.Vendor = vendor
		}

		providers, err := m.providerService.List(ctx, filter, nil)
		if err != nil {
			return nil, err
		}
		instance, err := m.pickBestProvider(ctx, providers, "project")
		if err != nil {
			return nil, err
		}
		if instance != nil {
			return instance, nil
		}
	}

	if selection.OrganizationID != nil {
		filter := modelprovider.ProviderFilter{
			OrganizationID: selection.OrganizationID,
			Active:         &active,
		}
		if providerType != nil {
			filter.Type = providerType
		}
		if vendor != nil {
			filter.Vendor = vendor
		}

		providers, err := m.providerService.List(ctx, filter, nil)
		if err != nil {
			return nil, err
		}

		scoped := make([]*modelprovider.ModelProvider, 0, len(providers))
		for _, provider := range providers {
			if provider == nil {
				continue
			}
			if provider.ProjectID != nil {
				continue
			}
			scoped = append(scoped, provider)
		}
		instance, err := m.pickBestProvider(ctx, scoped, "organization")
		if err != nil {
			return nil, err
		}
		if instance != nil {
			return instance, nil
		}
	}

	return nil, nil
}

func (m *MultiProviderInference) pickBestProvider(ctx context.Context, providers []*modelprovider.ModelProvider, scope string) (providerInstance, error) {
	if len(providers) == 0 {
		return nil, nil
	}

	unique := make([]*modelprovider.ModelProvider, 0, len(providers))
	seen := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		if provider == nil || !provider.Active {
			continue
		}
		if _, ok := seen[provider.PublicID]; ok {
			continue
		}
		seen[provider.PublicID] = struct{}{}
		unique = append(unique, provider)
	}
	if len(unique) == 0 {
		return nil, nil
	}

	sort.SliceStable(unique, func(i, j int) bool {
		return providerRecency(unique[i]).After(providerRecency(unique[j]))
	})

	for _, provider := range unique {
		instance, err := m.getProviderInstance(ctx, provider.PublicID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: unable to initialize %s provider %s: %v", scope, provider.PublicID, err)
			continue
		}
		return instance, nil
	}

	return nil, nil
}

func providerRecency(provider *modelprovider.ModelProvider) time.Time {
	if provider == nil {
		return time.Time{}
	}
	if provider.LastSyncedAt != nil && !provider.LastSyncedAt.IsZero() {
		return *provider.LastSyncedAt
	}
	return provider.UpdatedAt
}

func (m *MultiProviderInference) getProviderModels(ctx context.Context, providerID string) ([]inference.InferenceProviderModel, error) {
	instance, err := m.getProviderInstance(ctx, providerID)
	if err != nil {
		return nil, err
	}
	resp, err := instance.GetModels(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (m *MultiProviderInference) getProviderInstance(ctx context.Context, providerID string) (providerInstance, error) {
	if providerID == "" {
		return nil, fmt.Errorf("empty provider id")
	}
	if providerID == JanDefaultProviderID {
		return m.janProvider, nil
	}

	if provider := m.getCachedProvider(providerID); provider != nil {
		return provider, nil
	}

	if err := m.ensureCachedProvider(ctx, providerID); err != nil {
		return nil, err
	}

	if provider := m.getCachedProvider(providerID); provider != nil {
		return provider, nil
	}
	return nil, fmt.Errorf("provider %s not found", providerID)
}

func (m *MultiProviderInference) storeModelsInCache(ctx context.Context, key string, models []inference.InferenceProviderModel) {
	payload, err := json.Marshal(models)
	if err != nil {
		logger.GetLogger().Warnf("multi-provider inference: failed to marshal models for key %s: %v", key, err)
		return
	}
	if err := m.cache.Set(ctx, key, string(payload), aggregatedModelsCacheTTL); err != nil {
		logger.GetLogger().Warnf("multi-provider inference: failed to cache models for key %s: %v", key, err)
	}
}

func uniqueSortedProjectIDs(selection inference.ProviderSelection) []uint {
	ids := make([]uint, 0, len(selection.ProjectIDs)+1)
	if selection.ProjectID != nil {
		ids = append(ids, *selection.ProjectID)
	}
	ids = append(ids, selection.ProjectIDs...)
	if len(ids) == 0 {
		return ids
	}

	seen := make(map[uint]struct{}, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	return unique
}
