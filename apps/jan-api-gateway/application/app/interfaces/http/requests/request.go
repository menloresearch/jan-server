package requests

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetIntParam(reqCtx *gin.Context, paramName string) (int, error) {
	param := reqCtx.Param(paramName)
	if param == "" {
		return 0, fmt.Errorf("invalid param")
	}
	value, err := strconv.Atoi(param)
	return value, err
}

func GetTokenFromBearer(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	return token, true
}
