package dbschema

import (
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Organization{})
	database.RegisterSchemaForAutoMigrate(OrganizationMember{})
}

type Organization struct {
	BaseModel
	Name     string               `gorm:"size:128;not null;uniqueIndex"`
	PublicID string               `gorm:"size:64;not null;uniqueIndex"`
	Enabled  bool                 `gorm:"default:true;index"`
	Members  []OrganizationMember `gorm:"foreignKey:OrganizationID"`
	OwnerID  uint                 `gorm:"not null;index"`
	Owner    User                 `gorm:"foreignKey:OwnerID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

type OrganizationMember struct {
	BaseModel
	UserID         uint   `gorm:"primaryKey"`
	OrganizationID uint   `gorm:"primaryKey"`
	Role           string `gorm:"type:varchar(20);not null"`
}

func NewSchemaOrganization(o *organization.Organization) *Organization {
	return &Organization{
		BaseModel: BaseModel{
			ID: o.ID,
		},
		Name:     o.Name,
		PublicID: o.PublicID,
		OwnerID:  o.OwnerID,
		Enabled:  o.Enabled,
	}
}

func (o *Organization) EtoD() *organization.Organization {
	return &organization.Organization{
		ID:        o.ID,
		Name:      o.Name,
		PublicID:  o.PublicID,
		Enabled:   o.Enabled,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
		OwnerID:   o.OwnerID,
	}
}
