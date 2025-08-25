package user

import (
	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/organization"
)

type UserService struct {
	userrepo            UserRepository
	organizationService organization.OrganizationService
}

func NewService(userrepo UserRepository, organizationService organization.OrganizationService) *UserService {
	return &UserService{
		userrepo:            userrepo,
		organizationService: organizationService,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, user *User) (*User, error) {
	if err := s.userrepo.Create(ctx, user); err != nil {
		return nil, err
	}
	_, err := s.organizationService.CreateOrganizationWithPublicID(ctx, &organization.Organization{
		Name:    "Default Organization",
		Enabled: false,
		OwnerID: user.ID,
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) FindByEmail(ctx context.Context, email string) (*User, error) {
	return s.userrepo.FindByEmail(ctx, email)
}
