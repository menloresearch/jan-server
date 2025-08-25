package organization

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type Organization struct {
	ID        uint
	Name      string
	PublicID  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Enabled   bool
	OwnerID   uint
}

type OrganizationFilter struct {
	PublicID *string
	Enabled  *bool
	OwnerID  *uint
}

type OrganizationRepository interface {
	Create(ctx context.Context, o *Organization) error
	Update(ctx context.Context, o *Organization) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Organization, error)
	FindByPublicID(ctx context.Context, publicID string) (*Organization, error)
	FindByFilter(ctx context.Context, filter OrganizationFilter, pagination *query.Pagination) ([]*Organization, error)
	Count(ctx context.Context, filter OrganizationFilter) (int64, error)
}
