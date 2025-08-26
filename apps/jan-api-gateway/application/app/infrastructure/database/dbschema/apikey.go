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
	KeyHash       string `gorm:"size:128;uniqueIndex;not null"`
	PlaintextHint string `gorm:"size:16"`
	Description   string `gorm:"size:255"`
	Enabled       bool   `gorm:"default:true;index"`

	OwnerType      string `gorm:"size:32;index;not null"` // "admin","project","service","organization","ephemeral"
	OwnerID        *uint  `gorm:"index"`
	OrganizationID *uint  `gorm:"index"`

	Permissions string     `gorm:"type:json"`
	ExpiresAt   *time.Time `gorm:"index"`
}

func NewSchemaApiKey(a *apikey.ApiKey) *ApiKey {
	return &ApiKey{
		BaseModel: BaseModel{
			ID: a.ID,
		},
		KeyHash:        a.KeyHash,
		PlaintextHint:  a.PlaintextHint,
		Description:    a.Description,
		Enabled:        a.Enabled,
		OwnerType:      a.OwnerType,
		OwnerID:        a.OwnerID,
		OrganizationID: a.OrganizationID,
		Permissions:    a.Permissions,
		ExpiresAt:      a.ExpiresAt,
	}
}

func (a *ApiKey) EtoD() *apikey.ApiKey {
	return &apikey.ApiKey{
		ID:             a.ID,
		KeyHash:        a.KeyHash,
		PlaintextHint:  a.PlaintextHint,
		Description:    a.Description,
		Enabled:        a.Enabled,
		OwnerType:      a.OwnerType,
		OwnerID:        a.OwnerID,
		OrganizationID: a.OrganizationID,
		Permissions:    a.Permissions,
		ExpiresAt:      a.ExpiresAt,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}
