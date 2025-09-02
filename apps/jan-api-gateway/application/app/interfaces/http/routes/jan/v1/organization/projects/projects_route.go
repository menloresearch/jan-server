package projects

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	apikeys "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization/projects/api_keys"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type ProjectsRoute struct {
	userService         *user.UserService
	projectService      *project.ProjectService
	organizationService *organization.OrganizationService
	projectApiKeyRoute  *apikeys.ProjectApiKeyRoute
}

func NewProjectsRoute(
	userService *user.UserService,
	projectService *project.ProjectService,
	organizationService *organization.OrganizationService,
	projectApiKeyRoute *apikeys.ProjectApiKeyRoute,
) *ProjectsRoute {
	return &ProjectsRoute{
		userService:         userService,
		projectService:      projectService,
		organizationService: organizationService,
		projectApiKeyRoute:  projectApiKeyRoute,
	}
}

func (api *ProjectsRoute) RegisterRouter(router gin.IRouter) {
	projectRoute := router.Group("/projects")
	projectRoute.GET("", api.ListProjects)
	projectIdRouter := projectRoute.Group(fmt.Sprintf("/:%s", project.ProjectContextKeyPublicID), api.projectService.ProjectMiddleware())
	api.projectApiKeyRoute.RegisterRouter(projectIdRouter)
}

// @Summary List projects
// @Description List all projects within a given organization.
// @Security BearerAuth
// @Tags projects
// @Accept json
// @Produce json
// @Param org_public_id path string true "Organization Public ID"
// @Param limit query int false "Number of projects to return" default(10)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} responses.ListResponse[ProjectResponse] "Successfully retrieved projects"
// @Failure 400 {object} responses.ErrorResponse "Bad request, e.g., invalid pagination parameters or organization ID"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized, e.g., invalid or missing token"
// @Failure 404 {object} responses.ErrorResponse "Not Found, e.g., organization not found or no projects found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/organizations/{org_public_id}/projects [get]
func (api *ProjectsRoute) ListProjects(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, ok := api.userService.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "60a44edb-9127-48ad-aabd-20431289f73f",
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

	pagination, err := query.GetPaginationFromQuery(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "1f11f211-7f74-43c9-b7c3-df31fcd2cf4d",
		})
		return
	}
	filter := project.ProjectFilter{
		OrganizationID: &organizationEntity.ID,
	}
	projectEntities, err := api.projectService.Find(ctx, filter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "10f4132e-7042-4fc2-8675-b2fbae8158f9",
		})
		return
	}
	if len(projectEntities) == 0 {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code: "4c7e0459-f431-4fee-9e1b-d2b07013651e",
		})
		return
	}

	count, err := api.projectService.CountProjects(ctx, filter)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "5493a8e5-1367-4695-944b-d69e5d3069ea",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.ListResponse[ProjectResponse]{
		Status: responses.ResponseCodeOk,
		Total:  count,
		Results: functional.Map(projectEntities, func(entity *project.Project) ProjectResponse {
			return convertDomainToProjectResponse(entity)
		}),
	})
}

type ProjectResponse struct {
	Name           string
	PublicID       string
	Status         string
	OrganizationID uint
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ArchivedAt     *time.Time
}

func convertDomainToProjectResponse(entity *project.Project) ProjectResponse {
	return ProjectResponse{
		Name:       entity.Name,
		PublicID:   entity.PublicID,
		Status:     entity.Status,
		CreatedAt:  entity.CreatedAt,
		UpdatedAt:  entity.UpdatedAt,
		ArchivedAt: entity.ArchivedAt,
	}
}
