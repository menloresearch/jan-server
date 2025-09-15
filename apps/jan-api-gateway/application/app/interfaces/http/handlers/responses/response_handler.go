package responses

import (
	"encoding/json"
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
	"menlo.ai/jan-api-gateway/app/domain/response"
	"menlo.ai/jan-api-gateway/app/domain/user"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	janinference "menlo.ai/jan-api-gateway/app/utils/httpclients/jan_inference"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

const (
	// ClientCreatedRootConversationID is the special conversation ID that indicates a new conversation should be created
	ClientCreatedRootConversationID = "client-created-root"
)

// ResponseHandler handles the business logic for response API endpoints
type ResponseHandler struct {
	UserService         *user.UserService
	authService         *auth.AuthService
	apikeyService       *apikey.ApiKeyService
	conversationService *conversation.ConversationService
	responseService     *response.ResponseService
	streamHandler       *StreamHandler
	nonStreamHandler    *NonStreamHandler
}

// NewResponseHandler creates a new ResponseHandler instance
func NewResponseHandler(
	userService *user.UserService,
	authService *auth.AuthService,
	apikeyService *apikey.ApiKeyService,
	conversationService *conversation.ConversationService,
	responseService *response.ResponseService,
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

	// Convert response request to chat completion request
	chatCompletionRequest := h.convertToChatCompletionRequest(&request)
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

	// Handle conversation logic
	conversation, err := h.handleConversation(reqCtx, &request)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", err.Error())
		return
	}

	// If previous_response_id is provided, prepend conversation history to input messages
	if request.PreviousResponseID != nil && *request.PreviousResponseID != "" {
		conversationMessages, err := h.convertConversationItemsToMessages(reqCtx, conversation)
		if err != nil {
			h.sendErrorResponse(reqCtx, http.StatusBadRequest, "c3d4e5f6-g7h8-9012-cdef-345678901234", err.Error())
			return
		}
		// Prepend conversation history to the input messages
		chatCompletionRequest.Messages = append(conversationMessages, chatCompletionRequest.Messages...)
	}

	// Create response parameters
	responseParams := &response.ResponseParams{
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
		if err := h.appendMessagesToConversation(reqCtx, conversation, chatCompletionRequest.Messages, &responseEntity.ID); err != nil {
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
func (h *ResponseHandler) handleConversation(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest) (*conversation.Conversation, error) {
	// If store is explicitly set to false, don't create or use any conversation
	if request.Store != nil && !*request.Store {
		return nil, nil
	}

	// Get user from middleware context
	userEntity, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}

	// If previous_response_id is provided, load the conversation from the previous response
	if request.PreviousResponseID != nil && *request.PreviousResponseID != "" {
		// Load the previous response
		previousResponse, err := h.responseService.GetResponseByPublicID(reqCtx, *request.PreviousResponseID)
		if err != nil {
			return nil, fmt.Errorf("failed to load previous response: %w", err)
		}
		if previousResponse == nil {
			return nil, fmt.Errorf("previous response not found: %s", *request.PreviousResponseID)
		}

		// Validate that the previous response belongs to the same user
		if previousResponse.UserID != userEntity.ID {
			return nil, fmt.Errorf("previous response does not belong to the current user")
		}

		// Load the conversation from the previous response
		if previousResponse.ConversationID == nil {
			return nil, fmt.Errorf("previous response does not belong to any conversation")
		}

		conv, err := h.conversationService.GetConversationByID(reqCtx, *previousResponse.ConversationID)
		if err != nil {
			return nil, fmt.Errorf("failed to load conversation from previous response: %w", err)
		}
		return conv, nil
	}

	// Check if conversation is specified and not 'client-created-root'
	if request.Conversation != nil && *request.Conversation != "" && *request.Conversation != ClientCreatedRootConversationID {
		// Load existing conversation
		conv, err := h.conversationService.GetConversationByPublicIDAndUserID(reqCtx, *request.Conversation, userEntity.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load conversation: %w", err)
		}
		return conv, nil
	}

	// Create new conversation
	conv, err := h.conversationService.CreateConversation(reqCtx, userEntity.ID, nil, true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conv, nil
}

// appendMessagesToConversation converts chat completion messages to conversation items and appends them
func (h *ResponseHandler) appendMessagesToConversation(reqCtx *gin.Context, conv *conversation.Conversation, messages []openai.ChatCompletionMessage, responseID *uint) error {
	if len(messages) == 0 {
		return nil
	}

	// Convert messages to conversation items
	itemsToCreate := make([]*conversation.Item, len(messages))
	for i, msg := range messages {
		// Convert OpenAI role to conversation role
		var role *conversation.ItemRole
		switch msg.Role {
		case openai.ChatMessageRoleSystem:
			roleStr := conversation.ItemRole("system")
			role = &roleStr
		case openai.ChatMessageRoleUser:
			roleStr := conversation.ItemRole("user")
			role = &roleStr
		case openai.ChatMessageRoleAssistant:
			roleStr := conversation.ItemRole("assistant")
			role = &roleStr
		default:
			roleStr := conversation.ItemRole("user")
			role = &roleStr
		}

		// Create content
		content := []conversation.Content{
			{
				Type: "text",
				Text: &conversation.Text{
					Value: msg.Content,
				},
			},
		}

		itemsToCreate[i] = &conversation.Item{
			Type:       conversation.ItemType("message"),
			Role:       role,
			Content:    content,
			ResponseID: responseID,
		}
	}

	// Add items to conversation
	_, err := h.conversationService.AddMultipleItems(reqCtx, conv, conv.UserID, itemsToCreate)
	if err != nil {
		return fmt.Errorf("failed to add messages to conversation: %w", err)
	}

	return nil
}

// convertToChatCompletionRequest converts a CreateResponseRequest to a ChatCompletionRequest
func (h *ResponseHandler) convertToChatCompletionRequest(req *requesttypes.CreateResponseRequest) *openai.ChatCompletionRequest {
	var messages []openai.ChatCompletionMessage

	switch v := req.Input.(type) {
	case string:
		// Single string input
		messages = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: v,
			},
		}
	case []interface{}:
		// Array input - can be strings or message objects
		messages = []openai.ChatCompletionMessage{}
		for _, item := range v {
			switch itemVal := item.(type) {
			case string:
				// String item - treat as user message
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: itemVal,
				})
			case map[string]interface{}:
				// Message object
				role, hasRole := itemVal["role"]
				content, hasContent := itemVal["content"]

				if hasRole && hasContent {
					roleStr, roleOk := role.(string)
					contentStr, contentOk := content.(string)

					if roleOk && contentOk {
						// Map role to OpenAI role constants
						var openaiRole string
						switch roleStr {
						case "system":
							openaiRole = openai.ChatMessageRoleSystem
						case "user":
							openaiRole = openai.ChatMessageRoleUser
						case "assistant":
							openaiRole = openai.ChatMessageRoleAssistant
						default:
							openaiRole = openai.ChatMessageRoleUser
						}

						messages = append(messages, openai.ChatCompletionMessage{
							Role:    openaiRole,
							Content: contentStr,
						})
					}
				}
			}
		}
	default:
		return nil
	}

	if len(messages) == 0 {
		return nil
	}

	// Add system prompt if provided (only if no system message already exists)
	if req.SystemPrompt != nil && *req.SystemPrompt != "" {
		hasSystemMessage := false
		for _, msg := range messages {
			if msg.Role == openai.ChatMessageRoleSystem {
				hasSystemMessage = true
				break
			}
		}

		if !hasSystemMessage {
			systemMessage := openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: *req.SystemPrompt,
			}
			messages = append([]openai.ChatCompletionMessage{systemMessage}, messages...)
		}
	}

	chatReq := &openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream != nil && *req.Stream,
	}

	// Add optional parameters
	if req.MaxTokens != nil {
		chatReq.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		chatReq.Temperature = float32(*req.Temperature)
	}
	if req.TopP != nil {
		chatReq.TopP = float32(*req.TopP)
	}
	if req.Stop != nil {
		chatReq.Stop = req.Stop
	}
	if req.PresencePenalty != nil {
		chatReq.PresencePenalty = float32(*req.PresencePenalty)
	}
	if req.FrequencyPenalty != nil {
		chatReq.FrequencyPenalty = float32(*req.FrequencyPenalty)
	}

	return chatReq
}

// GetResponse handles the business logic for getting a response
func (h *ResponseHandler) GetResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := response.GetResponseFromContext(reqCtx)
	if !ok {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc", "response not found in context")
		return
	}

	// Convert domain response to API response
	apiResponse := h.convertDomainResponseToAPIResponse(reqCtx, responseEntity)
	h.sendSuccessResponse(reqCtx, apiResponse)
}

// DeleteResponse handles the business logic for deleting a response
func (h *ResponseHandler) DeleteResponse(reqCtx *gin.Context) {
	// Get response from middleware context
	responseEntity, ok := response.GetResponseFromContext(reqCtx)
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
	responseEntity, ok := response.GetResponseFromContext(reqCtx)
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
	responseEntity, ok := response.GetResponseFromContext(reqCtx)
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

	// Convert conversation items to input items
	inputItems := make([]responsetypes.InputItem, 0, len(items))
	for _, item := range items {
		inputItem := h.convertConversationItemToInputItem(item)
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

// convertConversationItemToInputItem converts a conversation item to an input item
func (h *ResponseHandler) convertConversationItemToInputItem(item *conversation.Item) responsetypes.InputItem {
	inputItem := responsetypes.InputItem{
		ID:      item.PublicID,
		Object:  "input_item",
		Created: item.CreatedAt.Unix(),
		Type:    requesttypes.InputType(item.Type),
	}

	// Extract text content from the item
	if len(item.Content) > 0 {
		for _, content := range item.Content {
			if content.Type == "text" && content.Text != nil {
				inputItem.Text = &content.Text.Value
				break
			} else if content.Type == "input_text" && content.InputText != nil {
				inputItem.Text = content.InputText
				break
			}
		}
	}

	// TODO: Handle other content types (image, file, web_search, etc.)
	// This would require mapping the conversation.Item content to the appropriate InputItem fields

	return inputItem
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

// convertDomainResponseToAPIResponse converts a domain response to API response format
func (h *ResponseHandler) convertDomainResponseToAPIResponse(reqCtx *gin.Context, responseEntity *response.Response) responsetypes.Response {
	apiResponse := responsetypes.Response{
		ID:      responseEntity.PublicID,
		Object:  "response",
		Created: responseEntity.CreatedAt.Unix(),
		Model:   responseEntity.Model,
		Status:  responsetypes.ResponseStatus(responseEntity.Status),
		Input:   responseEntity.Input, // This is already JSON string
	}

	// Parse and set output if available
	if responseEntity.Output != nil {
		apiResponse.Output = responseEntity.Output
	}

	// Set optional fields
	if responseEntity.SystemPrompt != nil {
		apiResponse.SystemPrompt = responseEntity.SystemPrompt
	}
	if responseEntity.MaxTokens != nil {
		apiResponse.MaxTokens = responseEntity.MaxTokens
	}
	if responseEntity.Temperature != nil {
		apiResponse.Temperature = responseEntity.Temperature
	}
	if responseEntity.TopP != nil {
		apiResponse.TopP = responseEntity.TopP
	}
	if responseEntity.TopK != nil {
		apiResponse.TopK = responseEntity.TopK
	}
	if responseEntity.RepetitionPenalty != nil {
		apiResponse.RepetitionPenalty = responseEntity.RepetitionPenalty
	}
	if responseEntity.Seed != nil {
		apiResponse.Seed = responseEntity.Seed
	}
	if responseEntity.PresencePenalty != nil {
		apiResponse.PresencePenalty = responseEntity.PresencePenalty
	}
	if responseEntity.FrequencyPenalty != nil {
		apiResponse.FrequencyPenalty = responseEntity.FrequencyPenalty
	}
	if responseEntity.Stream != nil {
		apiResponse.Stream = responseEntity.Stream
	}
	if responseEntity.Background != nil {
		apiResponse.Background = responseEntity.Background
	}
	if responseEntity.Timeout != nil {
		apiResponse.Timeout = responseEntity.Timeout
	}
	if responseEntity.User != nil {
		apiResponse.User = responseEntity.User
	}
	// Parse usage and error from JSON strings if available
	if responseEntity.Usage != nil {
		var usage responsetypes.DetailedUsage
		if err := json.Unmarshal([]byte(*responseEntity.Usage), &usage); err == nil {
			apiResponse.Usage = &usage
		}
	}
	if responseEntity.Error != nil {
		var errorResp responsetypes.ResponseError
		if err := json.Unmarshal([]byte(*responseEntity.Error), &errorResp); err == nil {
			apiResponse.Error = &errorResp
		}
	}

	// Set timestamps
	if responseEntity.CompletedAt != nil {
		completedAt := responseEntity.CompletedAt.Unix()
		apiResponse.CompletedAt = &completedAt
	}
	if responseEntity.CancelledAt != nil {
		cancelledAt := responseEntity.CancelledAt.Unix()
		apiResponse.CancelledAt = &cancelledAt
	}
	if responseEntity.FailedAt != nil {
		failedAt := responseEntity.FailedAt.Unix()
		apiResponse.FailedAt = &failedAt
	}

	// Set conversation info if available
	if responseEntity.ConversationID != nil {
		// Get conversation to retrieve its public ID
		conv, err := h.conversationService.GetConversationByID(reqCtx, *responseEntity.ConversationID)
		if err == nil && conv != nil {
			apiResponse.Conversation = &responsetypes.ConversationInfo{
				ID: conv.PublicID,
			}
		}
	}

	return apiResponse
}

// convertConversationItemsToMessages converts conversation items to OpenAI chat completion messages
func (h *ResponseHandler) convertConversationItemsToMessages(reqCtx *gin.Context, conv *conversation.Conversation) ([]openai.ChatCompletionMessage, error) {
	// Load conversation with items
	convWithItems, err := h.conversationService.GetConversationByPublicIDAndUserID(reqCtx, conv.PublicID, conv.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation with items: %w", err)
	}

	// Convert items to messages
	messages := make([]openai.ChatCompletionMessage, 0, len(convWithItems.Items))
	for _, item := range convWithItems.Items {
		// Skip items that don't have a role or content
		if item.Role == nil || len(item.Content) == 0 {
			continue
		}

		// Convert conversation role to OpenAI role
		var openaiRole string
		switch *item.Role {
		case conversation.ItemRoleSystem:
			openaiRole = openai.ChatMessageRoleSystem
		case conversation.ItemRoleUser:
			openaiRole = openai.ChatMessageRoleUser
		case conversation.ItemRoleAssistant:
			openaiRole = openai.ChatMessageRoleAssistant
		default:
			openaiRole = openai.ChatMessageRoleUser
		}

		// Extract text content from the item
		var content string
		for _, contentPart := range item.Content {
			if contentPart.Type == "text" && contentPart.Text != nil {
				content += contentPart.Text.Value
			}
		}

		// Only add messages with content
		if content != "" {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openaiRole,
				Content: content,
			})
		}
	}

	return messages, nil
}
