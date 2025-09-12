package apikey

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type ApikeyType string

const (
	ApikeyTypeAdmin        ApikeyType = "admin"
	ApikeyTypeProject      ApikeyType = "project"
	ApikeyTypeService      ApikeyType = "service"
	ApikeyTypeOrganization ApikeyType = "organization"
	ApikeyTypeEphemeral    ApikeyType = "ephemeral"
)

type ApiKey struct {
	ID             uint
	PublicID       string
	KeyHash        string
	PlaintextHint  string
	Description    string
	Enabled        bool
	ApikeyType     string // "admin","project","service","organization","ephemeral"
	OwnerPublicID  string
	ProjectID      *uint
	OrganizationID *uint
	Permissions    string //json
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
}

func (k *ApiKey) Revoke() {
	k.Enabled = false
	k.UpdatedAt = time.Now()
}

func (k *ApiKey) IsValid() bool {
	if !k.Enabled {
		return false
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

type ApiKeyFilter struct {
	KeyHash        *string
	PublicID       *string
	ApikeyType     *string
	OwnerPublicID  *string
	ProjectID      *uint
	UserID         *uint
	OrganizationID *uint
}

type ApiKeyRepository interface {
	Create(ctx context.Context, u *ApiKey) error
	Update(ctx context.Context, u *ApiKey) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ApiKey, error)
	FindByKeyHash(ctx context.Context, keyHash string) (*ApiKey, error)
	FindByFilter(ctx context.Context, filter ApiKeyFilter, pagination *query.Pagination) ([]*ApiKey, error)
	Count(ctx context.Context, filter ApiKeyFilter) (int64, error)
}
