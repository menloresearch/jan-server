package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

type CompletionAPI struct {
	completionNonStreamHandler *CompletionNonStreamHandler
	completionStreamHandler    *CompletionStreamHandler
	conversationService        *conversation.ConversationService
	authService                *auth.AuthService
}

func NewCompletionAPI(completionNonStreamHandler *CompletionNonStreamHandler, completionStreamHandler *CompletionStreamHandler, conversationService *conversation.ConversationService, authService *auth.AuthService) *CompletionAPI {
	return &CompletionAPI{
		completionNonStreamHandler: completionNonStreamHandler,
		completionStreamHandler:    completionStreamHandler,
		conversationService:        conversationService,
		authService:                authService,
	}
}

func (completionAPI *CompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

// ExtendedChatCompletionRequest extends OpenAI's request with conversation field
type ExtendedChatCompletionRequest struct {
	openai.ChatCompletionRequest
	Conversation string `json:"conversation,omitempty"`
}

// CreateChatCompletion
// @Summary Create a chat completion
// @Description Generates a model response for the given chat conversation. If `stream` is true, the response is sent as a stream of events. If `stream` is false or omitted, a single JSON response is returned.
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body ExtendedChatCompletionRequest true "Extended chat completion request payload"
// @Success 200 {object} CompletionResponse "Successful non-streaming response"
// @Success 200 {string} string "Successful streaming response (SSE format, event: 'data', data: JSON object per chunk)"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/chat/completions [post]
func (api *CompletionAPI) PostCompletion(reqCtx *gin.Context) {
	var request ExtendedChatCompletionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "cf237451-8932-48d1-9cf6-42c4db2d4805",
			Error: err.Error(),
		})
		return
	}

	// Get user ID for saving messages
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "0199506b-314d-70e2-a8aa-d5fde1569d1d",
			Error: "user not found",
		})
		return
	}
	// TODO: Implement admin API key check

	// Handle conversation management
	conv, conversationCreated, convErr := api.handleConversationManagement(reqCtx, request.Conversation, request.Messages)
	if convErr != nil {
		// Conversation doesn't exist, return error
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  convErr.GetCode(),
			Error: convErr.GetMessage(),
		})
		return
	}

	// Generate item IDs for tracking
	userItemID, _ := idgen.GenerateSecureID("msg", 42)
	assistantItemID, _ := idgen.GenerateSecureID("msg", 42)

	// Handle streaming vs non-streaming requests
	if request.Stream {

		// Send conversation metadata event
		api.sendConversationMetadata(reqCtx, conv, conversationCreated, userItemID, assistantItemID)

		// Handle streaming completion
		err := api.completionStreamHandler.StreamCompletion(reqCtx, "", request.ChatCompletionRequest, conv, user, userItemID, assistantItemID)
		if err != nil {
			// Check if context was cancelled (timeout)
			if reqCtx.Request.Context().Err() == context.DeadlineExceeded {
				reqCtx.AbortWithStatusJSON(
					http.StatusRequestTimeout,
					responses.ErrorResponse{
						Code: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
					})
			} else if reqCtx.Request.Context().Err() == context.Canceled {
				reqCtx.AbortWithStatusJSON(
					http.StatusRequestTimeout,
					responses.ErrorResponse{
						Code: "b2c3d4e5-f6g7-8901-bcde-f23456789012",
					})
			} else {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responses.ErrorResponse{
						Code:  err.GetCode(),
						Error: err.GetMessage(),
					})
			}
			return
		}
		return
	} else {

		response, err := api.completionNonStreamHandler.CreateCompletion(reqCtx.Request.Context(), "", request.ChatCompletionRequest)
		if err != nil {
			reqCtx.AbortWithStatusJSON(
				http.StatusBadRequest,
				responses.ErrorResponse{
					Code:  err.GetCode(),
					Error: err.GetMessage(),
				})
			return
		}

		// Save messages to conversation and get the assistant message item
		var latestMessage []openai.ChatCompletionMessage
		if len(request.Messages) > 0 {
			latestMessage = []openai.ChatCompletionMessage{request.Messages[len(request.Messages)-1]}
		}
		assistantItem, _ := api.saveMessagesToConversation(reqCtx.Request.Context(), conv, user.ID, latestMessage, userItemID, assistantItemID, response.Choices[0].Message.Content)

		// Modify response to include item ID and metadata
		modifiedResponse := api.completionNonStreamHandler.ModifyCompletionResponse(response, conv, conversationCreated, assistantItem, userItemID, assistantItemID)
		reqCtx.JSON(http.StatusOK, modifiedResponse)
		return
	}
}

// handleConversationManagement handles conversation loading or creation and returns conversation, created flag, and error
func (api *CompletionAPI) handleConversationManagement(reqCtx *gin.Context, conversationID string, messages []openai.ChatCompletionMessage) (*conversation.Conversation, bool, *common.Error) {
	if conversationID != "" {
		// Try to load existing conversation
		conv, convErr := api.loadConversation(reqCtx, conversationID)
		if convErr != nil {
			return nil, false, convErr
		}
		return conv, false, nil
	} else {
		// Create new conversation
		conv, conversationCreated := api.createNewConversation(reqCtx, messages)
		return conv, conversationCreated, nil
	}
}

// loadConversation loads an existing conversation by ID
func (api *CompletionAPI) loadConversation(reqCtx *gin.Context, conversationID string) (*conversation.Conversation, *common.Error) {
	ctx := reqCtx.Request.Context()

	// Get user from context (set by AppUserAuthMiddleware)
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return nil, common.NewErrorWithMessage("User not authenticated", "c1d2e3f4-g5h6-7890-cdef-123456789012")
	}

	conv, convErr := api.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if convErr != nil {
		return nil, common.NewErrorWithMessage(fmt.Sprintf("Conversation with ID '%s' not found", conversationID), "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}

	if conv == nil {
		return nil, common.NewErrorWithMessage(fmt.Sprintf("Conversation with ID '%s' not found", conversationID), "b2c3d4e5-f6g7-8901-bcde-f23456789012")
	}

	return conv, nil
}

// createNewConversation creates a new conversation
func (api *CompletionAPI) createNewConversation(reqCtx *gin.Context, messages []openai.ChatCompletionMessage) (*conversation.Conversation, bool) {
	ctx := reqCtx.Request.Context()

	// Get user from context (set by AppUserAuthMiddleware)
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		// If no user context, return nil
		return nil, false
	}

	title := api.generateTitleFromMessages(messages)
	conv, convErr := api.conversationService.CreateConversation(ctx, user.ID, &title, true, map[string]string{
		"model": "jan-v1-4b", // Default model
	})
	if convErr != nil {
		// If creation fails, return nil
		return nil, false
	}

	return conv, true // Created new conversation
}

// TODO should be generate from models, now we just use the first user message
// generateTitleFromMessages creates a title from the first user message
func (api *CompletionAPI) generateTitleFromMessages(messages []openai.ChatCompletionMessage) string {
	if len(messages) == 0 {
		return "New Conversation"
	}

	// Find the first user message
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			title := strings.TrimSpace(msg.Content)
			if len(title) > 50 {
				return title[:50] + "..."
			}
			return title
		}
	}

	return "New Conversation"
}

// sendConversationMetadata sends conversation metadata as SSE event
func (api *CompletionAPI) sendConversationMetadata(reqCtx *gin.Context, conv *conversation.Conversation, conversationCreated bool, userItemID string, assistantItemID string) {
	if conv == nil {
		return
	}

	metadata := map[string]any{
		"object":               "chat.completion.metadata",
		"conversation_id":      conv.PublicID,
		"conversation_created": conversationCreated,
		"conversation_title":   conv.Title,
		"user_item_id":         userItemID,
		"assistant_item_id":    assistantItemID,
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return
	}

	reqCtx.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(jsonData))))
	reqCtx.Writer.Flush()
}

// convertOpenAIMessageToConversationContent converts OpenAI message content to conversation content
func (api *CompletionAPI) convertOpenAIMessageToConversationContent(msg openai.ChatCompletionMessage) []conversation.Content {
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

	return content
}

// convertOpenAIRoleToConversationRole converts OpenAI role to conversation role
func (api *CompletionAPI) convertOpenAIRoleToConversationRole(role string) conversation.ItemRole {
	switch role {
	case openai.ChatMessageRoleSystem:
		return conversation.ItemRoleSystem
	case openai.ChatMessageRoleUser:
		return conversation.ItemRoleUser
	case openai.ChatMessageRoleAssistant:
		return conversation.ItemRoleAssistant
	default:
		return conversation.ItemRoleUser
	}
}

// saveMessagesToConversation saves messages to conversation with optional custom IDs
func (api *CompletionAPI) saveMessagesToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, messages []openai.ChatCompletionMessage, userItemID string, assistantItemID string, assistantContent string) (*conversation.Item, *common.Error) {
	if conv == nil {
		return nil, nil // No conversation to save to
	}

	var assistantItem *conversation.Item

	// Convert OpenAI messages to conversation items
	for i, msg := range messages {
		role := api.convertOpenAIRoleToConversationRole(msg.Role)
		content := api.convertOpenAIMessageToConversationContent(msg)

		// Add item to conversation - use userItemID for the last user message
		var item *conversation.Item
		var err *common.Error
		if i == len(messages)-1 && msg.Role == openai.ChatMessageRoleUser && userItemID != "" {
			item, err = api.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &role, content, userItemID)
		} else {
			item, err = api.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &role, content)
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
		var item *conversation.Item
		var err *common.Error
		if assistantItemID != "" {
			item, err = api.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &assistantRole, content, assistantItemID)
		} else {
			item, err = api.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &assistantRole, content)
		}
		if err != nil {
			return nil, common.NewError(err, "c3d4e5f6-g7h8-9012-cdef-345678901234")
		}
		assistantItem = item
	}

	return assistantItem, nil
}
