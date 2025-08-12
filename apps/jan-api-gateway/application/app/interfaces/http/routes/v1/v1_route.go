package v1

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
)

type V1Route struct {
	chatRoute *chat.ChatRoute
	modelAPI  *ModelAPI
}

func NewV1Route(
	chatRoute *chat.ChatRoute,
	modelAPI *ModelAPI,
) *V1Route {
	return &V1Route{
		chatRoute,
		modelAPI,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Route.chatRoute.RegisterRouter(v1Router)
	v1Route.modelAPI.RegisterRouter(v1Router)
}
