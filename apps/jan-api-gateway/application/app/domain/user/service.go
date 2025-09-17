package user

import (
	"fmt"

	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

type UserService struct {
	userrepo UserRepository
}

func NewService(userrepo UserRepository) *UserService {
	return &UserService{
		userrepo: userrepo,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, user *User) (*User, error) {
	publicId, err := s.generatePublicID()
	if err != nil {
		return nil, err
	}
	user.PublicID = publicId
	if err := s.userrepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, user *User) (*User, error) {
	if err := s.userrepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) FindByEmail(ctx context.Context, email string) (*User, error) {
	users, err := s.userrepo.FindByFilter(ctx, UserFilter{
		Email: &email,
	}, nil)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) != 1 {
		return nil, fmt.Errorf("invalid email")
	}
	return users[0], nil
}

func (s *UserService) FindByFilter(ctx context.Context, filter UserFilter) ([]*User, error) {
	return s.userrepo.FindByFilter(ctx, filter, nil)
}

func (s *UserService) FindByID(ctx context.Context, id uint) (*User, error) {
	return s.userrepo.FindByID(ctx, id)
}

func (s *UserService) FindByPublicID(ctx context.Context, publicID string) (*User, error) {
	userEntities, err := s.userrepo.FindByFilter(ctx, UserFilter{PublicID: &publicID}, nil)
	if err != nil {
		return nil, err
	}
	if len(userEntities) != 1 {
		return nil, fmt.Errorf("user does not exist")
	}
	return userEntities[0], nil
}

func (s *UserService) generatePublicID() (string, error) {
	return idgen.GenerateSecureID("user", 24)
}
