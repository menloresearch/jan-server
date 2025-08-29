package conversation

import (
	"context"
	"time"
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
	ID                uint               `json:"id"`
	Type              ItemType           `json:"type"`
	Role              *ItemRole          `json:"role,omitempty"`
	Content           []Content          `json:"content,omitempty"`
	Status            *string            `json:"status,omitempty"`
	IncompleteAt      *int64             `json:"incomplete_at,omitempty"`
	IncompleteDetails *IncompleteDetails `json:"incomplete_details,omitempty"`
	CompletedAt       *int64             `json:"completed_at,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
}

type Content struct {
	Type string `json:"type"`
	Text *Text  `json:"text,omitempty"`
}

type Text struct {
	Value       string       `json:"value"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

type Annotation struct {
	Type       string `json:"type"`
	Text       string `json:"text"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
}

type IncompleteDetails struct {
	Reason string `json:"reason"`
}

type Conversation struct {
	ID        uint               `json:"id"`
	PublicID  string             `json:"public_id"`
	Title     *string            `json:"title,omitempty"`
	UserID    uint               `json:"user_id"`
	Status    ConversationStatus `json:"status"`
	Items     []Item             `json:"items,omitempty"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
	IsPrivate bool               `json:"is_private"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
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
	FindByConversationID(ctx context.Context, conversationID uint) ([]*Item, error)
	Search(ctx context.Context, conversationID uint, query string) ([]*Item, error)
	Delete(ctx context.Context, id uint) error
}
