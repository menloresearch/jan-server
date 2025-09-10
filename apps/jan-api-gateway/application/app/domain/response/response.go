package response

import (
	"context"
	"time"

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
