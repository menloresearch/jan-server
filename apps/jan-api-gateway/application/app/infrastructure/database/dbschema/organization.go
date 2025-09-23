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
	Name     string               `gorm:"size:128;not null"`
	PublicID string               `gorm:"size:64;not null;uniqueIndex"`
	Enabled  bool                 `gorm:"default:true;index"`
	Members  []OrganizationMember `gorm:"foreignKey:OrganizationID"`
}

type OrganizationMember struct {
	BaseModel
	UserID         uint   `gorm:"not null;index:idx_user_org,unique"`
	OrganizationID uint   `gorm:"not null;index:idx_user_org,unique"`
	Role           string `gorm:"type:varchar(20);not null"`
}

func NewSchemaOrganization(o *organization.Organization) *Organization {
	return &Organization{
		BaseModel: BaseModel{
			ID: o.ID,
		},
		Name:     o.Name,
		PublicID: o.PublicID,
		Enabled:  o.Enabled,
	}
}

func NewSchemaOrganizationMember(o *organization.OrganizationMember) *OrganizationMember {
	return &OrganizationMember{
		BaseModel: BaseModel{
			ID: o.ID,
		},
		UserID:         o.UserID,
		OrganizationID: o.OrganizationID,
		Role:           string(o.Role),
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
	}
}

func (o *OrganizationMember) EtoD() *organization.OrganizationMember {
	return &organization.OrganizationMember{
		ID:             o.ID,
		UserID:         o.UserID,
		OrganizationID: o.OrganizationID,
		Role:           organization.OrganizationMemberRole(o.Role),
		CreatedAt:      o.CreatedAt,
	}
}
