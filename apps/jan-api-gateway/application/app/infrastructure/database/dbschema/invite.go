package dbschema

import (
	"time"

	"menlo.ai/jan-api-gateway/app/domain/invite"
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Invite{})
}

type Invite struct {
	BaseModel
	PublicID       string `gorm:"size:64;not null;uniqueIndex"`
	Email          string `gorm:"size:128;not null"`
	Role           string `gorm:"type:varchar(20);not null"`
	Status         string `gorm:"type:varchar(20);not null;index"`
	InvitedAt      time.Time
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	Secrets        *string `gorm:"type:text"`
	Projects       string  `gorm:"type:jsonb"`
	OrganizationID uint    `gorm:"not null;index"`
}

func NewSchemaInvite(i *invite.Invite) *Invite {
	return &Invite{
		BaseModel: BaseModel{
			ID: i.ID,
		},
		PublicID:       i.PublicID,
		Email:          i.Email,
		Role:           i.Role,
		Status:         i.Status,
		InvitedAt:      i.InvitedAt,
		ExpiresAt:      i.ExpiresAt,
		AcceptedAt:     i.AcceptedAt,
		Secrets:        i.Secrets,
		Projects:       i.Projects,
		OrganizationID: i.OrganizationID,
	}
}

func (i *Invite) EtoD() *invite.Invite {
	return &invite.Invite{
		ID:             i.ID,
		PublicID:       i.PublicID,
		Email:          i.Email,
		Role:           i.Role,
		Status:         i.Status,
		InvitedAt:      i.InvitedAt,
		ExpiresAt:      i.ExpiresAt,
		AcceptedAt:     i.AcceptedAt,
		Secrets:        i.Secrets,
		Projects:       i.Projects,
		OrganizationID: i.OrganizationID,
	}
}
