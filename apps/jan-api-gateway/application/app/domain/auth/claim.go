package auth

const RefreshTokenKey = "jan_refresh_token"
const OAuthStateKey = "jan_oauth_state"

type UserClaims struct {
	Email string
	Name  string
	Sub   string
}
