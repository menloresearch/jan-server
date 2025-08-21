package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth/google"
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
	authRouter.GET("/refresh-token", authRoute.RefreshToken)
	authRouter.GET("/me", middleware.AuthMiddleware(), authRoute.GetMe)
	authRoute.google.RegisterRouter(authRouter)

}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type GetMeResponse struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// @Summary Get user profile
// @Description Retrieves the profile of the authenticated user based on the provided JWT.
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} responses.GeneralResponse[GetMeResponse] "Successfully retrieved user profile"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., missing or invalid JWT)"
// @Router /jan/v1/auth/me [get]
func (authRoute *AuthRoute) GetMe(reqCtx *gin.Context) {
	userClaim, ok := reqCtx.Get(auth.ContextUserClaim)
	if !ok {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "fbc49daf-2f73-4778-9362-5680da391190",
		})
		return
	}
	u, ok := userClaim.(*auth.UserClaim)
	if !ok {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "e8a957c3-e107-4244-8625-3f3a1d29ce5c",
		})
		return
	}
	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[GetMeResponse]{
		Status: responses.ResponseCodeOk,
		Result: GetMeResponse{
			Email: u.Email,
			Name:  u.Name,
		},
	})
}

// @Summary Refresh an access token
// @Description Use a valid refresh token to obtain a new access token. The refresh token is typically sent in a cookie.
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} responses.GeneralResponse[RefreshTokenResponse] "Successfully refreshed the access token"
// @Failure 400 {object} responses.ErrorResponse "Bad Request (e.g., invalid refresh token)"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., expired or missing refresh token)"
// @Router /jan/v1/auth/refresh-token [get]
func (authRoute *AuthRoute) RefreshToken(reqCtx *gin.Context) {
	refreshTokenString, err := reqCtx.Cookie(auth.RefreshTokenKey)
	if err != nil {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b95e8123-2590-48ed-bbad-e02d88464513",
			Error: err.Error(),
		})
		return
	}

	token, err := jwt.ParseWithClaims(refreshTokenString, &auth.UserClaim{}, func(token *jwt.Token) (interface{}, error) {
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

	userClaim, ok := token.Claims.(*auth.UserClaim)
	if !ok {
		reqCtx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "c2019018-b71c-4f13-8ac6-854fbd61c9dd",
		})
		return
	}

	accessTokenExp := time.Now().Add(15 * time.Minute)
	accessTokenString, err := auth.CreateJwtSignedString(auth.UserClaim{
		Email: userClaim.Email,
		Name:  userClaim.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExp),
			Subject:   userClaim.Email,
		},
	})
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "79373f8e-d80e-489c-95ba-9e6099ef7539",
			Error: err.Error(),
		})
		return
	}

	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour)
	refreshTokenString, err = auth.CreateJwtSignedString(auth.UserClaim{
		Email: userClaim.Email,
		Name:  userClaim.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshTokenExp),
			Subject:   userClaim.Email,
		},
	})
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "0e596742-64bb-4904-8429-4c09ce8434b9",
			Error: err.Error(),
		})
		return
	}

	http.SetCookie(reqCtx.Writer, &http.Cookie{
		Name:     auth.RefreshTokenKey,
		Value:    refreshTokenString,
		Expires:  refreshTokenExp,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	reqCtx.JSON(http.StatusOK, &responses.GeneralResponse[RefreshTokenResponse]{
		Status: responses.ResponseCodeOk,
		Result: RefreshTokenResponse{
			accessTokenString,
			int(time.Until(accessTokenExp).Seconds()),
		},
	})
}
