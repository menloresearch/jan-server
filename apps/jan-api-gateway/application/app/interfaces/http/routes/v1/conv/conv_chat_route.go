package conv

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
)

type ConvChatRoute struct {
	authService       *auth.AuthService
	convCompletionAPI *ConvCompletionAPI
}

func NewConvChatRoute(
	authService *auth.AuthService,
	convCompletionAPI *ConvCompletionAPI,
) *ConvChatRoute {
	return &ConvChatRoute{
		authService:       authService,
		convCompletionAPI: convCompletionAPI,
	}
}

func (convChatRoute *ConvChatRoute) RegisterRouter(router gin.IRouter) {
	// Register /v1/conv/chat routes
	convChatRouter := router.Group("/conv/chat",
		convChatRoute.authService.AppUserAuthMiddleware(),
		convChatRoute.authService.RegisteredUserMiddleware(),
	)
	convChatRoute.convCompletionAPI.RegisterRouter(convChatRouter)
}
