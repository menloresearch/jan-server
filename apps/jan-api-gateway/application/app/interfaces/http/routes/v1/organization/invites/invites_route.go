package invites

import (
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
func (api *InvitesRoute) CreateInvite(reqCtx *gin.Context) {

}
func (api *InvitesRoute) RetrieveInvite(reqCtx *gin.Context) {

}
func (api *InvitesRoute) DeleteInvite(reqCtx *gin.Context) {

}
