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
	return uc.ConvertResponse(response), nil
}

// ConvertResponse converts OpenAI response to our domain response
func (uc *CompletionNonStreamHandler) ConvertResponse(response *openai.ChatCompletionResponse) *CompletionResponse {
	choices := make([]CompletionChoice, len(response.Choices))
	for i, choice := range response.Choices {
		choices[i] = CompletionChoice{
			Index: choice.Index,
			Message: CompletionMessage{
				Role:             choice.Message.Role,
				Content:          choice.Message.Content,
				ReasoningContent: choice.Message.ReasoningContent,
				FunctionCall:     choice.Message.FunctionCall,
				ToolCalls:        choice.Message.ToolCalls,
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

// CompletionResponse represents the response from chat completion
type CompletionResponse struct {
	ID       string             `json:"id"`
	Object   string             `json:"object"`
	Created  int64              `json:"created"`
	Model    string             `json:"model"`
	Choices  []CompletionChoice `json:"choices"`
	Usage    Usage              `json:"usage"`
	Metadata *ResponseMetadata  `json:"metadata,omitempty"`
}

// CompletionChoice represents a single completion choice from the AI model
type CompletionChoice struct {
	Index        int               `json:"index"`
	Message      CompletionMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

// CompletionMessage represents a message in the completion response
type CompletionMessage struct {
	Role             string               `json:"role"`
	Content          string               `json:"content"`
	ReasoningContent string               `json:"reasoning_content"`
	FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
	ToolCalls        []openai.ToolCall    `json:"tool_calls,omitempty"`
}

// Usage represents token usage statistics for the completion
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ResponseMetadata contains additional metadata about the completion response
type ResponseMetadata struct {
	ConversationID      string `json:"conversation_id"`
	ConversationCreated bool   `json:"conversation_created"`
	ConversationTitle   string `json:"conversation_title"`
	UserItemID          string `json:"user_item_id"`
	AssistantItemID     string `json:"assistant_item_id"`
	Store               bool   `json:"store"`
	StoreReasoning      bool   `json:"store_reasoning"`
}

// ModifyCompletionResponse modifies the completion response to include item ID and metadata
func (uc *CompletionNonStreamHandler) ModifyCompletionResponse(response *CompletionResponse, conv *conversation.Conversation, conversationCreated bool, assistantItem *conversation.Item, userItemID string, assistantItemID string, store bool, storeReasoning bool) *CompletionResponse {
	// Replace ID with item ID if assistant item exists
	if assistantItem != nil {
		response.ID = assistantItem.PublicID
	}

	// Add metadata if conversation exists
	if conv != nil {
		title := ""
		if conv.Title != nil {
			title = *conv.Title
		}
		response.Metadata = &ResponseMetadata{
			ConversationID:      conv.PublicID,
			ConversationCreated: conversationCreated,
			ConversationTitle:   title,
			UserItemID:          userItemID,
			AssistantItemID:     assistantItemID,
			Store:               store,
			StoreReasoning:      storeReasoning,
		}
	}

	return response
}
