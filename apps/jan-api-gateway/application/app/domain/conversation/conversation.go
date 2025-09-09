package conversation

import (
	"context"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusArchived ConversationStatus = "archived"
	ConversationStatusDeleted  ConversationStatus = "deleted"
)

// @Enum(message, function_call, function_call_output)
type ItemType string

const (
	ItemTypeMessage      ItemType = "message"
	ItemTypeFunction     ItemType = "function_call"
	ItemTypeFunctionCall ItemType = "function_call_output"
)

func ValidateItemType(input string) bool {
	switch ItemType(input) {
	case ItemTypeMessage, ItemTypeFunction, ItemTypeFunctionCall:
		return true
	default:
		return false
	}
}

// @Enum(system, user, assistant)
type ItemRole string

const (
	ItemRoleSystem    ItemRole = "system"
	ItemRoleUser      ItemRole = "user"
	ItemRoleAssistant ItemRole = "assistant"
)

func ValidateItemRole(input string) bool {
	switch ItemRole(input) {
	case ItemRoleSystem, ItemRoleUser, ItemRoleAssistant:
		return true
	default:
		return false
	}
}

type Item struct {
	ID                uint               `json:"-"` // Internal DB ID (hidden from JSON)
	ConversationID    uint               `json:"-"`
	PublicID          string             `json:"id"` // OpenAI-compatible string ID like "msg_abc123"
	Type              ItemType           `json:"type"`
	Role              *ItemRole          `json:"role,omitempty"`
	Content           []Content          `json:"content,omitempty"`
	Status            *string            `json:"status,omitempty"`
	IncompleteAt      *int64             `json:"incomplete_at,omitempty"`
	IncompleteDetails *IncompleteDetails `json:"incomplete_details,omitempty"`
	CompletedAt       *int64             `json:"completed_at,omitempty"`
	CreatedAt         int64              `json:"created_at"` // Unix timestamp for OpenAI compatibility
}

type Content struct {
	Type       string        `json:"type"`
	Text       *Text         `json:"text,omitempty"`        // Generic text content
	InputText  *string       `json:"input_text,omitempty"`  // User input text (simple)
	OutputText *OutputText   `json:"output_text,omitempty"` // AI output text (with annotations)
	Image      *ImageContent `json:"image,omitempty"`       // Image content
	File       *FileContent  `json:"file,omitempty"`        // File content
}

// Generic text content (backward compatibility)
type Text struct {
	Value       string       `json:"value"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

type OutputText struct {
	Text        string       `json:"text"`
	Annotations []Annotation `json:"annotations"`        // Required for OpenAI compatibility
	LogProbs    []LogProb    `json:"logprobs,omitempty"` // Token probabilities
}

// Image content for multimodal support
type ImageContent struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// File content for attachments
type FileContent struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type Annotation struct {
	Type       string `json:"type"`              // "file_citation", "url_citation", etc.
	Text       string `json:"text,omitempty"`    // Display text
	FileID     string `json:"file_id,omitempty"` // For file citations
	URL        string `json:"url,omitempty"`     // For URL citations
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Index      int    `json:"index,omitempty"` // Citation index
}

// Log probability for AI responses
type LogProb struct {
	Token       string       `json:"token"`
	LogProb     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes,omitempty"`
	TopLogProbs []TopLogProb `json:"top_logprobs,omitempty"`
}

type TopLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

type IncompleteDetails struct {
	Reason string `json:"reason"`
}

type Conversation struct {
	ID        uint               `json:"-"`  // Internal DB ID (hidden from JSON)
	PublicID  string             `json:"id"` // OpenAI-compatible string ID like "conv_abc123"
	Title     *string            `json:"title,omitempty"`
	UserID    uint               `json:"-"` // Internal user ID (hidden from JSON)
	Status    ConversationStatus `json:"status"`
	Items     []Item             `json:"items,omitempty"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
	IsPrivate bool               `json:"is_private"`
	CreatedAt int64              `json:"created_at"` // Unix timestamp for OpenAI compatibility
	UpdatedAt int64              `json:"updated_at"` // Unix timestamp for OpenAI compatibility
}

type ConversationFilter struct {
	PublicID *string
	UserID   *uint
}

type ItemFilter struct {
	PublicID       *string
	ConversationID *uint
}

type ConversationRepository interface {
	Create(ctx context.Context, conversation *Conversation) error
	FindByFilter(ctx context.Context, filter ConversationFilter, pagination *query.Pagination) ([]*Conversation, error)
	Count(ctx context.Context, filter ConversationFilter) (int64, error)
	FindByID(ctx context.Context, id uint) (*Conversation, error)
	FindByPublicID(ctx context.Context, publicID string) (*Conversation, error)
	Update(ctx context.Context, conversation *Conversation) error
	Delete(ctx context.Context, id uint) error
	AddItem(ctx context.Context, conversationID uint, item *Item) error
	SearchItems(ctx context.Context, conversationID uint, query string) ([]*Item, error)
	BulkAddItems(ctx context.Context, conversationID uint, items []*Item) error
}

type ItemRepository interface {
	Create(ctx context.Context, item *Item) error
	FindByID(ctx context.Context, id uint) (*Item, error)
	FindByPublicID(ctx context.Context, publicID string) (*Item, error) // Find by OpenAI-compatible string ID
	FindByConversationID(ctx context.Context, conversationID uint) ([]*Item, error)
	Search(ctx context.Context, conversationID uint, query string) ([]*Item, error)
	Delete(ctx context.Context, id uint) error
	BulkCreate(ctx context.Context, items []*Item) error
	CountByConversation(ctx context.Context, conversationID uint) (int64, error)
	ExistsByIDAndConversation(ctx context.Context, itemID uint, conversationID uint) (bool, error)
	FindByFilter(ctx context.Context, filter ItemFilter, pagination *query.Pagination) ([]*Item, error)
	Count(ctx context.Context, filter ItemFilter) (int64, error)
}
