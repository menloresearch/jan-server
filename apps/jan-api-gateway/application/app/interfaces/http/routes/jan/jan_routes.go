package jan

import (
	"github.com/gin-gonic/gin"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/chat"
	"menlo.ai/jan-api-gateway/config/environment_variables"
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
	if environment_variables.EnvironmentVariables.ENABLE_ADMIN_API {
		janRouter := router.Group("/jan")
		janRoute.v1Route.RegisterRouter(janRouter)
	}
}
