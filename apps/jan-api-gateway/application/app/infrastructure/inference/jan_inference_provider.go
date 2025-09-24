package inference

import (
	"context"
	"io"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
)

// JanInferenceProvider implements InferenceProvider using Jan Inference service
type JanInferenceProvider struct {
	client *janinference.JanInferenceClient
	cache  *cache.RedisCacheService
}

// NewJanInferenceProvider creates a new JanInferenceProvider
func NewJanInferenceProvider(client *janinference.JanInferenceClient, cacheService *cache.RedisCacheService) inference.InferenceProvider {
	return &JanInferenceProvider{
		client: client,
		cache:  cacheService,
	}
}

// CreateCompletion creates a non-streaming chat completion
func (p *JanInferenceProvider) CreateCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return p.client.CreateChatCompletion(ctx, apiKey, request)
}

// CreateCompletionStream creates a streaming chat completion
func (p *JanInferenceProvider) CreateCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (io.ReadCloser, error) {
	// Create a pipe for streaming
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		// Use the existing streaming logic but write to pipe instead of HTTP response
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

		// Stream data to pipe
		_, err = io.Copy(writer, resp.RawResponse.Body)
		if err != nil {
			writer.CloseWithError(err)
		}
	}()

	return reader, nil
}

// GetModels returns available models with Redis caching
func (p *JanInferenceProvider) GetModels(ctx context.Context) (*inference.ModelsResponse, error) {
	var response inference.ModelsResponse

	// Try to get from cache first
	err := p.cache.GetWithFallback(ctx, cache.ModelsCacheKey, &response, func() (any, error) {
		// Cache miss, call the underlying client
		clientResponse, err := p.client.GetModels(ctx)
		if err != nil {
			return nil, err
		}

		// Convert to domain models
		models := make([]inference.Model, len(clientResponse.Data))
		for i, model := range clientResponse.Data {
			models[i] = inference.Model{
				ID:      model.ID,
				Object:  model.Object,
				Created: model.Created,
				OwnedBy: model.OwnedBy,
			}
		}

		return &inference.ModelsResponse{
			Object: clientResponse.Object,
			Data:   models,
		}, nil
	}, 10*time.Minute) // Cache models list for 10 minutes

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ValidateModel checks if a model is supported
func (p *JanInferenceProvider) ValidateModel(model string) error {
	// For now, assume all models are supported by Jan Inference
	// In the future, this could check against a list of supported models
	return nil
}
