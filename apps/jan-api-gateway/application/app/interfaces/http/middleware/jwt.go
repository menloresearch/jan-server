package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "55312c8d-4fa4-4ecf-a0a2-6fee16c8d7e0",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "c6d6bafd-b9f3-4ebb-9c90-a21b07308ebc",
			})
			return
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, &auth.UserClaim{}, func(token *jwt.Token) (interface{}, error) {
			return environment_variables.EnvironmentVariables.JWT_SECRET, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "9d7a21c4-d94c-4451-841b-4d9333f86942",
			})
			return
		}

		claims, ok := token.Claims.(*auth.UserClaim)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "6cc0aa26-148d-4b8d-8f53-9d47b2a00ef1",
			})
			return
		}

		c.Set(auth.ContextUserClaim, claims)
		c.Next()
	}
}
