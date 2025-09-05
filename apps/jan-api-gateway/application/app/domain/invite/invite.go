package invite

import (
	"context"
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

type InviteProjectRole string

const (
	InviteProjectRoleMember InviteProjectRole = "member"
	InviteProjectRoleOwner  InviteProjectRole = "owner"
)

type InviteProject struct {
	ID   string
	Role string
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
