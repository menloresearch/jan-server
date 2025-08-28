package organization

import (
	"github.com/gin-gonic/gin"
)

type OrganizationRoute struct {
	adminApiKeyAPI *AdminApiKeyAPI
}

func NewOrganizationRoute(adminApiKeyAPI *AdminApiKeyAPI) *OrganizationRoute {
	return &OrganizationRoute{
		adminApiKeyAPI,
	}
}

func (organizationRoute *OrganizationRoute) RegisterRouter(router gin.IRouter) {
	organizationRouter := router.Group("/organization")
	organizationRoute.adminApiKeyAPI.RegisterRouter(organizationRouter)
}
