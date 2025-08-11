package chat

import "github.com/gin-gonic/gin"

type ChatRoute struct {
	completionAPI *CompletionAPI
}

func NewChatRoute(
	completionAPI *CompletionAPI,
) *ChatRoute {
	return &ChatRoute{
		completionAPI,
	}
}

func (chatRoute *ChatRoute) RegisterRouter(router gin.IRouter) {
	chatRouter := router.Group("/chat")
	chatRoute.completionAPI.RegisterRouter(chatRouter)
}
