package user

import (
	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

type UserService struct {
	userrepo            UserRepository
	organizationService *organization.OrganizationService
}

func NewService(userrepo UserRepository, organizationService *organization.OrganizationService) *UserService {
	return &UserService{
		userrepo:            userrepo,
		organizationService: organizationService,
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

func (s *UserService) FindByEmail(ctx context.Context, email string) (*User, error) {
	return s.userrepo.FindByEmail(ctx, email)
}

func (s *UserService) FindByID(ctx context.Context, id uint) (*User, error) {
	return s.userrepo.FindByID(ctx, id)
}

func (s *UserService) generatePublicID() (string, error) {
	return idgen.GenerateSecureID("user", 16)
}
