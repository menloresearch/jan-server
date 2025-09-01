package organization

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	apikeys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan-platform/v1/organization/api_keys"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan-platform/v1/organization/projects"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type OrganizationRoute struct {
	organizationService *organization.OrganizationService
	userService         *user.UserService
	projectsRoute       *projects.ProjectsRoute
	apiKeyRoute         *apikeys.OrganizationApiKeyRoute
}

func NewOrganizationRoute(
	organizationService *organization.OrganizationService,
	userService *user.UserService,
	projectsRoute *projects.ProjectsRoute,
	apiKeyRoute *apikeys.OrganizationApiKeyRoute) *OrganizationRoute {
	return &OrganizationRoute{
		organizationService,
		userService,
		projectsRoute,
		apiKeyRoute,
	}
}

func (o *OrganizationRoute) RegisterRouter(router gin.IRouter) {
	organizationRouter := router.Group("/organizations", middleware.AuthMiddleware())
	organizationIdRouter := organizationRouter.Group(fmt.Sprintf("/:%s", organization.OrganizationContextKeyPublicID), o.organizationService.OrganizationMiddleware())
	o.projectsRoute.RegisterRouter(organizationIdRouter)
	o.apiKeyRoute.RegisterRouter(organizationRouter)
	organizationRouter.GET("", o.ListOrganization)
}

func (o *OrganizationRoute) ListOrganization(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "9715151d-02ab-4759-bfb7-89d717f05cd3",
			Error: err.Error(),
		})
		return
	}
	user, err := o.userService.FindByEmailAndPlatform(ctx, userClaim.Email, string(user.UserPlatformTypePlatform))
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
	pagination, err := query.GetPaginationFromQuery(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "1f11f211-7f74-43c9-b7c3-df31fcd2cf4d",
		})
		return
	}

	filter := organization.OrganizationFilter{
		OwnerID: &user.ID,
	}

	organizationEntities, err := o.organizationService.FindOrganizations(ctx, filter, pagination)

	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "5000796f-2c68-49db-a1f2-a9c83197bde8",
		})
		return
	}

	count, err := o.organizationService.CountOrganizations(ctx, filter)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "5493a8e5-1367-4695-944b-d69e5d3069ea",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.ListResponse[OrganizationResponse]{
		Status: responses.ResponseCodeOk,
		Total:  count,
		Results: functional.Map(organizationEntities, func(entity *organization.Organization) OrganizationResponse {
			return *convertDomainToOrganizationResponse(entity)
		}),
	})
}

type OrganizationResponse struct {
	ID        uint
	Name      string
	PublicID  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Enabled   bool
}

func convertDomainToOrganizationResponse(entity *organization.Organization) *OrganizationResponse {
	return &OrganizationResponse{
		ID:        entity.ID,
		Name:      entity.Name,
		PublicID:  entity.PublicID,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
		Enabled:   entity.Enabled,
	}
}
