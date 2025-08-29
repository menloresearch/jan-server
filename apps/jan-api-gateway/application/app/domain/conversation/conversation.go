package conversation

import (
	"context"
)

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusArchived ConversationStatus = "archived"
	ConversationStatusDeleted  ConversationStatus = "deleted"
)

type ItemType string

const (
	ItemTypeMessage      ItemType = "message"
	ItemTypeFunction     ItemType = "function_call"
	ItemTypeFunctionCall ItemType = "function_call_output"
)

type ItemRole string

const (
	ItemRoleSystem    ItemRole = "system"
	ItemRoleUser      ItemRole = "user"
	ItemRoleAssistant ItemRole = "assistant"
)

type Item struct {
	ID                uint               `json:"-"`  // Internal DB ID (hidden from JSON)
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

// Enhanced AI output text with annotations and logprobs
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

// Enhanced annotation with more types
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

type ConversationRepository interface {
	Create(ctx context.Context, conversation *Conversation) error
	FindByID(ctx context.Context, id uint) (*Conversation, error)
	FindByPublicID(ctx context.Context, publicID string) (*Conversation, error)

	Update(ctx context.Context, conversation *Conversation) error
	Delete(ctx context.Context, id uint) error
	AddItem(ctx context.Context, conversationID uint, item *Item) error
	SearchItems(ctx context.Context, conversationID uint, query string) ([]*Item, error)
}

type ItemRepository interface {
	Create(ctx context.Context, item *Item) error
	FindByID(ctx context.Context, id uint) (*Item, error)
	FindByPublicID(ctx context.Context, publicID string) (*Item, error) // Find by OpenAI-compatible string ID
	FindByConversationID(ctx context.Context, conversationID uint) ([]*Item, error)
	Search(ctx context.Context, conversationID uint, query string) ([]*Item, error)
	Delete(ctx context.Context, id uint) error
	DeleteByPublicID(ctx context.Context, publicID string) error // Delete by OpenAI-compatible string ID
}
