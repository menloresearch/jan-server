package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/auth/google"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type AuthRoute struct {
	google      *google.GoogleAuthAPI
	userService *user.UserService
}

func NewAuthRoute(google *google.GoogleAuthAPI, userService *user.UserService) *AuthRoute {
	return &AuthRoute{
		google,
		userService,
	}
}

func (authRoute *AuthRoute) RegisterRouter(router gin.IRouter) {
	authRouter := router.Group("/auth")
	authRouter.GET("/refresh-token", authRoute.RefreshToken)
	authRouter.GET("/me",
		authRoute.userService.AppUserAuthMiddleware(),
		authRoute.userService.RegisteredUserMiddleware(),
		authRoute.GetMe,
	)
	authRouter.POST("/guest-login", authRoute.GuestLogin)
	authRoute.google.RegisterRouter(authRouter)

}

// @Enum(access.token)
type AccessTokenResponseObjectType string

const AccessTokenResponseObjectTypeObject = "access.token"

type AccessTokenResponse struct {
	Object      AccessTokenResponseObjectType `json:"object"`
	AccessToken string                        `json:"access_token"`
	ExpiresIn   int                           `json:"expires_in"`
}

type GetMeResponse struct {
	Object string `json:"object"`
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// @Summary Get user profile
// @Description Retrieves the profile of the authenticated user based on the provided JWT.
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} GetMeResponse "Successfully retrieved user profile"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., missing or invalid JWT)"
// @Router /v1/auth/me [get]
func (authRoute *AuthRoute) GetMe(reqCtx *gin.Context) {
	user, _ := user.GetUserFromContext(reqCtx)
	reqCtx.JSON(http.StatusOK, GetMeResponse{
		Object: "me",
		ID:     user.PublicID,
		Email:  user.Email,
		Name:   user.Name,
	})
}

// @Summary Refresh an access token
// @Description Use a valid refresh token to obtain a new access token. The refresh token is typically sent in a cookie.
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} AccessTokenResponse "Successfully refreshed the access token"
// @Failure 400 {object} responses.ErrorResponse "Bad Request (e.g., invalid refresh token)"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., expired or missing refresh token)"
// @Router /v1/auth/refresh-token [get]
func (authRoute *AuthRoute) RefreshToken(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	refreshTokenString, err := reqCtx.Cookie(auth.RefreshTokenKey)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b95e8123-2590-48ed-bbad-e02d88464513",
			Error: err.Error(),
		})
		return
	}

	token, err := jwt.ParseWithClaims(refreshTokenString, &auth.UserClaim{}, func(token *jwt.Token) (interface{}, error) {
		return environment_variables.EnvironmentVariables.JWT_SECRET, nil
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "7c7b8a48-311c-4beb-a2a1-1c13a87610bb",
			Error: err.Error(),
		})
		return
	}

	if !token.Valid {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "ec5fa88c-78bb-462a-ab90-b046f269d5eb",
		})
		return
	}

	userClaim, ok := token.Claims.(*auth.UserClaim)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "c2019018-b71c-4f13-8ac6-854fbd61c9dd",
		})
		return
	}
	if userClaim.ID == "" {
		user, err := authRoute.userService.FindByEmail(ctx, userClaim.Email)
		if err != nil || user == nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "58174ddb-ef9c-4a3c-a6ad-c880af070518",
			})
			return
		}
		userClaim.ID = user.PublicID
	}

	accessTokenExp := time.Now().Add(15 * time.Minute)
	accessTokenString, err := auth.CreateJwtSignedString(auth.UserClaim{
		Email: userClaim.Email,
		Name:  userClaim.Name,
		ID:    userClaim.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExp),
			Subject:   userClaim.Email,
		},
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "79373f8e-d80e-489c-95ba-9e6099ef7539",
			Error: err.Error(),
		})
		return
	}

	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour)
	refreshTokenString, err = auth.CreateJwtSignedString(auth.UserClaim{
		Email: userClaim.Email,
		Name:  userClaim.Name,
		ID:    userClaim.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshTokenExp),
			Subject:   userClaim.Email,
		},
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
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

	reqCtx.JSON(http.StatusOK, &AccessTokenResponse{
		AccessTokenResponseObjectTypeObject,
		accessTokenString,
		int(time.Until(accessTokenExp).Seconds()),
	})
}

// @Summary Guest Login
// @Description JWT-base Guest Login.
// @Tags Authentication
// @Produce json
// @Success 200 {object} AccessTokenResponse "Successfully refreshed the access token"
// @Failure 400 {object} responses.ErrorResponse "Bad Request (e.g., invalid refresh token)"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized (e.g., expired or missing refresh token)"
// @Router /v1/auth/guest-login [post]
func (authRoute *AuthRoute) GuestLogin(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	userClaim, ok := auth.GetUserClaimFromRefreshToken(reqCtx)
	email := ""
	name := ""
	var id string = ""
	if !ok {
		userService := authRoute.userService
		randomStr := strings.ToUpper(uuid.New().String())
		user, err := userService.RegisterUser(ctx, &user.User{
			Name:    fmt.Sprintf("Jan-%s", randomStr),
			Email:   fmt.Sprintf("Jan-%s@jan.ai", randomStr),
			Enabled: true,
		})
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusOK, responses.ErrorResponse{
				Code: "9576b6ba-fcc6-4bd2-b13a-33d59d6a71f1",
			})
			return
		}
		email = user.Email
		name = user.Name
		id = user.PublicID
	} else {
		email = userClaim.Email
		name = userClaim.Name
		id = userClaim.ID
	}

	accessTokenExp := time.Now().Add(15 * time.Minute)
	accessTokenString, err := auth.CreateJwtSignedString(auth.UserClaim{
		Email: email,
		Name:  name,
		ID:    id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExp),
			Subject:   email,
		},
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "79373f8e-d80e-489c-95ba-9e6099ef7539",
			Error: err.Error(),
		})
		return
	}

	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour)
	refreshTokenString, err := auth.CreateJwtSignedString(auth.UserClaim{
		Email: email,
		Name:  name,
		ID:    id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshTokenExp),
			Subject:   email,
		},
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
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

	reqCtx.JSON(http.StatusOK, &AccessTokenResponse{
		AccessTokenResponseObjectTypeObject,
		accessTokenString,
		int(time.Until(accessTokenExp).Seconds()),
	})
}
