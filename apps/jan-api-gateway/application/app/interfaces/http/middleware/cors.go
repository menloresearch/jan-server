package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Header.Get("Origin")
		isValidHost := false
		for _, allowedHost := range environment_variables.EnvironmentVariables.ALLOWED_CORS_HOSTS {
			if allowedHost == host {
				isValidHost = true
				break
			}
		}
		if isValidHost {
			c.Writer.Header().Set("Access-Control-Allow-Origin", host)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
