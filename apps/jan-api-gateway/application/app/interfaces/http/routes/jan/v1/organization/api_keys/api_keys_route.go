package apikeys

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type OrganizationApiKeyRoute struct {
	organizationService *organization.OrganizationService
	apikeyService       *apikey.ApiKeyService
	userService         *user.UserService
}

func NewOrganizationApiKeyRouteRoute(
	organizationService *organization.OrganizationService,
	apikeyService *apikey.ApiKeyService,
	userService *user.UserService,
) *OrganizationApiKeyRoute {
	return &OrganizationApiKeyRoute{
		organizationService,
		apikeyService,
		userService,
	}
}

func (api *OrganizationApiKeyRoute) RegisterRouter(router gin.IRouter) {
	apiKeyRouter := router.Group("/api_keys")
	apiKeyRouter.GET("", api.ListApiKeys)
	apiKeyRouter.POST("", api.CreateAdminKey)
}

type CreateAdminKeyRequest struct {
	Description string `json:"description"`
}

// @Summary Create a new organization-level admin key
// @Description Creates a new API key with administrative permissions for a specific organization.
// @Tags Jan, Jan-Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param org_public_id path string true "Organization Public ID"
// @Param requestBody body CreateAdminKeyRequest true "Request body for creating an admin key"
// @Success 200 {object} responses.GeneralResponse[ApiKeyResponse] "Admin API key created successfully"
// @Failure 400 {object} responses.ErrorResponse "Bad request, e.g., invalid payload or missing IDs"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized, e.g., invalid or missing token"
// @Failure 404 {object} responses.ErrorResponse "Not Found, e.g., organization not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/organizations/{org_public_id}/api_keys [post]
func (api *OrganizationApiKeyRoute) CreateAdminKey(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	var req CreateAdminKeyRequest

	// Bind the JSON payload to the struct
	if err := reqCtx.BindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "86820bbd-af19-4e4a-9958-b9ee72701cd1",
			Error: err.Error(),
		})
		return
	}

	user, ok := api.userService.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "a3be84ac-132e-4af1-a4ca-9f70aa49fd70",
		})
		return
	}

	organizationEntity, ok := api.organizationService.GetOrganizationFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "3d1ab99a-e2d3-4d13-9130-56eb662e5f92",
		})
		return
	}
	// TODO: Change the verification to users with organization read permission.
	if organizationEntity.OwnerID != user.ID {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "6d2d10f9-3bab-4d2d-8076-d573d829e397",
		})
		return
	}

	key, hash, err := api.apikeyService.GenerateKeyAndHash(ctx, apikey.ApikeyTypeProject)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "6d2d10f9-3bab-4d2d-8076-d573d829e397",
			Error: err.Error(),
		})
		return
	}

	apikeyEntity, err := api.apikeyService.CreateApiKey(ctx, &apikey.ApiKey{
		KeyHash:        hash,
		PlaintextHint:  fmt.Sprintf("sk-..%s", key[len(key)-4:]),
		Description:    req.Description,
		Enabled:        true,
		ApikeyType:     string(apikey.ApikeyTypeAdmin),
		OwnerID:        &user.ID,
		OrganizationID: &organizationEntity.ID,
		Permissions:    "{}",
	})

	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "d7bb0e84-72ba-41bd-8e71-8aec92ec8abe",
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[ApiKeyResponse]{
		Status: responses.ResponseCodeOk,
		Result: ApiKeyResponse{
			ID:            apikeyEntity.PublicID,
			Key:           key,
			PlaintextHint: apikeyEntity.PlaintextHint,
			Description:   apikeyEntity.Description,
			Enabled:       apikeyEntity.Enabled,
			ApikeyType:    apikeyEntity.ApikeyType,
			Permissions:   apikeyEntity.Permissions,
			ExpiresAt:     apikeyEntity.ExpiresAt,
			LastUsedAt:    apikeyEntity.LastUsedAt,
		},
	})
}

// @Summary List API keys for a specific organization
// @Description Retrieves a list of all API keys associated with an organization.
// @Tags Jan, Jan-Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param org_public_id path string true "Organization Public ID"
// @Param offset query int false "offset for pagination" default(0)
// @Param limit query int false "Number of items per page" default(10)
// @Success 200 {object} responses.ListResponse[ApiKeyResponse] "List of API keys retrieved successfully"
// @Failure 400 {object} responses.ErrorResponse "Bad request, e.g., invalid pagination parameters"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized, e.g., invalid or missing token"
// @Failure 404 {object} responses.ErrorResponse "Not Found, e.g., organization not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/organizations/{org_public_id}/api_keys [get]
func (api *OrganizationApiKeyRoute) ListApiKeys(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, ok := api.userService.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "ada5a8f9-d5e1-4761-9af1-a176473ff7eb",
		})
		return
	}

	organizationEntity, ok := api.organizationService.GetOrganizationFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "7d4c0c15-9ecb-4e9a-a7e7-e949b5acf723",
		})
		return
	}

	// TODO: Change the verification to users with organization read permission.
	if organizationEntity.OwnerID != user.ID {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "6d2d10f9-3bab-4d2d-8076-d573d829e397",
		})
		return
	}

	pagination, err := query.GetPaginationFromQuery(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "1f11f211-7f74-43c9-b7c3-df31fcd2cf4d",
		})
		return
	}

	apikeyService := api.apikeyService
	apikeys, err := apikeyService.Find(ctx, apikey.ApiKeyFilter{
		OrganizationID: &organizationEntity.ID,
	}, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "e598e0e5-92f8-46d3-b8b5-5c87d04ffef0",
		})
		return
	}
	reqCtx.JSON(http.StatusOK, responses.ListResponse[ApiKeyResponse]{
		Status:  responses.ResponseCodeOk,
		Results: functional.Map(apikeys, convertApiKeyDomainToResponse),
		Total:   int64(len(apikeys)),
	})
}

type ApiKeyResponse struct {
	ID            string     `json:"id"`
	Key           string     `json:"key,omitempty"`
	PlaintextHint string     `json:"plaintextHint"`
	Description   string     `json:"description"`
	Enabled       bool       `json:"enabled"`
	ApikeyType    string     `json:"apikeyType"`
	Permissions   string     `json:"permissions"`
	ExpiresAt     *time.Time `json:"expiresAt"`
	LastUsedAt    *time.Time `json:"last_usedAt"`
}

func convertApiKeyDomainToResponse(entity *apikey.ApiKey) ApiKeyResponse {
	return ApiKeyResponse{
		ID:            entity.PublicID,
		PlaintextHint: entity.PlaintextHint,
		Description:   entity.Description,
		Enabled:       entity.Enabled,
		ApikeyType:    entity.ApikeyType,
		Permissions:   entity.Permissions,
		ExpiresAt:     entity.ExpiresAt,
		LastUsedAt:    entity.LastUsedAt,
	}
}
