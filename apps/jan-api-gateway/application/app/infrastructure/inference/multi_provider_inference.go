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
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/gemini"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/openrouter"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

const aggregatedModelsCacheTTL = 10 * time.Minute
const providerCacheTTL = time.Minute

type providerCacheEntry struct {
	instance   providerInstance
	descriptor *modelprovider.ModelProvider
	apiKey     string
	loadedAt   time.Time
}

type MultiProviderInference struct {
	janProvider       *JanProvider
	providerService   *modelprovider.ModelProviderService
	cache             *cache.RedisCacheService
	openRouterClient  *openrouter.Client
	mu                sync.RWMutex
	geminiClient      *gemini.Client
	organizationCache map[string]*providerCacheEntry
}

func NewMultiProviderInference(jan *JanProvider, providerService *modelprovider.ModelProviderService, cacheService *cache.RedisCacheService, openRouterClient *openrouter.Client, geminiClient *gemini.Client) *MultiProviderInference {
	return &MultiProviderInference{
		janProvider:       jan,
		providerService:   providerService,
		cache:             cacheService,
		openRouterClient:  openRouterClient,
		geminiClient:      geminiClient,
		organizationCache: make(map[string]*providerCacheEntry),
	}
}

func (m *MultiProviderInference) CreateCompletion(ctx context.Context, selection ProviderSelection, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
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

func (m *MultiProviderInference) CreateCompletionStream(ctx context.Context, selection ProviderSelection, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	provider, err := m.resolveProvider(ctx, selection)
	if err != nil {
		return nil, err
	}
	return provider.CreateCompletionStream(ctx, request)
}

func (m *MultiProviderInference) GetModels(ctx context.Context, selection ProviderSelection) (*ModelsResponse, error) {
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

	return &ModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}

func (m *MultiProviderInference) ValidateModel(ctx context.Context, selection ProviderSelection, model string) error {
	provider, err := m.resolveProvider(ctx, selection)
	if err != nil {
		return err
	}
	return provider.ValidateModel(ctx, model)
}

func (m *MultiProviderInference) ListProviders(ctx context.Context, filter ProviderSummaryFilter) ([]ProviderSummary, error) {
	summaries := make([]ProviderSummary, 0)
	seen := make(map[string]struct{})

	if shouldIncludeJan(filter) {
		summaries = append(summaries, ProviderSummary{
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
		summaries = append(summaries, ProviderSummary{
			ProviderID: provider.PublicID,
			Name:       provider.Name,
			Type:       provider.Type,
			Vendor:     provider.Vendor,
			APIKeyHint: provider.APIKeyHint,
			Active:     provider.Active,
		})
		seen[provider.PublicID] = struct{}{}
		if provider.Active {
			if _, err := m.getProviderEntry(ctx, provider.PublicID); err != nil {
				logger.GetLogger().Warnf("multi-provider inference: failed to warm cache for provider %s: %v", provider.PublicID, err)
			}
		}
	}

	return summaries, nil
}

func shouldIncludeJan(filter ProviderSummaryFilter) bool {
	if filter.Type != nil && *filter.Type != modelprovider.ProviderTypeJan {
		return false
	}
	if filter.Vendor != nil && *filter.Vendor != modelprovider.ProviderVendorJan {
		return false
	}
	return true
}

func (m *MultiProviderInference) resolveProvider(ctx context.Context, selection ProviderSelection) (providerInstance, error) {
	providerID := strings.TrimSpace(selection.ProviderID)
	if providerID != "" {
		if providerID == JanDefaultProviderID {
			return m.janProvider, nil
		}

		entry, err := m.getProviderEntry(ctx, providerID)
		if err != nil {
			return nil, err
		}
		if err := m.validateProviderAccess(selection, entry.descriptor); err != nil {
			return nil, err
		}
		return entry.instance, nil
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

func (m *MultiProviderInference) getCachedProvider(providerID string) (*providerCacheEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.organizationCache[providerID]
	return entry, ok
}

func (m *MultiProviderInference) storeProviderInCache(providerID string, entry *providerCacheEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.organizationCache[providerID] = entry
}

func (m *MultiProviderInference) removeProviderFromCache(providerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.organizationCache, providerID)
}

func (m *MultiProviderInference) getProviderEntry(ctx context.Context, providerID string) (*providerCacheEntry, error) {
	if entry, ok := m.getCachedProvider(providerID); ok && entry != nil {
		if entry.descriptor != nil && !entry.descriptor.Active {
			return nil, fmt.Errorf("provider %s is disabled", providerID)
		}
		if time.Since(entry.loadedAt) < providerCacheTTL {
			return entry, nil
		}
	}

	descriptor, apiKey, err := m.providerService.GetByPublicIDWithKey(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if !descriptor.Active {
		m.removeProviderFromCache(providerID)
		return nil, fmt.Errorf("provider %s is disabled", providerID)
	}

	entry := &providerCacheEntry{
		instance:   NewOrganizationProvider(descriptor, apiKey, m.cache, m.openRouterClient, m.geminiClient),
		descriptor: descriptor,
		apiKey:     apiKey,
		loadedAt:   time.Now(),
	}
	m.storeProviderInCache(providerID, entry)
	return entry, nil
}

func (m *MultiProviderInference) validateProviderAccess(selection ProviderSelection, descriptor *modelprovider.ModelProvider) error {
	if descriptor == nil {
		return nil
	}
	if descriptor.ProjectID != nil {
		if selection.ProjectID != nil && *selection.ProjectID == *descriptor.ProjectID {
			return nil
		}
		for _, id := range selection.ProjectIDs {
			if id == *descriptor.ProjectID {
				return nil
			}
		}
		return fmt.Errorf("provider %s is not accessible in the current project context", descriptor.PublicID)
	}
	if descriptor.OrganizationID == nil {
		return fmt.Errorf("provider %s is not accessible in the current organization context", descriptor.PublicID)
	}
	if selection.OrganizationID != nil && *selection.OrganizationID == *descriptor.OrganizationID {
		return nil
	}
	return fmt.Errorf("provider %s is not accessible in the current organization context", descriptor.PublicID)
}

func (m *MultiProviderInference) findProviderByModel(ctx context.Context, selection ProviderSelection, modelID string) (providerInstance, error) {
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

func (m *MultiProviderInference) getAggregatedModels(ctx context.Context, selection ProviderSelection) ([]InferenceProviderModel, error) {
	janModels, janErr := m.loadJanModels(ctx)
	if janErr != nil {
		logger.GetLogger().Warnf("multi-provider inference: failed to load Jan models from cache: %v", janErr)
	}

	var organizationModels []InferenceProviderModel
	if selection.OrganizationID != nil {
		var err error
		organizationModels, err = m.loadOrganizationModels(ctx, *selection.OrganizationID)
		if err != nil {
			logger.GetLogger().Warnf("multi-provider inference: failed to load organization models for org %d: %v", *selection.OrganizationID, err)
		}
	}

	projectModels := make([][]InferenceProviderModel, 0)
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

	result := make([]InferenceProviderModel, 0, len(janModels)+len(organizationModels))
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

func (m *MultiProviderInference) loadJanModels(ctx context.Context) ([]InferenceProviderModel, error) {
	cached, err := m.cache.Get(ctx, cache.JanModelsCacheKey)
	if err == nil && cached != "" {
		var models []InferenceProviderModel
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

func (m *MultiProviderInference) loadOrganizationModels(ctx context.Context, organizationID uint) ([]InferenceProviderModel, error) {
	cacheKey := fmt.Sprintf(cache.OrganizationModelsCacheKeyPattern, organizationID)
	cached, err := m.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var models []InferenceProviderModel
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

func (m *MultiProviderInference) loadProjectModels(ctx context.Context, projectID uint) ([]InferenceProviderModel, error) {
	cacheKey := fmt.Sprintf(cache.ProjectModelsCacheKeyPattern, projectID)
	cached, err := m.cache.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var models []InferenceProviderModel
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

func (m *MultiProviderInference) rebuildOrganizationModels(ctx context.Context, organizationID uint) ([]InferenceProviderModel, error) {
	active := true
	filter := modelprovider.ProviderFilter{
		OrganizationID: &organizationID,
		Active:         &active,
	}

	providers, err := m.providerService.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}

	models := make([]InferenceProviderModel, 0)
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

func (m *MultiProviderInference) rebuildProjectModels(ctx context.Context, projectID uint) ([]InferenceProviderModel, error) {
	active := true
	filter := modelprovider.ProviderFilter{
		ProjectID: &projectID,
		Active:    &active,
	}

	providers, err := m.providerService.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}

	models := make([]InferenceProviderModel, 0)
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

func (m *MultiProviderInference) selectScopedProvider(ctx context.Context, selection ProviderSelection) (providerInstance, error) {
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

func (m *MultiProviderInference) getProviderModels(ctx context.Context, providerID string) ([]InferenceProviderModel, error) {
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

	entry, err := m.getProviderEntry(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return entry.instance, nil
}

func (m *MultiProviderInference) storeModelsInCache(ctx context.Context, key string, models []InferenceProviderModel) {
	payload, err := json.Marshal(models)
	if err != nil {
		logger.GetLogger().Warnf("multi-provider inference: failed to marshal models for key %s: %v", key, err)
		return
	}
	if err := m.cache.Set(ctx, key, string(payload), aggregatedModelsCacheTTL); err != nil {
		logger.GetLogger().Warnf("multi-provider inference: failed to cache models for key %s: %v", key, err)
	}
}

func uniqueSortedProjectIDs(selection ProviderSelection) []uint {
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
