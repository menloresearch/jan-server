package admin

import (
	"github.com/gin-gonic/gin"
	v1 "menlo.ai/jan-api-gateway/app/interfaces/http/routes/admin/v1"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

type AdminRoute struct {
	v1Route *v1.V1Route
}

func NewAdminRoute(v1Route *v1.V1Route) *AdminRoute {
	return &AdminRoute{
		v1Route,
	}
}

func (adminRoute *AdminRoute) RegisterRouter(router gin.IRouter) {
	if environment_variables.EnvironmentVariables.ENABLE_ADMIN_API {
		adminRouter := router.Group("/admin")
		adminRoute.v1Route.RegisterRouter(adminRouter)
	}
}
