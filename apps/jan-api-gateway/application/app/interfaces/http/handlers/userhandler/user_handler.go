package userhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type UserHandler struct {
	userService *user.UserService
}

func NewUserHandler(userService *user.UserService) *UserHandler {
	return &UserHandler{
		userService,
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
