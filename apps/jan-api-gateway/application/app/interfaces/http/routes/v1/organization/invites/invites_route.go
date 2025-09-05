package invites

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"menlo.ai/jan-api-gateway/app/domain/invite"
	"menlo.ai/jan-api-gateway/app/domain/query"
	apikeyHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/apikey"
	inviteHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/invite"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type InvitesRoute struct {
	apiKeyHandler *apikeyHandler.ApiKeyHandler
	inviteHandler *inviteHandler.InviteHandler
}

func NewInvitesRoute(apiKeyHandler *apikeyHandler.ApiKeyHandler, inviteHandler *inviteHandler.InviteHandler) *InvitesRoute {
	return &InvitesRoute{
		apiKeyHandler,
		inviteHandler,
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

type InviteListResponse struct {
	Object  string           `json:"object" example:"list" description:"The type of the object, 'list'"`
	Data    []InviteResponse `json:"data" description:"Array of invites"`
	FirstID *string          `json:"first_id,omitempty"`
	LastID  *string          `json:"last_id,omitempty"`
	HasMore bool             `json:"has_more"`
}

func (inviteRoute *InvitesRoute) RegisterRouter(router gin.IRouter) {
	inviteRouter := router.Group("/invites", inviteRoute.apiKeyHandler.AdminApiKeyMiddleware())
	inviteRouter.GET("", inviteRoute.ListInvites)
	inviteRouter.POST("", inviteRoute.CreateInvite)
	inviteRouter.GET(fmt.Sprintf("/:%s", inviteHandler.InviteContextKeyPublicID), inviteRoute.RetrieveInvite)
	inviteRouter.DELETE(fmt.Sprintf("/:%s", inviteHandler.InviteContextKeyPublicID), inviteRoute.RetrieveInvite)
}

func (api *InvitesRoute) ListInvites(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	adminKey, ok := api.apiKeyHandler.GetApiKeyFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "d16722a1-97ca-46b4-812a-678d25e47ef8",
		})
		return
	}
	pagination, err := query.GetPaginationFromQuery(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "321028be-970b-4dc2-92ed-11ce63284f08",
		})
		return
	}
	filter := invite.InvitesFilter{
		OrganizationID: &adminKey.ID,
	}
	inviteEntities, err := api.inviteHandler.ListInvites(ctx, filter, pagination)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "1f79e6aa-a25b-44af-bf9e-b9fbb6e1ceab",
		})
		return
	}
	var firstId *string
	var lastId *string
	hasMore := false

	if len(inviteEntities) > 0 {
		firstId = &inviteEntities[0].PublicID
		lastId = &inviteEntities[len(inviteEntities)-1].PublicID
		moreRecords, err := api.inviteHandler.ListInvites(ctx, filter, &query.Pagination{
			Order: pagination.Order,
			Limit: ptr.ToInt(1),
			After: &inviteEntities[len(inviteEntities)-1].ID,
		})
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "a34720d0-fe4f-4586-aff2-564e060faa99",
				Error: "failed to retrieve Invites",
			})
			return
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}

	reqCtx.JSON(http.StatusOK, InviteListResponse{
		Object:  "list",
		LastID:  lastId,
		FirstID: firstId,
		HasMore: hasMore,
		Data: functional.Map(inviteEntities, func(item *invite.Invite) InviteResponse {
			return InviteResponse{
				Object:     "organization.invite",
				ID:         item.PublicID,
				Email:      item.Email,
				Role:       item.Role,
				Status:     item.Status,
				InvitedAt:  item.InvitedAt,
				AcceptedAt: item.AcceptedAt,
				ExpiresAt:  item.ExpiresAt,
			}
		}),
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

func (api *InvitesRoute) CreateInvite(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	adminKey, ok := api.apiKeyHandler.GetApiKeyFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "d16722a1-97ca-46b4-812a-678d25e47ef8",
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

	ok, err := api.inviteHandler.VerifyUserInvited(ctx, requestPayload.Email, *adminKey.OrganizationID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "398c1de0-1a9f-47e2-8f56-c06e4510f884",
			Error: err.Error(),
		})
		return
	}
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "ac130c69-e9fd-4dfc-b246-4c6abfa44bbe",
		})
		return
	}
	projectIDs := functional.Map(requestPayload.Projects, func(proj InviteProject) string {
		return proj.ID
	})

	ok, err = api.inviteHandler.VerifyProjects(ctx, functional.Distinct[string](projectIDs))
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "ea649ae7-d82c-48b2-9ef1-626c139f180d",
			Error: err.Error(),
		})
		return
	}
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "a08c5ee3-651e-4465-a7c9-5009fec9d5c2",
		})
		return
	}

	// TODO: send a email here
	projectsStr, err := json.Marshal(requestPayload.Projects)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "f7957c66-77d6-494f-9ee9-8fa54408a604",
		})
		return
	}

	inviteEntity, err := api.inviteHandler.CreateInvite(ctx, &invite.Invite{
		Email:          requestPayload.Email,
		Role:           requestPayload.Role,
		Status:         string(invite.InviteStatusPending),
		OrganizationID: *adminKey.OrganizationID,
		Projects:       string(projectsStr),
	})

	reqCtx.JSON(http.StatusOK, InviteResponse{})
}
func (api *InvitesRoute) RetrieveInvite(reqCtx *gin.Context) {

}
func (api *InvitesRoute) DeleteInvite(reqCtx *gin.Context) {

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
