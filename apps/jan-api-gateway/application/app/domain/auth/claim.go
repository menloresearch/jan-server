package auth

import (
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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, u)
	return token.SignedString(environment_variables.EnvironmentVariables.JWT_SECRET)
}
