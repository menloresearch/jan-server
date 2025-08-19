package google

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type GoogleAuthAPI struct {
	oAuth2Config *oauth2.Config
	oidcProvider *oidc.Provider
	userService  *user.UserService
}

func NewGoogleAuthAPI(userService *user.UserService) *GoogleAuthAPI {
	oauth2Config := &oauth2.Config{
		ClientID:     environment_variables.EnvironmentVariables.OAUTH2_GOOGLE_CLIENT_ID,
		ClientSecret: environment_variables.EnvironmentVariables.OAUTH2_GOOGLE_CLIENT_SECRET,
		RedirectURL:  environment_variables.EnvironmentVariables.OAUTH2_GOOGLE_REDIRECT_URL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint:     google.Endpoint,
	}

	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		panic(err)
	}
	return &GoogleAuthAPI{
		oauth2Config,
		provider,
		userService,
	}
}

func (googleAuthAPI *GoogleAuthAPI) RegisterRouter(router *gin.RouterGroup) {
	googleRouter := router.Group("/google")
	googleRouter.POST("/callback", googleAuthAPI.HandleGoogleCallback)
	googleRouter.GET("/login", googleAuthAPI.GetGoogleLoginUrl)
}

type GoogleCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state"`
}

type GoogleCallbackResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (googleAuthAPI *GoogleAuthAPI) HandleGoogleCallback(reqCtx *gin.Context) {
	var req GoogleCallbackRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "f1ca221e-cc6e-4e31-92b0-7c59dd966536",
		})
		return
	}

	storedState, err := reqCtx.Cookie(auth.OAuthStateKey)
	if storedState != req.State {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "f845d325-fe49-4487-978b-543090f2ec42",
		})
		return
	}
	if err != nil {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "2a17e34c-95bd-4d03-95ee-01fd6172348d",
			Error: err.Error(),
		})
		return
	}

	token, err := googleAuthAPI.oAuth2Config.Exchange(reqCtx, req.Code)
	if err != nil {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "f9e2d2b5-45b5-4697-bb04-548b4290fdde",
		})
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "69137efa-bf46-456f-ab4c-bda9fa38aff0",
		})
		return
	}
	verifier := googleAuthAPI.oidcProvider.Verifier(&oidc.Config{ClientID: googleAuthAPI.oAuth2Config.ClientID})
	idToken, err := verifier.Verify(reqCtx, rawIDToken)
	if err != nil {
		reqCtx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "8ea31139-211e-4282-82de-9664814e6f46",
			Error: err.Error(),
		})
		return
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Sub   string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "f2ea83a6-36f6-4a87-ae50-e934f984f1c9",
			Error: err.Error(),
		})
		return
	}

	userService := googleAuthAPI.userService
	exists, err := userService.FindByEmail(reqCtx, claims.Email)
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "ad6e260d-b5ad-447b-8ab0-7e161c932b6a",
			Error: err.Error(),
		})
		return
	}
	if exists == nil {
		exists, err = userService.CreateUser(reqCtx, &user.User{
			Name:    claims.Name,
			Email:   claims.Email,
			Enabled: true,
		})
		if err != nil {
			reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "45f08e6d-4b0c-4718-9bf3-5974a14d5f25",
				Error: err.Error(),
			})
			return
		}
	}

	accessTokenExp := time.Now().Add(15 * time.Minute)
	accessTokenString, err := auth.CreateJwtSignedString(auth.UserClaim{
		Email: exists.Email,
		Name:  exists.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExp),
			Subject:   exists.Email,
		},
	})

	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "7b50f7ab-f3a1-4a3c-920a-41e387c2bc12",
			Error: err.Error(),
		})
		return
	}
	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour)
	refreshTokenString, err := auth.CreateJwtSignedString(auth.UserClaim{
		Email: exists.Email,
		Name:  exists.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshTokenExp),
			Subject:   exists.Email,
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

	reqCtx.JSON(http.StatusOK, &responses.GeneralResponse[GoogleCallbackResponse]{
		Status: responses.ResponseCodeOk,
		Data: GoogleCallbackResponse{
			accessTokenString,
			int(time.Until(accessTokenExp).Seconds()),
		},
	})
}

func (googleAuthAPI *GoogleAuthAPI) GetGoogleLoginUrl(reqCtx *gin.Context) {
	state, err := generateState()
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "e30d6d79-8126-4e76-bcff-49bbfaee3b06",
			Error: err.Error(),
		})
		return
	}

	// 5 minutes csrf token
	reqCtx.SetCookie(auth.OAuthStateKey, state, 300, "/", "", true, true)
	authURL := googleAuthAPI.oAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	reqCtx.Redirect(http.StatusTemporaryRedirect, authURL)
}
