package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	mcpimpl "menlo.ai/jan-api-gateway/app/interfaces/http/routes/v1/mcp/mcp_impl"

	mcpgolang "github.com/metoro-io/mcp-golang"
	mcpHttp "github.com/metoro-io/mcp-golang/transport/http"
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
		log.Println("Incoming method:", req.Method)

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
		c.Next()
	}
}

type MCPAPI struct {
	MCPTransport     *mcpHttp.GinTransport
	MCPServerHandler *mcpgolang.Server
	SerperMCP        *mcpimpl.SerperMCP
}

func NewMCPAPI(serperMCP *mcpimpl.SerperMCP) *MCPAPI {
	transport := mcpHttp.NewGinTransport()
	mcpServer := mcpgolang.NewServer(transport)
	return &MCPAPI{
		MCPTransport:     transport,
		MCPServerHandler: mcpServer,
		SerperMCP:        serperMCP,
	}
}

func (mcpAPI *MCPAPI) RegisterRouter(router *gin.RouterGroup) {
	mcpAPI.SerperMCP.RegisterTool(mcpAPI.MCPServerHandler)
	mcpAPI.MCPServerHandler.Serve()
	router.POST(
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
		mcpAPI.MCPTransport.Handler())
}
