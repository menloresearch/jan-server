package v1

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan-platform/v1/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan-platform/v1/organization"
)

type V1Route struct {
	auth         *auth.AuthRoute
	organization *organization.OrganizationRoute
}

func NewV1Route(
	auth *auth.AuthRoute,
	organization *organization.OrganizationRoute) *V1Route {
	return &V1Route{
		auth:         auth,
		organization: organization,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/jan-platform/v1")
	v1Route.auth.RegisterRouter(v1Router)
	v1Route.organization.RegisterRouter(v1Router)
}
