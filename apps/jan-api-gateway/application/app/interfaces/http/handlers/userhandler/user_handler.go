package userhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type UserHandler struct {
	userService   *user.UserService
	apikeyService *apikey.ApiKeyService
}

func NewUserHandler(userService *user.UserService, apikeyService *apikey.ApiKeyService) *UserHandler {
	return &UserHandler{
		userService,
		apikeyService,
	}
}

type UserContextKey string

const (
	UserContextKeyEntity UserContextKey = "UserContextKeyEntity"
)

func (handler *UserHandler) RegisteredUserMiddleware() gin.HandlerFunc {
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
		user, err := handler.userService.FindByEmail(ctx, userClaim.Email)
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
		SetUserToContext(reqCtx, user)
		reqCtx.Next()
	}
}

func (handler *UserHandler) RegisteredApiKeyUserMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		token, ok := requests.GetTokenFromBearer(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "e513c329-542f-414a-99bb-10d857b77d2e",
			})
			return
		}
		hashed := handler.apikeyService.HashKey(ctx, token)
		apikeyEntity, err := handler.apikeyService.FindByKeyHash(ctx, hashed)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:          "cd13cdcd-bfc4-4c61-a026-8dd38b3421d7",
				ErrorInstance: err,
			})
			return
		}
		if apikeyEntity == nil || apikeyEntity.ApikeyType == string(apikey.ApikeyTypeAdmin) {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "3309b530-0871-44eb-a280-4a82e9a40716",
			})
			return
		}

		user, err := handler.userService.FindByID(ctx, *apikeyEntity.OwnerID)
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
		SetUserToContext(reqCtx, user)
		reqCtx.Next()
	}
}

func SetUserToContext(reqCtx *gin.Context, u *user.User) {
	reqCtx.Set(string(UserContextKeyEntity), u)
}

func GetUserFromContext(reqCtx *gin.Context) (*user.User, bool) {
	v, ok := reqCtx.Get(string(UserContextKeyEntity))
	if !ok {
		return nil, false
	}
	u, ok := v.(*user.User)
	if !ok {
		return nil, false
	}
	return u, true
}
