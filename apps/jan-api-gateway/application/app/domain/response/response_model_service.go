package response

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	inferencemodelregistry "menlo.ai/jan-api-gateway/app/domain/inference_model_registry"
	"menlo.ai/jan-api-gateway/app/domain/user"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// ResponseCreationResult represents the result of creating a response
type ResponseCreationResult struct {
	Response              *Response
	Conversation          *conversation.Conversation
	ChatCompletionRequest *openai.ChatCompletionRequest
	APIKey                string
	IsStreaming           bool
}

// ResponseModelService handles the business logic for response API endpoints
type ResponseModelService struct {
	UserService           *user.UserService
	authService           *auth.AuthService
	apikeyService         *apikey.ApiKeyService
	conversationService   *conversation.ConversationService
	responseService       *ResponseService
	streamModelService    *StreamModelService
	nonStreamModelService *NonStreamModelService
}

// NewResponseModelService creates a new ResponseModelService instance
func NewResponseModelService(
	userService *user.UserService,
	authService *auth.AuthService,
	apikeyService *apikey.ApiKeyService,
	conversationService *conversation.ConversationService,
	responseService *ResponseService,
) *ResponseModelService {
	responseModelService := &ResponseModelService{
		UserService:         userService,
		authService:         authService,
		apikeyService:       apikeyService,
		conversationService: conversationService,
		responseService:     responseService,
	}

	// Initialize specialized handlers
	responseModelService.streamModelService = NewStreamModelService(responseModelService)
	responseModelService.nonStreamModelService = NewNonStreamModelService(responseModelService)

	return responseModelService
}

// CreateResponse handles the business logic for creating a response
// Returns domain objects and business logic results, no HTTP concerns
func (h *ResponseModelService) CreateResponse(ctx context.Context, userID uint, request *requesttypes.CreateResponseRequest) (*ResponseCreationResult, error) {
	// Validate the request
	if validationErrors := ValidateCreateResponseRequest(request); validationErrors != nil {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	// Get API key for the user
	key, err := h.getAPIKeyForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("invalid apikey: %w", err)
	}

	// Check if model exists in registry
	modelRegistry := inferencemodelregistry.GetInstance()
	mToE := modelRegistry.GetModelToEndpoints()
	endpoints, ok := mToE[request.Model]
	if !ok {
		return nil, fmt.Errorf("model %s does not exist", request.Model)
	}

	// Convert response request to chat completion request using domain service
	chatCompletionRequest := h.responseService.ConvertToChatCompletionRequest(request)
	if chatCompletionRequest == nil {
		return nil, fmt.Errorf("unsupported input type for chat completion")
	}

	// Check if model endpoint exists
	janInferenceClient := janinference.NewJanInferenceClient(ctx)
	endpointExists := false
	for _, endpoint := range endpoints {
		if endpoint == janInferenceClient.BaseURL {
			endpointExists = true
			break
		}
	}

	if !endpointExists {
		return nil, fmt.Errorf("client does not exist")
	}

	// Handle conversation logic using domain service
	conversation, err := h.responseService.HandleConversation(ctx, userID, request)
	if err != nil {
		return nil, fmt.Errorf("failed to handle conversation: %w", err)
	}

	// If previous_response_id is provided, prepend conversation history to input messages
	if request.PreviousResponseID != nil && *request.PreviousResponseID != "" {
		conversationMessages, err := h.responseService.ConvertConversationItemsToMessages(ctx, conversation)
		if err != nil {
			return nil, fmt.Errorf("failed to convert conversation items to messages: %w", err)
		}
		// Prepend conversation history to the input messages
		chatCompletionRequest.Messages = append(conversationMessages, chatCompletionRequest.Messages...)
	}

	// Create response parameters
	responseParams := &ResponseParams{
		MaxTokens:         request.MaxTokens,
		Temperature:       request.Temperature,
		TopP:              request.TopP,
		TopK:              request.TopK,
		RepetitionPenalty: request.RepetitionPenalty,
		Seed:              request.Seed,
		Stop:              request.Stop,
		PresencePenalty:   request.PresencePenalty,
		FrequencyPenalty:  request.FrequencyPenalty,
		LogitBias:         request.LogitBias,
		ResponseFormat:    request.ResponseFormat,
		Metadata:          request.Metadata,
		Stream:            request.Stream,
		Background:        request.Background,
		Timeout:           request.Timeout,
		User:              request.User,
	}

	// Create response record in database
	var conversationID *uint
	if conversation != nil {
		conversationID = &conversation.ID
	}
	responseEntity, err := h.responseService.CreateResponse(ctx, userID, conversationID, request.Model, request.Input, request.SystemPrompt, responseParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create response record: %w", err)
	}

	// Append input messages to conversation (only if conversation exists)
	if conversation != nil {
		if err := h.responseService.AppendMessagesToConversation(ctx, conversation, chatCompletionRequest.Messages, &responseEntity.ID); err != nil {
			return nil, fmt.Errorf("failed to append messages to conversation: %w", err)
		}
	}

	// Return the result for the interface layer to handle
	isStreaming := request.Stream != nil && *request.Stream
	return &ResponseCreationResult{
		Response:              responseEntity,
		Conversation:          conversation,
		ChatCompletionRequest: chatCompletionRequest,
		APIKey:                key,
		IsStreaming:           isStreaming,
	}, nil
}

// handleConversation handles conversation creation or loading based on the request

// GetResponse handles the business logic for getting a response
func (h *ResponseModelService) GetResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "response not found in context")
		return
	}

	// Convert domain response to API response using domain service
	apiResponse := h.responseService.ConvertDomainResponseToAPIResponse(responseEntity)
	h.sendSuccessResponse(reqCtx, apiResponse)
}

// DeleteResponse handles the business logic for deleting a response
func (h *ResponseModelService) DeleteResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "b2c3d4e5-f6g7-8901-bcde-f23456789012", "response not found in context")
		return
	}

	// Delete the response from database
	if err := h.responseService.DeleteResponse(reqCtx, responseEntity.ID); err != nil {
		h.sendErrorResponse(reqCtx, http.StatusInternalServerError, "c3d4e5f6-g7h8-9012-cdef-345678901234", "failed to delete response: "+err.Error())
		return
	}

	// Return the deleted response data
	deletedResponse := responsetypes.Response{
		ID:          responseEntity.PublicID,
		Object:      "response",
		Created:     responseEntity.CreatedAt.Unix(),
		Model:       responseEntity.Model,
		Status:      responsetypes.ResponseStatusCancelled,
		CancelledAt: ptr.ToInt64(time.Now().Unix()),
	}

	h.sendSuccessResponse(reqCtx, deletedResponse)
}

// CancelResponse handles the business logic for cancelling a response
func (h *ResponseModelService) CancelResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "d4e5f6g7-h8i9-0123-defg-456789012345", "response not found in context")
		return
	}

	// TODO: Implement actual cancellation logic
	// For now, return the response with cancelled status
	mockResponse := responsetypes.Response{
		ID:          responseEntity.PublicID,
		Object:      "response",
		Created:     responseEntity.CreatedAt.Unix(),
		Model:       responseEntity.Model,
		Status:      responsetypes.ResponseStatusCancelled,
		CancelledAt: ptr.ToInt64(time.Now().Unix()),
	}

	h.sendSuccessResponse(reqCtx, mockResponse)
}

// ListInputItems handles the business logic for listing input items
func (h *ResponseModelService) ListInputItems(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "e5f6g7h8-i9j0-1234-efgh-567890123456", "response not found in context")
		return
	}

	// Parse pagination parameters
	limit := 20 // default limit
	if limitStr := reqCtx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Get input items for the response (only user role messages)
	userRole := conversation.ItemRole("user")
	items, err := h.responseService.GetItemsForResponse(reqCtx, responseEntity.ID, &userRole)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusInternalServerError, "f6g7h8i9-j0k1-2345-fghi-678901234567", "failed to get input items: "+err.Error())
		return
	}

	// Convert conversation items to input items using domain service
	inputItems := make([]responsetypes.InputItem, 0, len(items))
	for _, item := range items {
		inputItem := h.responseService.ConvertConversationItemToInputItem(item)
		inputItems = append(inputItems, inputItem)
	}

	// Apply pagination (simple implementation - in production you'd want cursor-based pagination)
	after := reqCtx.Query("after")
	before := reqCtx.Query("before")

	var paginatedItems []responsetypes.InputItem
	var hasMore bool

	if after != "" {
		// Find items after the specified ID
		found := false
		for _, item := range inputItems {
			if found {
				paginatedItems = append(paginatedItems, item)
				if len(paginatedItems) >= limit {
					break
				}
			}
			if item.ID == after {
				found = true
			}
		}
	} else if before != "" {
		// Find items before the specified ID
		for _, item := range inputItems {
			if item.ID == before {
				break
			}
			paginatedItems = append(paginatedItems, item)
			if len(paginatedItems) >= limit {
				break
			}
		}
	} else {
		// No pagination, return first N items
		if len(inputItems) > limit {
			paginatedItems = inputItems[:limit]
			hasMore = true
		} else {
			paginatedItems = inputItems
		}
	}

	// Set pagination metadata
	var firstID, lastID *string
	if len(paginatedItems) > 0 {
		firstID = &paginatedItems[0].ID
		lastID = &paginatedItems[len(paginatedItems)-1].ID
	}

	status := responsetypes.ResponseCodeOk
	objectType := responsetypes.ObjectTypeList

	reqCtx.JSON(http.StatusOK, responsetypes.OpenAIListResponse[responsetypes.InputItem]{
		JanStatus: &status,
		Object:    &objectType,
		HasMore:   &hasMore,
		FirstID:   firstID,
		LastID:    lastID,
		T:         paginatedItems,
	})
}

// sendErrorResponse sends a standardized error response
func (h *ResponseModelService) sendErrorResponse(reqCtx *gin.Context, statusCode int, errorCode, errorMessage string) {
	reqCtx.AbortWithStatusJSON(statusCode, responsetypes.ErrorResponse{
		Code:  errorCode,
		Error: errorMessage,
	})
}

// sendSuccessResponse sends a standardized success response
func (h *ResponseModelService) sendSuccessResponse(reqCtx *gin.Context, data interface{}) {
	status := responsetypes.ResponseCodeOk
	objectType := responsetypes.ObjectTypeResponse
	reqCtx.JSON(http.StatusOK, responsetypes.OpenAIGeneralResponse[responsetypes.Response]{
		JanStatus: &status,
		Object:    &objectType,
		T:         data.(responsetypes.Response),
	})
}

// getAPIKeyForUser retrieves the API key for the user (domain method)
func (h *ResponseModelService) getAPIKeyForUser(ctx context.Context, userID uint) (string, error) {
	// Get user by ID
	user, err := h.UserService.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// Always generate a new ephemeral key for each request
	key, hash, err := h.apikeyService.GenerateKeyAndHash(ctx, apikey.ApikeyTypeEphemeral)
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	_, err = h.apikeyService.CreateApiKey(ctx, &apikey.ApiKey{
		KeyHash:        hash,
		PlaintextHint:  fmt.Sprintf("sk-..%s", key[len(key)-3:]),
		Description:    "Default Key For User",
		Enabled:        true,
		ApikeyType:     string(apikey.ApikeyTypeEphemeral),
		OwnerPublicID:  user.PublicID,
		OrganizationID: nil,
		Permissions:    "{}",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create API key: %w", err)
	}
	return key, nil
}
