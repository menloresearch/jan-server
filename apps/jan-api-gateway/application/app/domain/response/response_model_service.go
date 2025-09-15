package response

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

// ResponseHandler handles the business logic for response API endpoints
type ResponseHandler struct {
	UserService         *user.UserService
	authService         *auth.AuthService
	apikeyService       *apikey.ApiKeyService
	conversationService *conversation.ConversationService
	responseService     *ResponseService
	streamHandler       *StreamHandler
	nonStreamHandler    *NonStreamHandler
}

// NewResponseHandler creates a new ResponseHandler instance
func NewResponseHandler(
	userService *user.UserService,
	authService *auth.AuthService,
	apikeyService *apikey.ApiKeyService,
	conversationService *conversation.ConversationService,
	responseService *ResponseService,
) *ResponseHandler {
	responseHandler := &ResponseHandler{
		UserService:         userService,
		authService:         authService,
		apikeyService:       apikeyService,
		conversationService: conversationService,
		responseService:     responseService,
	}

	// Initialize specialized handlers
	responseHandler.streamHandler = NewStreamHandler(responseHandler)
	responseHandler.nonStreamHandler = NewNonStreamHandler(responseHandler)

	return responseHandler
}

// CreateResponse handles the business logic for creating a response
func (h *ResponseHandler) CreateResponse(reqCtx *gin.Context) {
	// Get user from middleware context
	userEntity, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "user not found in context")
		return
	}

	// Parse and validate the request body
	var request requesttypes.CreateResponseRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "j0k1l2m3-n4o5-6789-jklm-012345678901", "invalid request body: "+err.Error())
		return
	}

	// Validate the request
	if validationErrors := ValidateCreateResponseRequest(&request); validationErrors != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "k1l2m3n4-o5p6-7890-klmn-123456789012", "validation failed")
		return
	}

	// Get API key for the user
	key, err := h.getAPIKey(reqCtx)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "019929d1-1e85-72c1-a1cf-e151403692dc", "invalid apikey")
		return
	}

	// Check if model exists in registry
	modelRegistry := inferencemodelregistry.GetInstance()
	mToE := modelRegistry.GetModelToEndpoints()
	endpoints, ok := mToE[request.Model]
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "59253517-df33-44bf-9333-c927402e4e2e", fmt.Sprintf("Model: %s does not exist", request.Model))
		return
	}

	// Convert response request to chat completion request using domain service
	chatCompletionRequest := h.responseService.ConvertToChatCompletionRequest(&request)
	if chatCompletionRequest == nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "019929e6-c3ee-76e3-b0fd-b046611b79ad", "unsupported input type for chat completion")
		return
	}

	// Check if model endpoint exists
	janInferenceClient := janinference.NewJanInferenceClient(reqCtx)
	endpointExists := false
	for _, endpoint := range endpoints {
		if endpoint == janInferenceClient.BaseURL {
			endpointExists = true
			break
		}
	}

	if !endpointExists {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "6c6e4ea0-53d2-4c6c-8617-3a645af59f43", "Client does not exist")
		return
	}

	// Handle conversation logic using domain service
	conversation, err := h.responseService.HandleConversation(reqCtx, userEntity.ID, &request)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", err.Error())
		return
	}

	// If previous_response_id is provided, prepend conversation history to input messages
	if request.PreviousResponseID != nil && *request.PreviousResponseID != "" {
		conversationMessages, err := h.responseService.ConvertConversationItemsToMessages(reqCtx, conversation)
		if err != nil {
			h.sendErrorResponse(reqCtx, http.StatusBadRequest, "c3d4e5f6-g7h8-9012-cdef-345678901234", err.Error())
			return
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
	responseEntity, err := h.responseService.CreateResponse(reqCtx, userEntity.ID, conversationID, request.Model, request.Input, request.SystemPrompt, responseParams)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusInternalServerError, "c7d8e9f0-a1b2-3456-cdef-012345678901", "failed to create response record: "+err.Error())
		return
	}

	// Append input messages to conversation (only if conversation exists)
	if conversation != nil {
		if err := h.responseService.AppendMessagesToConversation(reqCtx, conversation, chatCompletionRequest.Messages, &responseEntity.ID); err != nil {
			h.sendErrorResponse(reqCtx, http.StatusBadRequest, "b2c3d4e5-f6g7-8901-bcde-f23456789012", err.Error())
			return
		}
	}

	// Delegate to specialized handlers based on streaming preference
	if request.Stream != nil && *request.Stream {
		// Handle streaming response
		h.streamHandler.CreateStreamResponse(reqCtx, &request, key, conversation, responseEntity, chatCompletionRequest)
	} else {
		// Handle non-streaming response
		h.nonStreamHandler.CreateNonStreamResponse(reqCtx, &request, key, conversation, responseEntity, chatCompletionRequest)
	}
}

// handleConversation handles conversation creation or loading based on the request

// GetResponse handles the business logic for getting a response
func (h *ResponseHandler) GetResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc", "response not found in context")
		return
	}

	// Convert domain response to API response using domain service
	apiResponse := h.responseService.ConvertDomainResponseToAPIResponse(responseEntity)
	h.sendSuccessResponse(reqCtx, apiResponse)
}

// DeleteResponse handles the business logic for deleting a response
func (h *ResponseHandler) DeleteResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc", "response not found in context")
		return
	}

	// Delete the response from database
	if err := h.responseService.DeleteResponse(reqCtx, responseEntity.ID); err != nil {
		h.sendErrorResponse(reqCtx, http.StatusInternalServerError, "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc", "failed to delete response: "+err.Error())
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
func (h *ResponseHandler) CancelResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc", "response not found in context")
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
func (h *ResponseHandler) ListInputItems(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc", "response not found in context")
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
		h.sendErrorResponse(reqCtx, http.StatusInternalServerError, "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc", "failed to get input items: "+err.Error())
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
func (h *ResponseHandler) sendErrorResponse(reqCtx *gin.Context, statusCode int, errorCode, errorMessage string) {
	reqCtx.AbortWithStatusJSON(statusCode, responsetypes.ErrorResponse{
		Code:  errorCode,
		Error: errorMessage,
	})
}

// sendSuccessResponse sends a standardized success response
func (h *ResponseHandler) sendSuccessResponse(reqCtx *gin.Context, data interface{}) {
	status := responsetypes.ResponseCodeOk
	objectType := responsetypes.ObjectTypeResponse
	reqCtx.JSON(http.StatusOK, responsetypes.OpenAIGeneralResponse[responsetypes.Response]{
		JanStatus: &status,
		Object:    &objectType,
		T:         data.(responsetypes.Response),
	})
}

// getAPIKey retrieves the API key for the user
func (h *ResponseHandler) getAPIKey(reqCtx *gin.Context) (string, error) {
	userClaim, _ := auth.GetUserClaimFromRequestContext(reqCtx)
	key := "AnonymousUserKey"

	if userClaim != nil {
		user, err := h.UserService.FindByEmail(reqCtx, userClaim.Email)
		if err != nil {
			return "", fmt.Errorf("failed to find user: %w", err)
		}

		apikeyEntities, err := h.apikeyService.Find(reqCtx, apikey.ApiKeyFilter{
			OwnerPublicID: &user.PublicID,
			ApikeyType:    ptr.ToString(string(apikey.ApikeyTypeAdmin)),
		}, nil)
		if err != nil {
			return "", fmt.Errorf("failed to find API keys: %w", err)
		}

		// Generate default key if none exists
		if len(apikeyEntities) == 0 {
			key, hash, err := h.apikeyService.GenerateKeyAndHash(reqCtx, apikey.ApikeyTypeEphemeral)
			if err != nil {
				return "", fmt.Errorf("failed to generate API key: %w", err)
			}

			entity, err := h.apikeyService.CreateApiKey(reqCtx, &apikey.ApiKey{
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
			apikeyEntities = []*apikey.ApiKey{entity}
		}
		key = apikeyEntities[0].KeyHash
	}

	return key, nil
}
