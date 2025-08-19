package user

import "context"

type User struct {
	ID      uint
	Name    string
	Email   string
	Enabled bool
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
}
