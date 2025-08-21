package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"menlo.ai/jan-api-gateway/app/domain/query"
)

const (
	ApiKeyServiceTypeJanApi   uint = 0
	ApiKeyServiceTypeJanCloud uint = 1
)

func NewApiKey(userID uint, description string, serviceType uint, expiresAt *time.Time) (*ApiKey, error) {
	if userID == 0 {
		return nil, errors.New("invalid userID")
	}

	key, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	return &ApiKey{
		Key:         key.String(),
		UserID:      userID,
		Description: description,
		Enabled:     true,
		ServiceType: serviceType,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

type ApiKey struct {
	ID          uint
	Key         string
	UserID      uint
	Description string
	Enabled     bool
	ServiceType uint
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	Key         *string
	ServiceType *uint
	UserID      *uint
}

type ApiKeyRepository interface {
	Create(ctx context.Context, u *ApiKey) error
	Update(ctx context.Context, u *ApiKey) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ApiKey, error)
	FindByKey(ctx context.Context, key string) (*ApiKey, error)
	FindByFilter(ctx context.Context, filter ApiKeyFilter, pagination *query.Pagination) ([]*ApiKey, error)
	Count(ctx context.Context, filter ApiKeyFilter) (int64, error)
}
