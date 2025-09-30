package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

type OrganizationProviderRoute struct {
	authService     *auth.AuthService
	providerService *modelprovider.ModelProviderService
	cache           *cache.RedisCacheService
}

type ProjectProviderRoute struct {
	authService     *auth.AuthService
	providerService *modelprovider.ModelProviderService
	projectService  *project.ProjectService
	cache           *cache.RedisCacheService
}

type ProviderRequest struct {
	Name     string         `json:"name" binding:"required"`
	Vendor   string         `json:"vendor" binding:"required"`
	BaseURL  string         `json:"base_url"`
	APIKey   string         `json:"api_key" binding:"required"`
	Metadata map[string]any `json:"metadata"`
	Active   *bool          `json:"active"`
}

type UpdateProviderRequest struct {
	Name     *string        `json:"name"`
	BaseURL  *string        `json:"base_url"`
	APIKey   *string        `json:"api_key"`
	Metadata map[string]any `json:"metadata"`
	Active   *bool          `json:"active"`
}

type ProviderResponse struct {
	ProviderID     string `json:"provider_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Vendor         string `json:"vendor"`
	APIKeyHint     string `json:"api_key_hint"`
	Active         bool   `json:"active"`
	Scope          string `json:"scope"`
	OrganizationID uint   `json:"organization_id"`
	ProjectID      *uint  `json:"project_id,omitempty"`
}

func NewOrganizationProviderRoute(authService *auth.AuthService, providerService *modelprovider.ModelProviderService, cacheService *cache.RedisCacheService) *OrganizationProviderRoute {
	return &OrganizationProviderRoute{
		authService:     authService,
		providerService: providerService,
		cache:           cacheService,
	}
}

func NewProjectProviderRoute(authService *auth.AuthService, providerService *modelprovider.ModelProviderService, projectService *project.ProjectService, cacheService *cache.RedisCacheService) *ProjectProviderRoute {
	return &ProjectProviderRoute{
		authService:     authService,
		providerService: providerService,
		projectService:  projectService,
		cache:           cacheService,
	}
}

func (route *OrganizationProviderRoute) RegisterRouter(router gin.IRouter) {
	providersRouter := router.Group("/providers",
		route.authService.AdminUserAuthMiddleware(),
		route.authService.RegisteredUserMiddleware(),
	)

	permissionAll := route.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleAll)
	permissionOwnerOnly := route.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleOwnerOnly)

	providersRouter.GET("", permissionAll, route.ListOrganizationProviders)
	providersRouter.POST("", permissionOwnerOnly, route.CreateOrganizationProvider)
	providersRouter.GET(":provider_id", permissionAll, route.GetOrganizationProvider)
	providersRouter.PATCH(":provider_id", permissionOwnerOnly, route.UpdateOrganizationProvider)
	providersRouter.DELETE(":provider_id", permissionOwnerOnly, route.DeleteOrganizationProvider)
}

func (route *ProjectProviderRoute) RegisterRouter(router gin.IRouter) {
	providersRouter := router.Group("/providers",
		route.authService.AdminUserAuthMiddleware(),
		route.authService.RegisteredUserMiddleware(),
	)
	permissionAll := route.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleAll)
	providersRouter.GET("", permissionAll, route.ListProjectProviders)

	providersRouter.POST("", permissionAll, route.CreateProjectProvider)
	providersRouter.GET(":provider_id", permissionAll, route.GetProjectProvider)
	providersRouter.PATCH(":provider_id", permissionAll, route.UpdateProjectProvider)
	providersRouter.DELETE(":provider_id", permissionAll, route.DeleteProjectProvider)
}

// CreateOrganizationProvider godoc
// @Summary Create organization provider
// @Description Registers a new organization-scoped inference provider. API keys are stored encrypted.
// @Tags Providers API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ProviderRequest true "Provider configuration"
// @Success 201 {object} ProviderResponse "Provider created"
// @Failure 400 {object} responses.ErrorResponse "Invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Insufficient permissions"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/providers [post]
func (route *OrganizationProviderRoute) CreateOrganizationProvider(reqCtx *gin.Context) {
	var req ProviderRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "1a2f5f54-ac1a-4f7b-b03a-5f0c5368e35d",
			Error: "invalid payload",
		})
		return
	}

	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	vendor, err := parseVendor(req.Vendor)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "59f7d2c2-2f22-40ef-aa1a-05f36b936a78",
			Error: err.Error(),
		})
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	ctx := reqCtx.Request.Context()
	provider, err := route.providerService.RegisterOrganizationProvider(ctx, modelprovider.CreateOrganizationProviderInput{
		OrganizationID: orgEntity.ID,
		Name:           req.Name,
		Vendor:         vendor,
		BaseURL:        req.BaseURL,
		APIKey:         req.APIKey,
		Metadata:       req.Metadata,
		Active:         active,
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "2b6aa7d4-03da-4a49-9b3b-935fe2b1bde2",
			Error: err.Error(),
		})
		return
	}

	route.invalidateOrganizationModelsCache(ctx, orgEntity.ID)
	unlinkProviderModelsCache(ctx, route.cache, provider.PublicID, "organization provider")

	reqCtx.JSON(http.StatusCreated, ProviderResponseFromModel(provider))
}

// ListOrganizationProviders godoc
// @Summary List organization providers
// @Description Returns organization-level providers visible to the authenticated user.
// @Tags Providers API
// @Security BearerAuth
// @Produce json
// @Success 200 {object} responses.GeneralResponse[[]ProviderResponse] "Providers retrieved"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/providers [get]
func (route *OrganizationProviderRoute) ListOrganizationProviders(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	providers, err := route.providerService.List(reqCtx.Request.Context(), modelprovider.ProviderFilter{
		OrganizationID: &orgEntity.ID,
	}, nil)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "3ec1e43d-77c2-4ac7-9c90-fbb82ff1c4ad",
		})
		return
	}

	payload := make([]ProviderResponse, 0, len(providers))
	for _, provider := range providers {
		if provider.ProjectID != nil {
			continue
		}
		payload = append(payload, ProviderResponseFromModel(provider))
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[[]ProviderResponse]{
		Status: responses.ResponseCodeOk,
		Result: payload,
	})
}

// CreateProjectProvider godoc
// @Summary Create project provider
// @Description Registers a new project-scoped provider under the specified project.
// @Tags Providers API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param proj_public_id path string true "Project public ID"
// @Param request body ProviderRequest true "Provider configuration"
// @Success 201 {object} ProviderResponse "Provider created"
// @Failure 400 {object} responses.ErrorResponse "Invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Insufficient permissions"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/projects/{proj_public_id}/providers [post]
func (route *ProjectProviderRoute) CreateProjectProvider(reqCtx *gin.Context) {
	var req ProviderRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "4d1f6b7a-0d20-4496-9529-75abda3d72b7",
			Error: "invalid payload",
		})
		return
	}

	ctx := reqCtx.Request.Context()
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	projectEntity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "f71c6385-8e35-4c9a-bd56-98c0a4f0d16a",
		})
		return
	}
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return
	}

	if !route.authorizeProjectOwner(reqCtx, projectEntity.ID, user.ID) {
		reqCtx.AbortWithStatusJSON(http.StatusForbidden, responses.ErrorResponse{
			Code:  "8f6e6d5c-ec0e-4b8c-9df4-6f696cb68f6a",
			Error: "insufficient permissions",
		})
		return
	}

	vendor, err := parseVendor(req.Vendor)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "90f2dbbe-3910-4ff8-8d74-0e6cb1877f54",
			Error: err.Error(),
		})
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	provider, err := route.providerService.RegisterOrganizationProvider(ctx, modelprovider.CreateOrganizationProviderInput{
		OrganizationID: orgEntity.ID,
		ProjectID:      &projectEntity.ID,
		Name:           req.Name,
		Vendor:         vendor,
		BaseURL:        req.BaseURL,
		APIKey:         req.APIKey,
		Metadata:       req.Metadata,
		Active:         active,
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "cc4d9f5a-9c47-4a2e-9dd0-77d0b6ba0eab",
			Error: err.Error(),
		})
		return
	}

	route.invalidateProjectModelsCache(ctx, projectEntity.ID)
	route.invalidateOrganizationModelsCache(ctx, orgEntity.ID)
	unlinkProviderModelsCache(ctx, route.cache, provider.PublicID, "project provider")

	reqCtx.JSON(http.StatusCreated, ProviderResponseFromModel(provider))
}

// ListProjectProviders godoc
// @Summary List project providers
// @Description Returns all providers scoped to the specified project.
// @Tags Providers API
// @Security BearerAuth
// @Produce json
// @Param proj_public_id path string true "Project public ID"
// @Success 200 {object} responses.GeneralResponse[[]ProviderResponse] "Providers retrieved"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/projects/{proj_public_id}/providers [get]
func (route *ProjectProviderRoute) ListProjectProviders(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	projectEntity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "253f4841-07ad-4d71-9d76-5af6d1dcb1f4",
		})
		return
	}

	ctx := reqCtx.Request.Context()
	projectIDs := []uint{projectEntity.ID}
	providers, err := route.providerService.List(ctx, modelprovider.ProviderFilter{
		OrganizationID: &orgEntity.ID,
		ProjectIDs:     &projectIDs,
	}, nil)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "2da2cf5b-0f09-4cb5-86a6-0a7d367ce9f9",
		})
		return
	}

	payload := make([]ProviderResponse, 0, len(providers))
	for _, provider := range providers {
		payload = append(payload, ProviderResponseFromModel(provider))
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[[]ProviderResponse]{
		Status: responses.ResponseCodeOk,
		Result: payload,
	})
}

// GetOrganizationProvider godoc
// @Summary Get organization provider
// @Description Fetches details for a single organization-scoped provider.
// @Tags Providers API
// @Security BearerAuth
// @Produce json
// @Param provider_id path string true "Provider public ID"
// @Success 200 {object} responses.GeneralResponse[ProviderResponse] "Provider details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/providers/{provider_id} [get]
func (route *OrganizationProviderRoute) GetOrganizationProvider(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	provider, err := route.fetchOrganizationProvider(reqCtx.Request.Context(), orgEntity.ID, reqCtx.Param("provider_id"))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "5c6c9f1f-0d6a-40a4-9fa6-823f19f8391d",
			Error: "failed to load provider",
		})
		return
	}
	if provider == nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "c6cfcb04-39b9-45da-8c48-8a6196b36c05",
			Error: "provider not found",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[ProviderResponse]{
		Status: responses.ResponseCodeOk,
		Result: ProviderResponseFromModel(provider),
	})
}

// UpdateOrganizationProvider godoc
// @Summary Update organization provider
// @Description Updates metadata, activation state, or credentials for an organization-scoped provider.
// @Tags Providers API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param provider_id path string true "Provider public ID"
// @Param request body UpdateProviderRequest true "Fields to update"
// @Success 200 {object} responses.GeneralResponse[ProviderResponse] "Provider updated"
// @Failure 400 {object} responses.ErrorResponse "Invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Insufficient permissions"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/providers/{provider_id} [patch]
func (route *OrganizationProviderRoute) UpdateOrganizationProvider(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	providerID := strings.TrimSpace(reqCtx.Param("provider_id"))
	if providerID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b5a59f38-43bb-40af-a5c0-dc94c3b3f2cb",
			Error: "provider_id is required",
		})
		return
	}

	var req UpdateProviderRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b1e8f971-dc5e-4b24-86b0-9b6ea6e2d14f",
			Error: "invalid payload",
		})
		return
	}

	provider, err := route.fetchOrganizationProvider(reqCtx.Request.Context(), orgEntity.ID, providerID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "bf9e6554-9741-4fbf-a5f4-0a6a6838df5d",
			Error: "failed to load provider",
		})
		return
	}
	if provider == nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "54c0b807-eb78-4e9f-8bc4-5f7c89b0bd49",
			Error: "provider not found",
		})
		return
	}

	input := modelprovider.UpdateOrganizationProviderInput{PublicID: provider.PublicID}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		input.Name = &name
	}
	if req.BaseURL != nil {
		baseURL := strings.TrimSpace(*req.BaseURL)
		input.BaseURL = &baseURL
	}
	if req.APIKey != nil {
		apiKey := strings.TrimSpace(*req.APIKey)
		input.APIKey = &apiKey
	}
	if req.Metadata != nil {
		input.Metadata = req.Metadata
	}
	if req.Active != nil {
		input.Active = req.Active
	}

	updated, err := route.providerService.UpdateOrganizationProvider(reqCtx.Request.Context(), input)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "8a19c120-e2f6-4835-8bd7-96b6aae7523d",
			Error: "failed to update provider",
		})
		return
	}

	route.invalidateOrganizationModelsCache(reqCtx.Request.Context(), orgEntity.ID)
	unlinkProviderModelsCache(reqCtx.Request.Context(), route.cache, updated.PublicID, "organization provider")

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[ProviderResponse]{
		Status: responses.ResponseCodeOk,
		Result: ProviderResponseFromModel(updated),
	})
}

// DeleteOrganizationProvider godoc
// @Summary Delete organization provider
// @Description Removes an organization-scoped provider and invalidates cached models.
// @Tags Providers API
// @Security BearerAuth
// @Param provider_id path string true "Provider public ID"
// @Success 204 {string} string "Provider deleted"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Insufficient permissions"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/providers/{provider_id} [delete]
func (route *OrganizationProviderRoute) DeleteOrganizationProvider(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}

	provider, err := route.fetchOrganizationProvider(reqCtx.Request.Context(), orgEntity.ID, reqCtx.Param("provider_id"))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "e02bb4a5-2cbb-429c-9c5b-2fa03236f1e8",
			Error: "failed to load provider",
		})
		return
	}
	if provider == nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "20cb8c4b-81e1-4d1d-8f5c-4b68cf81c5a9",
			Error: "provider not found",
		})
		return
	}

	if err := route.providerService.DeleteByPublicID(reqCtx.Request.Context(), provider.PublicID); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "53f17dee-6754-4d8e-8471-92cbf6b88a27",
			Error: "failed to delete provider",
		})
		return
	}

	route.invalidateOrganizationModelsCache(reqCtx.Request.Context(), orgEntity.ID)
	if provider.ProjectID != nil {
		cacheKey := fmt.Sprintf(cache.ProjectModelsCacheKeyPattern, *provider.ProjectID)
		if err := route.cache.Unlink(reqCtx.Request.Context(), cacheKey); err != nil {
			logger.GetLogger().Warnf("organization provider: failed to invalidate project cache for project %d: %v", *provider.ProjectID, err)
		}
	}
	unlinkProviderModelsCache(reqCtx.Request.Context(), route.cache, provider.PublicID, "organization provider")

	reqCtx.Status(http.StatusNoContent)
}

// GetProjectProvider godoc
// @Summary Get project provider
// @Description Fetches details for a project-scoped provider.
// @Tags Providers API
// @Security BearerAuth
// @Produce json
// @Param proj_public_id path string true "Project public ID"
// @Param provider_id path string true "Provider public ID"
// @Success 200 {object} responses.GeneralResponse[ProviderResponse] "Provider details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/projects/{proj_public_id}/providers/{provider_id} [get]
func (route *ProjectProviderRoute) GetProjectProvider(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	projectEntity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "6c6c85b7-75c0-4f13-8564-1245c2448a34",
			Error: "project context missing",
		})
		return
	}

	provider, err := route.fetchProjectProvider(reqCtx.Request.Context(), orgEntity.ID, projectEntity.ID, reqCtx.Param("provider_id"))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "24d6dab2-1e92-4cc0-b532-4a188dd276aa",
			Error: "failed to load provider",
		})
		return
	}
	if provider == nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "96d7fe70-6e74-476c-bc22-11b2a31bb71f",
			Error: "provider not found",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[ProviderResponse]{
		Status: responses.ResponseCodeOk,
		Result: ProviderResponseFromModel(provider),
	})
}

// UpdateProjectProvider godoc
// @Summary Update project provider
// @Description Updates metadata, activation state, or credentials for a project-scoped provider.
// @Tags Providers API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param proj_public_id path string true "Project public ID"
// @Param provider_id path string true "Provider public ID"
// @Param request body UpdateProviderRequest true "Fields to update"
// @Success 200 {object} responses.GeneralResponse[ProviderResponse] "Provider updated"
// @Failure 400 {object} responses.ErrorResponse "Invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Insufficient permissions"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/projects/{proj_public_id}/providers/{provider_id} [patch]
func (route *ProjectProviderRoute) UpdateProjectProvider(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	projectEntity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "aa7c564b-403a-49b4-8bfb-6f9ec1bfaf0c",
			Error: "project context missing",
		})
		return
	}
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return
	}
	if !route.authorizeProjectOwner(reqCtx, projectEntity.ID, user.ID) {
		reqCtx.AbortWithStatusJSON(http.StatusForbidden, responses.ErrorResponse{
			Code:  "f8b5cf72-c77b-4f61-a724-f4782555026e",
			Error: "insufficient permissions",
		})
		return
	}

	providerID := strings.TrimSpace(reqCtx.Param("provider_id"))
	if providerID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b5d61b17-b024-47fb-9156-5448f513a01b",
			Error: "provider_id is required",
		})
		return
	}

	var req UpdateProviderRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "d7019267-7ff4-4c74-af5d-d2d4d73d3532",
			Error: "invalid payload",
		})
		return
	}

	provider, err := route.fetchProjectProvider(reqCtx.Request.Context(), orgEntity.ID, projectEntity.ID, providerID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "ab23414a-3f4e-4506-8e19-8ebacc7be5b6",
			Error: "failed to load provider",
		})
		return
	}
	if provider == nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "6b21570c-9bba-4d74-aed7-3d4ffa5fbc75",
			Error: "provider not found",
		})
		return
	}

	input := modelprovider.UpdateOrganizationProviderInput{PublicID: provider.PublicID}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		input.Name = &name
	}
	if req.BaseURL != nil {
		baseURL := strings.TrimSpace(*req.BaseURL)
		input.BaseURL = &baseURL
	}
	if req.APIKey != nil {
		apiKey := strings.TrimSpace(*req.APIKey)
		input.APIKey = &apiKey
	}
	if req.Metadata != nil {
		input.Metadata = req.Metadata
	}
	if req.Active != nil {
		input.Active = req.Active
	}

	updated, err := route.providerService.UpdateOrganizationProvider(reqCtx.Request.Context(), input)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "7e72d402-9d91-40e7-9d62-2d2e8ad1e078",
			Error: "failed to update provider",
		})
		return
	}

	route.invalidateProjectModelsCache(reqCtx.Request.Context(), projectEntity.ID)
	route.invalidateOrganizationModelsCache(reqCtx.Request.Context(), orgEntity.ID)
	unlinkProviderModelsCache(reqCtx.Request.Context(), route.cache, updated.PublicID, "project provider")

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[ProviderResponse]{
		Status: responses.ResponseCodeOk,
		Result: ProviderResponseFromModel(updated),
	})
}

// DeleteProjectProvider godoc
// @Summary Delete project provider
// @Description Removes a project-scoped provider and clears cached models.
// @Tags Providers API
// @Security BearerAuth
// @Param proj_public_id path string true "Project public ID"
// @Param provider_id path string true "Provider public ID"
// @Success 204 {string} string "Provider deleted"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Insufficient permissions"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/projects/{proj_public_id}/providers/{provider_id} [delete]
func (route *ProjectProviderRoute) DeleteProjectProvider(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	projectEntity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "c8cbeb9b-173f-44b5-94ea-3e04ac36f979",
			Error: "project context missing",
		})
		return
	}
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return
	}
	if !route.authorizeProjectOwner(reqCtx, projectEntity.ID, user.ID) {
		reqCtx.AbortWithStatusJSON(http.StatusForbidden, responses.ErrorResponse{
			Code:  "b8a7f80f-8bc3-40b0-9d12-dfd597240f1c",
			Error: "insufficient permissions",
		})
		return
	}

	provider, err := route.fetchProjectProvider(reqCtx.Request.Context(), orgEntity.ID, projectEntity.ID, reqCtx.Param("provider_id"))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "f6bf2f18-9ff1-4ddd-81f5-7a4c0a4c1d6b",
			Error: "failed to load provider",
		})
		return
	}
	if provider == nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "22427e8a-fd46-49e7-9c2f-8bc578c32d01",
			Error: "provider not found",
		})
		return
	}

	if err := route.providerService.DeleteByPublicID(reqCtx.Request.Context(), provider.PublicID); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "4bdeac78-9af7-4f94-bc65-07977cd736bf",
			Error: "failed to delete provider",
		})
		return
	}

	route.invalidateProjectModelsCache(reqCtx.Request.Context(), projectEntity.ID)
	route.invalidateOrganizationModelsCache(reqCtx.Request.Context(), orgEntity.ID)
	unlinkProviderModelsCache(reqCtx.Request.Context(), route.cache, provider.PublicID, "project provider")

	reqCtx.Status(http.StatusNoContent)
}

func (route *OrganizationProviderRoute) fetchOrganizationProvider(ctx context.Context, organizationID uint, providerID string) (*modelprovider.ModelProvider, error) {
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return nil, nil
	}
	provider, err := route.providerService.GetByPublicID(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if provider == nil || provider.OrganizationID == nil || *provider.OrganizationID != organizationID {
		return nil, nil
	}
	if provider.ProjectID != nil {
		return nil, nil
	}
	return provider, nil
}

func (route *ProjectProviderRoute) fetchProjectProvider(ctx context.Context, organizationID uint, projectID uint, providerID string) (*modelprovider.ModelProvider, error) {
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return nil, nil
	}
	provider, err := route.providerService.GetByPublicID(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	if provider == nil || provider.OrganizationID == nil || *provider.OrganizationID != organizationID {
		return nil, nil
	}
	if provider.ProjectID == nil || *provider.ProjectID != projectID {
		return nil, nil
	}
	return provider, nil
}
func ProviderResponseFromModel(provider *modelprovider.ModelProvider) ProviderResponse {
	scope := "organization"
	var projectID *uint
	if provider.ProjectID != nil {
		scope = "project"
		projectID = provider.ProjectID
	}
	orgID := uint(0)
	if provider.OrganizationID != nil {
		orgID = *provider.OrganizationID
	}
	return ProviderResponse{
		ProviderID:     provider.PublicID,
		Name:           provider.Name,
		Type:           provider.Type.String(),
		Vendor:         provider.Vendor.String(),
		APIKeyHint:     provider.APIKeyHint,
		Active:         provider.Active,
		Scope:          scope,
		OrganizationID: orgID,
		ProjectID:      projectID,
	}
}

func parseVendor(input string) (modelprovider.ProviderVendor, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	vendor := modelprovider.ProviderVendor(normalized)
	switch vendor {
	case modelprovider.ProviderVendorOpenRouter, modelprovider.ProviderVendorGemini:
		return vendor, nil
	default:
		return "", fmt.Errorf("unsupported vendor: %s", input)
	}
}

func (route *ProjectProviderRoute) authorizeProjectOwner(reqCtx *gin.Context, projectID uint, userID uint) bool {
	if orgMember, ok := auth.GetAdminOrganizationMemberFromContext(reqCtx); ok {
		if orgMember.Role == organization.OrganizationMemberRoleOwner {
			return true
		}
	}
	ctx := reqCtx.Request.Context()
	member, err := route.projectService.FindOneMemberByFilter(ctx, project.ProjectMemberFilter{
		ProjectID: &projectID,
		UserID:    &userID,
	})
	if err != nil || member == nil {
		return false
	}
	return member.Role == string(project.ProjectMemberRoleOwner)
}

func (route *OrganizationProviderRoute) invalidateOrganizationModelsCache(ctx context.Context, organizationID uint) {
	if route.cache == nil {
		return
	}
	cacheKey := fmt.Sprintf(cache.OrganizationModelsCacheKeyPattern, organizationID)
	if err := route.cache.Unlink(ctx, cacheKey); err != nil {
		logger.GetLogger().Warnf("organization provider: failed to invalidate cache for org %d: %v", organizationID, err)
	}
}

func (route *ProjectProviderRoute) invalidateProjectModelsCache(ctx context.Context, projectID uint) {
	if route.cache == nil {
		return
	}
	cacheKey := fmt.Sprintf(cache.ProjectModelsCacheKeyPattern, projectID)
	if err := route.cache.Unlink(ctx, cacheKey); err != nil {
		logger.GetLogger().Warnf("project provider: failed to invalidate cache for project %d: %v", projectID, err)
	}
}
func (route *ProjectProviderRoute) invalidateOrganizationModelsCache(ctx context.Context, organizationID uint) {
	if route.cache == nil {
		return
	}
	cacheKey := fmt.Sprintf(cache.OrganizationModelsCacheKeyPattern, organizationID)
	if err := route.cache.Unlink(ctx, cacheKey); err != nil {
		logger.GetLogger().Warnf("project provider: failed to invalidate organization cache for org %d: %v", organizationID, err)
	}
}

func unlinkProviderModelsCache(ctx context.Context, cacheService *cache.RedisCacheService, providerID string, logPrefix string) {
	if cacheService == nil {
		return
	}
	trimmed := strings.TrimSpace(providerID)
	if trimmed == "" {
		return
	}
	cacheKey := fmt.Sprintf("%s:%s", cache.ModelsCacheKey, trimmed)
	if err := cacheService.Unlink(ctx, cacheKey); err != nil {
		logger.GetLogger().Warnf("%s: failed to invalidate provider models cache for provider %s: %v", logPrefix, trimmed, err)
	}
}
