package dbschema

import (
	"time"

	"menlo.ai/jan-api-gateway/app/domain/response"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

// Response represents the response table in the database
type Response struct {
	BaseModel
	PublicID           string  `gorm:"size:255;not null;uniqueIndex"`
	UserID             uint    `gorm:"not null;index"`
	ConversationID     *uint   `gorm:"index"`
	PreviousResponseID *string `gorm:"size:255;index"`
	Model              string  `gorm:"size:255;not null;index"`
	Status             string  `gorm:"size:50;not null;default:'pending';index"`
	Input              string  `gorm:"type:text;not null"`
	Output             *string `gorm:"type:text"`
	SystemPrompt       *string `gorm:"type:text"`
	MaxTokens          *int
	Temperature        *float64
	TopP               *float64
	TopK               *int
	RepetitionPenalty  *float64
	Seed               *int
	Stop               *string `gorm:"type:text"`
	PresencePenalty    *float64
	FrequencyPenalty   *float64
	LogitBias          *string `gorm:"type:text"`
	ResponseFormat     *string `gorm:"type:text"`
	Tools              *string `gorm:"type:text"`
	ToolChoice         *string `gorm:"type:text"`
	Metadata           *string `gorm:"type:text"`
	Stream             *bool
	Background         *bool
	Timeout            *int
	User               *string `gorm:"size:255"`
	Usage              *string `gorm:"type:text"`
	Error              *string `gorm:"type:text"`
	CompletedAt        *time.Time
	CancelledAt        *time.Time
	FailedAt           *time.Time

	// Relationships
	UserEntity   User          `gorm:"foreignKey:UserID;references:ID"`
	Conversation *Conversation `gorm:"foreignKey:ConversationID;references:ID"`
	Items        []Item        `gorm:"foreignKey:ResponseID;references:ID"`
}

// TableName returns the table name for the Response model
func (Response) TableName() string {
	return "responses"
}

func init() {
	database.RegisterSchemaForAutoMigrate(Response{})
}

// NewSchemaResponse converts domain Response to database Response
func NewSchemaResponse(r *response.Response) *Response {
	return &Response{
		BaseModel: BaseModel{
			ID: r.ID,
		},
		PublicID:           r.PublicID,
		UserID:             r.UserID,
		ConversationID:     r.ConversationID,
		PreviousResponseID: r.PreviousResponseID,
		Model:              r.Model,
		Status:             string(r.Status),
		Input:              r.Input,
		Output:             r.Output,
		SystemPrompt:       r.SystemPrompt,
		MaxTokens:          r.MaxTokens,
		Temperature:        r.Temperature,
		TopP:               r.TopP,
		TopK:               r.TopK,
		RepetitionPenalty:  r.RepetitionPenalty,
		Seed:               r.Seed,
		Stop:               r.Stop,
		PresencePenalty:    r.PresencePenalty,
		FrequencyPenalty:   r.FrequencyPenalty,
		LogitBias:          r.LogitBias,
		ResponseFormat:     r.ResponseFormat,
		Tools:              r.Tools,
		ToolChoice:         r.ToolChoice,
		Metadata:           r.Metadata,
		Stream:             r.Stream,
		Background:         r.Background,
		Timeout:            r.Timeout,
		User:               r.User,
		Usage:              r.Usage,
		Error:              r.Error,
		CompletedAt:        r.CompletedAt,
		CancelledAt:        r.CancelledAt,
		FailedAt:           r.FailedAt,
	}
}

// EtoD converts database Response to domain Response
func (r *Response) EtoD() *response.Response {
	return &response.Response{
		ID:                 r.ID,
		PublicID:           r.PublicID,
		UserID:             r.UserID,
		ConversationID:     r.ConversationID,
		PreviousResponseID: r.PreviousResponseID,
		Model:              r.Model,
		Status:             response.ResponseStatus(r.Status),
		Input:              r.Input,
		Output:             r.Output,
		SystemPrompt:       r.SystemPrompt,
		MaxTokens:          r.MaxTokens,
		Temperature:        r.Temperature,
		TopP:               r.TopP,
		TopK:               r.TopK,
		RepetitionPenalty:  r.RepetitionPenalty,
		Seed:               r.Seed,
		Stop:               r.Stop,
		PresencePenalty:    r.PresencePenalty,
		FrequencyPenalty:   r.FrequencyPenalty,
		LogitBias:          r.LogitBias,
		ResponseFormat:     r.ResponseFormat,
		Tools:              r.Tools,
		ToolChoice:         r.ToolChoice,
		Metadata:           r.Metadata,
		Stream:             r.Stream,
		Background:         r.Background,
		Timeout:            r.Timeout,
		User:               r.User,
		Usage:              r.Usage,
		Error:              r.Error,
		CompletedAt:        r.CompletedAt,
		CancelledAt:        r.CancelledAt,
		FailedAt:           r.FailedAt,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}
