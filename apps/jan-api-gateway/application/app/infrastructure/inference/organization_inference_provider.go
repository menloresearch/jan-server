package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	openai "github.com/sashabaranov/go-openai"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/gemini"
	"menlo.ai/jan-api-gateway/app/utils/httpclients/openrouter"
)

const organizationModelsCacheTTL = 10 * time.Minute

type OrganizationProvider struct {
	descriptor       *modelprovider.ModelProvider
	apiKey           string
	cache            *cache.RedisCacheService
	openRouterClient *openrouter.Client
	geminiClient     *gemini.Client
}

func NewOrganizationProvider(descriptor *modelprovider.ModelProvider, apiKey string, cacheService *cache.RedisCacheService, openRouterClient *openrouter.Client, geminiClient *gemini.Client) *OrganizationProvider {
	return &OrganizationProvider{
		descriptor:       descriptor,
		apiKey:           apiKey,
		cache:            cacheService,
		openRouterClient: openRouterClient,
		geminiClient:     geminiClient,
	}
}

func (p *OrganizationProvider) ID() string {
	return p.descriptor.PublicID
}

func (p *OrganizationProvider) Name() string {
	return p.descriptor.Name
}

func (p *OrganizationProvider) Type() modelprovider.ProviderType {
	return p.descriptor.Type
}

func (p *OrganizationProvider) Vendor() modelprovider.ProviderVendor {
	return p.descriptor.Vendor
}

func (p *OrganizationProvider) APIKeyHint() string {
	return p.descriptor.APIKeyHint
}

func (p *OrganizationProvider) Active() bool {
	return p.descriptor.Active
}

func (p *OrganizationProvider) CreateCompletion(ctx context.Context, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	switch p.descriptor.Vendor {
	case modelprovider.ProviderVendorOpenRouter:
		if p.openRouterClient == nil {
			return nil, fmt.Errorf("openrouter client not configured")
		}
		return p.openRouterClient.CreateChatCompletion(ctx, p.apiKey, request)
	case modelprovider.ProviderVendorGemini:
		if p.geminiClient == nil {
			return nil, fmt.Errorf("gemini client not configured")
		}
		return p.geminiClient.CreateChatCompletion(ctx, p.apiKey, request)
	default:
		return nil, fmt.Errorf("unsupported vendor: %s", p.descriptor.Vendor)
	}
}

func (p *OrganizationProvider) CreateCompletionStream(ctx context.Context, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	switch p.descriptor.Vendor {
	case modelprovider.ProviderVendorOpenRouter:
		if p.openRouterClient == nil {
			return nil, fmt.Errorf("openrouter client not configured")
		}
		return p.openRouterClient.CreateChatCompletionStream(ctx, p.apiKey, request)
	case modelprovider.ProviderVendorGemini:
		return nil, fmt.Errorf("gemini streaming not supported")
	default:
		return nil, fmt.Errorf("unsupported vendor: %s", p.descriptor.Vendor)
	}
}

func (p *OrganizationProvider) GetModels(ctx context.Context) (*ModelsResponse, error) {
	cacheKey := fmt.Sprintf("%s:%s", cache.ModelsCacheKey, p.ID())
	cachedResponseJSON, err := p.cache.GetWithFallback(ctx, cacheKey, func() (string, error) {
		response, err := p.fetchModels(ctx)
		if err != nil {
			return "", err
		}
		payload, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			return "", marshalErr
		}
		return string(payload), nil
	}, organizationModelsCacheTTL)
	if err != nil {
		return nil, err
	}
	var models ModelsResponse
	if err := json.Unmarshal([]byte(cachedResponseJSON), &models); err != nil {
		return nil, err
	}
	return &models, nil
}

func (p *OrganizationProvider) fetchModels(ctx context.Context) (*ModelsResponse, error) {
	switch p.descriptor.Vendor {
	case modelprovider.ProviderVendorOpenRouter:
		if p.openRouterClient == nil {
			return nil, fmt.Errorf("openrouter client not configured")
		}
		resp, err := p.openRouterClient.GetModels(ctx, p.apiKey)
		if err != nil {
			return nil, err
		}
		models := make([]InferenceProviderModel, len(resp.Data))
		for i, model := range resp.Data {
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
		return &ModelsResponse{Object: resp.Object, Data: models}, nil
	case modelprovider.ProviderVendorGemini:
		if p.geminiClient == nil {
			return nil, fmt.Errorf("gemini client not configured")
		}
		resp, err := p.geminiClient.GetModels(ctx, p.apiKey)
		if err != nil {
			return nil, err
		}
		models := make([]InferenceProviderModel, len(resp.Data))
		for i, model := range resp.Data {
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
		return &ModelsResponse{Object: resp.Object, Data: models}, nil
	default:
		return nil, fmt.Errorf("unsupported vendor: %s", p.descriptor.Vendor)
	}
}

func (p *OrganizationProvider) ValidateModel(ctx context.Context, model string) error {
	modelsResp, err := p.GetModels(ctx)
	if err != nil {
		return err
	}
	for _, m := range modelsResp.Data {
		if m.ID == model {
			return nil
		}
	}
	return fmt.Errorf("model %s not found for provider %s", model, p.ID())
}
