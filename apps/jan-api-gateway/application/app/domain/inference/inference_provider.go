package inference

import (
	"context"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// InferenceProvider defines the interface for AI inference services
type InferenceProvider interface {
	// CreateCompletion creates a non-streaming chat completion
	CreateCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error)

	// CreateCompletionStream creates a streaming chat completion
	CreateCompletionStream(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (io.ReadCloser, error)

	// GetModels returns available models
	GetModels(ctx context.Context) (*ModelsResponse, error)

	// ValidateModel checks if a model is supported
	ValidateModel(model string) error
}

// ModelsResponse represents the response from GetModels
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model represents an AI model
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}
