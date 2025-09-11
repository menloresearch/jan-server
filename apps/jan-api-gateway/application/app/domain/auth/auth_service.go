package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"menlo.ai/jan-api-gateway/app/domain/apikey"

	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type AuthService struct {
	userService         *user.UserService
	apiKeyService       *apikey.ApiKeyService
	organizationService *organization.OrganizationService
}

func NewAuthService(
	userService *user.UserService,
	apiKeyService *apikey.ApiKeyService,
	organizationService *organization.OrganizationService,
) *AuthService {
	return &AuthService{
		userService,
		apiKeyService,
		organizationService,
	}
}

type UserContextKey string

const (
	UserContextKeyEntity UserContextKey = "UserContextKeyEntity"
	UserContextKeyID     UserContextKey = "UserContextKeyID"
)

func (s *AuthService) JWTAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserPublicIDFromJWT(reqCtx)
		if !ok {
			return
		}

		SetUserIDToContext(reqCtx, userId)
		reqCtx.Next()
	}
}

// Retrieve the user's public ID from the header.
func (s *AuthService) AppUserAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserPublicIDFromJWT(reqCtx)
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

func (s *AuthService) AdminUserAuthMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userId, ok := s.getUserPublicIDFromJWT(reqCtx)
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

// Verify user from public ID
func (s *AuthService) RegisteredUserMiddleware() gin.HandlerFunc {
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
		user, err := s.userService.FindByPublicID(ctx, userPublicId)
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

func (s *AuthService) getUserPublicIDFromJWT(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	token, err := jwt.ParseWithClaims(tokenString, &UserClaim{}, func(token *jwt.Token) (interface{}, error) {
		return environment_variables.EnvironmentVariables.JWT_SECRET, nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	claims, ok := token.Claims.(*UserClaim)
	if !ok {
		return "", false
	}
	return claims.ID, true
}

func (s *AuthService) getUserIDFromApikey(reqCtx *gin.Context) (string, bool) {
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
	hashed := s.apiKeyService.HashKey(reqCtx, token)
	apikeyEntity, err := s.apiKeyService.FindByKeyHash(ctx, hashed)
	if err != nil {
		return "", false
	}
	if apikeyEntity == nil || apikeyEntity.ApikeyType == string(apikey.ApikeyTypeAdmin) {
		return "", false
	}
	return apikeyEntity.OwnerPublicID, true
}

func (s *AuthService) getUserIDFromAdminkey(reqCtx *gin.Context) (string, bool) {
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
	hashed := s.apiKeyService.HashKey(reqCtx, token)
	apikeyEntity, err := s.apiKeyService.FindByKeyHash(ctx, hashed)
	if err != nil {
		return "", false
	}
	if apikeyEntity == nil || apikeyEntity.ApikeyType != string(apikey.ApikeyTypeAdmin) {
		return "", false
	}
	return apikeyEntity.OwnerPublicID, true
}

func GetUserFromContext(reqCtx *gin.Context) (*user.User, bool) {
	v, ok := reqCtx.Get(string(UserContextKeyEntity))
	if !ok {
		return nil, false
	}
	return v.(*user.User), true
}

func SetUserToContext(reqCtx *gin.Context, user *user.User) {
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

func SetUserIDToContext(reqCtx *gin.Context, v string) {
	reqCtx.Set(string(UserContextKeyID), v)
}
