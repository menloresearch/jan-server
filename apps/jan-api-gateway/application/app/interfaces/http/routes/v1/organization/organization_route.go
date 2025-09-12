package organization

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
)

type OrganizationRoute struct {
	adminApiKeyAPI *AdminApiKeyAPI
	projectsRoute  *projects.ProjectsRoute
	authService    *auth.AuthService
}

func NewOrganizationRoute(adminApiKeyAPI *AdminApiKeyAPI, projectsRoute *projects.ProjectsRoute, authService *auth.AuthService) *OrganizationRoute {
	return &OrganizationRoute{
		adminApiKeyAPI,
		projectsRoute,
		authService,
	}
}

func (organizationRoute *OrganizationRoute) RegisterRouter(router gin.IRouter) {
	organizationRouter := router.Group("/organization",
		organizationRoute.authService.AdminUserAuthMiddleware(),
		organizationRoute.authService.RegisteredUserMiddleware(),
	)
	// TODO: Access via admin key instead of API key.
	organizationRoute.adminApiKeyAPI.RegisterRouter(organizationRouter)
	organizationRoute.projectsRoute.RegisterRouter(organizationRouter)
}
