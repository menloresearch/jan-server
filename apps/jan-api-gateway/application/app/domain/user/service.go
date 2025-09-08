package user

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

type UserService struct {
	userrepo            UserRepository
	organizationService *organization.OrganizationService
	apiKeyService       *apikey.ApiKeyService
}

func NewService(userrepo UserRepository, organizationService *organization.OrganizationService, apiKeyService *apikey.ApiKeyService) *UserService {
	return &UserService{
		userrepo:            userrepo,
		organizationService: organizationService,
		apiKeyService:       apiKeyService,
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

// RegisteredApiKeyUserMiddleware validates API key and ensures the associated user is registered
func (s *UserService) RegisteredApiKeyUserMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()

		// Extract API key from Bearer token
		apiKeyString, ok := requests.GetTokenFromBearer(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "019928bd-2c6d-74bb-a03a-02a4fbcf292c",
				Error: "API key not provided",
			})
			return
		}

		// Find API key by hash
		apiKeyEntity, err := s.apiKeyService.FindByKey(ctx, apiKeyString)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "019928bd-57f8-7418-9c17-51c21cbf0f17",
				Error: "Invalid API key",
			})
			return
		}

		// Validate API key
		if !apiKeyEntity.IsValid() {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "019928bd-67a8-743d-9843-785936bebc54",
				Error: "API key is disabled or expired",
			})
			return
		}

		// Check if API key has an owner (user)
		if apiKeyEntity.OwnerID == nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "019928bd-78cb-734e-8b2c-d0ba0c43cb73",
				Error: "API key has no associated user",
			})
			return
		}

		// Fetch the user by ID
		userEntity, err := s.FindByID(ctx, *apiKeyEntity.OwnerID)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "019928bd-89dc-735f-8c3d-e1cb1d44dc84",
				Error: "User not found",
			})
			return
		}

		// Store user in context for later use
		reqCtx.Set(string(UserContextKeyEntity), userEntity)
		reqCtx.Next()
	}
}
