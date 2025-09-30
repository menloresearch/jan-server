package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/admin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/conv"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/conversations"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/organization"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/responses"
	"menlo.ai/jan-api-gateway/config"
)

type V1Route struct {
	organizationRoute *organization.OrganizationRoute
	adminRoute        *admin.CacheRoute
	chatRoute         *chat.ChatRoute
	convChatRoute     *conv.ConvChatRoute
	conversationAPI   *conversations.ConversationAPI
	modelAPI          *ModelAPI
	mcpAPI            *mcp.MCPAPI
	authRoute         *auth.AuthRoute
	responsesRoute    *responses.ResponseRoute
}

func NewV1Route(
	organizationRoute *organization.OrganizationRoute,
	adminRoute *admin.CacheRoute,
	chatRoute *chat.ChatRoute,
	convChatRoute *conv.ConvChatRoute,
	conversationAPI *conversations.ConversationAPI,
	modelAPI *ModelAPI,
	mcpAPI *mcp.MCPAPI,
	authRoute *auth.AuthRoute,
	responsesRoute *responses.ResponseRoute,
) *V1Route {
	return &V1Route{
		organizationRoute: organizationRoute,
		adminRoute:        adminRoute,
		chatRoute:         chatRoute,
		convChatRoute:     convChatRoute,
		conversationAPI:   conversationAPI,
		modelAPI:          modelAPI,
		mcpAPI:            mcpAPI,
		authRoute:         authRoute,
		responsesRoute:    responsesRoute,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Router.GET("/version", GetVersion)
	v1Route.chatRoute.RegisterRouter(v1Router)
	v1Route.convChatRoute.RegisterRouter(v1Router)
	v1Route.conversationAPI.RegisterRouter(v1Router)
	v1Route.modelAPI.RegisterRouter(v1Router)
	v1Route.mcpAPI.RegisterRouter(v1Router)
	v1Route.organizationRoute.RegisterRouter(v1Router)
	v1Route.adminRoute.RegisterRouter(v1Router)
	v1Route.authRoute.RegisterRouter(v1Router)
	v1Route.responsesRoute.RegisterRouter(v1Router)
}

// GetVersion godoc
// @Summary     Get API build version
// @Description Returns the current build version of the API server.
// @Tags        Server API
// @Produce     json
// @Success     200 {object} map[string]string "version info"
// @Router      /v1/version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":         config.Version,
		"env_reloaded_at": config.EnvReloadedAt,
	})
}
