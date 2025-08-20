package v1

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/apikeys"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/chat"
)

type V1Route struct {
	auth    *auth.AuthRoute
	apikeys *apikeys.ApiKeyAPI
	chat    *chat.ChatRoute
}

func NewV1Route(auth *auth.AuthRoute, apikeys *apikeys.ApiKeyAPI, chat *chat.ChatRoute) *V1Route {
	return &V1Route{
		auth,
		apikeys,
		chat,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Route.auth.RegisterRouter(v1Router)
	v1Route.apikeys.RegisterRouter(v1Router)
	v1Route.chat.RegisterRouter(v1Router)
}
