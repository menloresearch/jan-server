package chat

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
)

type ChatRoute struct {
	authService   *auth.AuthService
	completionAPI *CompletionAPI
}

func NewChatRoute(
	authService *auth.AuthService,
	completionAPI *CompletionAPI,
) *ChatRoute {
	return &ChatRoute{
		authService:   authService,
		completionAPI: completionAPI,
	}
}

func (chatRoute *ChatRoute) RegisterRouter(router gin.IRouter) {
	// Register /v1/chat routes
	chatRouter := router.Group("/chat",
		chatRoute.authService.AppUserAuthMiddleware(),
		chatRoute.authService.RegisteredUserMiddleware(),
	)
	chatRoute.completionAPI.RegisterRouter(chatRouter)
}
