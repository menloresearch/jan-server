package projects

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses/openai"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ProjectsRoute struct {
	projectService *project.ProjectService
	apiKeyService  *apikey.ApiKeyService
}

func NewProjectsRoute(projectService *project.ProjectService, apiKeyService *apikey.ApiKeyService) *ProjectsRoute {
	return &ProjectsRoute{
		projectService,
		apiKeyService,
	}
}

func (projectsRoute *ProjectsRoute) RegisterRouter(router gin.IRouter) {
	projectsRouter := router.Group("/projects")
	projectsRouter.GET("", projectsRoute.GetProjects)
	projectsRouter.POST("", projectsRoute.CreateProject)
	projectsRouter.GET("/:project_id", projectsRoute.GetProject)
	projectsRouter.POST("/:project_id", projectsRoute.UpdateProject)
	projectsRouter.POST("/:project_id/archive", projectsRoute.ArchiveProject)
}

// GetProjects godoc
// @Summary List Projects
// @Description Retrieves a paginated list of all projects for the authenticated organization.
// @Tags Platform, Platform-Organizations
// @Security BearerAuth
// @Param Authorization header string true "Bearer token" default("Bearer <api_key>")
// @Param limit query int false "The maximum number of items to return" default(20)
// @Param after query string false "A cursor for use in pagination. The ID of the last object from the previous page"
// @Param include_archived query string false "Whether to include archived projects."
// @Success 200 {object} ProjectListResponse "Successfully retrieved the list of projects"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /v1/organization/projects [get]
func (api *ProjectsRoute) GetProjects(reqCtx *gin.Context) {
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
	adminKey, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	pagination, err := query.GetPaginationFromQuery(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "4434f5ed-89f4-4a62-9fef-8ca53336dcda",
			Error: "invalid or missing query parameter",
		})
		return
	}

	afterStr := reqCtx.Query("after")
	if afterStr != "" {
		entity, err := projectService.Find(ctx, project.ProjectFilter{
			PublicID: &afterStr,
		}, &query.Pagination{
			Limit: ptr.ToInt(1),
		})
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "20f37a43-1c2e-4efe-9f5b-c1d0b1ccdd58",
				Error: err.Error(),
			})
			return
		}
		if len(entity) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "a8a65f59-7b92-4b09-87eb-993eb19188e6",
				Error: "failed to retrieve projects",
			})
			return
		}
		pagination.After = &entity[0].ID
	}

	projectFilter := project.ProjectFilter{
		OrganizationID: adminKey.OrganizationID,
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

	var firstId *string
	var lastId *string
	hasMore := false
	if len(projects) > 0 {
		firstId = &projects[0].PublicID
		lastId = &projects[len(projects)-1].PublicID
		moreRecords, err := projectService.Find(ctx, projectFilter, &query.Pagination{
			Order: pagination.Order,
			Limit: ptr.ToInt(1),
			After: &projects[len(projects)-1].ID,
		})
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "1be3dfc2-f2ce-4b0e-a385-1cc6f4324398",
				Error: "failed to retrieve API keys",
			})
			return
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}

	result := functional.Map(projects, func(project *project.Project) ProjectResponse {
		return domainToProjectResponse(project)
	})

	response := ProjectListResponse{
		Object:  "list",
		Data:    result,
		HasMore: hasMore,
		FirstID: firstId,
		LastID:  lastId,
	}
	reqCtx.JSON(http.StatusOK, response)
}

// CreateProject godoc
// @Summary Create Project
// @Description Creates a new project for an organization.
// @Tags Platform, Platform-Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token" default("Bearer <api_key>")
// @Param body body CreateProjectRequest true "Project creation request"
// @Success 200 {object} ProjectResponse "Successfully created project"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /v1/organization/projects [post]
func (api *ProjectsRoute) CreateProject(reqCtx *gin.Context) {
	projectService := api.projectService
	ctx := reqCtx.Request.Context()

	var requestPayload CreateProjectRequest
	if err := reqCtx.ShouldBindJSON(&requestPayload); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "db8142f8-dc78-4581-a238-6e32288a54ec",
			Error: err.Error(),
		})
		return
	}

	adminKey, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	projectEntity, err := projectService.CreateProjectWithPublicID(ctx, &project.Project{
		Name:           requestPayload.Name,
		OrganizationID: *adminKey.OrganizationID,
		Status:         string(project.ProjectStatusActive),
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "e00e6ab3-1b43-490e-90df-aae030697f74",
			Error: err.Error(),
		})
		return
	}

	response := domainToProjectResponse(projectEntity)
	reqCtx.JSON(http.StatusOK, response)
}

// GetProject godoc
// @Summary Get Project
// @Description Retrieves a specific project by its ID.
// @Tags Platform, Platform-Organizations
// @Security BearerAuth
// @Param Authorization header string true "Bearer token" default("Bearer <api_key>")
// @Param project_id path string true "ID of the project"
// @Success 200 {object} ProjectResponse "Successfully retrieved the project"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - project with the given ID does not exist or does not belong to the organization"
// @Router /v1/organization/projects/{project_id} [get]
func (api *ProjectsRoute) GetProject(reqCtx *gin.Context) {
	projectService := api.projectService
	ctx := reqCtx.Request.Context()

	adminKey, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	projectID := reqCtx.Param("project_id")
	if projectID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "e8100503-698f-4ee6-8f5c-3274f5476e67",
			Error: "invalid or missing project ID",
		})
		return
	}

	entity, err := projectService.FindProjectByPublicID(ctx, projectID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "42ad3a04-6c17-40db-a10f-640be569c93f",
			Error: "project not found",
		})
		return
	}

	if entity.OrganizationID != *adminKey.OrganizationID {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "752f93d3-21f1-45a3-ba13-0157d069aca2",
			Error: "project not found in organization",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, domainToProjectResponse(entity))
}

// UpdateProject godoc
// @Summary Update Project
// @Description Updates a specific project by its ID.
// @Tags Platform, Platform-Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer token" default("Bearer <api_key>")
// @Param project_id path string true "ID of the project to update"
// @Param body body UpdateProjectRequest true "Project update request"
// @Success 200 {object} ProjectResponse "Successfully updated the project"
// @Failure 400 {object} responses.ErrorResponse "Bad request - invalid payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - project with the given ID does not exist"
// @Router /v1/organization/projects/{project_id} [post]
func (api *ProjectsRoute) UpdateProject(reqCtx *gin.Context) {
	projectService := api.projectService
	ctx := reqCtx.Request.Context()

	var requestPayload UpdateProjectRequest
	if err := reqCtx.ShouldBindJSON(&requestPayload); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b6cb35be-8a53-478d-95d1-5e1f64f35c09",
			Error: err.Error(),
		})
		return
	}

	adminKey, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	projectID := reqCtx.Param("project_id")
	if projectID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "a50a180b-75ab-4292-9397-781f66f1502d",
			Error: "invalid or missing project ID",
		})
		return
	}

	entity, err := projectService.FindProjectByPublicID(ctx, projectID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "4ee156ce-6425-425b-b9fd-d95165456b6c",
			Error: "project not found",
		})
		return
	}

	if entity.OrganizationID != *adminKey.OrganizationID {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "bd69b6c5-2a54-421e-b3b9-b740d4a92f19",
			Error: "project not found in organization",
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
// @Tags Platform, Platform-Organizations
// @Security BearerAuth
// @Param Authorization header string true "Bearer token" default("Bearer <api_key>")
// @Param project_id path string true "ID of the project to archive"
// @Success 200 {object} ProjectResponse "Successfully archived the project"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Not Found - project with the given ID does not exist"
// @Router /v1/organization/projects/{project_id}/archive [post]
func (api *ProjectsRoute) ArchiveProject(reqCtx *gin.Context) {
	projectService := api.projectService
	ctx := reqCtx.Request.Context()

	adminKey, err := api.validateAdminKey(reqCtx)
	if err != nil {
		return
	}

	projectID := reqCtx.Param("project_id")
	if projectID == "" {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "2ab393c5-708d-42bc-a785-dcbdcf429ad1",
			Error: "invalid or missing project ID",
		})
		return
	}

	entity, err := projectService.FindProjectByPublicID(ctx, projectID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "26b68f41-0eb0-4fca-8365-613742ef9204",
			Error: "project not found",
		})
		return
	}

	if entity.OrganizationID != *adminKey.OrganizationID {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "4b656858-4212-451a-9ab6-23bc09dcc357",
			Error: "project not found in organization",
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

// TODO: move to middleware
func (api *ProjectsRoute) validateAdminKey(reqCtx *gin.Context) (*apikey.ApiKey, error) {
	apikeyService := api.apiKeyService
	ctx := reqCtx.Request.Context()
	adminKey, ok := requests.GetTokenFromBearer(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "704ff768-2681-4ba0-bc6b-600c6f10df25",
			Error: "invalid or missing API key",
		})
		return nil, fmt.Errorf("invalid token")
	}

	// Verify the provided admin API key
	adminKeyEntity, err := apikeyService.FindByKey(ctx, adminKey)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "af7eb57f-8a57-45c8-8ad7-e60c1c68cb6f",
			Error: "invalid or missing API key",
		})
		return nil, err
	}

	if adminKeyEntity.ApikeyType != string(apikey.ApikeyTypeAdmin) {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "2f83ee2f-3054-40de-afd0-82f06d3fb6cb",
			Error: "invalid or missing API key",
		})
		return nil, fmt.Errorf("invalid or missing API key")
	}
	return adminKeyEntity, nil
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
