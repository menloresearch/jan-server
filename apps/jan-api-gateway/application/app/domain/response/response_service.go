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

// CreateResponse creates a new response
func (s *ResponseService) CreateResponse(ctx context.Context, userID uint, conversationID *uint, model string, input interface{}, systemPrompt *string, params *ResponseParams) (*Response, *common.Error) {
	return s.CreateResponseWithPrevious(ctx, userID, conversationID, nil, model, input, systemPrompt, params)
}

// CreateResponseWithPrevious creates a new response, optionally linking to a previous response
func (s *ResponseService) CreateResponseWithPrevious(ctx context.Context, userID uint, conversationID *uint, previousResponseID *string, model string, input interface{}, systemPrompt *string, params *ResponseParams) (*Response, *common.Error) {
	// Convert input to JSON string
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, common.NewError("a1b2c3d4-e5f6-7890-abcd-ef1234567890", "Failed to marshal input")
	}

	// Handle previous_response_id logic
	var finalConversationID *uint = conversationID
	if previousResponseID != nil {
		// Load the previous response
		previousResponse, err := s.responseRepo.FindByPublicID(ctx, *previousResponseID)
		if err != nil {
			return nil, common.NewError("b2c3d4e5-f6g7-8901-bcde-f23456789012", "Failed to find previous response")
		}
		if previousResponse == nil {
			return nil, common.NewError("c3d4e5f6-g7h8-9012-cdef-345678901234", "Previous response not found")
		}

		// Validate that the previous response belongs to the same user
		if previousResponse.UserID != userID {
			return nil, common.NewError("d4e5f6g7-h8i9-0123-defg-456789012345", "Previous response does not belong to the current user")
		}

		// Use the previous response's conversation ID
		finalConversationID = previousResponse.ConversationID
		if finalConversationID == nil {
			return nil, common.NewError("e5f6g7h8-i9j0-1234-efgh-567890123456", "Previous response does not belong to any conversation")
		}
	}

	// Generate public ID
	publicID, err := idgen.GenerateSecureID("resp", 42)
	if err != nil {
		return nil, common.NewError("f6g7h8i9-j0k1-2345-fghi-678901234567", "Failed to generate response ID")
	}

	response := &Response{
		PublicID:           publicID,
		UserID:             userID,
		ConversationID:     finalConversationID,
		PreviousResponseID: previousResponseID,
		Model:              model,
		Status:             ResponseStatusPending,
		Input:              string(inputJSON),
		SystemPrompt:       systemPrompt,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Apply parameters if provided
	if params != nil {
		response.MaxTokens = params.MaxTokens
		response.Temperature = params.Temperature
		response.TopP = params.TopP
		response.TopK = params.TopK
		response.RepetitionPenalty = params.RepetitionPenalty
		response.Seed = params.Seed
		response.PresencePenalty = params.PresencePenalty
		response.FrequencyPenalty = params.FrequencyPenalty
		response.Stream = params.Stream
		response.Background = params.Background
		response.Timeout = params.Timeout
		response.User = params.User

		// Convert complex fields to JSON strings
		if params.Stop != nil {
			stopJSON, err := json.Marshal(params.Stop)
			if err != nil {
				return nil, common.NewError("g7h8i9j0-k1l2-3456-ghij-789012345678", "Failed to marshal stop sequences")
			}
			stopStr := string(stopJSON)
			// For JSON columns, use null for empty arrays/objects
			if stopStr == "[]" || stopStr == "{}" {
				response.Stop = nil
			} else {
				response.Stop = &stopStr
			}
		}

		if params.LogitBias != nil {
			logitBiasJSON, err := json.Marshal(params.LogitBias)
			if err != nil {
				return nil, common.NewError("h8i9j0k1-l2m3-4567-hijk-890123456789", "Failed to marshal logit bias")
			}
			logitBiasStr := string(logitBiasJSON)
			// For JSON columns, use null for empty arrays/objects
			if logitBiasStr == "[]" || logitBiasStr == "{}" {
				response.LogitBias = nil
			} else {
				response.LogitBias = &logitBiasStr
			}
		}

		if params.ResponseFormat != nil {
			responseFormatJSON, err := json.Marshal(params.ResponseFormat)
			if err != nil {
				return nil, common.NewError("i9j0k1l2-m3n4-5678-ijkl-901234567890", "Failed to marshal response format")
			}
			responseFormatStr := string(responseFormatJSON)
			// For JSON columns, use null for empty arrays/objects
			if responseFormatStr == "[]" || responseFormatStr == "{}" {
				response.ResponseFormat = nil
			} else {
				response.ResponseFormat = &responseFormatStr
			}
		}

		if params.Tools != nil {
			toolsJSON, err := json.Marshal(params.Tools)
			if err != nil {
				return nil, common.NewError("j0k1l2m3-n4o5-6789-jklm-012345678901", "Failed to marshal tools")
			}
			toolsStr := string(toolsJSON)
			// For JSON columns, use null for empty arrays/objects
			if toolsStr == "[]" || toolsStr == "{}" {
				response.Tools = nil
			} else {
				response.Tools = &toolsStr
			}
		}

		if params.ToolChoice != nil {
			toolChoiceJSON, err := json.Marshal(params.ToolChoice)
			if err != nil {
				return nil, common.NewError("k1l2m3n4-o5p6-7890-klmn-123456789012", "Failed to marshal tool choice")
			}
			toolChoiceStr := string(toolChoiceJSON)
			// For JSON columns, use null for empty arrays/objects
			if toolChoiceStr == "[]" || toolChoiceStr == "{}" {
				response.ToolChoice = nil
			} else {
				response.ToolChoice = &toolChoiceStr
			}
		}

		if params.Metadata != nil {
			metadataJSON, err := json.Marshal(params.Metadata)
			if err != nil {
				return nil, common.NewError("l2m3n4o5-p6q7-8901-lmno-234567890123", "Failed to marshal metadata")
			}
			metadataStr := string(metadataJSON)
			// For JSON columns, use null for empty arrays/objects
			if metadataStr == "[]" || metadataStr == "{}" {
				response.Metadata = nil
			} else {
				response.Metadata = &metadataStr
			}
		}
	}

	if err := s.responseRepo.Create(ctx, response); err != nil {
		return nil, common.NewError("m3n4o5p6-q7r8-9012-mnop-345678901234", "Failed to create response")
	}

	return response, nil
}

// UpdateResponseStatus updates the status of a response
func (s *ResponseService) UpdateResponseStatus(ctx context.Context, responseID uint, status ResponseStatus) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError("n4o5p6q7-r8s9-0123-nopq-456789012345", "Failed to find response")
	}
	if response == nil {
		return false, common.NewError("o5p6q7r8-s9t0-1234-opqr-567890123456", "Response not found")
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
		return false, common.NewError("p6q7r8s9-t0u1-2345-pqrs-678901234567", "Failed to update response status")
	}

	return true, nil
}

// UpdateResponseOutput updates the output of a response
func (s *ResponseService) UpdateResponseOutput(ctx context.Context, responseID uint, output interface{}) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError("q7r8s9t0-u1v2-3456-qrst-789012345678", "Failed to find response")
	}
	if response == nil {
		return false, common.NewError("r8s9t0u1-v2w3-4567-rstu-890123456789", "Response not found")
	}

	// Convert output to JSON string
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return false, common.NewError("s9t0u1v2-w3x4-5678-stuv-901234567890", "Failed to marshal output")
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
		return false, common.NewError("t0u1v2w3-x4y5-6789-tuvw-012345678901", "Failed to update response output")
	}

	return true, nil
}

// UpdateResponseUsage updates the usage statistics of a response
func (s *ResponseService) UpdateResponseUsage(ctx context.Context, responseID uint, usage interface{}) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError("u1v2w3x4-y5z6-7890-uvwx-123456789012", "Failed to find response")
	}
	if response == nil {
		return false, common.NewError("v2w3x4y5-z6a7-8901-vwxy-234567890123", "Response not found")
	}

	// Convert usage to JSON string
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return false, common.NewError("w3x4y5z6-a7b8-9012-wxyz-345678901234", "Failed to marshal usage")
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
		return false, common.NewError("x4y5z6a7-b8c9-0123-xyza-456789012345", "Failed to update response usage")
	}

	return true, nil
}

// UpdateResponseError updates the error information of a response
func (s *ResponseService) UpdateResponseError(ctx context.Context, responseID uint, error interface{}) (bool, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return false, common.NewError("y5z6a7b8-c9d0-1234-yzab-567890123456", "Failed to find response")
	}
	if response == nil {
		return false, common.NewError("z6a7b8c9-d0e1-2345-zabc-678901234567", "Response not found")
	}

	// Convert error to JSON string
	errorJSON, err := json.Marshal(error)
	if err != nil {
		return false, common.NewError("a7b8c9d0-e1f2-3456-abcd-789012345678", "Failed to marshal error")
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
		return false, common.NewError("b8c9d0e1-f2g3-4567-bcde-890123456789", "Failed to update response error")
	}

	return true, nil
}

// GetResponseByPublicID gets a response by public ID
func (s *ResponseService) GetResponseByPublicID(ctx context.Context, publicID string) (*Response, *common.Error) {
	response, err := s.responseRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, common.NewError("c9d0e1f2-g3h4-5678-cdef-901234567890", "Failed to get response")
	}
	return response, nil
}

// GetResponsesByUserID gets responses for a specific user
func (s *ResponseService) GetResponsesByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*Response, *common.Error) {
	responses, err := s.responseRepo.FindByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, common.NewError("d0e1f2g3-h4i5-6789-defg-012345678901", "Failed to get responses by user ID")
	}
	return responses, nil
}

// GetResponsesByConversationID gets responses for a specific conversation
func (s *ResponseService) GetResponsesByConversationID(ctx context.Context, conversationID uint, pagination *query.Pagination) ([]*Response, *common.Error) {
	responses, err := s.responseRepo.FindByConversationID(ctx, conversationID, pagination)
	if err != nil {
		return nil, common.NewError("e1f2g3h4-i5j6-7890-efgh-123456789012", "Failed to get responses by conversation ID")
	}
	return responses, nil
}

// DeleteResponse deletes a response
func (s *ResponseService) DeleteResponse(ctx context.Context, responseID uint) (bool, *common.Error) {
	if err := s.responseRepo.DeleteByID(ctx, responseID); err != nil {
		return false, common.NewError("f2g3h4i5-j6k7-8901-fghi-234567890123", "Failed to delete response")
	}
	return true, nil
}

// CreateItemsForResponse creates items for a specific response
func (s *ResponseService) CreateItemsForResponse(ctx context.Context, responseID uint, conversationID uint, items []*conversation.Item) ([]*conversation.Item, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return nil, common.NewError("g3h4i5j6-k7l8-9012-ghij-345678901234", "Failed to find response")
	}
	if response == nil {
		return nil, common.NewError("h4i5j6k7-l8m9-0123-hijk-456789012345", "Response not found")
	}

	// Validate that the response belongs to the specified conversation
	if response.ConversationID == nil || *response.ConversationID != conversationID {
		return nil, common.NewError("i5j6k7l8-m9n0-1234-ijkl-567890123456", "Response does not belong to the specified conversation")
	}

	var createdItems []*conversation.Item
	for _, itemData := range items {
		// Generate public ID for the item
		publicID, err := idgen.GenerateSecureID("msg", 42)
		if err != nil {
			return nil, common.NewError("j6k7l8m9-n0o1-2345-jklm-678901234567", "Failed to generate item ID")
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
			return nil, common.NewError("k7l8m9n0-o1p2-3456-klmn-789012345678", "Failed to create item")
		}

		createdItems = append(createdItems, item)
	}

	return createdItems, nil
}

// GetItemsForResponse gets items that belong to a specific response, optionally filtered by role
func (s *ResponseService) GetItemsForResponse(ctx context.Context, responseID uint, itemRole *conversation.ItemRole) ([]*conversation.Item, *common.Error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return nil, common.NewError("l8m9n0o1-p2q3-4567-lmno-890123456789", "Failed to find response")
	}
	if response == nil {
		return nil, common.NewError("m9n0o1p2-q3r4-5678-mnop-901234567890", "Response not found")
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
		return nil, common.NewError("n0o1p2q3-r4s5-6789-nopq-012345678901", "Failed to get items")
	}

	return items, nil
}

// CreateResponseFromRequest creates a response from an API request structure
func (s *ResponseService) CreateResponseFromRequest(ctx context.Context, userID uint, req *ResponseRequest) (*Response, *common.Error) {
	// Convert the request to ResponseParams
	params := &ResponseParams{
		Stream: req.Stream,
	}

	// Create the response with previous_response_id handling
	return s.CreateResponseWithPrevious(ctx, userID, nil, req.PreviousResponseID, req.Model, req.Input, nil, params)
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
			return nil, common.NewError("o1p2q3r4-s5t6-7890-opqr-123456789012", "Previous response not found")
		}

		// Validate that the previous response belongs to the same user
		if previousResponse.UserID != userID {
			return nil, common.NewError("p2q3r4s5-t6u7-8901-pqrs-234567890123", "Previous response does not belong to the current user")
		}

		// Load the conversation from the previous response
		if previousResponse.ConversationID == nil {
			return nil, common.NewError("q3r4s5t6-u7v8-9012-qrst-345678901234", "Previous response does not belong to any conversation")
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
			return false, common.NewError("u7v8w9x0-y1z2-3456-uvwx-789012345678", "Failed to generate item ID")
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
