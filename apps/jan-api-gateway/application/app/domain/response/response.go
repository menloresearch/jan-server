package response

import (
	"context"
	"encoding/json"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"
)

// Response represents a model response stored in the database
type Response struct {
	ID                 uint
	PublicID           string
	UserID             uint
	ConversationID     *uint
	PreviousResponseID *string // Public ID of the previous response
	Model              string
	Status             ResponseStatus
	Input              string  // JSON string of the input
	Output             *string // JSON string of the output
	SystemPrompt       *string
	MaxTokens          *int
	Temperature        *float64
	TopP               *float64
	TopK               *int
	RepetitionPenalty  *float64
	Seed               *int
	Stop               *string // JSON string of stop sequences
	PresencePenalty    *float64
	FrequencyPenalty   *float64
	LogitBias          *string // JSON string of logit bias
	ResponseFormat     *string // JSON string of response format
	Tools              *string // JSON string of tools
	ToolChoice         *string // JSON string of tool choice
	Metadata           *string // JSON string of metadata
	Stream             *bool
	Background         *bool
	Timeout            *int
	User               *string
	Usage              *string // JSON string of usage statistics
	Error              *string // JSON string of error details
	CompletedAt        *time.Time
	CancelledAt        *time.Time
	FailedAt           *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Items              []conversation.Item // Items that belong to this response
}

// ResponseStatus represents the status of a response
type ResponseStatus string

const (
	ResponseStatusPending   ResponseStatus = "pending"
	ResponseStatusRunning   ResponseStatus = "running"
	ResponseStatusCompleted ResponseStatus = "completed"
	ResponseStatusCancelled ResponseStatus = "cancelled"
	ResponseStatusFailed    ResponseStatus = "failed"
)

// ResponseFilter represents filters for querying responses
type ResponseFilter struct {
	PublicID       *string
	UserID         *uint
	ConversationID *uint
	Model          *string
	Status         *ResponseStatus
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
}

// ResponseRepository defines the interface for response data operations
type ResponseRepository interface {
	Create(ctx context.Context, r *Response) error
	Update(ctx context.Context, r *Response) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Response, error)
	FindByPublicID(ctx context.Context, publicID string) (*Response, error)
	FindByFilter(ctx context.Context, filter ResponseFilter, pagination *query.Pagination) ([]*Response, error)
	Count(ctx context.Context, filter ResponseFilter) (int64, error)
	FindByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*Response, error)
	FindByConversationID(ctx context.Context, conversationID uint, pagination *query.Pagination) ([]*Response, error)
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
	ResponseFormat    any
	Tools             any
	ToolChoice        any
	Metadata          map[string]any
	Stream            *bool
	Background        *bool
	Timeout           *int
	User              *string
}

// NewResponse creates a new Response object with the given parameters
func NewResponse(userID uint, conversationID *uint, model, input string, systemPrompt *string, params *ResponseParams) *Response {
	response := &Response{
		UserID:         userID,
		ConversationID: conversationID,
		Model:          model,
		Input:          input,
		SystemPrompt:   systemPrompt,
		Status:         ResponseStatusPending,
	}

	// Apply response parameters
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
			if stopJSON, err := json.Marshal(params.Stop); err == nil {
				stopStr := string(stopJSON)
				if stopStr != "[]" && stopStr != "{}" {
					response.Stop = &stopStr
				}
			}
		}

		if params.LogitBias != nil {
			if logitBiasJSON, err := json.Marshal(params.LogitBias); err == nil {
				logitBiasStr := string(logitBiasJSON)
				if logitBiasStr != "[]" && logitBiasStr != "{}" {
					response.LogitBias = &logitBiasStr
				}
			}
		}

		if params.ResponseFormat != nil {
			if responseFormatJSON, err := json.Marshal(params.ResponseFormat); err == nil {
				responseFormatStr := string(responseFormatJSON)
				if responseFormatStr != "[]" && responseFormatStr != "{}" {
					response.ResponseFormat = &responseFormatStr
				}
			}
		}

		if params.Tools != nil {
			if toolsJSON, err := json.Marshal(params.Tools); err == nil {
				toolsStr := string(toolsJSON)
				if toolsStr != "[]" && toolsStr != "{}" {
					response.Tools = &toolsStr
				}
			}
		}

		if params.ToolChoice != nil {
			if toolChoiceJSON, err := json.Marshal(params.ToolChoice); err == nil {
				toolChoiceStr := string(toolChoiceJSON)
				if toolChoiceStr != "[]" && toolChoiceStr != "{}" {
					response.ToolChoice = &toolChoiceStr
				}
			}
		}

		if params.Metadata != nil {
			if metadataJSON, err := json.Marshal(params.Metadata); err == nil {
				metadataStr := string(metadataJSON)
				if metadataStr != "[]" && metadataStr != "{}" {
					response.Metadata = &metadataStr
				}
			}
		}
	}

	return response
}

// ResponseUpdates represents multiple updates to be applied to a response
type ResponseUpdates struct {
	Status *string `json:"status,omitempty"`
	Output any     `json:"output,omitempty"`
	Usage  any     `json:"usage,omitempty"`
	Error  any     `json:"error,omitempty"`
}

// ApplyResponseUpdates applies multiple updates to a response object (no DB access)
func ApplyResponseUpdates(response *Response, updates *ResponseUpdates) *common.Error {
	// Update status if provided
	if updates.Status != nil {
		UpdateResponseStatusOnObject(response, ResponseStatus(*updates.Status))
	}

	// Update output if provided
	if updates.Output != nil {
		if err := UpdateResponseOutputOnObject(response, updates.Output); err != nil {
			return err
		}
	}

	// Update usage if provided
	if updates.Usage != nil {
		if err := UpdateResponseUsageOnObject(response, updates.Usage); err != nil {
			return err
		}
	}

	// Update error if provided
	if updates.Error != nil {
		if err := UpdateResponseErrorOnObject(response, updates.Error); err != nil {
			return err
		}
	}

	return nil
}

// UpdateResponseStatusOnObject updates the status on a response object (no DB access)
func UpdateResponseStatusOnObject(response *Response, status ResponseStatus) {
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
}

// UpdateResponseOutputOnObject updates the output on a response object (no DB access)
func UpdateResponseOutputOnObject(response *Response, output any) *common.Error {
	// Convert output to JSON string
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return common.NewError(err, "s9t0u1v2-w3x4-5678-stuv-901234567890")
	}

	outputStr := string(outputJSON)
	// For JSON columns, use null for empty arrays/objects
	if outputStr == "[]" || outputStr == "{}" {
		response.Output = nil
	} else {
		response.Output = &outputStr
	}
	response.UpdatedAt = time.Now()

	return nil
}

// UpdateResponseUsageOnObject updates the usage statistics on a response object (no DB access)
func UpdateResponseUsageOnObject(response *Response, usage any) *common.Error {
	// Convert usage to JSON string
	usageJSON, err := json.Marshal(usage)
	if err != nil {
		return common.NewError(err, "w3x4y5z6-a7b8-9012-wxyz-345678901234")
	}

	usageStr := string(usageJSON)
	// For JSON columns, use null for empty arrays/objects
	if usageStr == "[]" || usageStr == "{}" {
		response.Usage = nil
	} else {
		response.Usage = &usageStr
	}
	response.UpdatedAt = time.Now()

	return nil
}

// UpdateResponseErrorOnObject updates the error information on a response object (no DB access)
func UpdateResponseErrorOnObject(response *Response, error any) *common.Error {
	// Convert error to JSON string
	errorJSON, err := json.Marshal(error)
	if err != nil {
		return common.NewError(err, "a7b8c9d0-e1f2-3456-abcd-789012345678")
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

	return nil
}
