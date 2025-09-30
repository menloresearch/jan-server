package organization

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/invites"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/providers"
)

type OrganizationRoute struct {
	adminApiKeyAPI *AdminApiKeyAPI
	projectsRoute  *projects.ProjectsRoute
	inviteRoute    *invites.InvitesRoute
	providerRoute  *providers.OrganizationProviderRoute
	authService    *auth.AuthService
}

func NewOrganizationRoute(adminApiKeyAPI *AdminApiKeyAPI, projectsRoute *projects.ProjectsRoute, inviteRoute *invites.InvitesRoute, providerRoute *providers.OrganizationProviderRoute, authService *auth.AuthService) *OrganizationRoute {
	return &OrganizationRoute{
		adminApiKeyAPI,
		projectsRoute,
		inviteRoute,
		providerRoute,
		authService,
	}
}

func (organizationRoute *OrganizationRoute) RegisterRouter(router gin.IRouter) {
	organizationRouter := router.Group("/organization")
	organizationRoute.adminApiKeyAPI.RegisterRouter(organizationRouter)
	organizationRoute.projectsRoute.RegisterRouter(organizationRouter)
	organizationRoute.inviteRoute.RegisterRouter(organizationRouter)
	organizationRoute.providerRoute.RegisterRouter(organizationRouter)
}
