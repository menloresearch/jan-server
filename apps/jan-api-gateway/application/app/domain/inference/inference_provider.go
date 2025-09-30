package inference

import (
	"context"
	"io"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
)

type ProviderSelection struct {
	ProviderID     string
	ProviderType   modelprovider.ProviderType
	Vendor         modelprovider.ProviderVendor
	OrganizationID *uint
	ProjectID      *uint
	ProjectIDs     []uint
	Model          string
}

type ProviderSummary struct {
	ProviderID string
	Name       string
	Type       modelprovider.ProviderType
	Vendor     modelprovider.ProviderVendor
	APIKeyHint string
	Active     bool
}

type ProviderSummaryFilter struct {
	OrganizationID *uint
	ProjectID      *uint
	ProjectIDs     *[]uint
	Type           *modelprovider.ProviderType
	Vendor         *modelprovider.ProviderVendor
	Active         *bool
}

type InferenceProvider interface {
	CreateCompletion(ctx context.Context, selection ProviderSelection, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)
	CreateCompletionStream(ctx context.Context, selection ProviderSelection, request openai.ChatCompletionRequest) (io.ReadCloser, error)
	GetModels(ctx context.Context, selection ProviderSelection) (*ModelsResponse, error)
	ValidateModel(ctx context.Context, selection ProviderSelection, model string) error
	ListProviders(ctx context.Context, filter ProviderSummaryFilter) ([]ProviderSummary, error)
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID           string                       `json:"id"`
	Object       string                       `json:"object"`
	Created      int                          `json:"created"`
	OwnedBy      string                       `json:"owned_by"`
	ProviderID   string                       `json:"provider_id"`
	ProviderType modelprovider.ProviderType   `json:"provider_type"`
	Vendor       modelprovider.ProviderVendor `json:"vendor"`
}
