package apikeys

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type ProjectApiKeyRoute struct {
	organizationService *organization.OrganizationService
	projectService      *project.ProjectService
	apikeyService       *apikey.ApiKeyService
	userService         *user.UserService
}

func NewProjectApiKeyRoute(
	organizationService *organization.OrganizationService,
	projectService *project.ProjectService,
	apikeyService *apikey.ApiKeyService,
	userService *user.UserService,
) *ProjectApiKeyRoute {
	return &ProjectApiKeyRoute{
		organizationService,
		projectService,
		apikeyService,
		userService,
	}
}

func (api *ProjectApiKeyRoute) RegisterRouter(router gin.IRouter) {
	apiKeyRouter := router.Group("/api_keys")
	apiKeyRouter.POST("", api.CreateProjectApiKey)
	apiKeyRouter.GET("", api.ListProjectApiKey)
}

// @Summary List new project API key
// @Description List API keys for a specific project.
// @Tags project-api-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param org_public_id path string true "Organization Public ID"
// @Param project_public_id path string true "Project Public ID"
// @Success 200 {object} responses.GeneralResponse[ApiKeyResponse] "API key created successfully"
// @Failure 400 {object} responses.ErrorResponse "Bad request, e.g., invalid payload or missing IDs"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized, e.g., invalid or missing token"
// @Failure 404 {object} responses.ErrorResponse "Not Found, e.g., project or organization not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/organizations/{org_public_id}/projects/{project_public_id}/api_keys [get]
func (api *ProjectApiKeyRoute) ListProjectApiKey(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
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

	if organizationEntity.OwnerID != user.ID {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "6d2d10f9-3bab-4d2d-8076-d573d829e397",
		})
		return
	}

	project, ok := api.projectService.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "e50b3d93-f508-401a-b55e-50ffec69e087",
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
	filter := apikey.ApiKeyFilter{
		OrganizationID: &organizationEntity.ID,
		ProjectID:      &project.ID,
	}

	apikeyEntities, err := api.apikeyService.Find(ctx, filter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "d6a2ac93-49e9-4d42-8487-c384209adce0",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.ListResponse[ApiKeyResponse]{
		Status: responses.ResponseCodeOk,
		Results: functional.Map(apikeyEntities, func(apikeyEntity *apikey.ApiKey) ApiKeyResponse {
			return ApiKeyResponse{
				ID:            apikeyEntity.PublicID,
				PlaintextHint: apikeyEntity.PlaintextHint,
				Description:   apikeyEntity.Description,
				Enabled:       apikeyEntity.Enabled,
				ApikeyType:    apikeyEntity.ApikeyType,
				Permissions:   apikeyEntity.Permissions,
				ExpiresAt:     apikeyEntity.ExpiresAt,
				LastUsedAt:    apikeyEntity.LastUsedAt,
			}
		}),
	})
}

type CreateApiKeyRequest struct {
	Description string     `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

// @Summary Create a new project API key
// @Description Creates a new API key for a specific project.
// @Tags project-api-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param org_public_id path string true "Organization Public ID"
// @Param project_public_id path string true "Project Public ID"
// @Param requestBody body CreateApiKeyRequest true "Request body for creating an API key"
// @Success 200 {object} responses.GeneralResponse[ApiKeyResponse] "API key created successfully"
// @Failure 400 {object} responses.ErrorResponse "Bad request, e.g., invalid payload or missing IDs"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized, e.g., invalid or missing token"
// @Failure 404 {object} responses.ErrorResponse "Not Found, e.g., project or organization not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/organizations/{org_public_id}/projects/{project_public_id}/api_keys [post]
func (api *ProjectApiKeyRoute) CreateProjectApiKey(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	var req CreateApiKeyRequest

	// Bind the JSON payload to the struct
	if err := reqCtx.BindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "fa1d1cea-7229-446f-9de8-fa254fe6733c",
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

	project, ok := api.projectService.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "e50b3d93-f508-401a-b55e-50ffec69e087",
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
		PlaintextHint:  fmt.Sprintf("sk-..%s", key[len(key)-3:]),
		Description:    req.Description,
		Enabled:        true,
		ApikeyType:     string(apikey.ApikeyTypeProject),
		OwnerID:        &user.ID,
		ProjectID:      &project.ID,
		OrganizationID: &organizationEntity.ID,
		Permissions:    "{}",
		ExpiresAt:      req.ExpiresAt,
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
