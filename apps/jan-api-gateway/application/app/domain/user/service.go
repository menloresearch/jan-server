package user

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
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
	s.organizationService.CreateOrganizationWithPublicID(ctx, &organization.Organization{
		Name:    "Default",
		Enabled: true,
		OwnerID: user.ID,
	})
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

func (s *UserService) FindByID(ctx context.Context, id uint) (*User, error) {
	return s.userrepo.FindByID(ctx, id)
}

func (s *UserService) generatePublicID() (string, error) {
	return idgen.GenerateSecureID("user", 16)
}

type UserContextKey string

const (
	UserContextKeyEntity UserContextKey = "UserContextKeyEntity"
)

func (s *UserService) RegisteredUserMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "9715151d-02ab-4759-bfb7-89d717f05cd3",
				Error: err.Error(),
			})
			return
		}
		user, err := s.FindByEmail(ctx, userClaim.Email)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "edf9dd05-aad4-4c1e-9795-98bf60ecf57c",
				Error: err.Error(),
			})
			return
		}
		if user == nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "417cff16-0325-45f7-9826-8ab24d2fef29",
			})
			return
		}
		reqCtx.Set(string(UserContextKeyEntity), user)
		reqCtx.Next()
	}
}

func (s *UserService) GetUserFromContext(reqCtx *gin.Context) (*User, bool) {
	user, ok := reqCtx.Get(string(UserContextKeyEntity))
	if !ok {
		return nil, false
	}
	return user.(*User), true
}
