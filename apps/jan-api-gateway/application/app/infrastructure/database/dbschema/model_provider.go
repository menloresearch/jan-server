package dbschema

import (
	"time"

	domain "menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(ModelProvider{})
}

type ModelProvider struct {
	BaseModel
	PublicID        string     `gorm:"type:varchar(60);uniqueIndex;not null"`
	OrganizationID  *uint      `gorm:"index"`
	ProjectID       *uint      `gorm:"index"`
	Name            string     `gorm:"type:varchar(150);not null"`
	Type            string     `gorm:"type:varchar(32);not null"`
	Vendor          string     `gorm:"type:varchar(64);not null"`
	BaseURL         string     `gorm:"type:text"`
	EncryptedAPIKey string     `gorm:"type:text"`
	APIKeyHint      string     `gorm:"type:varchar(64)"`
	MetadataJSON    string     `gorm:"type:jsonb;default:'{}'"`
	Active          bool       `gorm:"not null;default:true"`
	LastSyncedAt    *time.Time `gorm:"index"`
}

func NewSchemaModelProvider(p *domain.ModelProvider) *ModelProvider {
	if p == nil {
		return nil
	}
	return &ModelProvider{
		BaseModel: BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		PublicID:        p.PublicID,
		OrganizationID:  p.OrganizationID,
		ProjectID:       p.ProjectID,
		Name:            p.Name,
		Type:            p.Type.String(),
		Vendor:          p.Vendor.String(),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		MetadataJSON:    p.MetadataJSON,
		Active:          p.Active,
		LastSyncedAt:    p.LastSyncedAt,
	}
}

func (p *ModelProvider) EtoD() *domain.ModelProvider {
	if p == nil {
		return nil
	}
	return &domain.ModelProvider{
		ID:              p.ID,
		PublicID:        p.PublicID,
		OrganizationID:  p.OrganizationID,
		ProjectID:       p.ProjectID,
		Name:            p.Name,
		Type:            domain.ProviderType(p.Type),
		Vendor:          domain.ProviderVendor(p.Vendor),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		MetadataJSON:    p.MetadataJSON,
		Active:          p.Active,
		LastSyncedAt:    p.LastSyncedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
