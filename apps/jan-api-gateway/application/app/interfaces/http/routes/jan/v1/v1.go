package v1

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/conversations"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization"
)

type V1Route struct {
	auth          *auth.AuthRoute
	chat          *chat.ChatRoute
	conversations *conversations.ConversationAPI
	organizations *organization.OrganizationRoute
}

func NewV1Route(
	auth *auth.AuthRoute,
	chat *chat.ChatRoute,
	conversations *conversations.ConversationAPI,
	organizations *organization.OrganizationRoute) *V1Route {
	return &V1Route{
		auth,
		chat,
		conversations,
		organizations,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Route.auth.RegisterRouter(v1Router)
	v1Route.chat.RegisterRouter(v1Router)
	v1Route.conversations.RegisterRouter(v1Router)
	v1Route.organizations.RegisterRouter(v1Router)
}
