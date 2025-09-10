package user

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type UserService struct {
	userrepo            UserRepository
	organizationService *organization.OrganizationService
	apikeyService       *apikey.ApiKeyService
}

func NewService(userrepo UserRepository, organizationService *organization.OrganizationService, apikeyService *apikey.ApiKeyService) *UserService {
	return &UserService{
		userrepo:            userrepo,
		organizationService: organizationService,
		apikeyService:       apikeyService,
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

type UserContextKey string

const (
	UserContextKeyEntity      UserContextKey = "UserContextKeyEntity"
	UserContextKeyID          UserContextKey = "UserContextKeyID"
	UserContextKeyAdminEntity UserContextKey = "UserContextKeyAdminEntity"
)

// Verify user from public ID
func (s *UserService) RegisteredUserMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		userPublicId, ok := GetUserIDFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "3296ce86-783b-4c05-9fdb-930d3713024e",
			})
			return
		}
		if userPublicId == "" {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "80e1017d-038a-48c1-9de7-c3cdffdddb95",
			})
			return
		}
		user, err := s.FindByPublicID(ctx, userPublicId)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "6272df83-f538-421b-93ba-c2b6f6d39f39",
			})
			return
		}
		if user == nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "b1ef40e7-9db9-477d-bb59-f3783585195d",
			})
			return
		}
		reqCtx.Set(string(UserContextKeyEntity), user)
		reqCtx.Next()
	}
}

func (s *UserService) JWTAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserIDFromJWT(reqCtx)
		if !ok {
			return
		}

		SetUserIDToContext(reqCtx, userId)
		reqCtx.Next()
	}
}

func (s *UserService) getUserIDFromJWT(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	token, err := jwt.ParseWithClaims(tokenString, &auth.UserClaim{}, func(token *jwt.Token) (interface{}, error) {
		return environment_variables.EnvironmentVariables.JWT_SECRET, nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	claims, ok := token.Claims.(*auth.UserClaim)
	if !ok {
		return "", false
	}
	return claims.ID, true
}

func (s *UserService) ApikeyAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userID, ok := s.getUserIDFromApikey(reqCtx)
		if !ok {
			return
		}
		SetUserIDToContext(reqCtx, userID)
		reqCtx.Next()
	}
}

func (s *UserService) getUserIDFromApikey(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	if !strings.HasPrefix(tokenString, apikey.ApikeyPrefix) {
		return "", false
	}
	token, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	ctx := reqCtx.Request.Context()
	hashed := s.apikeyService.HashKey(reqCtx, token)
	apikeyEntity, err := s.apikeyService.FindByKeyHash(ctx, hashed)
	if err != nil {
		return "", false
	}
	if apikeyEntity == nil || apikeyEntity.ApikeyType == string(apikey.ApikeyTypeAdmin) {
		return "", false
	}
	return apikeyEntity.OwnerPublicID, true
}

// Retrieve the user's public ID from the header.
func (s *UserService) AppUserAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserIDFromJWT(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}
		userId, ok = s.getUserIDFromApikey(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}

		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "4026757e-d5a4-4cf7-8914-2c96f011084f",
		})
	}
}

func (s *UserService) AdminUserAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserIDFromJWT(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}
		userId, ok = s.getUserIDFromAdminkey(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userId)
			reqCtx.Next()
			return
		}

		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "4026757e-d5a4-4cf7-8914-2c96f011084f",
		})
	}
}

func (s *UserService) getUserIDFromAdminkey(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	if !strings.HasPrefix(tokenString, apikey.ApikeyPrefix) {
		return "", false
	}
	token, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	ctx := reqCtx.Request.Context()
	hashed := s.apikeyService.HashKey(reqCtx, token)
	apikeyEntity, err := s.apikeyService.FindByKeyHash(ctx, hashed)
	if err != nil {
		return "", false
	}
	if apikeyEntity == nil || apikeyEntity.ApikeyType != string(apikey.ApikeyTypeAdmin) {
		return "", false
	}
	return apikeyEntity.OwnerPublicID, true
}

func GetUserFromContext(reqCtx *gin.Context) (*User, bool) {
	user, ok := reqCtx.Get(string(UserContextKeyEntity))
	if !ok {
		return nil, false
	}
	return user.(*User), true
}

func SetUserToContext(reqCtx *gin.Context, user *User) {
	reqCtx.Set(string(UserContextKeyEntity), user)
}

func GetUserIDFromContext(reqCtx *gin.Context) (string, bool) {
	userId, ok := reqCtx.Get(string(UserContextKeyID))
	if !ok {
		return "", false
	}
	v, ok := userId.(string)
	if !ok {
		return "", false
	}
	return v, true
}

func SetUserIDToContext(reqCtx *gin.Context, userID string) {
	reqCtx.Set(string(UserContextKeyID), userID)
}
