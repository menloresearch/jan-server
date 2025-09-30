package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

const (
	JanDefaultProviderID = "jan-default"
	janModelsCacheTTL    = 10 * time.Minute
	janClientTimeout     = 20 * time.Second
)

type JanProvider struct {
	client       *janinference.JanInferenceClient
	cache        *cache.RedisCacheService
	providerName string
}

func NewJanProvider(client *janinference.JanInferenceClient, cacheService *cache.RedisCacheService) *JanProvider {
	return &JanProvider{
		client:       client,
		cache:        cacheService,
		providerName: "Jan",
	}
}

func (p *JanProvider) ID() string {
	return JanDefaultProviderID
}

func (p *JanProvider) Name() string {
	return p.providerName
}

func (p *JanProvider) Type() modelprovider.ProviderType {
	return modelprovider.ProviderTypeJan
}

func (p *JanProvider) Vendor() modelprovider.ProviderVendor {
	return modelprovider.ProviderVendorJan
}

func (p *JanProvider) APIKeyHint() string {
	return ""
}

func (p *JanProvider) Active() bool {
	return true
}

func (p *JanProvider) CreateCompletion(ctx context.Context, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return p.client.CreateChatCompletion(ctx, "", request)
}

func (p *JanProvider) CreateCompletionStream(ctx context.Context, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		req := janinference.JanInferenceRestyClient.R().SetBody(request)
		resp, err := req.
			SetContext(ctx).
			SetDoNotParseResponse(true).
			Post("/v1/chat/completions")
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		defer resp.RawResponse.Body.Close()

		if resp.IsError() {
			body, readErr := io.ReadAll(resp.RawResponse.Body)
			if readErr != nil {
				writer.CloseWithError(fmt.Errorf("jan provider: streaming request failed with status %d", resp.StatusCode()))
				return
			}
			writer.CloseWithError(fmt.Errorf("jan provider: streaming request failed with status %d: %s", resp.StatusCode(), strings.TrimSpace(string(body))))
			return
		}

		if _, err = io.Copy(writer, resp.RawResponse.Body); err != nil {
			writer.CloseWithError(err)
		}
	}()

	return reader, nil
}

func (p *JanProvider) GetModels(ctx context.Context) (*ModelsResponse, error) {
	cacheKey := cache.JanModelsCacheKey
	cachedResponseJSON, err := p.cache.GetWithFallback(ctx, cacheKey, func() (string, error) {
		response, fetchErr := p.fetchModels(ctx)
		if fetchErr != nil {
			return "", fetchErr
		}
		payload, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			return "", marshalErr
		}
		return string(payload), nil
	}, janModelsCacheTTL)

	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if jsonErr := json.Unmarshal([]byte(cachedResponseJSON), &response); jsonErr != nil {
		return nil, jsonErr
	}
	return &response, nil
}

func (p *JanProvider) RefreshModels(ctx context.Context) (*ModelsResponse, error) {
	response, err := p.fetchModels(ctx)
	if err != nil {
		p.clearJanModelsCache(ctx)
		return nil, err
	}

	payload, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	if err := p.cache.Set(ctx, cache.JanModelsCacheKey, string(payload), janModelsCacheTTL); err != nil {
		return nil, err
	}

	p.invalidateAggregatedCaches(ctx)

	if err := p.validateJanCache(ctx, response); err != nil {
		p.clearJanModelsCache(ctx)
		return nil, err
	}

	return response, nil
}

func (p *JanProvider) ValidateModel(ctx context.Context, model string) error {
	modelID := strings.TrimSpace(model)
	if modelID == "" {
		return fmt.Errorf("model id cannot be empty")
	}

	resp, err := p.GetModels(ctx)
	if err == nil && p.containsModel(resp, modelID) {
		return nil
	}

	if err != nil {
		logger.GetLogger().Warnf("jan provider: failed to load models from cache: %v", err)
	}

	refreshed, refreshErr := p.RefreshModels(ctx)
	if refreshErr != nil {
		if err != nil {
			return fmt.Errorf("failed to validate model %s: cache error %v, refresh error %w", modelID, err, refreshErr)
		}
		return fmt.Errorf("failed to validate model %s: %w", modelID, refreshErr)
	}

	if p.containsModel(refreshed, modelID) {
		return nil
	}

	return fmt.Errorf("model %s not found for provider %s", modelID, p.ID())
}

func (p *JanProvider) fetchModels(ctx context.Context) (*ModelsResponse, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, janClientTimeout)
	defer cancel()

	clientResponse, err := p.client.GetModels(fetchCtx)
	if err != nil {
		return nil, err
	}

	return p.toModelsResponse(clientResponse), nil
}

func (p *JanProvider) toModelsResponse(clientResponse *janinference.ModelsResponse) *ModelsResponse {
	models := make([]InferenceProviderModel, len(clientResponse.Data))
	for i, model := range clientResponse.Data {
		models[i] = InferenceProviderModel{
			Model: inferencemodel.Model{
				ID:      model.ID,
				Object:  model.Object,
				Created: model.Created,
				OwnedBy: model.OwnedBy,
			},
			ProviderID:   p.ID(),
			ProviderType: p.Type(),
			Vendor:       p.Vendor(),
		}
	}

	return &ModelsResponse{
		Object: clientResponse.Object,
		Data:   models,
	}
}

func (p *JanProvider) invalidateAggregatedCaches(ctx context.Context) {
	patterns := []string{
		strings.ReplaceAll(cache.OrganizationModelsCacheKeyPattern, "%d", "*"),
		strings.ReplaceAll(cache.ProjectModelsCacheKeyPattern, "%d", "*"),
	}

	for _, pattern := range patterns {
		if err := p.cache.DeletePattern(ctx, pattern); err != nil {
			logger.GetLogger().Warnf("jan provider: failed to clear cache pattern %s: %v", pattern, err)
		}
	}
}

func (p *JanProvider) clearJanModelsCache(ctx context.Context) {
	if err := p.cache.Unlink(ctx, cache.JanModelsCacheKey); err != nil {
		logger.GetLogger().Warnf("jan provider: failed to remove cache key %s: %v", cache.JanModelsCacheKey, err)
	}
	p.invalidateAggregatedCaches(ctx)
}

func (p *JanProvider) validateJanCache(ctx context.Context, expected *ModelsResponse) error {
	cached, err := p.cache.Get(ctx, cache.JanModelsCacheKey)
	if err != nil {
		return fmt.Errorf("failed to read jan models cache: %w", err)
	}

	var decoded ModelsResponse
	if err := json.Unmarshal([]byte(cached), &decoded); err != nil {
		return fmt.Errorf("failed to decode jan models cache: %w", err)
	}

	if len(decoded.Data) != len(expected.Data) {
		return fmt.Errorf("jan models cache mismatch: expected %d models, got %d", len(expected.Data), len(decoded.Data))
	}

	return nil
}

func (p *JanProvider) containsModel(response *ModelsResponse, id string) bool {
	if response == nil {
		return false
	}
	for _, model := range response.Data {
		if model.ID == id {
			return true
		}
	}
	return false
}
