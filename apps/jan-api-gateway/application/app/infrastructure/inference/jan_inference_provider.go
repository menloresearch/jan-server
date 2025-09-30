package inference

import (
	"context"
	"encoding/json"
	"io"
	"time"

	openai "github.com/sashabaranov/go-openai"
	inference "menlo.ai/jan-api-gateway/app/domain/inference"
	inferencemodel "menlo.ai/jan-api-gateway/app/domain/inference_model"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

const JanDefaultProviderID = "jan-default"

const janModelsCacheTTL = 10 * time.Minute

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

		if _, err = io.Copy(writer, resp.RawResponse.Body); err != nil {
			writer.CloseWithError(err)
		}
	}()

	return reader, nil
}

func (p *JanProvider) GetModels(ctx context.Context) (*inference.ModelsResponse, error) {
	cacheKey := cache.JanModelsCacheKey
	cachedResponseJSON, err := p.cache.GetWithFallback(ctx, cacheKey, func() (string, error) {
		clientResponse, err := p.client.GetModels(ctx)
		if err != nil {
			return "", err
		}

		models := make([]inference.InferenceProviderModel, len(clientResponse.Data))
		for i, model := range clientResponse.Data {
			models[i] = inference.InferenceProviderModel{
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

		response := &inference.ModelsResponse{
			Object: clientResponse.Object,
			Data:   models,
		}

		responseJSON, jsonErr := json.Marshal(response)
		if jsonErr != nil {
			return "", jsonErr
		}

		return string(responseJSON), nil
	}, janModelsCacheTTL)

	if err != nil {
		return nil, err
	}

	var response inference.ModelsResponse
	if jsonErr := json.Unmarshal([]byte(cachedResponseJSON), &response); jsonErr != nil {
		return nil, jsonErr
	}
	return &response, nil
}

func (p *JanProvider) ValidateModel(ctx context.Context, model string) error {
	return nil
}
