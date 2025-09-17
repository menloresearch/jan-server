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

// SaveMessagesToConversation saves all messages from the completion request to the conversation
func (c *CompletionNonStreamHandler) SaveMessagesToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage) *common.Error {
	_, err := c.saveMessagesToConversationWithAssistant(ctx, conv, userID, messages, "")
	return err
}

// SaveMessagesToConversationWithAssistant saves all messages including the assistant response and returns the assistant item
func (c *CompletionNonStreamHandler) SaveMessagesToConversationWithAssistant(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, assistantContent string) (*conversation.Item, *common.Error) {
	return c.saveMessagesToConversationWithAssistant(ctx, conv, userID, messages, assistantContent)
}

// SaveMessagesToConversationWithAssistantAndIDs saves all messages including the assistant response with custom item IDs
func (c *CompletionNonStreamHandler) SaveMessagesToConversationWithAssistantAndIDs(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, assistantContent string, userItemID string, assistantItemID string) (*conversation.Item, *common.Error) {
	return c.saveMessagesToConversationWithAssistantAndIDs(ctx, conv, userID, messages, assistantContent, userItemID, assistantItemID)
}

// saveMessagesToConversationWithAssistant internal method that saves messages and optionally the assistant response
func (c *CompletionNonStreamHandler) saveMessagesToConversationWithAssistant(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, assistantContent string) (*conversation.Item, *common.Error) {
	if conv == nil {
		return nil, nil // No conversation to save to
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
		if err != nil {
			return nil, common.NewError(err, "b2c3d4e5-f6g7-8901-bcde-f23456789012")
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
		if err != nil {
			return nil, common.NewError(err, "c3d4e5f6-g7h8-9012-cdef-345678901234")
		}
		assistantItem = item
	}

	return assistantItem, nil
}

// saveMessagesToConversationWithAssistantAndIDs internal method that saves messages with custom item IDs
func (c *CompletionNonStreamHandler) saveMessagesToConversationWithAssistantAndIDs(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, assistantContent string, userItemID string, assistantItemID string) (*conversation.Item, *common.Error) {
	if conv == nil {
		return nil, nil // No conversation to save to
	}

	var assistantItem *conversation.Item

	// Convert OpenAI messages to conversation items
	for i, msg := range messages {
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

		// Add item to conversation - use userItemID for the last user message
		var item *conversation.Item
		var err *common.Error
		if i == len(messages)-1 && msg.Role == openai.ChatMessageRoleUser {
			item, err = c.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &role, content, userItemID)
		} else {
			item, err = c.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &role, content)
		}

		if err != nil {
			return nil, common.NewError(err, "b2c3d4e5-f6g7-8901-bcde-f23456789012")
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
		item, err := c.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &assistantRole, content, assistantItemID)
		if err != nil {
			return nil, common.NewError(err, "c3d4e5f6-g7h8-9012-cdef-345678901234")
		}
		assistantItem = item
	}

	return assistantItem, nil
}

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
