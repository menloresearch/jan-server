package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"

	"menlo.ai/jan-api-gateway/app/domain/project"
	infrainference "menlo.ai/jan-api-gateway/app/infrastructure/inference"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

type ModelAPI struct {
	multiProvider  *infrainference.MultiProviderInference
	authService    *auth.AuthService
	projectService *project.ProjectService
}

func NewModelAPI(multiProvider *infrainference.MultiProviderInference, authService *auth.AuthService, projectService *project.ProjectService) *ModelAPI {
	return &ModelAPI{
		multiProvider:  multiProvider,
		authService:    authService,
		projectService: projectService,
	}
}

func (modelAPI *ModelAPI) RegisterRouter(router *gin.RouterGroup) {
	modelsRouter := router.Group("",
		modelAPI.authService.AppUserAuthMiddleware(),
		modelAPI.authService.RegisteredUserMiddleware(),
		modelAPI.authService.DefaultOrganizationMemberOptionalMiddleware(),
	)
	modelsRouter.GET("models", modelAPI.GetModels)
	modelsRouter.GET("models/providers", modelAPI.GetProviders)
}

func (modelAPI *ModelAPI) GetModels(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	filter, err := modelAPI.buildProviderFilter(reqCtx)
	if err != nil {
		logger.GetLogger().Errorf("failed to build provider filter: %v", err)
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "e3d2b60d-8b49-4e6d-9e8d-9f0c1adbe4fd",
			Error: "failed to list providers",
		})
		return
	}
	providers, err := modelAPI.multiProvider.ListProviders(ctx, filter)
	if err != nil {
		logger.GetLogger().Errorf("failed to list providers: %v", err)
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "6c3e5c9d-1f50-4f58-a8d9-47bce682b6d0",
			Error: "failed to list providers",
		})
		return
	}

	providerNames := make(map[string]string, len(providers))
	for _, provider := range providers {
		providerNames[provider.ProviderID] = provider.Name
	}

	selection := infrainference.ProviderSelection{
		OrganizationID: filter.OrganizationID,
	}
	if filter.ProjectID != nil {
		selection.ProjectID = filter.ProjectID
	}
	if filter.ProjectIDs != nil {
		selection.ProjectIDs = append(selection.ProjectIDs, (*filter.ProjectIDs)...)
	}

	modelsResp, err := modelAPI.multiProvider.GetModels(ctx, selection)
	if err != nil {
		logger.GetLogger().Errorf("failed to aggregate models: %v", err)
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "27c47e9f-4143-4691-91f7-635f87f291ec",
			Error: "failed to list models",
		})
		return
	}

	data := make([]ModelResponse, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		name := providerNames[m.ProviderID]
		if name == "" {
			name = m.ProviderID
		}
		data = append(data, ModelResponse{
			ID:             m.ID,
			Object:         m.Object,
			Created:        m.Created,
			OwnedBy:        m.OwnedBy,
			ProviderID:     m.ProviderID,
			ProviderType:   m.ProviderType.String(),
			ProviderVendor: m.Vendor.String(),
			ProviderName:   name,
		})
	}

	reqCtx.JSON(http.StatusOK, ModelsResponse{
		Object: "list",
		Data:   data,
	})
}

type ModelResponse struct {
	ID             string `json:"id"`
	Object         string `json:"object"`
	Created        int    `json:"created"`
	OwnedBy        string `json:"owned_by"`
	ProviderID     string `json:"provider_id"`
	ProviderType   string `json:"provider_type"`
	ProviderVendor string `json:"provider_vendor"`
	ProviderName   string `json:"provider_name"`
}

type ModelsResponse struct {
	Object string          `json:"object"`
	Data   []ModelResponse `json:"data"`
}

type ProvidersResponse struct {
	Object string                    `json:"object"`
	Data   []ProviderSummaryResponse `json:"data"`
}

type ProviderSummaryResponse struct {
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Vendor     string `json:"vendor"`
	APIKeyHint string `json:"api_key_hint,omitempty"`
	Active     bool   `json:"active"`
}

func (modelAPI *ModelAPI) GetProviders(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	filter, err := modelAPI.buildProviderFilter(reqCtx)
	if err != nil {
		logger.GetLogger().Errorf("failed to build provider filter: %v", err)
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "205cddfb-9568-4a1d-9f43-f3c949015135",
			Error: "failed to list providers",
		})
		return
	}
	providers, err := modelAPI.multiProvider.ListProviders(ctx, filter)
	if err != nil {
		logger.GetLogger().Errorf("failed to list providers: %v", err)
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "9d1a67cf-8c02-4e3f-8f6e-3b381b18cba7",
			Error: "failed to list providers",
		})
		return
	}

	summaries := make([]ProviderSummaryResponse, len(providers))
	for i, provider := range providers {
		summaries[i] = ProviderSummaryResponse{
			ProviderID: provider.ProviderID,
			Name:       provider.Name,
			Type:       provider.Type.String(),
			Vendor:     provider.Vendor.String(),
			APIKeyHint: provider.APIKeyHint,
			Active:     provider.Active,
		}
	}

	reqCtx.JSON(http.StatusOK, ProvidersResponse{
		Object: "list",
		Data:   summaries,
	})
}

func (modelAPI *ModelAPI) buildProviderFilter(reqCtx *gin.Context) (infrainference.ProviderSummaryFilter, error) {
	filter := infrainference.ProviderSummaryFilter{}
	if org, ok := auth.GetAdminOrganizationFromContext(reqCtx); ok && org != nil {
		filter.OrganizationID = &org.ID
	}
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok || user == nil {
		return filter, nil
	}
	if modelAPI.projectService == nil {
		return filter, nil
	}
	ctx := reqCtx.Request.Context()
	projects, err := modelAPI.projectService.Find(ctx, project.ProjectFilter{
		MemberID: &user.ID,
	}, nil)
	if err != nil {
		return filter, err
	}
	if len(projects) == 0 {
		return filter, nil
	}
	ids := make([]uint, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ID)
	}
	filter.ProjectIDs = &ids
	return filter, nil
}
