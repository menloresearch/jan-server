package dbschema

import (
	"time"

	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(ApiKey{})
}

type ApiKey struct {
	BaseModel
	PublicID      string `gorm:"size:128;uniqueIndex;not null"`
	KeyHash       string `gorm:"size:128;uniqueIndex;not null"`
	PlaintextHint string `gorm:"size:16"`
	Description   string `gorm:"size:255"`
	Enabled       bool   `gorm:"default:true;index"`

	ApikeyType     string `gorm:"size:32;index;not null"` // "admin","project","service","organization","ephemeral"
	OwnerPublicID  string `gorm:"type:varchar(50);not null"`
	OrganizationID *uint  `gorm:"index"`
	ProjectID      *uint  `gorm:"index"`

	Permissions string     `gorm:"type:json"`
	ExpiresAt   *time.Time `gorm:"type:timestamp"`
	LastUsedAt  *time.Time `gorm:"type:timestamp"`
}

func NewSchemaApiKey(a *apikey.ApiKey) *ApiKey {
	return &ApiKey{
		BaseModel: BaseModel{
			ID: a.ID,
		},
		PublicID:       a.PublicID,
		KeyHash:        a.KeyHash,
		PlaintextHint:  a.PlaintextHint,
		Description:    a.Description,
		Enabled:        a.Enabled,
		ApikeyType:     a.ApikeyType,
		OwnerPublicID:  a.OwnerPublicID,
		ProjectID:      a.ProjectID,
		OrganizationID: a.OrganizationID,
		Permissions:    a.Permissions,
		ExpiresAt:      a.ExpiresAt,
		LastUsedAt:     a.LastUsedAt,
	}
}

func (a *ApiKey) EtoD() *apikey.ApiKey {
	return &apikey.ApiKey{
		ID:             a.ID,
		PublicID:       a.PublicID,
		KeyHash:        a.KeyHash,
		PlaintextHint:  a.PlaintextHint,
		Description:    a.Description,
		Enabled:        a.Enabled,
		ApikeyType:     a.ApikeyType,
		OwnerPublicID:  a.OwnerPublicID,
		ProjectID:      a.ProjectID,
		OrganizationID: a.OrganizationID,
		Permissions:    a.Permissions,
		ExpiresAt:      a.ExpiresAt,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
		LastUsedAt:     a.LastUsedAt,
	}
}
