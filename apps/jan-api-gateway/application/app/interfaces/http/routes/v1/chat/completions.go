package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	chatdomain "menlo.ai/jan-api-gateway/app/domain/chat"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type CompletionAPI struct {
	apikeyService       *apikey.ApiKeyService
	chatUseCase         *chatdomain.ChatUseCase
	streamingService    *chatdomain.StreamingService
	conversationService *conversation.ConversationService
	authService         *auth.AuthService
}

func NewCompletionAPI(apikeyService *apikey.ApiKeyService, chatUseCase *chatdomain.ChatUseCase, streamingService *chatdomain.StreamingService, conversationService *conversation.ConversationService, authService *auth.AuthService) *CompletionAPI {
	return &CompletionAPI{
		apikeyService:       apikeyService,
		chatUseCase:         chatUseCase,
		streamingService:    streamingService,
		conversationService: conversationService,
		authService:         authService,
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
// @Description Generates a model response for the given chat conversation.
// @Tags Chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ExtendedChatCompletionRequest true "Extended chat completion request payload"
// @Success 200 {object} CompletionResponse "Successful response"
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
	key := ""
	// if environment_variables.EnvironmentVariables.ENABLE_ADMIN_API {
	// 	key, ok := requests.GetTokenFromBearer(reqCtx)
	// 	if !ok {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "4284adb3-7af4-428b-8064-7073cb9ca2ca",
	// 			Error: "invalid apikey",
	// 		})
	// 		return
	// 	}
	// 	hashed := api.apikeyService.HashKey(reqCtx, key)
	// 	apikeyEntity, err := api.apikeyService.FindByKeyHash(reqCtx, hashed)
	// 	if err != nil {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "d14ab75b-586b-4b55-ba65-e520a76d6559",
	// 			Error: "invalid apikey",
	// 		})
	// 		return
	// 	}
	// 	if !apikeyEntity.Enabled {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "42bd6104-28a1-45bd-a164-8e32d12b0378",
	// 			Error: "invalid apikey",
	// 		})
	// 		return
	// 	}
	// 	if apikeyEntity.ExpiresAt != nil && apikeyEntity.ExpiresAt.Before(time.Now()) {
	// 		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
	// 			Code:  "f8f2733d-c76f-40e4-95b1-584a5d054225",
	// 			Error: "apikey expired",
	// 		})
	// 		return
	// 	}
	// }

	// Convert to domain request
	domainRequest := &chatdomain.CompletionRequest{
		Model:    request.Model,
		Messages: request.Messages,
		Stream:   request.Stream,
	}

	if request.Temperature != 0 {
		domainRequest.Temperature = &request.Temperature
	}
	if request.MaxTokens != 0 {
		domainRequest.MaxTokens = &request.MaxTokens
	}
	if request.TopP != 0 {
		domainRequest.TopP = &request.TopP
	}
	if request.Metadata != nil {
		// Convert map[string]string to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range request.Metadata {
			metadata[k] = v
		}
		domainRequest.Metadata = metadata
	}

	// Handle conversation management
	var conv *conversation.Conversation
	var conversationCreated bool
	var convErr *common.Error

	if request.Conversation != "" {
		// Try to load existing conversation
		conv, convErr = api.loadConversation(reqCtx, request.Conversation)
		conversationCreated = false
		if !convErr.IsEmpty() {
			// Conversation doesn't exist, return error
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  convErr.Code,
				Error: convErr.Message,
			})
			return
		}
	} else {
		// Create new conversation
		conv, conversationCreated = api.createNewConversation(reqCtx, request.Messages)
	}

	// Always send conversation metadata event for streaming requests
	if request.Stream {
		api.sendConversationMetadata(reqCtx, conv, conversationCreated)
	}

	// Handle streaming vs non-streaming requests
	if request.Stream {
		err := api.streamingService.StreamCompletion(reqCtx, key, request.ChatCompletionRequest, conv, user)
		if !err.IsEmpty() {
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
						Code:  err.Code,
						Error: err.Message,
					})
			}
			return
		}
		return
	} else {
		// Inject conversation ID into domain request for non-streaming
		if conv != nil {
			domainRequest.ConversationID = &conv.PublicID
		}

		response, err := api.chatUseCase.CreateCompletion(reqCtx.Request.Context(), key, domainRequest)
		if !err.IsEmpty() {
			reqCtx.AbortWithStatusJSON(
				http.StatusBadRequest,
				responses.ErrorResponse{
					Code:  err.Code,
					Error: err.Message,
				})
			return
		}

		// Save messages to conversation and get the assistant message item
		assistantItem, saveErr := api.chatUseCase.SaveMessagesToConversationWithAssistant(reqCtx.Request.Context(), conv, user.ID, request.Messages, response.Choices[0].Message.Content)
		if !saveErr.IsEmpty() {
			// Log error but don't fail the request since completion was successful
			fmt.Printf("Warning: Failed to save messages to conversation: %s\n", saveErr.Message)
		}

		// Modify response to include item ID and metadata
		modifiedResponse := api.modifyCompletionResponse(response, conv, conversationCreated, assistantItem)
		reqCtx.JSON(http.StatusOK, modifiedResponse)
		return
	}
}

// modifyCompletionResponse modifies the completion response to include item ID and metadata
func (api *CompletionAPI) modifyCompletionResponse(response *chatdomain.CompletionResponse, conv *conversation.Conversation, conversationCreated bool, assistantItem *conversation.Item) *CompletionResponse {
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
		}
	}

	return modifiedResponse
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

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PostChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

// loadConversation loads an existing conversation by ID
func (api *CompletionAPI) loadConversation(reqCtx *gin.Context, conversationID string) (*conversation.Conversation, *common.Error) {
	ctx := reqCtx.Request.Context()

	// Get user from context (set by AppUserAuthMiddleware)
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return nil, common.NewError("c1d2e3f4-g5h6-7890-cdef-123456789012", "User not authenticated")
	}

	conv, convErr := api.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if !convErr.IsEmpty() {
		return nil, common.NewError("a1b2c3d4-e5f6-7890-abcd-ef1234567890", fmt.Sprintf("Conversation with ID '%s' not found", conversationID))
	}

	if conv == nil {
		return nil, common.NewError("b2c3d4e5-f6g7-8901-bcde-f23456789012", fmt.Sprintf("Conversation with ID '%s' not found", conversationID))
	}

	return conv, common.EmptyError
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
	if !convErr.IsEmpty() {
		// If creation fails, return nil
		return nil, false
	}

	return conv, true // Created new conversation
}

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
func (api *CompletionAPI) sendConversationMetadata(reqCtx *gin.Context, conv *conversation.Conversation, conversationCreated bool) {
	if conv == nil {
		return
	}

	metadata := map[string]interface{}{
		"object":               "chat.completion.metadata",
		"conversation_id":      conv.PublicID,
		"conversation_created": conversationCreated,
		"conversation_title":   conv.Title,
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return
	}

	reqCtx.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(jsonData))))
	reqCtx.Writer.Flush()
}

type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type PostChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}
