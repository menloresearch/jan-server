package auth

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"menlo.ai/jan-api-gateway/config/environment_variables"
)

const RefreshTokenKey = "jan_refresh_token"
const OAuthStateKey = "jan_oauth_state"
const ContextUserClaim = "context_user_claim"

type UserClaim struct {
	Email string
	Name  string
	jwt.RegisteredClaims
}

func CreateJwtSignedString(u UserClaim) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, u)
	return token.SignedString(environment_variables.EnvironmentVariables.JWT_SECRET)
}

func GetUserClaimFromRequestContext(reqCtx *gin.Context) (*UserClaim, error) {
	userClaim, ok := reqCtx.Get(ContextUserClaim)
	if !ok {
		return nil, fmt.Errorf("userclaim not found in context")
	}
	u, ok := userClaim.(*UserClaim)
	if !ok {
		return nil, fmt.Errorf("invalid user claim in context: expected *auth.UserClaim, got %T", userClaim)
	}
	return u, nil
}
