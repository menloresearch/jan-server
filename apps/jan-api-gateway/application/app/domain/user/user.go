package user

import (
	"context"
	"time"
)

type UserPlatformType string

const (
	UserPlatformTypePlatform UserPlatformType = "plaftorm"
	UserPlatformTypeAskJanAI UserPlatformType = "ask.jan.ai"
)

type User struct {
	ID           uint
	Name         string
	Email        string
	Enabled      bool
	PlatformType string
	PublicID     string
	CreatedAt    time.Time
}

type UserFilter struct {
	Email        *string
	Enabled      *bool
	PlatformType *string
	PublicID     *string
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindFirst(ctx context.Context, filter UserFilter) (*User, error)
	FindByID(ctx context.Context, id uint) (*User, error)
}
