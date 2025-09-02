package organization

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	apikeys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization/api_keys"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization/projects"
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
	organizationRouter := router.Group("/organizations", middleware.AuthMiddleware(), o.userService.RegisteredUserMiddleware())
	organizationIdRouter := organizationRouter.Group(fmt.Sprintf("/:%s", organization.OrganizationContextKeyPublicID), o.organizationService.OrganizationMiddleware())
	o.projectsRoute.RegisterRouter(organizationIdRouter)
	o.apiKeyRoute.RegisterRouter(organizationIdRouter)
	organizationRouter.GET("", o.ListOrganization)
}

// @Summary List organizations
// @Description Retrieves a list of organizations owned by the authenticated user.
// @Tags organizations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Number of organizations to return" default(10)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} responses.ListResponse[OrganizationResponse] "Successfully retrieved organizations."
// @Failure 400 {object} responses.ErrorResponse "Bad request, e.g., invalid pagination parameters."
// @Failure 401 {object} responses.ErrorResponse "Unauthorized, e.g., invalid or missing token."
// @Failure 500 {object} responses.ErrorResponse "Internal server error."
// @Router /jan/v1/organizations [get]
func (o *OrganizationRoute) ListOrganization(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, ok := o.userService.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "746ca453-519f-453e-8855-f523379ab306",
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
	Name      string
	PublicID  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Enabled   bool
}

func convertDomainToOrganizationResponse(entity *organization.Organization) *OrganizationResponse {
	return &OrganizationResponse{
		Name:      entity.Name,
		PublicID:  entity.PublicID,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
		Enabled:   entity.Enabled,
	}
}
