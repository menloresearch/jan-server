package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/chat"
	"menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp"
	"menlo.ai/jan-api-gateway/config"
)

type V1Route struct {
	chatRoute *chat.ChatRoute
	modelAPI  *ModelAPI
	mcpAPI    *mcp.MCPAPI
}

func NewV1Route(
	chatRoute *chat.ChatRoute,
	modelAPI *ModelAPI,
	mcpAPI *mcp.MCPAPI,
) *V1Route {
	return &V1Route{
		chatRoute,
		modelAPI,
		mcpAPI,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Router.GET("/version", GetVersion)
	v1Route.chatRoute.RegisterRouter(v1Router)
	v1Route.modelAPI.RegisterRouter(v1Router)
	v1Route.mcpAPI.RegisterRouter(v1Router)
}

// GetVersion godoc
// @Summary     Get API build version
// @Description Returns the current build version of the API server.
// @Tags        system
// @Produce     json
// @Success     200 {object} map[string]string "version info"
// @Router      /v1/version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": config.Version,
	})
}
