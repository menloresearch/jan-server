package apikeys

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
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
}

func (api *OrganizationApiKeyRoute) ListApiKeys(reqCtx *gin.Context) {
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

	orgPublicId := reqCtx.Param("org_public_id")
	if orgPublicId == "" {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "ada5a8f9-d5e1-4761-9af1-a176473ff7eb",
			Error: "invalid or missing project ID",
		})
		return
	}

	organization, err := api.organizationService.FindOrganizationByPublicID(ctx, orgPublicId)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "61b57adf-32be-4532-9090-e2f8e576491d",
		})
		return
	}
	if organization == nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "67a03e01-2797-4a5e-b2cd-f2893d4a14b2",
		})
		return
	}

	// TODO: Change the verification to users with organization read permission.
	if organization.OwnerID != user.ID {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "6d2d10f9-3bab-4d2d-8076-d573d829e397",
		})
		return
	}

	apikeyService := api.apikeyService
	apikeys, err := apikeyService.Find(ctx, apikey.ApiKeyFilter{
		OrganizationID: &organization.ID,
	}, nil)
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
}

func convertApiKeyDomainToResponse(entity *apikey.ApiKey) ApiKeyResponse {
	return ApiKeyResponse{}
}
