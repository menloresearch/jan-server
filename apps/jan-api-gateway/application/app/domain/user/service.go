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
	user.PlatformType = string(UserPlatformTypeAskJanAI)
	if err := s.userrepo.Create(ctx, user); err != nil {
		return nil, err
	}
	s.organizationService.CreateOrganizationWithPublicID(ctx, &organization.Organization{
		Name:    "Default",
		Enabled: true,
		OwnerID: user.ID,
	})
	return user, nil
}

func (s *UserService) RegisterPlatformUser(ctx context.Context, user *User) (*User, error) {
	publicId, err := s.generatePublicID()
	if err != nil {
		return nil, err
	}
	user.PublicID = publicId
	user.PlatformType = string(UserPlatformTypePlatform)
	if err := s.userrepo.Create(ctx, user); err != nil {
		return nil, err
	}
	s.organizationService.CreateOrganizationWithPublicID(ctx, &organization.Organization{
		Name:    "Default",
		Enabled: true,
		OwnerID: user.ID,
	})
	return user, nil
}

func (s *UserService) FindByEmailAndPlatform(ctx context.Context, email string, platform string) (*User, error) {
	return s.userrepo.FindFirst(ctx, UserFilter{
		Email:        &email,
		PlatformType: &platform,
	})
}

func (s *UserService) FindByID(ctx context.Context, id uint) (*User, error) {
	return s.userrepo.FindByID(ctx, id)
}

func (s *UserService) generatePublicID() (string, error) {
	return idgen.GenerateSecureID("user", 16)
}
