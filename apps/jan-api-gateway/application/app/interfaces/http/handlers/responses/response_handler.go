package responses

import (
	"fmt"
	"net/http"

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

// ResponseHandler handles the business logic for response API endpoints
type ResponseHandler struct {
	UserService         *user.UserService
	apikeyService       *apikey.ApiKeyService
	conversationService *conversation.ConversationService
	streamHandler       *StreamHandler
	nonStreamHandler    *NonStreamHandler
}

// NewResponseHandler creates a new ResponseHandler instance
func NewResponseHandler(
	userService *user.UserService,
	apikeyService *apikey.ApiKeyService,
	conversationService *conversation.ConversationService,
) *ResponseHandler {
	responseHandler := &ResponseHandler{
		UserService:         userService,
		apikeyService:       apikeyService,
		conversationService: conversationService,
	}

	// Initialize specialized handlers
	responseHandler.streamHandler = NewStreamHandler(responseHandler)
	responseHandler.nonStreamHandler = NewNonStreamHandler(responseHandler)

	return responseHandler
}

// CreateResponse handles the business logic for creating a response
func (h *ResponseHandler) CreateResponse(reqCtx *gin.Context) {
	// Get user from middleware context
	_, ok := h.UserService.GetUserFromContext(reqCtx)
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
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "019929d1-1e85-72c1-a1cf-e151403692dc", err.Error())
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

	// Append input messages to conversation
	if err := h.appendMessagesToConversation(reqCtx, conversation, chatCompletionRequest.Messages); err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "b2c3d4e5-f6g7-8901-bcde-f23456789012", err.Error())
		return
	}

	// Delegate to specialized handlers based on streaming preference
	if request.Stream != nil && *request.Stream {
		// Handle streaming response
		h.streamHandler.CreateStreamResponse(reqCtx, &request, key, conversation)
	} else {
		// Handle non-streaming response
		h.nonStreamHandler.CreateNonStreamResponse(reqCtx, &request, key, conversation)
	}
}

// handleConversation handles conversation creation or loading based on the request
func (h *ResponseHandler) handleConversation(reqCtx *gin.Context, request *requesttypes.CreateResponseRequest) (*conversation.Conversation, error) {
	// Get user from middleware context
	userEntity, ok := h.UserService.GetUserFromContext(reqCtx)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}

	// Check if conversation is specified and not 'client-created-root'
	if request.Conversation != nil && *request.Conversation != "" && *request.Conversation != "client-created-root" {
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
func (h *ResponseHandler) appendMessagesToConversation(reqCtx *gin.Context, conv *conversation.Conversation, messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	// Convert messages to conversation items
	itemsToCreate := make([]conversation.ItemCreationData, len(messages))
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

		itemsToCreate[i] = conversation.ItemCreationData{
			Type:    conversation.ItemType("message"),
			Role:    role,
			Content: content,
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
	responseID := reqCtx.Param("response_id")

	// Validate user and response ID
	userEntity, err := h.validateUserAndResponseID(reqCtx, responseID)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "b2c3d4e5-f6g7-8901-bcde-f23456789012", err.Error())
		return
	}

	_ = userEntity // TODO: Use user info if needed
	// TODO: Get response logic here

	// Create mock response data
	mockResponse := responsetypes.Response{
		ID:      responseID,
		Object:  "response",
		Created: 1234567890,
		Model:   "gpt-4",
		Status:  responsetypes.ResponseStatusCompleted,
		Input: requesttypes.CreateResponseInput{
			Type: requesttypes.InputTypeText,
			Text: ptr.ToString("Hello, world!"),
		},
		Output: &responsetypes.ResponseOutput{
			Type: responsetypes.OutputTypeText,
			Text: &responsetypes.TextOutput{
				Value: "Hello! How can I help you today?",
			},
		},
		Conversation: &responsetypes.ConversationInfo{
			ID: "mock-conversation-id",
		},
	}

	h.sendSuccessResponse(reqCtx, mockResponse)
}

// DeleteResponse handles the business logic for deleting a response
func (h *ResponseHandler) DeleteResponse(reqCtx *gin.Context) {
	responseID := reqCtx.Param("response_id")

	// Validate user and response ID
	userEntity, err := h.validateUserAndResponseID(reqCtx, responseID)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "d4e5f6g7-h8i9-0123-defg-456789012345", err.Error())
		return
	}

	_ = userEntity // TODO: Use user info if needed
	// TODO: Delete response logic here

	// Create mock deleted response data
	mockResponse := responsetypes.Response{
		ID:      responseID,
		Object:  "response",
		Created: 1234567890,
		Model:   "gpt-4",
		Status:  responsetypes.ResponseStatusCancelled,
		Input: requesttypes.CreateResponseInput{
			Type: requesttypes.InputTypeText,
			Text: ptr.ToString("Hello, world!"),
		},
		CancelledAt: ptr.ToInt64(1234567890),
		Conversation: &responsetypes.ConversationInfo{
			ID: "mock-conversation-id",
		},
	}

	h.sendSuccessResponse(reqCtx, mockResponse)
}

// CancelResponse handles the business logic for cancelling a response
func (h *ResponseHandler) CancelResponse(reqCtx *gin.Context) {
	responseID := reqCtx.Param("response_id")

	// Validate user and response ID
	userEntity, err := h.validateUserAndResponseID(reqCtx, responseID)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "f6g7h8i9-j0k1-2345-fghi-678901234567", err.Error())
		return
	}

	_ = userEntity // TODO: Use user info if needed
	// TODO: Cancel response logic here

	// Create mock cancelled response data
	mockResponse := responsetypes.Response{
		ID:      responseID,
		Object:  "response",
		Created: 1234567890,
		Model:   "gpt-4",
		Status:  responsetypes.ResponseStatusCancelled,
		Input: requesttypes.CreateResponseInput{
			Type: requesttypes.InputTypeText,
			Text: ptr.ToString("Hello, world!"),
		},
		CancelledAt: ptr.ToInt64(1234567890),
		Conversation: &responsetypes.ConversationInfo{
			ID: "mock-conversation-id",
		},
	}

	h.sendSuccessResponse(reqCtx, mockResponse)
}

// ListInputItems handles the business logic for listing input items
func (h *ResponseHandler) ListInputItems(reqCtx *gin.Context) {
	responseID := reqCtx.Param("response_id")

	// Validate user and response ID
	userEntity, err := h.validateUserAndResponseID(reqCtx, responseID)
	if err != nil {
		h.sendErrorResponse(reqCtx, http.StatusBadRequest, "h8i9j0k1-l2m3-4567-hijk-890123456789", err.Error())
		return
	}

	_ = userEntity // TODO: Use user info if needed
	// TODO: List input items logic here

	// Create mock input items data
	status := responsetypes.ResponseCodeOk
	objectType := responsetypes.ObjectTypeList
	hasMore := false
	reqCtx.JSON(http.StatusOK, responsetypes.OpenAIListResponse[responsetypes.InputItem]{
		JanStatus: &status,
		Object:    &objectType,
		HasMore:   &hasMore,
		T: []responsetypes.InputItem{
			{
				ID:      "input_1234567890",
				Object:  "input_item",
				Created: 1234567890,
				Type:    requesttypes.InputTypeText,
				Text:    ptr.ToString("Hello, world!"),
			},
		},
	})
}

// validateUserAndResponseID validates user context and response ID
func (h *ResponseHandler) validateUserAndResponseID(reqCtx *gin.Context, responseID string) (*user.User, error) {
	// Get user from middleware context
	userEntity, ok := h.UserService.GetUserFromContext(reqCtx)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}

	// Validate response ID
	if validationError := ValidateResponseID(responseID); validationError != nil {
		return nil, fmt.Errorf("validation error: %s", validationError.Message)
	}

	return userEntity, nil
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
			OwnerID:    &user.ID,
			ApikeyType: ptr.ToString(string(apikey.ApikeyTypeAdmin)),
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
				OwnerID:        &user.ID,
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
