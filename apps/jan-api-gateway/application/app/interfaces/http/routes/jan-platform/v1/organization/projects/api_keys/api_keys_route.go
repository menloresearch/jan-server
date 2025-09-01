package apikeys

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
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
	apiKeyRouter.GET("", api.CreateProjectApiKey)
}

func (api *ProjectApiKeyRoute) CreateProjectApiKey(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "9715151d-02ab-4759-bfb7-89d717f05cd3",
			Error: err.Error(),
		})
		return
	}
	user, err := api.userService.FindByEmailAndPlatform(ctx, userClaim.Email, string(user.UserPlatformTypePlatform))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "edf9dd05-aad4-4c1e-9795-98bf60ecf57c",
			Error: err.Error(),
		})
		return
	}
	if user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "417cff16-0325-45f7-9826-8ab24d2fef29",
		})
		return
	}

	organizationEntity, _ := api.organizationService.GetOrganizationFromContext(reqCtx)

	// TODO: Change the verification to users with organization read permission.
	if organizationEntity.OwnerID != user.ID {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "6d2d10f9-3bab-4d2d-8076-d573d829e397",
		})
		return
	}
	api.apikeyService.CreateApiKey(ctx, &apikey.ApiKey{
		
	})
}
