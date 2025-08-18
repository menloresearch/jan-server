package google

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/golang-jwt/jwt/v5"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type GoogleAuthAPI struct {
	oAuth2Config *oauth2.Config
	oidcProvider *oidc.Provider
}

func NewGoogleAuthAPI() *GoogleAuthAPI {
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
	}
}

func (googleAuthAPI *GoogleAuthAPI) RegisterRouter(router *gin.RouterGroup) {
	googleRouter := router.Group("/google")
	googleRouter.POST("/callback", googleAuthAPI.HandleGoogleCallback)
	googleRouter.GET("/login", googleAuthAPI.GetGoogleLogin)
}

type GoogleCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state"`
}

type GoogleCallbackResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Email       string `json:"email"`
	Name        string `json:"name"`
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

	accessTokenExp := time.Now().Add(15 * time.Minute)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   claims.Sub,
		"email": claims.Email,
		"exp":   accessTokenExp.Unix(),
	})
	accessTokenString, err := accessToken.SignedString(environment_variables.EnvironmentVariables.JWT_SECRET)
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "7b50f7ab-f3a1-4a3c-920a-41e387c2bc12",
			Error: err.Error(),
		})
		return
	}

	refreshTokenExp := time.Now().Add(7 * 24 * time.Hour)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": claims.Sub,
		"exp": refreshTokenExp.Unix(),
	})
	refreshTokenString, err := refreshToken.SignedString(environment_variables.EnvironmentVariables.JWT_SECRET)
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "de0f019c-2f92-45c6-9698-66f7906f8cdc",
			Error: err.Error(),
		})
		return
	}
	reqCtx.SetCookie(auth.RefreshTokenKey, refreshTokenString, int(7*24*time.Hour.Seconds()), "/", "", true, true)
	reqCtx.JSON(http.StatusOK, &responses.GeneralResponse[GoogleCallbackResponse]{
		Status: "000000",
		Data: GoogleCallbackResponse{
			accessTokenString,
			int(time.Until(accessTokenExp).Seconds()),
			claims.Email,
			claims.Name,
		},
	})
}

func (googleAuthAPI *GoogleAuthAPI) GetGoogleLogin(reqCtx *gin.Context) {
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
