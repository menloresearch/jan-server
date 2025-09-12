package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Header.Get("Origin")
		isValidHost := false
		for _, allowedHost := range environment_variables.EnvironmentVariables.ALLOWED_CORS_HOSTS {
			// wildcard
			if strings.HasPrefix(allowedHost, "*") {
				suffix := strings.TrimPrefix(allowedHost, "*")
				if strings.HasSuffix(host, suffix) {
					isValidHost = true
					break
				}
			}
			if allowedHost == host {
				isValidHost = true
				break
			}
		}
		if isValidHost {
			c.Writer.Header().Set("Access-Control-Allow-Origin", host)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, MCP-Protocol-Version, Mcp-Session-Id, X-User-ID, X-User-Email, X-User-Role, MCP-Client-Id, X-Request-Id")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")
			c.Writer.Header().Set("Vary", "Origin")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
