package chat

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/inference"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
)

// ChatUseCase handles chat completion business logic
type ChatUseCase struct {
	inferenceProvider   inference.InferenceProvider
	modelRegistry       *inferencemodelregistry.InferenceModelRegistry
	conversationService *conversation.ConversationService
}

// NewChatUseCase creates a new ChatUseCase instance
func NewChatUseCase(inferenceProvider inference.InferenceProvider, conversationService *conversation.ConversationService) *ChatUseCase {
	return &ChatUseCase{
		inferenceProvider:   inferenceProvider,
		modelRegistry:       inferencemodelregistry.GetInstance(),
		conversationService: conversationService,
	}
}

// CompletionRequest represents a chat completion request
type CompletionRequest struct {
	Model          string                         `json:"model"`
	Messages       []openai.ChatCompletionMessage `json:"messages"`
	Temperature    *float32                       `json:"temperature,omitempty"`
	MaxTokens      *int                           `json:"max_tokens,omitempty"`
	Stream         bool                           `json:"stream"`
	TopP           *float32                       `json:"top_p,omitempty"`
	Metadata       map[string]interface{}         `json:"metadata,omitempty"`
	ConversationID *string                        `json:"conversation_id,omitempty"`
}

// CompletionResponse represents a chat completion response
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

// CompletionChoice represents a completion choice
type CompletionChoice struct {
	Index        int               `json:"index"`
	Message      CompletionMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

// CompletionMessage represents a completion message
type CompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ValidateRequest validates the completion request
func (uc *ChatUseCase) ValidateRequest(req *CompletionRequest) *common.Error {
	if req.Model == "" {
		return common.NewError("a1b2c3d4-e5f6-7890-abcd-ef1234567890", "Model is required")
	}

	if len(req.Messages) == 0 {
		return common.NewError("b2c3d4e5-f6g7-8901-bcde-f23456789012", "Messages are required")
	}

	// Validate model exists in registry
	if err := uc.validateModel(req.Model); !err.IsEmpty() {
		return err
	}

	return common.EmptyError
}

// validateModel checks if the model is available
func (uc *ChatUseCase) validateModel(model string) *common.Error {
	mToE := uc.modelRegistry.GetModelToEndpoints()
	_, ok := mToE[model]
	if !ok {
		return common.NewError("59253517-df33-44bf-9333-c927402e4e2e", fmt.Sprintf("Model: %s does not exist", model))
	}

	// Check if the model is available for this provider
	if err := uc.inferenceProvider.ValidateModel(model); err != nil {
		return common.NewError("6c6e4ea0-53d2-4c6c-8617-3a645af59f43", "Client does not exist")
	}

	return common.EmptyError
}

// CreateCompletion creates a non-streaming completion
func (uc *ChatUseCase) CreateCompletion(ctx context.Context, apiKey string, req *CompletionRequest) (*CompletionResponse, *common.Error) {
	// Validate request
	if err := uc.ValidateRequest(req); !err.IsEmpty() {
		return nil, err
	}

	// Convert to OpenAI format
	openaiReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}

	if req.Temperature != nil {
		openaiReq.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		openaiReq.MaxTokens = *req.MaxTokens
	}
	if req.TopP != nil {
		openaiReq.TopP = *req.TopP
	}

	// Call inference provider
	response, err := uc.inferenceProvider.CreateCompletion(ctx, apiKey, openaiReq)
	if err != nil {
		return nil, common.NewError("c7d8e9f0-g1h2-3456-cdef-789012345678", fmt.Sprintf("Inference failed: %v", err))
	}

	// Convert response
	return uc.convertResponse(response), common.EmptyError
}

// convertResponse converts OpenAI response to our domain response
func (uc *ChatUseCase) convertResponse(response *openai.ChatCompletionResponse) *CompletionResponse {
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

// SaveMessagesToConversation saves all messages from the completion request to the conversation
func (c *ChatUseCase) SaveMessagesToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage) *common.Error {
	_, err := c.saveMessagesToConversationWithAssistant(ctx, conv, userID, messages, "")
	return err
}

// SaveMessagesToConversationWithAssistant saves all messages including the assistant response and returns the assistant item
func (c *ChatUseCase) SaveMessagesToConversationWithAssistant(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, assistantContent string) (*conversation.Item, *common.Error) {
	return c.saveMessagesToConversationWithAssistant(ctx, conv, userID, messages, assistantContent)
}

// saveMessagesToConversationWithAssistant internal method that saves messages and optionally the assistant response
func (c *ChatUseCase) saveMessagesToConversationWithAssistant(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, assistantContent string) (*conversation.Item, *common.Error) {
	if conv == nil {
		return nil, common.EmptyError // No conversation to save to
	}

	var assistantItem *conversation.Item

	// Convert OpenAI messages to conversation items
	for _, msg := range messages {
		// Convert role
		var role conversation.ItemRole
		switch msg.Role {
		case openai.ChatMessageRoleSystem:
			role = conversation.ItemRoleSystem
		case openai.ChatMessageRoleUser:
			role = conversation.ItemRoleUser
		case openai.ChatMessageRoleAssistant:
			role = conversation.ItemRoleAssistant
		default:
			role = conversation.ItemRoleUser
		}

		// Convert content
		content := make([]conversation.Content, 0, len(msg.MultiContent))
		for _, contentPart := range msg.MultiContent {
			if contentPart.Type == openai.ChatMessagePartTypeText {
				content = append(content, conversation.Content{
					Type: "text",
					Text: &conversation.Text{
						Value: contentPart.Text,
					},
				})
			}
		}

		// If no multi-content, use simple text content
		if len(content) == 0 && msg.Content != "" {
			content = append(content, conversation.Content{
				Type: "text",
				Text: &conversation.Text{
					Value: msg.Content,
				},
			})
		}

		// Add item to conversation
		item, err := c.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &role, content)
		if !err.IsEmpty() {
			return nil, common.NewError("b2c3d4e5-f6g7-8901-bcde-f23456789012", fmt.Sprintf("Failed to save message: %s", err.Message))
		}

		// If this is an assistant message, store it for return
		if msg.Role == openai.ChatMessageRoleAssistant {
			assistantItem = item
		}
	}

	// If assistant content is provided and no assistant message was found in the input, create one
	if assistantContent != "" && assistantItem == nil {
		content := []conversation.Content{
			{
				Type: "text",
				Text: &conversation.Text{
					Value: assistantContent,
				},
			},
		}

		assistantRole := conversation.ItemRoleAssistant
		item, err := c.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &assistantRole, content)
		if !err.IsEmpty() {
			return nil, common.NewError("c3d4e5f6-g7h8-9012-cdef-345678901234", fmt.Sprintf("Failed to save assistant message: %s", err.Message))
		}
		assistantItem = item
	}

	return assistantItem, common.EmptyError
}
