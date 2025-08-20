package dbschema

import (
	"time"

	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(ApiKey{})
}

func NewSchemaApiKey(a *apikey.ApiKey) *ApiKey {
	return &ApiKey{
		BaseModel: BaseModel{
			ID: a.ID,
		},
		Key:         a.Key,
		UserID:      a.UserID,
		Description: a.Description,
		Enabled:     a.Enabled,
		ServiceType: a.ServiceType,
		ExpiresAt:   a.ExpiresAt,
	}
}

type ApiKey struct {
	BaseModel
	Key         string `gorm:"uniqueIndex;not null"`
	UserID      uint   `gorm:"index:idx_user_service_type,not null"`
	Description string `gorm:"size:255"`
	Enabled     bool   `gorm:"default:true;index"`
	ServiceType uint   `gorm:"index:idx_user_service_type,not null"`
	ExpiresAt   *time.Time
}

func (a *ApiKey) EtoD() *apikey.ApiKey {
	return &apikey.ApiKey{
		ID:          a.ID,
		Key:         a.Key,
		UserID:      a.UserID,
		Description: a.Description,
		Enabled:     a.Enabled,
		ServiceType: a.ServiceType,
		ExpiresAt:   a.ExpiresAt,
		CreatedAt:   a.CreatedAt,
	}
}
