package organization

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
)

type OrganizationRoute struct {
	adminApiKeyAPI *AdminApiKeyAPI
	projectsRoute  *projects.ProjectsRoute
	userService    *user.UserService
}

func NewOrganizationRoute(adminApiKeyAPI *AdminApiKeyAPI, projectsRoute *projects.ProjectsRoute, userService *user.UserService) *OrganizationRoute {
	return &OrganizationRoute{
		adminApiKeyAPI,
		projectsRoute,
		userService,
	}
}

func (organizationRoute *OrganizationRoute) RegisterRouter(router gin.IRouter) {
	organizationRouter := router.Group("/organization",
		organizationRoute.userService.AdminUserAuthMiddleware(),
		organizationRoute.userService.RegisteredUserMiddleware(),
	)
	// TODO: Access via admin key instead of API key.
	organizationRoute.adminApiKeyAPI.RegisterRouter(organizationRouter)
	organizationRoute.projectsRoute.RegisterRouter(organizationRouter)
}
