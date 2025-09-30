package response

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/user"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
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
func (h *ResponseModelService) CreateResponse(ctx context.Context, userID uint, request *requesttypes.CreateResponseRequest) (*ResponseCreationResult, *common.Error) {
	// Validate the request
	success, err := ValidateCreateResponseRequest(request)
	if !success {
		return nil, err
	}

	// TODO add the logic to get the API key for the user
	key := ""

	// Convert response request to chat completion request using domain service
	chatCompletionRequest := h.responseService.ConvertToChatCompletionRequest(request)
	if chatCompletionRequest == nil {
		return nil, common.NewErrorWithMessage("Input validation error", "i9j0k1l2-m3n4-5678-ijkl-901234567890")
	}

	// Handle conversation logic using domain service
	conversation, err := h.responseService.HandleConversation(ctx, userID, request)
	if err != nil {
		return nil, err
	}

	// If previous_response_id is provided, prepend conversation history to input messages
	if request.PreviousResponseID != nil && *request.PreviousResponseID != "" {
		conversationMessages, err := h.responseService.ConvertConversationItemsToMessages(ctx, conversation)
		if err != nil {
			return nil, err
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

	// Convert input to JSON string
	inputJSON, jsonErr := json.Marshal(request.Input)
	if jsonErr != nil {
		return nil, common.NewError(jsonErr, "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}

	// Build Response object from parameters
	response := NewResponse(userID, conversationID, request.Model, string(inputJSON), request.SystemPrompt, responseParams)

	responseEntity, err := h.responseService.CreateResponse(ctx, response)
	if err != nil {
		return nil, err
	}

	// Append input messages to conversation (only if conversation exists)
	if conversation != nil {
		success, err := h.responseService.AppendMessagesToConversation(ctx, conversation, chatCompletionRequest.Messages, &responseEntity.ID)
		if !success {
			return nil, err
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
func (h *ResponseModelService) GetResponseHandler(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "response not found in context")
		return
	}

	result, err := h.GetResponse(responseEntity)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, err.GetCode(), err.Error())
		return
	}

	h.sendSuccessResponse(reqCtx, result)
}

// doGetResponse performs the business logic for getting a response
func (h *ResponseModelService) GetResponse(responseEntity *Response) (responsetypes.Response, *common.Error) {
	// Convert domain response to API response using domain service
	apiResponse := h.responseService.ConvertDomainResponseToAPIResponse(responseEntity)
	return apiResponse, nil
}

// DeleteResponse handles the business logic for deleting a response
func (h *ResponseModelService) DeleteResponseHandler(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "b2c3d4e5-f6g7-8901-bcde-f23456789012", "response not found in context")
		return
	}

	result, err := h.DeleteResponse(reqCtx, responseEntity)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, err.GetCode(), err.Error())
		return
	}

	h.sendSuccessResponse(reqCtx, result)
}

// doDeleteResponse performs the business logic for deleting a response
func (h *ResponseModelService) DeleteResponse(reqCtx *gin.Context, responseEntity *Response) (responsetypes.Response, *common.Error) {
	// Delete the response from database
	success, err := h.responseService.DeleteResponse(reqCtx, responseEntity.ID)
	if !success {
		return responsetypes.Response{}, err
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

	return deletedResponse, nil
}

// CancelResponse handles the business logic for cancelling a response
func (h *ResponseModelService) CancelResponseHandler(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "d4e5f6g7-h8i9-0123-defg-456789012345", "response not found in context")
		return
	}

	result, err := h.CancelResponse(responseEntity)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, err.GetCode(), err.Error())
		return
	}

	h.sendSuccessResponse(reqCtx, result)
}

// doCancelResponse performs the business logic for cancelling a response
func (h *ResponseModelService) CancelResponse(responseEntity *Response) (responsetypes.Response, *common.Error) {
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

	return mockResponse, nil
}

// ListInputItems handles the business logic for listing input items
func (h *ResponseModelService) ListInputItemsHandler(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "e5f6g7h8-i9j0-1234-efgh-567890123456", "response not found in context")
		return
	}

	result, err := h.ListInputItems(reqCtx, responseEntity)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, err.GetCode(), err.Error())
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doListInputItems performs the business logic for listing input items
func (h *ResponseModelService) ListInputItems(reqCtx *gin.Context, responseEntity *Response) (responsetypes.OpenAIListResponse[responsetypes.InputItem], *common.Error) {
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
		return responsetypes.OpenAIListResponse[responsetypes.InputItem]{}, err
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

	return responsetypes.OpenAIListResponse[responsetypes.InputItem]{
		JanStatus: &status,
		Object:    &objectType,
		HasMore:   &hasMore,
		FirstID:   firstID,
		LastID:    lastID,
		T:         paginatedItems,
	}, nil
}

// sendErrorResponse sends a standardized error response
func (h *ResponseModelService) sendErrorResponse(reqCtx *gin.Context, statusCode int, errorCode, errorMessage string) {
	reqCtx.AbortWithStatusJSON(statusCode, responsetypes.ErrorResponse{
		Code:  errorCode,
		Error: errorMessage,
	})
}

// sendSuccessResponse sends a standardized success response
func (h *ResponseModelService) sendSuccessResponse(reqCtx *gin.Context, data any) {
	reqCtx.JSON(http.StatusOK, data.(responsetypes.Response))
}
