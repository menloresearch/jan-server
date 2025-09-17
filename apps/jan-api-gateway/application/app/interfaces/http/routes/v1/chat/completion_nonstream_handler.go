package chat

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/inference"
)

// CompletionNonStreamHandler handles non-streaming completion business logic
type CompletionNonStreamHandler struct {
	inferenceProvider   inference.InferenceProvider
	conversationService *conversation.ConversationService
}

// NewCompletionNonStreamHandler creates a new CompletionNonStreamHandler instance
func NewCompletionNonStreamHandler(inferenceProvider inference.InferenceProvider, conversationService *conversation.ConversationService) *CompletionNonStreamHandler {
	return &CompletionNonStreamHandler{
		inferenceProvider:   inferenceProvider,
		conversationService: conversationService,
	}
}

// CreateCompletion creates a non-streaming completion
func (uc *CompletionNonStreamHandler) CreateCompletion(ctx context.Context, apiKey string, request openai.ChatCompletionRequest) (*CompletionResponse, *common.Error) {

	// Call inference provider
	response, err := uc.inferenceProvider.CreateCompletion(ctx, apiKey, request)
	if err != nil {
		return nil, common.NewError(err, "c7d8e9f0-g1h2-3456-cdef-789012345678")
	}

	// Convert response
	return uc.convertResponse(response), nil
}

// convertResponse converts OpenAI response to our domain response
func (uc *CompletionNonStreamHandler) convertResponse(response *openai.ChatCompletionResponse) *CompletionResponse {
	choices := make([]CompletionChoice, len(response.Choices))
	for i, choice := range response.Choices {
		choices[i] = CompletionChoice{
			Index: choice.Index,
			Message: CompletionMessage{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: string(choice.FinishReason),
		}
	}

	return &CompletionResponse{
		ID:      response.ID,
		Object:  response.Object,
		Created: response.Created,
		Model:   response.Model,
		Choices: choices,
		Usage: Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}
}

// Note: Message saving logic has been moved to completion_route.go to eliminate duplication
// These methods are kept for backward compatibility but should be deprecated

// CompletionResponse represents the response from chat completion
type CompletionResponse struct {
	ID       string                 `json:"id"`
	Object   string                 `json:"object"`
	Created  int64                  `json:"created"`
	Model    string                 `json:"model"`
	Choices  []CompletionChoice     `json:"choices"`
	Usage    Usage                  `json:"usage"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type CompletionChoice struct {
	Index        int               `json:"index"`
	Message      CompletionMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type CompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModifyCompletionResponse modifies the completion response to include item ID and metadata
func (uc *CompletionNonStreamHandler) ModifyCompletionResponse(response *CompletionResponse, conv *conversation.Conversation, conversationCreated bool, assistantItem *conversation.Item, userItemID string, assistantItemID string) *CompletionResponse {
	// Create modified response
	modifiedResponse := &CompletionResponse{
		ID:      response.ID, // Default to original ID
		Object:  response.Object,
		Created: response.Created,
		Model:   response.Model,
		Choices: make([]CompletionChoice, len(response.Choices)),
		Usage: Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}

	// Copy choices
	for i, choice := range response.Choices {
		modifiedResponse.Choices[i] = CompletionChoice{
			Index: choice.Index,
			Message: CompletionMessage{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			},
			FinishReason: choice.FinishReason,
		}
	}

	// Replace ID with item ID if assistant item exists
	if assistantItem != nil {
		modifiedResponse.ID = assistantItem.PublicID
	}

	// Add metadata if conversation exists
	if conv != nil {
		modifiedResponse.Metadata = map[string]interface{}{
			"conversation_id":      conv.PublicID,
			"conversation_created": conversationCreated,
			"conversation_title":   conv.Title,
			"user_item_id":         userItemID,
			"assistant_item_id":    assistantItemID,
		}
	}

	return modifiedResponse
}
