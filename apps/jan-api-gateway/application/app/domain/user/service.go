package user

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/organization"
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
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	key := base64.URLEncoding.EncodeToString(bytes)
	key = strings.TrimRight(key, "=")

	if len(key) > 16 {
		key = key[:16]
	} else if len(key) < 16 {
		extra := make([]byte, 16-len(key))
		_, err := rand.Read(extra)
		if err != nil {
			return "", err
		}
		key += base64.URLEncoding.EncodeToString(extra)[:16-len(key)]
	}

	return fmt.Sprintf("user-%s", key), nil
}
