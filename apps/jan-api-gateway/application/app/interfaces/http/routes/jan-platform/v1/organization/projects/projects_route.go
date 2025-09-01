package projects

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type ProjectsRoute struct {
	userService         *user.UserService
	projectService      *project.ProjectService
	organizationService *organization.OrganizationService
}

func NewProjectsRoute(
	userService *user.UserService,
	projectService *project.ProjectService,
	organizationService *organization.OrganizationService,
) *ProjectsRoute {
	return &ProjectsRoute{
		userService:         userService,
		projectService:      projectService,
		organizationService: organizationService,
	}
}

func (api *ProjectsRoute) RegisterRouter(router gin.IRouter) {
	projectRoute := router.Group("/:org_public_id/projects")
	projectRoute.GET("", api.ListProjects)
}

func (api *ProjectsRoute) ListProjects(reqCtx *gin.Context) {
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
