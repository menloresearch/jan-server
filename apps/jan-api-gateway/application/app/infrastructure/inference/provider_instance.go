package inference

import (
	"context"
	"io"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
)

type providerInstance interface {
	ID() string
	Name() string
	Type() modelprovider.ProviderType
	Vendor() modelprovider.ProviderVendor
	APIKeyHint() string
	Active() bool
	CreateCompletion(ctx context.Context, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)
	CreateCompletionStream(ctx context.Context, request openai.ChatCompletionRequest) (io.ReadCloser, error)
	GetModels(ctx context.Context) (*ModelsResponse, error)
	ValidateModel(ctx context.Context, model string) error
}
