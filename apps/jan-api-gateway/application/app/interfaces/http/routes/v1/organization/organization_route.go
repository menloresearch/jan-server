package organization

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/invites"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/models"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization/projects"
)

type OrganizationRoute struct {
	adminApiKeyAPI *AdminApiKeyAPI
	projectsRoute  *projects.ProjectsRoute
	inviteRoute    *invites.InvitesRoute
	modelsAPI      *models.ModelsAPI
	kubernetesAPI  *models.KubernetesAPI
	authService    *auth.AuthService
}

func NewOrganizationRoute(
	adminApiKeyAPI *AdminApiKeyAPI,
	projectsRoute *projects.ProjectsRoute,
	inviteRoute *invites.InvitesRoute,
	modelsAPI *models.ModelsAPI,
	kubernetesAPI *models.KubernetesAPI,
	authService *auth.AuthService,
) *OrganizationRoute {
	return &OrganizationRoute{
		adminApiKeyAPI,
		projectsRoute,
		inviteRoute,
		modelsAPI,
		kubernetesAPI,
		authService,
	}
}

func (organizationRoute *OrganizationRoute) RegisterRouter(router gin.IRouter) {
	organizationRouter := router.Group("/organization")
	organizationRoute.adminApiKeyAPI.RegisterRouter(organizationRouter)
	organizationRoute.projectsRoute.RegisterRouter(organizationRouter)
	organizationRoute.inviteRoute.RegisterRouter(organizationRouter)
	organizationRoute.modelsAPI.RegisterRouter(organizationRouter)
	organizationRoute.kubernetesAPI.RegisterRouter(organizationRouter)
}
