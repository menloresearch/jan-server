package invite

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/invite"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type InviteHandler struct {
	inviteService *invite.InviteService
}

func NewInviteHandler(inviteService *invite.InviteService) *InviteHandler {
	return &InviteHandler{
		inviteService,
	}
}

type InviteContextKey string

const (
	InviteContextKeyPublicID InviteContextKey = "invite_public_id"
	InviteContextKeyEntity   InviteContextKey = "InviteContextKeyEntity"
)

func (handler *InviteHandler) ListInvites(ctx context.Context, filter invite.InvitesFilter, pagination *query.Pagination) ([]*invite.Invite, error) {
	return handler.inviteService.FindInvites(ctx, filter, pagination)
}

func (handler *InviteHandler) AdminInviteMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		publicID := reqCtx.Param(string(InviteContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "3bbbadee-a055-473b-a69b-6c00fd31bacc",
				Error: "missing invite public ID",
			})
			return
		}
		inviteEntities, err := handler.inviteService.FindInvites(ctx, invite.InvitesFilter{
			PublicID: &publicID,
		}, nil)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "355b0e80-5b99-47c2-8e9b-9fb4d80e7716",
				Error: err.Error(),
			})
			return
		}
		if len(inviteEntities) != 1 {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code: "3bbbadee-a055-473b-a69b-6c00fd31bacc",
			})
			return
		}
		reqCtx.Set(string(InviteContextKeyEntity), inviteEntities[0])
		reqCtx.Next()
	}
}

func (handler *InviteHandler) GetInviteFromContext(reqCtx *gin.Context) (*invite.Invite, bool) {
	key, ok := reqCtx.Get(string(InviteContextKeyEntity))
	if !ok {
		return nil, false
	}
	return key.(*invite.Invite), true
}
