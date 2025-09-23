package projects

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses/openai"
	projectApikeyRoute "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects/api_keys"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ProjectsRoute struct {
	projectService     *project.ProjectService
	apiKeyService      *apikey.ApiKeyService
	authService        *auth.AuthService
	projectApiKeyRoute *projectApikeyRoute.ProjectApiKeyRoute
}

func NewProjectsRoute(
	projectService *project.ProjectService,
	apiKeyService *apikey.ApiKeyService,
	authService *auth.AuthService,
	projectApiKeyRoute *projectApikeyRoute.ProjectApiKeyRoute,
) *ProjectsRoute {
	return &ProjectsRoute{
		projectService,
		apiKeyService,
		authService,
		projectApiKeyRoute,
	}
}

func (projectsRoute *ProjectsRoute) RegisterRouter(router gin.IRouter) {
	permissionOptional := projectsRoute.authService.OrganizationMemberOptionalMiddleware()
	permissionOwnerOnly := projectsRoute.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleOwnerOnly)
	projectsRouter := router.Group(
		"/projects",
		projectsRoute.authService.AdminUserAuthMiddleware(),
		projectsRoute.authService.RegisteredUserMiddleware(),
	)
	projectsRouter.GET("",
		permissionOptional,
		projectsRoute.GetProjects,
	)
	projectsRouter.POST("",
		permissionOwnerOnly,
		projectsRoute.CreateProject,
	)

	projectIdRouter := projectsRouter.Group(
		fmt.Sprintf("/:%s", auth.ProjectContextKeyPublicID),
		permissionOptional,
		projectsRoute.authService.AdminProjectMiddleware(),
	)
	projectIdRouter.GET("",
		projectsRoute.GetProject)
	projectIdRouter.POST("",
		permissionOwnerOnly,
		projectsRoute.UpdateProject,
	)
	projectIdRouter.POST("/archive",
		permissionOwnerOnly,
		projectsRoute.ArchiveProject,
	)
	projectsRoute.projectApiKeyRoute.RegisterRouter(projectIdRouter)
}

// GetProjects godoc
// @Summary List Projects
// @Description Retrieves a paginated list of all projects for the authenticated organization.
// @Tags Administration API
// @Security BearerAuth
// @Param limit query int false "The maximum number of items to return" default(20)
// @Param after query string false "A cursor for use in pagination. The ID of the last object from the previous page"
// @Param include_archived query string false "Whether to include archived projects."
// @Success 200 {object} ProjectListResponse "Successfully retrieved the list of projects"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /v1/organization/projects [get]
func (api *ProjectsRoute) GetProjects(reqCtx *gin.Context) {
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		return
	}
	projectService := api.projectService
	includeArchivedStr := reqCtx.DefaultQuery("include_archived", "false")
	includeArchived, err := strconv.ParseBool(includeArchivedStr)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "65e69a2c-5ce0-4a9c-bb61-ee5cc494f948",
			Error: "invalid or missing query parameter",
		})
		return
	}
	ctx := reqCtx.Request.Context()
	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(after string) (*uint, error) {
		entity, err := projectService.FindOne(ctx, project.ProjectFilter{
			PublicID: &after,
		})
		if err != nil {
			return nil, err
		}
		return &entity.ID, nil
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "4434f5ed-89f4-4a62-9fef-8ca53336dcda",
			Error: "invalid or missing query parameter",
		})
		return
	}
	projectFilter := project.ProjectFilter{
		OrganizationID: &orgEntity.ID,
	}
	_, ok = auth.GetAdminOrganizationMemberFromContext(reqCtx)
	if !ok {
		projectFilter.MemberID = &user.ID
	}
	if !includeArchived {
		projectFilter.Archived = ptr.ToBool(false)
	}
	projects, err := projectService.Find(ctx, projectFilter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "29d3d0b0-e587-4f20-9adb-1ab9aa666b38",
			Error: "failed to retrieve projects",
		})
		return
	}

	pageCursor, err := responses.BuildCursorPage(
		projects,
		func(t *project.Project) *string {
			return &t.PublicID
		},
		func() ([]*project.Project, error) {
			return projectService.Find(ctx, projectFilter, &query.Pagination{
				Order: pagination.Order,
				Limit: ptr.ToInt(1),
				After: &projects[len(projects)-1].ID,
			})
		},
		func() (int64, error) {
			return projectService.CountProjects(ctx, projectFilter)
		},
	)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "6a0ee74e-d6fd-4be8-91b3-03a594b8cd2e",
		})
		return
	}

	result := functional.Map(projects, func(project *project.Project) ProjectResponse {
		return domainToProjectResponse(project)
	})

	response := openai.ListResponse[ProjectResponse]{
		Object:  "list",
		Data:    result,
		HasMore: pageCursor.HasMore,
		FirstID: pageCursor.FirstID,
		LastID:  pageCursor.LastID,
		Total:   int64(pageCursor.Total),
	}
	reqCtx.JSON(http.StatusOK, response)
}

// CreateProject godoc
// @Summary Create Project
// @Description Creates a new project for an organization.
// @Tags Administration API
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateProjectRequest true "Project creation request"
// @Success 200 {object} ProjectResponse "Successfully created project"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /v1/organization/projects [post]
func (api *ProjectsRoute) CreateProject(reqCtx *gin.Context) {
	projectService := api.projectService
	ctx := reqCtx.Request.Context()
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	orgMember, ok := auth.GetAdminOrganizationMemberFromContext(reqCtx)
	if !ok || orgMember.Role != organization.OrganizationMemberRoleOwner {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "6054bc14-ea67-4f27-b649-0d03050cc25f",
		})
		return
	}

	var requestPayload CreateProjectRequest
	if err := reqCtx.ShouldBindJSON(&requestPayload); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "db8142f8-dc78-4581-a238-6e32288a54ec",
			Error: err.Error(),
		})
		return
	}

	projectEntity, err := projectService.CreateProjectWithPublicID(ctx, &project.Project{
		Name:           requestPayload.Name,
		OrganizationID: orgEntity.ID,
		Status:         string(project.ProjectStatusActive),
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "e00e6ab3-1b43-490e-90df-aae030697f74",
			Error: err.Error(),
		})
		return
	}
	err = projectService.AddMember(ctx, &project.ProjectMember{
		UserID:    orgMember.UserID,
		ProjectID: projectEntity.ID,
		Role:      string(project.ProjectMemberRoleOwner),
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "e29ddee3-77ea-4ac5-b474-00e2311b68ab",
			ErrorInstance: err,
		})
		return
	}
	response := domainToProjectResponse(projectEntity)
	reqCtx.JSON(http.StatusOK, response)
}

// GetProject godoc
// @Summary Get Project
// @Description Retrieves a specific project by its ID.
// @Tags Administration API
// @Security BearerAuth
// @Param project_id path string true "ID of the project"
// @Success 200 {object} ProjectResponse "Successfully retrieved the project"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - project with the given ID does not exist or does not belong to the organization"
// @Router /v1/organization/projects/{project_id} [get]
func (api *ProjectsRoute) GetProject(reqCtx *gin.Context) {
	projectEntity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "42ad3a04-6c17-40db-a10f-640be569c93f",
			Error: "project not found",
		})
		return
	}
	reqCtx.JSON(http.StatusOK, domainToProjectResponse(projectEntity))
}

// UpdateProject godoc
// @Summary Update Project
// @Description Updates a specific project by its ID.
// @Tags Administration API
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "ID of the project to update"
// @Param body body UpdateProjectRequest true "Project update request"
// @Success 200 {object} ProjectResponse "Successfully updated the project"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - project with the given ID does not exist"
// @Router /v1/organization/projects/{project_id} [post]
func (api *ProjectsRoute) UpdateProject(reqCtx *gin.Context) {
	orgMember, ok := auth.GetAdminOrganizationMemberFromContext(reqCtx)
	if !ok || orgMember.Role != organization.OrganizationMemberRoleOwner {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "2e531704-2e55-4d55-9ca3-d60e245f75b4",
		})
		return
	}
	projectService := api.projectService
	ctx := reqCtx.Request.Context()
	var requestPayload UpdateProjectRequest
	if err := reqCtx.ShouldBindJSON(&requestPayload); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "b6cb35be-8a53-478d-95d1-5e1f64f35c09",
			ErrorInstance: err,
		})
		return
	}

	entity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "42ad3a04-6c17-40db-a10f-640be569c93f",
			Error: "project not found",
		})
		return
	}

	// Update the project name if provided
	if requestPayload.Name != nil {
		entity.Name = *requestPayload.Name
	}

	updatedEntity, err := projectService.UpdateProject(ctx, entity)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "c9a103b2-985c-44b7-9ccd-38e914a2c82b",
			Error: "failed to update project",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, domainToProjectResponse(updatedEntity))
}

// ArchiveProject godoc
// @Summary Archive Project
// @Description Archives a specific project by its ID, making it inactive.
// @Tags Administration API
// @Security BearerAuth
// @Param project_id path string true "ID of the project to archive"
// @Success 200 {object} ProjectResponse "Successfully archived the project"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - project with the given ID does not exist"
// @Router /v1/organization/projects/{project_id}/archive [post]
func (api *ProjectsRoute) ArchiveProject(reqCtx *gin.Context) {
	projectService := api.projectService
	ctx := reqCtx.Request.Context()

	entity, ok := auth.GetProjectFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "42ad3a04-6c17-40db-a10f-640be569c93f",
			Error: "project not found",
		})
		return
	}

	// Set archived status
	entity.Status = string(project.ProjectStatusArchived)
	entity.ArchivedAt = ptr.ToTime(time.Now())
	updatedEntity, err := projectService.UpdateProject(ctx, entity)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "c9a103b2-985c-44b7-9ccd-38e914a2c82b",
			Error: "failed to archive project",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, domainToProjectResponse(updatedEntity))
}

// ProjectResponse defines the response structure for a project.
type ProjectResponse struct {
	Object     string `json:"object" example:"project" description:"The type of the object, 'project'"`
	ID         string `json:"id" example:"proj_1234567890" description:"Unique identifier for the project"`
	Name       string `json:"name" example:"My First Project" description:"The name of the project"`
	CreatedAt  int64  `json:"created_at" example:"1698765432" description:"Unix timestamp when the project was created"`
	ArchivedAt *int64 `json:"archived_at,omitempty" example:"1698765432" description:"Unix timestamp when the project was archived, if applicable"`
	Status     string `json:"status"`
}

// CreateProjectRequest defines the request payload for creating a project.
type CreateProjectRequest struct {
	Name string `json:"name" binding:"required" example:"New AI Project" description:"The name of the project to be created"`
}

// UpdateProjectRequest defines the request payload for updating a project.
type UpdateProjectRequest struct {
	Name *string `json:"name" example:"Updated AI Project" description:"The new name for the project"`
}

// ProjectListResponse defines the response structure for a list of projects.
type ProjectListResponse struct {
	Object  string            `json:"object" example:"list" description:"The type of the object, 'list'"`
	Data    []ProjectResponse `json:"data" description:"Array of projects"`
	FirstID *string           `json:"first_id,omitempty"`
	LastID  *string           `json:"last_id,omitempty"`
	HasMore bool              `json:"has_more"`
}

func domainToProjectResponse(p *project.Project) ProjectResponse {
	var archivedAt *int64
	if p.ArchivedAt != nil {
		archivedAt = ptr.ToInt64(p.CreatedAt.Unix())
	}
	return ProjectResponse{
		Object:     string(openai.ObjectKeyProject),
		ID:         p.PublicID,
		Name:       p.Name,
		CreatedAt:  p.CreatedAt.Unix(),
		ArchivedAt: archivedAt,
		Status:     p.Status,
	}
}
