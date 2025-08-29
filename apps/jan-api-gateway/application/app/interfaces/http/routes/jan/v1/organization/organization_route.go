package organization

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/jan/v1/organization/projects"
)

type OrganizationRoute struct {
	organizationService *organization.OrganizationService
	userService         *user.UserService
	projectsRoute       *projects.ProjectsRoute
}

func NewOrganizationRoute(
	organizationService *organization.OrganizationService,
	userService *user.UserService,
	projectsRoute *projects.ProjectsRoute) *OrganizationRoute {
	return &OrganizationRoute{
		organizationService,
		userService,
		projectsRoute,
	}
}

func (o *OrganizationRoute) RegisterRouter(router gin.IRouter) {
	organizationRouter := router.Group("/organizations", middleware.AuthMiddleware())
	o.projectsRoute.RegisterRouter(organizationRouter)
	organizationRouter.GET("")
}

func (o *OrganizationRoute) ListOrganizationRouter(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	userClaim, err := auth.GetUserClaimFromRequestContext(reqCtx)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "9715151d-02ab-4759-bfb7-89d717f05cd3",
			Error: err.Error(),
		})
		return
	}
	user, err := o.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "edf9dd05-aad4-4c1e-9795-98bf60ecf57c",
			Error: err.Error(),
		})
		return
	}
	if user == nil {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "417cff16-0325-45f7-9826-8ab24d2fef29",
		})
		return
	}
}
