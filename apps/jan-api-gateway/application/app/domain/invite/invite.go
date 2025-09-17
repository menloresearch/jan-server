package invite

import (
	"context"
	"encoding/json"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type Invite struct {
	ID             uint
	PublicID       string
	Email          string
	Role           string
	Status         string
	InvitedAt      time.Time
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	OrganizationID uint
	Secrets        *string
	Projects       string
}

func (i *Invite) GetProjects() ([]InviteProject, error) {
	var projects []InviteProject
	byteData := []byte(i.Projects)
	err := json.Unmarshal(byteData, &projects)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

type InviteStatus string

const (
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusPending  InviteStatus = "pending"
)

type InviteProjectRole string

const (
	InviteProjectRoleMember InviteProjectRole = "member"
	InviteProjectRoleOwner  InviteProjectRole = "owner"
)

type InviteProject struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type InvitesFilter struct {
	PublicID       *string
	OrganizationID *uint
}

type InviteRepository interface {
	Create(ctx context.Context, p *Invite) error
	Update(ctx context.Context, p *Invite) error
	DeleteByID(ctx context.Context, id uint) error
	FindByFilter(ctx context.Context, filter InvitesFilter, p *query.Pagination) ([]*Invite, error)
	Count(ctx context.Context, filter InvitesFilter) (int64, error)
}
