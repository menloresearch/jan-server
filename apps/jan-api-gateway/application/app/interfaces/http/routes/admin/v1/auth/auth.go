package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1/auth/google"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type AuthRoute struct {
	google *google.GoogleAuthAPI
}

func NewAuthRoute(google *google.GoogleAuthAPI) *AuthRoute {
	return &AuthRoute{
		google,
	}
}

func (authRoute *AuthRoute) RegisterRouter(router gin.IRouter) {
	authRouter := router.Group("/auth")
	authRoute.google.RegisterRouter(authRouter)
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (authRoute *AuthRoute) RefreshToken(reqCtx *gin.Context) {
	refreshTokenString, err := reqCtx.Cookie(auth.RefreshTokenKey)
	if err != nil {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b95e8123-2590-48ed-bbad-e02d88464513",
			Error: err.Error(),
		})
		return
	}

	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		return environment_variables.EnvironmentVariables.JWT_SECRET, nil
	})
	if err != nil {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "7c7b8a48-311c-4beb-a2a1-1c13a87610bb",
			Error: err.Error(),
		})
		return
	}

	if !token.Valid {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "ec5fa88c-78bb-462a-ab90-b046f269d5eb",
		})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "c2019018-b71c-4f13-8ac6-854fbd61c9dd",
		})
		return
	}

	accessTokenExp := time.Now().Add(15 * time.Minute)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   claims["sub"],
		"email": claims["email"],
		"exp":   accessTokenExp.Unix(),
	})

	accessTokenString, err := accessToken.SignedString(environment_variables.EnvironmentVariables.JWT_SECRET)
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "79373f8e-d80e-489c-95ba-9e6099ef7539",
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, &responses.GeneralResponse[RefreshTokenResponse]{
		Status: "000000",
		Data: RefreshTokenResponse{
			accessTokenString,
			int(time.Until(accessTokenExp).Seconds()),
		},
	})
}
