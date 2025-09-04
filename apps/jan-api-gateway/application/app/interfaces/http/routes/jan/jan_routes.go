package jan

import (
	"github.com/gin-gonic/gin"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/chat"
)

type JanRoute struct {
	v1Route   *v1.V1Route
	chatRoute *chat.ChatRoute
}

func NewJanRoute(v1Route *v1.V1Route, chatRoute *chat.ChatRoute) *JanRoute {
	return &JanRoute{
		v1Route,
		chatRoute,
	}
}

func (janRoute *JanRoute) RegisterRouter(router gin.IRouter) {
	janRouter := router.Group("/jan")
	janRoute.v1Route.RegisterRouter(janRouter)
}
