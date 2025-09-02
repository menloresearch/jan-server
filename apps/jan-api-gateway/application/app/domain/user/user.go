package user

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type UserPlatformType string

type User struct {
	ID        uint
	Name      string
	Email     string
	Enabled   bool
	PublicID  string
	CreatedAt time.Time
}

type UserFilter struct {
	Email    *string
	Enabled  *bool
	PublicID *string
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindFirst(ctx context.Context, filter UserFilter) (*User, error)
	FindByFilter(ctx context.Context, filter UserFilter, p *query.Pagination) ([]*User, error)
	FindByID(ctx context.Context, id uint) (*User, error)
}
