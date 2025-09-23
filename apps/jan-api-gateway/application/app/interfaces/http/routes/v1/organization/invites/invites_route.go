package invites

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/invite"
	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/config/environment_variables"

	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses/openai"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type InvitesRoute struct {
	inviteService  *invite.InviteService
	projectService *project.ProjectService
	authService    *auth.AuthService
}

func NewInvitesRoute(
	inviteService *invite.InviteService,
	projectService *project.ProjectService,
	authService *auth.AuthService,
) *InvitesRoute {
	return &InvitesRoute{
		inviteService,
		projectService,
		authService,
	}
}

type InviteResponse struct {
	Object     string     `json:"object"`
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	Status     string     `json:"status"`
	InvitedAt  time.Time  `json:"invited_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	Projects   []InviteProject
}

func (inviteRoute *InvitesRoute) RegisterRouter(router gin.IRouter) {
	permissionAll := inviteRoute.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleAll)
	permissionOwnerOnly := inviteRoute.authService.OrganizationMemberRoleMiddleware(auth.OrganizationMemberRuleOwnerOnly)
	inviteRouter := router.Group(
		"/invites",
		inviteRoute.authService.AdminUserAuthMiddleware(),
		inviteRoute.authService.RegisteredUserMiddleware(),
	)
	inviteRouter.POST("",
		permissionOwnerOnly,
		inviteRoute.CreateInvite,
	)
	inviteRouter.GET(
		"",
		permissionAll,
		inviteRoute.ListInvites,
	)
	inviteIdRoute := inviteRouter.Group(fmt.Sprintf("/:%s", auth.InviteContextKeyPublicID), inviteRoute.authService.AdminInviteMiddleware())
	inviteIdRoute.GET("",
		permissionAll,
		inviteRoute.RetrieveInvite)
	inviteIdRoute.DELETE("",
		permissionOwnerOnly,
		inviteRoute.DeleteInvite,
	)
}

// ListInvites godoc
// @Summary List Organization Invites
// @Description Retrieves a paginated list of invites for the current organization.
// @Tags Administration API
// @Security BearerAuth
// @Param after query string false "Cursor pointing to a record after which to fetch results"
// @Param limit query int false "Maximum number of results to return"
// @Success 200 {object} openai.ListResponse[InviteResponse] "Successfully retrieved list of invites"
// @Failure 400 {object} responses.ErrorResponse "Invalid or missing query parameter"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/invites [get]
func (api *InvitesRoute) ListInvites(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		return
	}
	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(after string) (*uint, error) {
		entity, err := api.inviteService.FindOne(ctx, invite.InvitesFilter{
			PublicID: &after,
		})
		if err != nil {
			return nil, err
		}
		if entity == nil {
			return nil, fmt.Errorf("record not found")
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

	filter := invite.InvitesFilter{
		OrganizationID: &orgEntity.ID,
	}
	inviteEntities, err := api.inviteService.FindInvites(ctx, filter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "1f79e6aa-a25b-44af-bf9e-b9fbb6e1ceab",
		})
		return
	}
	pageCursor, err := responses.BuildCursorPage(
		inviteEntities,
		func(t *invite.Invite) *string {
			return &t.PublicID
		},
		func() ([]*invite.Invite, error) {
			return api.inviteService.FindInvites(ctx, filter, &query.Pagination{
				Order: pagination.Order,
				Limit: ptr.ToInt(1),
				After: &inviteEntities[len(inviteEntities)-1].ID,
			})
		},
		func() (int64, error) {
			return api.inviteService.CountInvites(ctx, filter)
		},
	)
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "59c1efc5-d6a1-4da1-baf8-d7ed0497e088",
			ErrorInstance: err,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, openai.ListResponse[InviteResponse]{
		Object:  "list",
		LastID:  pageCursor.LastID,
		FirstID: pageCursor.FirstID,
		HasMore: pageCursor.HasMore,
		Total:   pageCursor.Total,
		Data:    functional.Map(inviteEntities, convertInviteEntityToResponse),
	})
}

type InviteProject struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type CreateInviteUserRequest struct {
	Email    string          `json:"email"`
	Role     string          `json:"role"`
	Projects []InviteProject `json:"projects,omitempty"`
}

// CreateInvite godoc
// @Summary Create Invite
// @Description Creates a new invite for a user to join the organization.
// @Tags Administration API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param invite body CreateInviteUserRequest true "Invite request payload"
// @Success 200 {object} InviteResponse "Successfully created invite"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload or user already exists"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/organization/invites [post]
func (api *InvitesRoute) CreateInvite(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	userEntity, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "0c781396-68a9-4177-97a8-342af883f7c3",
		})
		return
	}
	orgEntity, ok := auth.GetAdminOrganizationFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "470ad74e-f9bc-4e8d-b42b-9d506ff11a0a",
		})
		return
	}
	var requestPayload CreateInviteUserRequest
	if err := reqCtx.ShouldBindJSON(&requestPayload); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "470ad74e-f9bc-4e8d-b42b-9d506ff11a0a",
			Error: err.Error(),
		})
		return
	}

	exists, err := api.authService.HasOrganizationUser(ctx, requestPayload.Email, orgEntity.ID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "398c1de0-1a9f-47e2-8f56-c06e4510f884",
			Error: err.Error(),
		})
		return
	}
	if exists {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "ac130c69-e9fd-4dfc-b246-4c6abfa44bbe",
		})
		return
	}
	projectIDs := functional.Map(requestPayload.Projects, func(proj InviteProject) string {
		return proj.ID
	})

	if len(projectIDs) > 0 {
		projects, err := api.projectService.Find(ctx, project.ProjectFilter{
			PublicIDs: &projectIDs,
		}, nil)

		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:          "ea649ae7-d82c-48b2-9ef1-626c139f180d",
				ErrorInstance: err,
			})
			return
		}
		if len(projects) != len(projectIDs) {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code: "a08c5ee3-651e-4465-a7c9-5009fec9d5c2",
			})
			return
		}
	}

	projectsStr, err := json.Marshal(requestPayload.Projects)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "f7957c66-77d6-494f-9ee9-8fa54408a604",
		})
		return
	}

	inviteEntity, err := api.inviteService.CreateInviteWithPublicID(ctx, &invite.Invite{
		Email:          requestPayload.Email,
		Role:           requestPayload.Role,
		Status:         string(invite.InviteStatusPending),
		OrganizationID: orgEntity.ID,
		Projects:       string(projectsStr),
		Secrets:        ptr.ToString(uuid.New().String()),
	})

	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "f7957c66-77d6-494f-9ee9-8fa54408a604",
		})
		return
	}

	err = api.inviteService.SendInviteEmail(ctx, invite.EmailMetadata{
		InviterEmail: userEntity.Email,
		OrgName:      orgEntity.Name,
		OrgPublicID:  orgEntity.PublicID,
		InviteLink: fmt.Sprintf(
			"%s?code=%s",
			environment_variables.EnvironmentVariables.INVITE_REDIRECT_URL,
			*inviteEntity.Secrets,
		),
	}, requestPayload.Email)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "8432e05b-bc3e-4432-b3cb-ade6353edacc",
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(http.StatusOK, convertInviteEntityToResponse(inviteEntity))
}

// RetrieveInvite godoc
// @Summary Retrieve Invite
// @Description Retrieves a specific invite by its ID.
// @Tags Administration API
// @Security BearerAuth
// @Param invite_id path string true "Public ID of the invite"
// @Success 200 {object} InviteResponse "Successfully retrieved invite"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 404 {object} responses.ErrorResponse "Invite not found"
// @Router /v1/organization/invites/{invite_id} [get]
func (api *InvitesRoute) RetrieveInvite(reqCtx *gin.Context) {
	inviteEntity, ok := auth.GetAdminInviteFromContext(reqCtx)
	if !ok {
		return
	}
	reqCtx.JSON(http.StatusOK, convertInviteEntityToResponse(inviteEntity))
}

// DeleteInvite godoc
// @Summary Delete Invite
// @Description Deletes a specific invite by its ID. Only organization owners can delete invites.
// @Tags Administration API
// @Security BearerAuth
// @Param invite_id path string true "Public ID of the invite"
// @Success 200 {object} openai.DeleteResponse "Successfully deleted invite"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 403 {object} responses.ErrorResponse "Forbidden - only owners can delete invites"
// @Failure 404 {object} responses.ErrorResponse "Invite not found"
// @Router /v1/organization/invites/{invite_id} [delete]
func (api *InvitesRoute) DeleteInvite(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	inviteEntity, ok := auth.GetAdminInviteFromContext(reqCtx)
	if !ok {
		return
	}

	err := api.inviteService.DeleteInviteByID(ctx, inviteEntity.ID)
	if err != nil {
		reqCtx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "ea8900d2-cf26-461a-a985-64760e300be1",
		})
		return
	}

	reqCtx.JSON(http.StatusOK, openai.DeleteResponse{
		Object:  "organization.invite.deleted",
		ID:      inviteEntity.PublicID,
		Deleted: true,
	})
}

func convertInviteEntityToResponse(entity *invite.Invite) InviteResponse {
	projectEntities, err := entity.GetProjects()
	if err != nil {
		projectEntities = make([]invite.InviteProject, 0)
	}
	return InviteResponse{
		Object:     "organization.invite",
		ID:         entity.PublicID,
		Email:      entity.Email,
		Role:       entity.Role,
		Status:     entity.Status,
		InvitedAt:  entity.InvitedAt,
		AcceptedAt: entity.AcceptedAt,
		ExpiresAt:  entity.ExpiresAt,
		Projects: functional.Map(projectEntities, func(item invite.InviteProject) InviteProject {
			return InviteProject{
				Role: item.Role,
				ID:   item.ID,
			}
		}),
	}
}
