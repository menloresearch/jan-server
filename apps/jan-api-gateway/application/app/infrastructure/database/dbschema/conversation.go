package dbschema

import (
	"encoding/json"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Conversation{})
	database.RegisterSchemaForAutoMigrate(Item{})
}

type Conversation struct {
	BaseModel
	PublicID  string `gorm:"type:varchar(50);uniqueIndex;not null"`
	Title     string `gorm:"type:varchar(255)"`
	UserID    uint   `gorm:"not null;index"`
	Status    string `gorm:"type:varchar(20);not null;default:'active';index"`
	Metadata  string `gorm:"type:text"`
	IsPrivate bool   `gorm:"not null;default:true;index"`
	Items     []Item `gorm:"foreignKey:ConversationID"`
	User      User   `gorm:"foreignKey:UserID"`
}

type Item struct {
	BaseModel
	PublicID          string       `gorm:"type:varchar(50);uniqueIndex;not null"`
	ConversationID    uint         `gorm:"not null;index"`
	ResponseID        *uint        `gorm:"index"`
	Type              string       `gorm:"type:varchar(50);not null;index"`
	Role              string       `gorm:"type:varchar(20);index"`
	Content           string       `gorm:"type:text"`
	Status            string       `gorm:"type:varchar(50);index"`
	IncompleteAt      *time.Time   `gorm:"type:timestamp"`
	IncompleteDetails string       `gorm:"type:text"`
	CompletedAt       *time.Time   `gorm:"type:timestamp"`
	Conversation      Conversation `gorm:"foreignKey:ConversationID"`
	Response          *Response    `gorm:"foreignKey:ResponseID"`
}

func NewSchemaConversation(c *conversation.Conversation) *Conversation {
	var metadataJSON string
	if c.Metadata != nil {
		metadataBytes, err := json.Marshal(c.Metadata)
		if err != nil {
			metadataJSON = "{}"
		} else {
			metadataJSON = string(metadataBytes)
		}
	}

	return &Conversation{
		BaseModel: BaseModel{
			ID: c.ID,
		},
		PublicID:  c.PublicID,
		Title:     ptr.FromString(c.Title),
		UserID:    c.UserID,
		Status:    string(c.Status),
		Metadata:  metadataJSON,
		IsPrivate: c.IsPrivate,
	}
}

func (c *Conversation) EtoD() *conversation.Conversation {
	var metadata map[string]string
	if c.Metadata != "" {
		json.Unmarshal([]byte(c.Metadata), &metadata)
	}

	title := ptr.ToString(c.Title)

	return &conversation.Conversation{
		ID:        c.ID,
		PublicID:  c.PublicID,
		Title:     title,
		UserID:    c.UserID,
		Status:    conversation.ConversationStatus(c.Status),
		Metadata:  metadata,
		IsPrivate: c.IsPrivate,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func NewSchemaItem(i *conversation.Item) *Item {
	// Convert Content slice to JSON string for storage
	var contentJSON string
	if i.Content != nil {
		contentBytes, _ := json.Marshal(i.Content)
		contentJSON = string(contentBytes)
	}

	// Convert IncompleteDetails to JSON string
	var incompleteDetailsJSON string
	if i.IncompleteDetails != nil {
		incompleteDetailsBytes, _ := json.Marshal(i.IncompleteDetails)
		incompleteDetailsJSON = string(incompleteDetailsBytes)
	}

	return &Item{
		BaseModel: BaseModel{
			ID: i.ID,
		},
		PublicID:          i.PublicID,
		ConversationID:    i.ConversationID,
		ResponseID:        i.ResponseID,
		Type:              string(i.Type),
		Role:              string(*i.Role),
		Content:           contentJSON,
		Status:            string(*i.Status),
		IncompleteAt:      i.IncompleteAt,
		IncompleteDetails: incompleteDetailsJSON,
		CompletedAt:       i.CompletedAt,
	}
}

func (i *Item) EtoD() *conversation.Item {
	// Parse Content JSON back to slice
	var content []conversation.Content
	if i.Content != "" {
		json.Unmarshal([]byte(i.Content), &content)
	}

	// Parse IncompleteDetails JSON
	var incompleteDetails *conversation.IncompleteDetails
	if i.IncompleteDetails != "" {
		incompleteDetails = &conversation.IncompleteDetails{}
		json.Unmarshal([]byte(i.IncompleteDetails), incompleteDetails)
	}

	return &conversation.Item{
		ID:                i.ID,
		PublicID:          i.PublicID, // Add PublicID field
		Type:              conversation.ItemType(i.Type),
		Role:              (*conversation.ItemRole)(ptr.ToString(i.Role)),
		Content:           content,
		Status:            (*conversation.ItemStatus)(ptr.ToString(i.Status)),
		IncompleteAt:      i.IncompleteAt,
		IncompleteDetails: incompleteDetails,
		CompletedAt:       i.CompletedAt,
		ConversationID:    i.ConversationID,
		ResponseID:        i.ResponseID,
		CreatedAt:         i.CreatedAt,
	}
}
