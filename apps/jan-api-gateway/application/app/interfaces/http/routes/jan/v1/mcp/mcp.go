package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	mcpimpl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"
)

func MCPMethodGuard(allowedMethods map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    -32600,
					"message": "Invalid request body",
				},
				"id": nil,
			})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		var req struct {
			Method string `json:"method"`
		}

		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    -32600,
					"message": "Invalid JSON-RPC request",
				},
				"id": nil,
			})
			c.Abort()
			return
		}

		if !allowedMethods[req.Method] {
			c.JSON(http.StatusOK, gin.H{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    -32601, // Method not found
					"message": fmt.Sprintf("Method '%s' not found", req.Method),
				},
				"id": nil,
			})
			c.Abort()
			return
		}

		if req.Method == "initialize" {
			middleware.AuthMiddleware()(c)
		}

		c.Next()
	}
}

type MCPAPI struct {
	SerperMCP *mcpimpl.SerperMCP
	MCPServer *mcpserver.MCPServer
}

func NewMCPAPI(serperMCP *mcpimpl.SerperMCP) *MCPAPI {
	mcpSrv := mcpserver.NewMCPServer("demo", "0.1.0",
		mcpserver.WithToolCapabilities(true),
	)
	return &MCPAPI{
		SerperMCP: serperMCP,
		MCPServer: mcpSrv,
	}
}

// TODO: move it to mcp handler

// MCPStream
// @Summary MCP streamable endpoint
// @Description Handles Model Context Protocol (MCP) requests over an HTTP stream. The response is sent as a continuous stream of data.
// @Tags Jan, Jan-MCP
// @Accept json
// @Security BearerAuth
// @Produce text/event-stream
// @Param request body any true "MCP request payload"
// @Success 200 {string} string "Streamed response (SSE or chunked transfer)"
// @Router /jan/v1/mcp [post]
func (mcpAPI *MCPAPI) RegisterRouter(router *gin.RouterGroup) {
	mcpAPI.SerperMCP.RegisterTool(mcpAPI.MCPServer)

	mcpHttpHandler := mcpserver.NewStreamableHTTPServer(mcpAPI.MCPServer)
	router.Any(
		"/mcp",
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
