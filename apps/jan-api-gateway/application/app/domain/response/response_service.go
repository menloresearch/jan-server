package response

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"
	requesttypes "menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	responsetypes "menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// ResponseService handles business logic for responses
type ResponseService struct {
	responseRepo        ResponseRepository
	itemRepo            conversation.ItemRepository
	conversationService *conversation.ConversationService
}

// ResponseContextKey represents context keys for responses
type ResponseContextKey string

const (
	ResponseContextKeyPublicID ResponseContextKey = "response_id"
	ResponseContextEntity      ResponseContextKey = "ResponseContextEntity"

	// ClientCreatedRootConversationID is the special conversation ID that indicates a new conversation should be created
	ClientCreatedRootConversationID = "client-created-root"
)

// NewResponseService creates a new response service
func NewResponseService(responseRepo ResponseRepository, itemRepo conversation.ItemRepository, conversationService *conversation.ConversationService) *ResponseService {
	return &ResponseService{
		responseRepo:        responseRepo,
		itemRepo:            itemRepo,
		conversationService: conversationService,
	}
}

// CreateResponse creates a new response using a Response domain object
func (s *ResponseService) CreateResponse(ctx context.Context, response *Response) (*Response, *common.Error) {
	return s.CreateResponseWithPrevious(ctx, response, nil)
}

// CreateResponseWithPrevious creates a new response, optionally linking to a previous response
func (s *ResponseService) CreateResponseWithPrevious(ctx context.Context, response *Response, previousResponseID *string) (*Response, *common.Error) {
	// Handle previous_response_id logic
	if previousResponseID != nil {
		// Load the previous response
		previousResponse, err := s.responseRepo.FindByPublicID(ctx, *previousResponseID)
		if err != nil {
			return nil, common.NewError(err, "b2c3d4e5-f6g7-8901-bcde-f23456789012")
		}
		if previousResponse == nil {
			return nil, common.NewErrorWithMessage("Previous response not found", "c3d4e5f6-g7h8-9012-cdef-345678901234")
		}

		// Validate that the previous response belongs to the same user
		if previousResponse.UserID != response.UserID {
			return nil, common.NewErrorWithMessage("Previous response does not belong to the current user", "d4e5f6g7-h8i9-0123-defg-456789012345")
		}

		// Use the previous response's conversation ID
		response.ConversationID = previousResponse.ConversationID
		if response.ConversationID == nil {
			return nil, common.NewErrorWithMessage("Previous response does not belong to any conversation", "e5f6g7h8-i9j0-1234-efgh-567890123456")
		}
	}

	// Set the previous response ID
	response.PreviousResponseID = previousResponseID

	// Generate public ID if not already set
	if response.PublicID == "" {
		publicID, err := idgen.GenerateSecureID("resp", 42)
		if err != nil {
			return nil, common.NewError(err, "f6g7h8i9-j0k1-2345-fghi-678901234567")
		}
		response.PublicID = publicID
	}

	// Set default values
	if response.Status == "" {
		response.Status = ResponseStatusPending
	}
	if response.CreatedAt.IsZero() {
		response.CreatedAt = time.Now()
	}
	response.UpdatedAt = time.Now()

	// Validate required fields
	if response.UserID == 0 {
		return nil, common.NewErrorWithMessage("UserID is required", "m3n4o5p6-q7r8-9012-mnop-345678901234")
	}
	if response.Model == "" {
		return nil, common.NewErrorWithMessage("Model is required", "n4o5p6q7-r8s9-0123-nopq-456789012345")
	}
	if response.Input == "" {
		return nil, common.NewErrorWithMessage("Input is required", "o5p6q7r8-s9t0-1234-opqr-567890123456")
	}

	if err := s.responseRepo.Create(ctx, response); err != nil {
		return nil, common.NewError(err, "m3n4o5p6-q7r8-9012-mnop-345678901234")
	}

	return response, nil
}

// UpdateResponseStatus updates the status of a response
func (s *ResponseService) UpdateResponseStatus(ctx context.Context, responseID uint, status ResponseStatus) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError(err, "n4o5p6q7-r8s9-0123-nopq-456789012345")
	}
	if response == nil {
		return false, common.NewErrorWithMessage("Response not found", "o5p6q7r8-s9t0-1234-opqr-567890123456")
	}

	response.Status = status
	response.UpdatedAt = time.Now()

	// Set completion timestamps based on status
	now := time.Now()
	switch status {
	case ResponseStatusCompleted:
		response.CompletedAt = &now
	case ResponseStatusCancelled:
		response.CancelledAt = &now
	case ResponseStatusFailed:
		response.FailedAt = &now
	}

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return false, common.NewError(err, "p6q7r8s9-t0u1-2345-pqrs-678901234567")
	}

	return true, nil
}

// UpdateResponseOutput updates the output of a response
func (s *ResponseService) UpdateResponseOutput(ctx context.Context, responseID uint, output interface{}) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError(err, "q7r8s9t0-u1v2-3456-qrst-789012345678")
	}
	if response == nil {
		return false, common.NewErrorWithMessage("Response not found", "r8s9t0u1-v2w3-4567-rstu-890123456789")
	}

	// Convert output to JSON string
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return false, common.NewError(err, "s9t0u1v2-w3x4-5678-stuv-901234567890")
	}

	outputStr := string(outputJSON)
	// For JSON columns, use null for empty arrays/objects
	if outputStr == "[]" || outputStr == "{}" {
		response.Output = nil
	} else {
		response.Output = &outputStr
	}
	response.UpdatedAt = time.Now()

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return false, common.NewError(err, "t0u1v2w3-x4y5-6789-tuvw-012345678901")
	}

	return true, nil
}

// UpdateResponseUsage updates the usage statistics of a response
func (s *ResponseService) UpdateResponseUsage(ctx context.Context, responseID uint, usage interface{}) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError(err, "u1v2w3x4-y5z6-7890-uvwx-123456789012")
	}
	if response == nil {
		return false, common.NewErrorWithMessage("Response not found", "v2w3x4y5-z6a7-8901-vwxy-234567890123")
	}

	// Convert usage to JSON string
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return false, common.NewError(err, "w3x4y5z6-a7b8-9012-wxyz-345678901234")
	}

	usageStr := string(usageJSON)
	// For JSON columns, use null for empty arrays/objects
	if usageStr == "[]" || usageStr == "{}" {
		response.Usage = nil
	} else {
		response.Usage = &usageStr
	}
	response.UpdatedAt = time.Now()

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return false, common.NewError(err, "x4y5z6a7-b8c9-0123-xyza-456789012345")
	}

	return true, nil
}

// UpdateResponseError updates the error information of a response
func (s *ResponseService) UpdateResponseError(ctx context.Context, responseID uint, error interface{}) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError(err, "y5z6a7b8-c9d0-1234-yzab-567890123456")
	}
	if response == nil {
		return false, common.NewErrorWithMessage("Response not found", "z6a7b8c9-d0e1-2345-zabc-678901234567")
	}

	// Convert error to JSON string
	errorJSON, err := json.Marshal(error)
	if err != nil {
		return false, common.NewError(err, "a7b8c9d0-e1f2-3456-abcd-789012345678")
	}

	errorStr := string(errorJSON)
	// For JSON columns, use null for empty arrays/objects
	if errorStr == "[]" || errorStr == "{}" {
		response.Error = nil
	} else {
		response.Error = &errorStr
	}
	response.Status = ResponseStatusFailed
	response.UpdatedAt = time.Now()
	now := time.Now()
	response.FailedAt = &now

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return false, common.NewError(err, "b8c9d0e1-f2g3-4567-bcde-890123456789")
	}

	return true, nil
}

// GetResponseByPublicID gets a response by public ID
func (s *ResponseService) GetResponseByPublicID(ctx context.Context, publicID string) (*Response, *common.Error) {
	response, err := s.responseRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, common.NewError(err, "c9d0e1f2-g3h4-5678-cdef-901234567890")
	}
	return response, nil
}

// GetResponsesByUserID gets responses for a specific user
func (s *ResponseService) GetResponsesByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*Response, *common.Error) {
	responses, err := s.responseRepo.FindByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, common.NewError(err, "d0e1f2g3-h4i5-6789-defg-012345678901")
	}
	return responses, nil
}

// GetResponsesByConversationID gets responses for a specific conversation
func (s *ResponseService) GetResponsesByConversationID(ctx context.Context, conversationID uint, pagination *query.Pagination) ([]*Response, *common.Error) {
	responses, err := s.responseRepo.FindByConversationID(ctx, conversationID, pagination)
	if err != nil {
		return nil, common.NewError(err, "e1f2g3h4-i5j6-7890-efgh-123456789012")
	}
	return responses, nil
}

// DeleteResponse deletes a response
func (s *ResponseService) DeleteResponse(ctx context.Context, responseID uint) (bool, *common.Error) {
	if err := s.responseRepo.DeleteByID(ctx, responseID); err != nil {
		return false, common.NewError(err, "f2g3h4i5-j6k7-8901-fghi-234567890123")
	}
	return true, nil
}

// CreateItemsForResponse creates items for a specific response
func (s *ResponseService) CreateItemsForResponse(ctx context.Context, responseID uint, conversationID uint, items []*conversation.Item) ([]*conversation.Item, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return nil, common.NewError(err, "g3h4i5j6-k7l8-9012-ghij-345678901234")
	}
	if response == nil {
		return nil, common.NewErrorWithMessage("Response not found", "h4i5j6k7-l8m9-0123-hijk-456789012345")
	}

	// Validate that the response belongs to the specified conversation
	if response.ConversationID == nil || *response.ConversationID != conversationID {
		return nil, common.NewErrorWithMessage("Response does not belong to the specified conversation", "i5j6k7l8-m9n0-1234-ijkl-567890123456")
	}

	var createdItems []*conversation.Item
	for _, itemData := range items {
		// Generate public ID for the item
		publicID, err := idgen.GenerateSecureID("msg", 42)
		if err != nil {
			return nil, common.NewError(err, "j6k7l8m9-n0o1-2345-jklm-678901234567")
		}

		item := &conversation.Item{
			PublicID:       publicID,
			Type:           itemData.Type,
			Role:           itemData.Role,
			Content:        itemData.Content,
			ConversationID: conversationID,
			ResponseID:     &responseID,
			CreatedAt:      time.Now(),
		}

		if err := s.itemRepo.Create(ctx, item); err != nil {
			return nil, common.NewError(err, "k7l8m9n0-o1p2-3456-klmn-789012345678")
		}

		createdItems = append(createdItems, item)
	}

	return createdItems, nil
}

// GetItemsForResponse gets items that belong to a specific response, optionally filtered by role
func (s *ResponseService) GetItemsForResponse(ctx context.Context, responseID uint, itemRole *conversation.ItemRole) ([]*conversation.Item, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return nil, common.NewError(err, "l8m9n0o1-p2q3-4567-lmno-890123456789")
	}
	if response == nil {
		return nil, common.NewErrorWithMessage("Response not found", "m9n0o1p2-q3r4-5678-mnop-901234567890")
	}

	// Create filter for database query
	filter := conversation.ItemFilter{
		ConversationID: response.ConversationID,
		ResponseID:     &responseID,
		Role:           itemRole,
	}

	// Get items using database filter (more efficient than in-memory filtering)
	items, err := s.itemRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, common.NewError(err, "n0o1p2q3-r4s5-6789-nopq-012345678901")
	}

	return items, nil
}

// CreateResponseFromRequest creates a response from an API request structure
func (s *ResponseService) CreateResponseFromRequest(ctx context.Context, userID uint, req *ResponseRequest) (*Response, *common.Error) {
	// Convert input to JSON string
	inputJSON, jsonErr := json.Marshal(req.Input)
	if jsonErr != nil {
		return nil, common.NewError(jsonErr, "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}

	// Build Response object from request
	response := &Response{
		UserID:             userID,
		ConversationID:     nil, // Will be set by CreateResponseWithPrevious if previousResponseID is provided
		PreviousResponseID: req.PreviousResponseID,
		Model:              req.Model,
		Input:              string(inputJSON),
		SystemPrompt:       nil,
		Status:             ResponseStatusPending,
		Stream:             req.Stream,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Create the response with previous_response_id handling
	return s.CreateResponseWithPrevious(ctx, response, req.PreviousResponseID)
}

// ResponseRequest represents the API request structure for creating a response
type ResponseRequest struct {
	Model              string      `json:"model"`
	PreviousResponseID *string     `json:"previous_response_id,omitempty"`
	Input              interface{} `json:"input"`
	Stream             *bool       `json:"stream,omitempty"`
}

// ResponseParams represents parameters for creating a response
type ResponseParams struct {
	MaxTokens         *int
	Temperature       *float64
	TopP              *float64
	TopK              *int
	RepetitionPenalty *float64
	Seed              *int
	Stop              []string
	PresencePenalty   *float64
	FrequencyPenalty  *float64
	LogitBias         map[string]float64
	ResponseFormat    interface{}
	Tools             interface{}
	ToolChoice        interface{}
	Metadata          map[string]interface{}
	Stream            *bool
	Background        *bool
	Timeout           *int
	User              *string
}

// GetResponseMiddleWare creates middleware to load response by public ID and set it in context
func (s *ResponseService) GetResponseMiddleWare() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		publicID := reqCtx.Param(string(ResponseContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responsetypes.ErrorResponse{
				Code:  "r8s9t0u1-v2w3-4567-rstu-890123456789",
				Error: "missing response public ID",
			})
			return
		}
		user, ok := auth.GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responsetypes.ErrorResponse{
				Code: "s9t0u1v2-w3x4-5678-stuv-901234567890",
			})
			return
		}
		entities, err := s.responseRepo.FindByFilter(ctx, ResponseFilter{
			PublicID: &publicID,
			UserID:   &user.ID,
		}, nil)

		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responsetypes.ErrorResponse{
				Code:  "t0u1v2w3-x4y5-6789-tuvw-012345678901",
				Error: err.Error(),
			})
			return
		}

		if len(entities) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responsetypes.ErrorResponse{
				Code: "u1v2w3x4-y5z6-7890-uvwx-123456789012",
			})
			return
		}

		SetResponseFromContext(reqCtx, entities[0])
		reqCtx.Next()
	}
}

// SetResponseFromContext sets a response in the gin context
func SetResponseFromContext(reqCtx *gin.Context, resp *Response) {
	reqCtx.Set(string(ResponseContextEntity), resp)
}

// GetResponseFromContext gets a response from the gin context
func GetResponseFromContext(reqCtx *gin.Context) (*Response, bool) {
	resp, ok := reqCtx.Get(string(ResponseContextEntity))
	if !ok {
		return nil, false
	}
	response, ok := resp.(*Response)
	return response, ok
}

// ProcessResponseRequest processes a response request and returns the appropriate handler
func (s *ResponseService) ProcessResponseRequest(ctx context.Context, userID uint, req *ResponseRequest) (*Response, *common.Error) {
	// Create response from request
	responseEntity, err := s.CreateResponseFromRequest(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	return responseEntity, nil
}

// ConvertDomainResponseToAPIResponse converts a domain response to API response format
func (s *ResponseService) ConvertDomainResponseToAPIResponse(responseEntity *Response) responsetypes.Response {
	apiResponse := responsetypes.Response{
		ID:      responseEntity.PublicID,
		Object:  "response",
		Created: responseEntity.CreatedAt.Unix(),
		Model:   responseEntity.Model,
		Status:  responsetypes.ResponseStatus(responseEntity.Status),
		Input:   responseEntity.Input,
	}

	// Add conversation if exists
	if responseEntity.ConversationID != nil {
		apiResponse.Conversation = &responsetypes.ConversationInfo{
			ID: fmt.Sprintf("conv_%d", *responseEntity.ConversationID),
		}
	}

	// Add timestamps
	if responseEntity.CompletedAt != nil {
		apiResponse.CompletedAt = ptr.ToInt64(responseEntity.CompletedAt.Unix())
	}
	if responseEntity.CancelledAt != nil {
		apiResponse.CancelledAt = ptr.ToInt64(responseEntity.CancelledAt.Unix())
	}
	if responseEntity.FailedAt != nil {
		apiResponse.FailedAt = ptr.ToInt64(responseEntity.FailedAt.Unix())
	}

	// Parse output if exists
	if responseEntity.Output != nil {
		var output interface{}
		if err := json.Unmarshal([]byte(*responseEntity.Output), &output); err == nil {
			apiResponse.Output = output
		}
	}

	// Parse usage if exists
	if responseEntity.Usage != nil {
		var usage responsetypes.DetailedUsage
		if err := json.Unmarshal([]byte(*responseEntity.Usage), &usage); err == nil {
			apiResponse.Usage = &usage
		}
	}

	// Parse error if exists
	if responseEntity.Error != nil {
		var errorData responsetypes.ResponseError
		if err := json.Unmarshal([]byte(*responseEntity.Error), &errorData); err == nil {
			apiResponse.Error = &errorData
		}
	}

	return apiResponse
}

// ConvertConversationItemToInputItem converts a conversation item to input item format
func (s *ResponseService) ConvertConversationItemToInputItem(item *conversation.Item) responsetypes.InputItem {
	inputItem := responsetypes.InputItem{
		ID:      item.PublicID,
		Object:  "input_item",
		Created: item.CreatedAt.Unix(),
		Type:    requesttypes.InputType(item.Type),
	}

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

	return inputItem
}

// HandleConversation handles conversation creation and management for responses
func (s *ResponseService) HandleConversation(ctx context.Context, userID uint, request *requesttypes.CreateResponseRequest) (*conversation.Conversation, *common.Error) {
	// If store is explicitly set to false, don't create or use any conversation
	if request.Store != nil && !*request.Store {
		return nil, nil
	}

	// If previous_response_id is provided, load the conversation from the previous response
	if request.PreviousResponseID != nil && *request.PreviousResponseID != "" {
		// Load the previous response
		previousResponse, err := s.GetResponseByPublicID(ctx, *request.PreviousResponseID)
		if err != nil {
			return nil, err
		}
		if previousResponse == nil {
			return nil, common.NewErrorWithMessage("Previous response not found", "o1p2q3r4-s5t6-7890-opqr-123456789012")
		}

		// Validate that the previous response belongs to the same user
		if previousResponse.UserID != userID {
			return nil, common.NewErrorWithMessage("Previous response does not belong to the current user", "p2q3r4s5-t6u7-8901-pqrs-234567890123")
		}

		// Load the conversation from the previous response
		if previousResponse.ConversationID == nil {
			return nil, common.NewErrorWithMessage("Previous response does not belong to any conversation", "q3r4s5t6-u7v8-9012-qrst-345678901234")
		}

		conv, err := s.conversationService.GetConversationByID(ctx, *previousResponse.ConversationID)
		if err != nil {
			return nil, err
		}
		return conv, nil
	}

	// Check if conversation is specified and not 'client-created-root'
	if request.Conversation != nil && *request.Conversation != "" && *request.Conversation != ClientCreatedRootConversationID {
		// Load existing conversation
		conv, err := s.conversationService.GetConversationByPublicIDAndUserID(ctx, *request.Conversation, userID)
		if err != nil {
			return nil, err
		}
		return conv, nil
	}

	// Create new conversation
	conv, err := s.conversationService.CreateConversation(ctx, userID, nil, true, nil)
	if err != nil {
		return nil, err
	}

	return conv, nil
}

// AppendMessagesToConversation appends messages to a conversation
func (s *ResponseService) AppendMessagesToConversation(ctx context.Context, conv *conversation.Conversation, messages []openai.ChatCompletionMessage, responseID *uint) (bool, *common.Error) {
	// Convert OpenAI messages to conversation items
	items := make([]*conversation.Item, 0, len(messages))
	for _, msg := range messages {
		// Generate public ID for the item
		publicID, err := idgen.GenerateSecureID("msg", 42)
		if err != nil {
			return false, common.NewErrorWithMessage("Failed to generate item ID", "u7v8w9x0-y1z2-3456-uvwx-789012345678")
		}

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

		item := &conversation.Item{
			PublicID:       publicID,
			Type:           conversation.ItemType("message"),
			Role:           &role,
			Content:        content,
			ConversationID: conv.ID,
			ResponseID:     responseID,
			CreatedAt:      time.Now(),
		}

		items = append(items, item)
	}

	// Add items to conversation
	if len(items) > 0 {
		_, err := s.conversationService.AddMultipleItems(ctx, conv, conv.UserID, items)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// ConvertToChatCompletionRequest converts a response request to OpenAI chat completion request
func (s *ResponseService) ConvertToChatCompletionRequest(req *requesttypes.CreateResponseRequest) *openai.ChatCompletionRequest {
	chatReq := &openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: make([]openai.ChatCompletionMessage, 0),
	}

	// Add system message if provided
	if req.SystemPrompt != nil && *req.SystemPrompt != "" {
		chatReq.Messages = append(chatReq.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: *req.SystemPrompt,
		})
	}

	// Add user input as message
	if req.Input != nil {
		// Try to parse input as JSON array of messages first
		var messages []openai.ChatCompletionMessage
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", req.Input)), &messages); err == nil {
			// Input is an array of messages
			chatReq.Messages = append(chatReq.Messages, messages...)
		} else {
			// Input is a single string message
			chatReq.Messages = append(chatReq.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("%v", req.Input),
			})
		}
	}

	// Set optional parameters
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
	if req.User != nil {
		chatReq.User = *req.User
	}

	return chatReq
}

// ConvertConversationItemsToMessages converts conversation items to OpenAI chat completion messages
func (s *ResponseService) ConvertConversationItemsToMessages(ctx context.Context, conv *conversation.Conversation) ([]openai.ChatCompletionMessage, *common.Error) {
	// Load conversation with items
	convWithItems, err := s.conversationService.GetConversationByPublicIDAndUserID(ctx, conv.PublicID, conv.UserID)
	if err != nil {
		return nil, err
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

		// Only add message if it has content
		if content != "" {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openaiRole,
				Content: content,
			})
		}
	}

	return messages, nil
}
