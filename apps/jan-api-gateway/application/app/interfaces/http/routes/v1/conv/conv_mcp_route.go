package conv

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	mcpimpl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"
)

// ConvMCPAPI handles MCP (Model Context Protocol) endpoints for conversation-aware chat
type ConvMCPAPI struct {
	authService *auth.AuthService
	serperMCP   *mcpimpl.SerperMCP
	mcpServer   *mcpserver.MCPServer
}

// NewConvMCPAPI creates a new ConvMCPAPI instance
func NewConvMCPAPI(authService *auth.AuthService, serperMCP *mcpimpl.SerperMCP) *ConvMCPAPI {
	mcpSrv := mcpserver.NewMCPServer("conv-mcp-demo", "0.1.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithRecovery(),
	)
	return &ConvMCPAPI{
		authService: authService,
		serperMCP:   serperMCP,
		mcpServer:   mcpSrv,
	}
}

// RegisterRouter registers the MCP routes for conversation-aware chat
// ConvMCP
// @Summary MCP streamable endpoint for conversation-aware chat
// @Description Handles Model Context Protocol (MCP) requests over an HTTP stream for conversation-aware chat functionality. The response is sent as a continuous stream of data with conversation context.
// @Tags Conversation-aware Chat API
// @Security BearerAuth
// @Accept json
// @Produce text/event-stream
// @Param request body any true "MCP request payload"
// @Success 200 {string} string "Streamed response (SSE or chunked transfer)"
// @Router /v1/conv/mcp [post]
func (api *ConvMCPAPI) RegisterRouter(router *gin.RouterGroup) {
	// Register MCP endpoint (without RegisteredUserMiddleware to avoid content type conflicts)
	api.serperMCP.RegisterTool(api.mcpServer)
	mcpHttpHandler := mcpserver.NewStreamableHTTPServer(api.mcpServer)

	// Create a separate router group for MCP that only uses AppUserAuthMiddleware
	// This avoids the content type conflicts that RegisteredUserMiddleware can cause
	mcpRouter := router.Group("")
	mcpRouter.Any(
		"/mcp",
		api.authService.AppUserAuthMiddleware(),
		MCPMethodGuard(map[string]bool{
			// Initialization / handshake
			"initialize":                true,
			"notifications/initialized": true,
			"ping":                      true,

			// Tools
			"tools/list": true,
			"tools/call": true,

			// Prompts
			"prompts/list": true,
			"prompts/call": true,

			// Resources
			"resources/list":           true,
			"resources/templates/list": true,
			"resources/read":           true,

			// If you support subscription:
			"resources/subscribe": true,
		}),
		gin.WrapH(mcpHttpHandler))
}

// MCPMethodGuard is a middleware that guards MCP methods
func MCPMethodGuard(allowedMethods map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		var req struct {
			Method string `json:"method"`
		}

		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			c.Abort()
			return
		}

		if !allowedMethods[req.Method] {
			c.Abort()
			return
		}
		c.Next()
	}
}
