package v1

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1/apikeys"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1/auth"
)

type V1Route struct {
	auth    *auth.AuthRoute
	apikeys *apikeys.ApiKeyAPI
}

func NewV1Route(auth *auth.AuthRoute, apikeys *apikeys.ApiKeyAPI) *V1Route {
	return &V1Route{
		auth,
		apikeys,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Route.auth.RegisterRouter(v1Router)
	v1Route.apikeys.RegisterRouter(v1Router)
}
