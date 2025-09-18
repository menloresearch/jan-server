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
	Conversation   string `json:"conversation,omitempty"`
	Store          bool   `json:"store,omitempty"`           // If true, the response will be stored in the conversation
	StoreReasoning bool   `json:"store_reasoning,omitempty"` // If true, the reasoning will be stored in the conversation
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
// @Success 200 {object} ExtendedCompletionResponse "Successful non-streaming response"
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

		var assistantItem *conversation.Item

		// Store messages conditionally based on store flag
		if request.Store {
			// Store user message
			if storeErr := api.StoreMessagesIfRequested(reqCtx.Request.Context(), request.ChatCompletionRequest, conv, user.ID, userItemID, assistantItemID, request.Store, request.StoreReasoning); storeErr != nil {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responses.ErrorResponse{
						Code:  storeErr.GetCode(),
						Error: storeErr.GetMessage(),
					})
				return
			}

			// Store assistant response
			if assistantItem, err = api.StoreAssistantResponseIfRequested(reqCtx.Request.Context(), response, conv, user.ID, assistantItemID, request.Store, request.StoreReasoning); err != nil {
				reqCtx.AbortWithStatusJSON(
					http.StatusBadRequest,
					responses.ErrorResponse{
						Code:  err.GetCode(),
						Error: err.GetMessage(),
					})
				return
			}
		}

		// Always handle completion response for other logic (like function calls, tool calls, etc.)
		// This ensures the response is properly set up regardless of store flag
		// Skip storage if we already handled it with the new store logic
		api.handleCompletionResponseAndUpdateConversation(reqCtx.Request.Context(), response, conv, user.ID, request.Store)

		// Modify response to include item ID and metadata
		modifiedResponse := api.completionNonStreamHandler.ModifyCompletionResponse(response, conv, conversationCreated, assistantItem, userItemID, assistantItemID, request.Store, request.StoreReasoning)
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

// isFunctionToolResult checks if a message is a function tool result
func (api *CompletionAPI) isFunctionToolResult(msg openai.ChatCompletionMessage) bool {
	// Check if the message has tool role (MCP function results)
	if msg.Role == openai.ChatMessageRoleTool {
		return true
	}

	// Check if the message has tool_calls or function_call_result indicators
	if len(msg.ToolCalls) > 0 {
		return true
	}

	// Check content for function result patterns
	if msg.Content != "" {
		content := strings.ToLower(msg.Content)
		// Look for common function result patterns
		if strings.Contains(content, "function_result") ||
			strings.Contains(content, "tool_result") ||
			strings.Contains(content, "call_id") {
			return true
		}
	}

	return false
}

// parseFunctionToolResult parses function tool result from message content
func (api *CompletionAPI) parseFunctionToolResult(msg openai.ChatCompletionMessage) *conversation.Content {
	if !api.isFunctionToolResult(msg) {
		return nil
	}

	// Handle tool role messages (MCP function results)
	if msg.Role == openai.ChatMessageRoleTool {
		// Try to parse the JSON content to extract function name and result
		var toolResult map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Content), &toolResult); err == nil {
			// Extract function name from the result structure
			var functionName string
			if searchParams, ok := toolResult["searchParameters"].(map[string]interface{}); ok {
				if query, exists := searchParams["q"]; exists {
					functionName = fmt.Sprintf("google_search (query: %v)", query)
				} else {
					functionName = "google_search"
				}
			} else {
				functionName = "unknown_function"
			}

			resultContent := fmt.Sprintf("MCP Function Result - %s\nResult: %s", functionName, msg.Content)

			return &conversation.Content{
				Type: "text",
				Text: &conversation.Text{
					Value: resultContent,
				},
			}
		}

		// Fallback for non-JSON tool results
		resultContent := fmt.Sprintf("MCP Function Result: %s", msg.Content)

		return &conversation.Content{
			Type: "text",
			Text: &conversation.Text{
				Value: resultContent,
			},
		}
	}

	// If message has tool_calls, create function call result content
	if len(msg.ToolCalls) > 0 {
		toolCall := msg.ToolCalls[0]
		resultContent := fmt.Sprintf("Function Result - Call ID: %s\nType: %s\nResult: %s",
			toolCall.ID, toolCall.Type, msg.Content)
		reasoningContent := fmt.Sprintf("Tool call %s (%s) executed. Call ID: %s. Result: %s",
			toolCall.ID, toolCall.Type, toolCall.ID, msg.Content)

		return &conversation.Content{
			Type: "text",
			Text: &conversation.Text{
				Value: resultContent,
			},
			ReasoningContent: &reasoningContent,
		}
	}

	// For text-based function results, format them appropriately
	resultContent := fmt.Sprintf("Function Result: %s", msg.Content)
	reasoningContent := fmt.Sprintf("Generic function execution completed. Result: %s", msg.Content)

	return &conversation.Content{
		Type: "text",
		Text: &conversation.Text{
			Value: resultContent,
		},
		ReasoningContent: &reasoningContent,
	}
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
	case openai.ChatMessageRoleTool:
		return conversation.ItemRoleTool // Tool results use dedicated tool role
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

		// Check if this is a function tool result
		var itemType conversation.ItemType = conversation.ItemTypeMessage
		if api.isFunctionToolResult(msg) {
			itemType = conversation.ItemTypeFunctionCall
			// Parse function tool result content
			if functionResultContent := api.parseFunctionToolResult(msg); functionResultContent != nil {
				content = []conversation.Content{*functionResultContent}
			}
		}

		// Add item to conversation - use userItemID for the last user message
		var item *conversation.Item
		var err *common.Error
		if i == len(messages)-1 && msg.Role == openai.ChatMessageRoleUser && userItemID != "" {
			item, err = api.conversationService.AddItemWithID(ctx, conv, userID, itemType, &role, content, userItemID)
		} else {
			item, err = api.conversationService.AddItem(ctx, conv, userID, itemType, &role, content)
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

// handleCompletionResponseAndUpdateConversation handles completion response based on finish_reason and updates conversation
func (api *CompletionAPI) handleCompletionResponseAndUpdateConversation(ctx context.Context, response *ExtendedCompletionResponse, conv *conversation.Conversation, userID uint, skipStorage bool) {
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
			// TODO: Add logging for length finish reason
		case "content_filter":
			// Do nothing -> tracking via log
			// TODO: Add logging for content filter finish reason
		default:
			// Handle unknown finish reasons
			// TODO: Add logging for unknown finish reasons
		}
	}
}

// saveFunctionCallToConversation saves a function call to the conversation
func (api *CompletionAPI) saveFunctionCallToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, functionCall *openai.FunctionCall, reasoningContent string) {
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
func (api *CompletionAPI) saveToolCallsToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, toolCalls []openai.ToolCall, reasoningContent string) {
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
func (api *CompletionAPI) saveAssistantMessageToConversation(ctx context.Context, conv *conversation.Conversation, userID uint, content string, reasoningContent string) {
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

// saveLatestMessageToConversation saves the latest message from request to conversation
func (api *CompletionAPI) saveLatestMessageToConversation(ctx context.Context, request openai.ChatCompletionRequest, conv *conversation.Conversation, userID uint, userItemID string) {
	if conv == nil || len(request.Messages) == 0 {
		return
	}

	// Get the latest message (last message in the request)
	latestMessage := request.Messages[len(request.Messages)-1]

	// Convert OpenAI message to conversation content
	content := api.convertOpenAIMessageToConversationContent(latestMessage)

	// Determine item type
	var itemType conversation.ItemType = conversation.ItemTypeMessage
	if api.isFunctionToolResult(latestMessage) {
		itemType = conversation.ItemTypeFunctionCall
		// Parse function tool result content
		if functionResultContent := api.parseFunctionToolResult(latestMessage); functionResultContent != nil {
			content = []conversation.Content{*functionResultContent}
		}
	}

	// Convert role
	role := api.convertOpenAIRoleToConversationRole(latestMessage.Role)

	// Save the latest message to conversation
	var err *common.Error
	if userItemID != "" {
		_, err = api.conversationService.AddItemWithID(ctx, conv, userID, itemType, &role, content, userItemID)
	} else {
		_, err = api.conversationService.AddItem(ctx, conv, userID, itemType, &role, content)
	}

	if err != nil {
		// TODO: Add proper logging here
		return
	}
}

// StoreMessagesIfRequested conditionally stores messages based on the store flag
func (api *CompletionAPI) StoreMessagesIfRequested(ctx context.Context, request openai.ChatCompletionRequest, conv *conversation.Conversation, userID uint, userItemID string, assistantItemID string, store bool, storeReasoning bool) *common.Error {
	if !store {
		return nil // Don't store if store flag is false
	}

	// Validate required parameters
	if conv == nil {
		return common.NewError(nil, "c1d2e3f4-g5h6-7890-abcd-ef1234567890")
	}

	// Store the latest user message
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

	if _, err := api.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &role, content, userItemID); err != nil {
		return err
	}

	return nil
}

// StoreAssistantResponseIfRequested conditionally stores the assistant response based on the store flag
func (api *CompletionAPI) StoreAssistantResponseIfRequested(ctx context.Context, response *ExtendedCompletionResponse, conv *conversation.Conversation, userID uint, assistantItemID string, store bool, storeReasoning bool) (*conversation.Item, *common.Error) {
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
	createdItem, err := api.conversationService.AddItemWithID(ctx, conv, userID, conversation.ItemTypeMessage, &role, contentArray, assistantItemID)
	if err != nil {
		return nil, err
	}

	return createdItem, nil
}

// createContentArray creates the content array based on finish reason and choice
func (api *CompletionAPI) createContentArray(choice openai.ChatCompletionChoice, finishReason, content string) ([]conversation.Content, *common.Error) {
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
