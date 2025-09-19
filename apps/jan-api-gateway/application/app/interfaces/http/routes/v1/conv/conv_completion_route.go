package conv

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
	userdomain "menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

const (
	DefaultConversationTitle = "New Conversation"
	MaxTitleLength           = 50
)

type ConvCompletionAPI struct {
	completionNonStreamHandler *CompletionNonStreamHandler
	completionStreamHandler    *CompletionStreamHandler
	conversationService        *conversation.ConversationService
	authService                *auth.AuthService
}

func NewConvCompletionAPI(completionNonStreamHandler *CompletionNonStreamHandler, completionStreamHandler *CompletionStreamHandler, conversationService *conversation.ConversationService, authService *auth.AuthService) *ConvCompletionAPI {
	return &ConvCompletionAPI{
		completionNonStreamHandler: completionNonStreamHandler,
		completionStreamHandler:    completionStreamHandler,
		conversationService:        conversationService,
		authService:                authService,
	}
}

func (completionAPI *ConvCompletionAPI) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/completions", completionAPI.PostCompletion)
}

// ExtendedChatCompletionRequest extends OpenAI's request with conversation field and store and store_reasoning fields
type ExtendedChatCompletionRequest struct {
	openai.ChatCompletionRequest
	Conversation   string `json:"conversation,omitempty"`
	Store          bool   `json:"store,omitempty"`           // If true, the response will be stored in the conversation, default is false
	StoreReasoning bool   `json:"store_reasoning,omitempty"` // If true, the reasoning will be stored in the conversation, default is false
}

// ResponseMetadata contains additional metadata about the completion response
type ResponseMetadata struct {
	ConversationID      string `json:"conversation_id"`
	ConversationCreated bool   `json:"conversation_created"`
	ConversationTitle   string `json:"conversation_title"`
	AskItemId           string `json:"ask_item_id"`
	CompletionItemId    string `json:"completion_item_id"`
	Store               bool   `json:"store"`
	StoreReasoning      bool   `json:"store_reasoning"`
}

// ExtendedCompletionResponse extends OpenAI's ChatCompletionResponse with additional metadata
type ExtendedCompletionResponse struct {
	openai.ChatCompletionResponse
	Metadata *ResponseMetadata `json:"metadata,omitempty"`
}

// PostCompletion
// @Summary Create a conversation-aware chat completion
// @Description Generates a model response for the given chat conversation with conversation persistence and management. This is the conversation-aware version of the chat completion API that supports both streaming and non-streaming modes with conversation management and storage options.
// @Description
// @Description **Streaming Mode (stream=true):**
// @Description - Returns Server-Sent Events (SSE) with real-time streaming
// @Description - First event contains conversation metadata
// @Description - Subsequent events contain completion chunks
// @Description - Final event contains "[DONE]" marker
// @Description
// @Description **Non-Streaming Mode (stream=false or omitted):**
// @Description - Returns single JSON response with complete completion
// @Description - Includes conversation metadata in response
// @Description
// @Description **Storage Options:**
// @Description - `store=true`: Saves user message and assistant response to conversation
// @Description - `store_reasoning=true`: Includes reasoning content in stored messages
// @Description - `conversation`: ID of existing conversation or empty for new conversation
// @Description
// @Description **Features:**
// @Description - Conversation persistence and history management
// @Description - Extended request format with conversation and storage options
// @Description - User authentication required
// @Description - Automatic conversation creation and management
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body ExtendedChatCompletionRequest true "Extended chat completion request with streaming, storage, and conversation options"
// @Success 200 {object} ExtendedCompletionResponse "Successful non-streaming response (when stream=false)"
// @Success 200 {string} string "Successful streaming response (when stream=true) - SSE format with data: {json} events"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload or conversation not found"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found or user not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conv/chat/completions [post]
func (api *ConvCompletionAPI) PostCompletion(reqCtx *gin.Context) {
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
			Code:          convErr.GetCode(),
			ErrorInstance: convErr.GetError(),
		})
		return
	}

	// Generate item IDs for tracking
	askItemID, _ := idgen.GenerateSecureID("msg", 42)
	completionItemID, _ := idgen.GenerateSecureID("msg", 42)

	// Handle streaming vs non-streaming requests
	var response *ExtendedCompletionResponse
	var err *common.Error

	if request.Stream {
		// Handle streaming completion - streams SSE events and accumulates response
		response, err = api.completionStreamHandler.StreamCompletionAndAccumulateResponse(reqCtx, "", request.ChatCompletionRequest, conv, conversationCreated, askItemID, completionItemID)
	} else {
		// Handle non-streaming completion
		response, err = api.completionNonStreamHandler.CallCompletionAndGetRestResponse(reqCtx.Request.Context(), "", request.ChatCompletionRequest)
	}

	if err != nil {
		reqCtx.AbortWithStatusJSON(
			http.StatusBadRequest,
			responses.ErrorResponse{
				Code:          err.GetCode(),
				ErrorInstance: err.GetError(),
			})
		return
	}

	// Process response (common logic for both streaming and non-streaming)
	modifiedResponse := api.processCompletionResponse(reqCtx, response, request, conv, user, askItemID, completionItemID, conversationCreated)

	// Only send JSON response for non-streaming requests (streaming uses SSE)
	if !request.Stream && modifiedResponse != nil {
		reqCtx.JSON(http.StatusOK, modifiedResponse)
	}
}

// processCompletionResponse handles the common response processing logic for both streaming and non-streaming
func (api *ConvCompletionAPI) processCompletionResponse(reqCtx *gin.Context, response *ExtendedCompletionResponse, request ExtendedChatCompletionRequest, conv *conversation.Conversation, user *userdomain.User, askItemID string, completionItemID string, conversationCreated bool) *ExtendedCompletionResponse {
	var assistantItem *conversation.Item

	// Store messages conditionally based on store flag
	if request.Store {
		// Store last input message (user or tool)
		if storeErr := api.StoreLastInputMessageIfRequested(reqCtx.Request.Context(), request.ChatCompletionRequest, conv, user.ID, askItemID, completionItemID, request.Store, request.StoreReasoning); storeErr != nil {
			reqCtx.AbortWithStatusJSON(
				http.StatusBadRequest,
				responses.ErrorResponse{
					Code:          storeErr.GetCode(),
					ErrorInstance: storeErr.GetError(),
				})
			return nil
		}

		// Store assistant response
		if item, err := api.StoreAssistantResponseIfRequested(reqCtx.Request.Context(), response, conv, user.ID, completionItemID, request.Store, request.StoreReasoning); err != nil {
			reqCtx.AbortWithStatusJSON(
				http.StatusBadRequest,
				responses.ErrorResponse{
					Code:          err.GetCode(),
					ErrorInstance: err.GetError(),
				})
			return nil
		} else {
			assistantItem = item
		}
	}

	// Always handle completion response for other logic (like function calls, tool calls, etc.)
	// This ensures the response is properly set up regardless of store flag
	// Skip storage if we already handled it with the new store logic
	api.handleCompletionResponseAndUpdateConversation(reqCtx.Request.Context(), response, conv, user.ID, request.Store)

	// Modify response to include item ID and metadata
	return api.completionNonStreamHandler.ModifyCompletionResponse(response, conv, conversationCreated, assistantItem, askItemID, completionItemID, request.Store, request.StoreReasoning)
}

// handleConversationManagement handles conversation loading or creation and returns conversation, created flag, and error
func (api *ConvCompletionAPI) handleConversationManagement(reqCtx *gin.Context, conversationID string, messages []openai.ChatCompletionMessage) (*conversation.Conversation, bool, *common.Error) {
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
func (api *ConvCompletionAPI) loadConversation(reqCtx *gin.Context, conversationID string) (*conversation.Conversation, *common.Error) {
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
func (api *ConvCompletionAPI) createNewConversation(reqCtx *gin.Context, messages []openai.ChatCompletionMessage) (*conversation.Conversation, bool) {
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
func (api *ConvCompletionAPI) generateTitleFromMessages(messages []openai.ChatCompletionMessage) string {
	if len(messages) == 0 {
		return DefaultConversationTitle
	}

	// Find the first user message
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			title := strings.TrimSpace(msg.Content)
			if len(title) > MaxTitleLength {
				return title[:MaxTitleLength] + "..."
			}
			return title
		}
	}

	return DefaultConversationTitle
}

// handleCompletionResponseAndUpdateConversation handles completion response based on finish_reason and updates conversation
func (api *ConvCompletionAPI) handleCompletionResponseAndUpdateConversation(ctx context.Context, response *ExtendedCompletionResponse, conv *conversation.Conversation, userID uint, skipStorage bool) {
	if conv == nil || len(response.Choices) == 0 {
		return
	}

	// Loop through all choices in the response
	for _, choice := range response.Choices {
		finishReason := choice.FinishReason
		message := choice.Message

		// Skip storage if already handled by new store logic
		if skipStorage {
			continue
		}

		switch finishReason {
		case "function_call":
			// Save the function call to the conversation
			if message.FunctionCall != nil {
				api.saveFunctionCallToConversation(ctx, conv, userID, message.FunctionCall, message.ReasoningContent)
			}
		case "tool_calls":
			// Save the tool calls to the conversation
			if len(message.ToolCalls) > 0 {
				api.saveToolCallsToConversation(ctx, conv, userID, message.ToolCalls, message.ReasoningContent)
			}
		case "stop":
			// Save the response as assistant message to the conversation
			if message.Content != "" {
				api.saveAssistantMessageToConversation(ctx, conv, userID, message.Content, message.ReasoningContent)
			}
		case "length":
			// Do nothing -> tracking via log
			logger.GetLogger().Error("length finish reason: " + message.Content)
		case "content_filter":
			// Do nothing -> tracking via log
			logger.GetLogger().Error("content filter finish reason: " + message.Content)
		default:
			// Handle unknown finish reasons
			logger.GetLogger().Error("unknown finish reason: " + message.Content)
		}
	}
}

// saveFunctionCallToConversation saves a function call to the conversation
func (api *ConvCompletionAPI) saveFunctionCallToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, functionCall *openai.FunctionCall, reasoningContent string) {
	if conv == nil || functionCall == nil {
		return
	}

	functionCallContent := []conversation.Content{
		{
			Type: "text",
			Text: &conversation.Text{
				Value: fmt.Sprintf("Function: %s\nArguments: %s", functionCall.Name, functionCall.Arguments),
			},
		},
	}

	// Add reasoning content if present
	if reasoningContent != "" {
		functionCallContent[0].ReasoningContent = &reasoningContent
	}

	// Add the function call to conversation as a separate item
	assistantRole := conversation.ItemRoleAssistant
	api.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeFunction, &assistantRole, functionCallContent)
}

// saveToolCallsToConversation saves tool calls to the conversation
func (api *ConvCompletionAPI) saveToolCallsToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, toolCalls []openai.ToolCall, reasoningContent string) {
	if conv == nil || len(toolCalls) == 0 {
		return
	}

	// Save each tool call as a separate conversation item
	for _, toolCall := range toolCalls {
		toolCallContent := []conversation.Content{
			{
				Type: "text",
				Text: &conversation.Text{
					Value: fmt.Sprintf("Tool Call ID: %s\nType: %s\nFunction: %s\nArguments: %s",
						toolCall.ID, toolCall.Type, toolCall.Function.Name, toolCall.Function.Arguments),
				},
			},
		}

		// Add reasoning content if present
		if reasoningContent != "" {
			toolCallContent[0].ReasoningContent = &reasoningContent
		}

		// Add the tool call to conversation as a separate item
		assistantRole := conversation.ItemRoleAssistant
		api.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeFunction, &assistantRole, toolCallContent)
	}
}

// saveAssistantMessageToConversation saves assistant message to the conversation
func (api *ConvCompletionAPI) saveAssistantMessageToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, content string, reasoningContent string) {
	if conv == nil || content == "" {
		return
	}

	// Create content structure
	conversationContent := []conversation.Content{
		{
			Type: "text",
			Text: &conversation.Text{
				Value: content,
			},
		},
	}

	// Add reasoning content if present
	if reasoningContent != "" {
		conversationContent[0].ReasoningContent = &reasoningContent
	}

	// Add the assistant message to conversation
	assistantRole := conversation.ItemRoleAssistant
	api.conversationService.AddItem(ctx, conv, userID, conversation.ItemTypeMessage, &assistantRole, conversationContent)
}

// StoreLastInputMessageIfRequested conditionally stores the last input message (user or tool) based on the store flag
func (api *ConvCompletionAPI) StoreLastInputMessageIfRequested(ctx context.Context, request openai.ChatCompletionRequest, conv *conversation.Conversation, userID uint, askItemID string, completionItemID string, store bool, storeReasoning bool) *common.Error {
	if !store {
		return nil // Don't store if store flag is false
	}

	// Validate required parameters
	if conv == nil {
		return common.NewError(nil, "c1d2e3f4-g5h6-7890-abcd-ef1234567890")
	}

	// Store the latest input message (user or tool)
	if len(request.Messages) == 0 {
		return nil // No messages to store
	}

	latestMessage := request.Messages[len(request.Messages)-1]
	role := conversation.ItemRole(latestMessage.Role)

	content := []conversation.Content{
		{
			Type: "text",
			Text: &conversation.Text{
				Value: latestMessage.Content,
			},
		},
	}

	if _, err := api.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &role, content, askItemID); err != nil {
		return err
	}

	return nil
}

// StoreAssistantResponseIfRequested conditionally stores the assistant response based on the store flag
func (api *ConvCompletionAPI) StoreAssistantResponseIfRequested(ctx context.Context, response *ExtendedCompletionResponse, conv *conversation.Conversation, userID uint, completionItemID string, store bool, storeReasoning bool) (*conversation.Item, *common.Error) {
	if !store {
		return nil, nil // Don't store if store flag is false
	}

	// Validate required parameters
	if response == nil {
		return nil, common.NewErrorWithMessage("Response is nil", "d2e3f4g5-h6i7-8901-bcde-f23456789012")
	}
	if conv == nil {
		return nil, common.NewErrorWithMessage("Conversation is nil", "e3f4g5h6-i7j8-9012-cdef-345678901234")
	}

	if len(response.Choices) == 0 {
		return nil, common.NewErrorWithMessage("No choices to store", "01995b18-1638-719d-8ee2-01375bb2a19c")
	}

	choice := response.Choices[0]
	content := choice.Message.Content
	reasoningContent := choice.Message.ReasoningContent
	finishReason := string(choice.FinishReason)

	// Don't store if no content available
	if content == "" && reasoningContent == "" {
		return nil, nil
	}

	// Create content array based on finish reason
	contentArray, err := api.createContentArray(choice, finishReason, content)
	if err != nil {
		return nil, err
	}

	// Add reasoning content if requested
	if storeReasoning && reasoningContent != "" {
		contentArray[0].ReasoningContent = &reasoningContent
	}

	role := conversation.ItemRoleAssistant
	createdItem, err := api.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &role, contentArray, completionItemID)
	if err != nil {
		return nil, err
	}

	return createdItem, nil
}

// createContentArray creates the content array based on finish reason and choice
func (api *ConvCompletionAPI) createContentArray(choice openai.ChatCompletionChoice, finishReason, content string) ([]conversation.Content, *common.Error) {
	switch finishReason {
	case "tool_calls":
		if len(choice.Message.ToolCalls) > 0 {
			toolCallsJSON, err := json.Marshal(choice.Message.ToolCalls)
			if err != nil {
				return nil, common.NewError(err, "f4g5h6i7-j8k9-0123-defg-456789012345")
			}
			return []conversation.Content{
				{
					Type:         "text",
					FinishReason: &finishReason,
					Text: &conversation.Text{
						Value: string(toolCallsJSON),
					},
				},
			}, nil
		}
	case "function_call":
		if choice.Message.FunctionCall != nil {
			functionCallJSON, err := json.Marshal(choice.Message.FunctionCall)
			if err != nil {
				return nil, common.NewError(err, "g5h6i7j8-k9l0-1234-efgh-567890123456")
			}
			return []conversation.Content{
				{
					Type:         "text",
					FinishReason: &finishReason,
					Text: &conversation.Text{
						Value: string(functionCallJSON),
					},
				},
			}, nil
		}
	}

	// Default case: store regular content (for "stop" and other finish reasons)
	return []conversation.Content{
		{
			Type:         "text",
			FinishReason: &finishReason,
			Text: &conversation.Text{
				Value: content,
			},
		},
	}, nil
}
