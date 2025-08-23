package organization

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type Organization struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:128;not null;uniqueIndex"`
	PublicID  string `gorm:"size:64;not null;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Enabled   bool `gorm:"default:true;index"`
}

type OrganizationFilter struct {
	PublicID *string
	Enabled  *bool
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
