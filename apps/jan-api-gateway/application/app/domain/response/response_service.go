package response

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

// ResponseService handles business logic for responses
type ResponseService struct {
	responseRepo ResponseRepository
	itemRepo     conversation.ItemRepository
}

// NewResponseService creates a new response service
func NewResponseService(responseRepo ResponseRepository, itemRepo conversation.ItemRepository) *ResponseService {
	return &ResponseService{
		responseRepo: responseRepo,
		itemRepo:     itemRepo,
	}
}

// CreateResponse creates a new response
func (s *ResponseService) CreateResponse(ctx context.Context, userID uint, conversationID *uint, model string, input interface{}, systemPrompt *string, params *ResponseParams) (*Response, error) {
	return s.CreateResponseWithPrevious(ctx, userID, conversationID, nil, model, input, systemPrompt, params)
}

// CreateResponseWithPrevious creates a new response, optionally linking to a previous response
func (s *ResponseService) CreateResponseWithPrevious(ctx context.Context, userID uint, conversationID *uint, previousResponseID *string, model string, input interface{}, systemPrompt *string, params *ResponseParams) (*Response, error) {
	// Convert input to JSON string
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Handle previous_response_id logic
	var finalConversationID *uint = conversationID
	if previousResponseID != nil {
		// Load the previous response
		previousResponse, err := s.responseRepo.FindByPublicID(ctx, *previousResponseID)
		if err != nil {
			return nil, fmt.Errorf("failed to find previous response: %w", err)
		}
		if previousResponse == nil {
			return nil, fmt.Errorf("previous response not found: %s", *previousResponseID)
		}

		// Validate that the previous response belongs to the same user
		if previousResponse.UserID != userID {
			return nil, fmt.Errorf("previous response does not belong to the current user")
		}

		// Use the previous response's conversation ID
		finalConversationID = previousResponse.ConversationID
		if finalConversationID == nil {
			return nil, fmt.Errorf("previous response does not belong to any conversation")
		}
	}

	// Generate public ID
	publicID, err := idgen.GenerateSecureID("resp", 16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response ID: %w", err)
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
				return nil, fmt.Errorf("failed to marshal stop sequences: %w", err)
			}
			stopStr := string(stopJSON)
			response.Stop = &stopStr
		}

		if params.LogitBias != nil {
			logitBiasJSON, err := json.Marshal(params.LogitBias)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal logit bias: %w", err)
			}
			logitBiasStr := string(logitBiasJSON)
			response.LogitBias = &logitBiasStr
		}

		if params.ResponseFormat != nil {
			responseFormatJSON, err := json.Marshal(params.ResponseFormat)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response format: %w", err)
			}
			responseFormatStr := string(responseFormatJSON)
			response.ResponseFormat = &responseFormatStr
		}

		if params.Tools != nil {
			toolsJSON, err := json.Marshal(params.Tools)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tools: %w", err)
			}
			toolsStr := string(toolsJSON)
			response.Tools = &toolsStr
		}

		if params.ToolChoice != nil {
			toolChoiceJSON, err := json.Marshal(params.ToolChoice)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool choice: %w", err)
			}
			toolChoiceStr := string(toolChoiceJSON)
			response.ToolChoice = &toolChoiceStr
		}

		if params.Metadata != nil {
			metadataJSON, err := json.Marshal(params.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}
			metadataStr := string(metadataJSON)
			response.Metadata = &metadataStr
		}
	}

	if err := s.responseRepo.Create(ctx, response); err != nil {
		return nil, fmt.Errorf("failed to create response: %w", err)
	}

	return response, nil
}

// UpdateResponseStatus updates the status of a response
func (s *ResponseService) UpdateResponseStatus(ctx context.Context, responseID uint, status ResponseStatus) error {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return fmt.Errorf("failed to find response: %w", err)
	}
	if response == nil {
		return fmt.Errorf("response not found")
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
		return fmt.Errorf("failed to update response status: %w", err)
	}

	return nil
}

// UpdateResponseOutput updates the output of a response
func (s *ResponseService) UpdateResponseOutput(ctx context.Context, responseID uint, output interface{}) error {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return fmt.Errorf("failed to find response: %w", err)
	}
	if response == nil {
		return fmt.Errorf("response not found")
	}

	// Convert output to JSON string
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	outputStr := string(outputJSON)
	response.Output = &outputStr
	response.UpdatedAt = time.Now()

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return fmt.Errorf("failed to update response output: %w", err)
	}

	return nil
}

// UpdateResponseUsage updates the usage statistics of a response
func (s *ResponseService) UpdateResponseUsage(ctx context.Context, responseID uint, usage interface{}) error {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return fmt.Errorf("failed to find response: %w", err)
	}
	if response == nil {
		return fmt.Errorf("response not found")
	}

	// Convert usage to JSON string
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	usageStr := string(usageJSON)
	response.Usage = &usageStr
	response.UpdatedAt = time.Now()

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return fmt.Errorf("failed to update response usage: %w", err)
	}

	return nil
}

// UpdateResponseError updates the error information of a response
func (s *ResponseService) UpdateResponseError(ctx context.Context, responseID uint, error interface{}) error {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return fmt.Errorf("failed to find response: %w", err)
	}
	if response == nil {
		return fmt.Errorf("response not found")
	}

	// Convert error to JSON string
	errorJSON, err := json.Marshal(error)
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	errorStr := string(errorJSON)
	response.Error = &errorStr
	response.Status = ResponseStatusFailed
	response.UpdatedAt = time.Now()
	now := time.Now()
	response.FailedAt = &now

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return fmt.Errorf("failed to update response error: %w", err)
	}

	return nil
}

// GetResponseByPublicID gets a response by public ID
func (s *ResponseService) GetResponseByPublicID(ctx context.Context, publicID string) (*Response, error) {
	response, err := s.responseRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}
	return response, nil
}

// GetResponsesByUserID gets responses for a specific user
func (s *ResponseService) GetResponsesByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*Response, error) {
	responses, err := s.responseRepo.FindByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get responses by user ID: %w", err)
	}
	return responses, nil
}

// GetResponsesByConversationID gets responses for a specific conversation
func (s *ResponseService) GetResponsesByConversationID(ctx context.Context, conversationID uint, pagination *query.Pagination) ([]*Response, error) {
	responses, err := s.responseRepo.FindByConversationID(ctx, conversationID, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get responses by conversation ID: %w", err)
	}
	return responses, nil
}

// DeleteResponse deletes a response
func (s *ResponseService) DeleteResponse(ctx context.Context, responseID uint) error {
	if err := s.responseRepo.DeleteByID(ctx, responseID); err != nil {
		return fmt.Errorf("failed to delete response: %w", err)
	}
	return nil
}

// CreateItemsForResponse creates items for a specific response
func (s *ResponseService) CreateItemsForResponse(ctx context.Context, responseID uint, conversationID uint, items []conversation.ItemCreationData) ([]*conversation.Item, error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return nil, fmt.Errorf("failed to find response: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("response not found")
	}

	// Validate that the response belongs to the specified conversation
	if response.ConversationID == nil || *response.ConversationID != conversationID {
		return nil, fmt.Errorf("response does not belong to the specified conversation")
	}

	var createdItems []*conversation.Item
	for _, itemData := range items {
		// Generate public ID for the item
		publicID, err := idgen.GenerateSecureID("msg", 16)
		if err != nil {
			return nil, fmt.Errorf("failed to generate item ID: %w", err)
		}

		item := &conversation.Item{
			PublicID:       publicID,
			Type:           itemData.Type,
			Role:           itemData.Role,
			Content:        itemData.Content,
			ConversationID: conversationID,
			ResponseID:     &responseID,
			CreatedAt:      time.Now().Unix(),
		}

		if err := s.itemRepo.Create(ctx, item); err != nil {
			return nil, fmt.Errorf("failed to create item: %w", err)
		}

		createdItems = append(createdItems, item)
	}

	return createdItems, nil
}

// GetItemsForResponse gets all items that belong to a specific response
func (s *ResponseService) GetItemsForResponse(ctx context.Context, responseID uint) ([]*conversation.Item, error) {
	response, err := s.responseRepo.FindByID(ctx, responseID)
	if err != nil {
		return nil, fmt.Errorf("failed to find response: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("response not found")
	}

	// Get items by conversation ID and filter by response ID
	items, err := s.itemRepo.FindByConversationID(ctx, *response.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	// Filter items that belong to this response
	var responseItems []*conversation.Item
	for _, item := range items {
		if item.ResponseID != nil && *item.ResponseID == responseID {
			responseItems = append(responseItems, item)
		}
	}

	return responseItems, nil
}

// CreateResponseFromRequest creates a response from an API request structure
func (s *ResponseService) CreateResponseFromRequest(ctx context.Context, userID uint, req *ResponseRequest) (*Response, error) {
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
