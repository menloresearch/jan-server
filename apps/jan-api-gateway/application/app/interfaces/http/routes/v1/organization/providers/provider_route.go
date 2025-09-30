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
}

func (route *ProjectProviderRoute) RegisterRouter(router gin.IRouter) {
	providersRouter := router.Group("/providers",
		route.authService.AdminUserAuthMiddleware(),
		route.authService.RegisteredUserMiddleware(),
	)
	permissionAll := route.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleAll)
	providersRouter.GET("", permissionAll, route.ListProjectProviders)

	providersRouter.POST("", permissionAll, route.CreateProjectProvider)
}

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

	reqCtx.JSON(http.StatusCreated, ProviderResponseFromModel(provider))
}

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

	reqCtx.JSON(http.StatusCreated, ProviderResponseFromModel(provider))
}

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

	providers, err := route.providerService.List(reqCtx.Request.Context(), modelprovider.ProviderFilter{
		OrganizationID: &orgEntity.ID,
		ProjectID:      &projectEntity.ID,
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
