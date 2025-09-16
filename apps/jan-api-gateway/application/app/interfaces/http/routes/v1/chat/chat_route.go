package chat

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
)

type ChatRoute struct {
	completionAPI *CompletionAPI
	authService   *auth.AuthService
}

func NewChatRoute(
	completionAPI *CompletionAPI,
	authService *auth.AuthService,
) *ChatRoute {
	return &ChatRoute{
		completionAPI: completionAPI,
		authService:   authService,
	}
}

func (chatRoute *ChatRoute) RegisterRouter(router gin.IRouter) {
	chatRouter := router.Group("/chat",
		chatRoute.authService.AppUserAuthMiddleware(),
		chatRoute.authService.RegisteredUserMiddleware(),
	)
	chatRoute.completionAPI.RegisterRouter(chatRouter)
}
