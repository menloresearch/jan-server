package dbschema

import (
	"encoding/json"

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
	Status    string `gorm:"type:varchar(20);not null;default:'active'"`
	Metadata  string `gorm:"type:text"`
	IsPrivate bool   `gorm:"not null;default:true"`
	Items     []Item `gorm:"foreignKey:ConversationID"`
	User      User   `gorm:"foreignKey:UserID"`
}

type Item struct {
	BaseModel
	PublicID          string       `gorm:"type:varchar(50);uniqueIndex;not null"` // OpenAI-compatible string ID
	ConversationID    uint         `gorm:"not null;index"`
	ResponseID        *uint        `gorm:"index"` // Optional response ID for items that belong to a specific response
	Type              string       `gorm:"type:varchar(50);not null"`
	Role              string       `gorm:"type:varchar(20)"`
	Content           string       `gorm:"type:text"`
	Status            string       `gorm:"type:varchar(50)"`
	IncompleteAt      *int64       `gorm:"type:bigint"`
	IncompleteDetails string       `gorm:"type:text"`
	CompletedAt       *int64       `gorm:"type:bigint"`
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
		Title:     stringPtrToString(c.Title),
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
		CreatedAt: c.CreatedAt.Unix(), // Convert time.Time to Unix timestamp
		UpdatedAt: c.UpdatedAt.Unix(), // Convert time.Time to Unix timestamp
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
		PublicID:          i.PublicID, // Add PublicID field
		ConversationID:    i.ConversationID,
		ResponseID:        i.ResponseID,
		Type:              string(i.Type),
		Role:              stringPtrToString((*string)(i.Role)),
		Content:           contentJSON,
		Status:            stringPtrToString(i.Status),
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
		Status:            ptr.ToString(i.Status),
		IncompleteAt:      i.IncompleteAt,
		IncompleteDetails: incompleteDetails,
		CompletedAt:       i.CompletedAt,
		ConversationID:    i.ConversationID,
		ResponseID:        i.ResponseID,
		CreatedAt:         i.CreatedAt.Unix(), // Convert time.Time to Unix timestamp
	}
}

// Helper functions
func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
