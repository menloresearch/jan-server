package conv

import (
	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
)

// ConvChatRoute handles conversation-aware chat completion routes
// This route provides chat completion functionality with conversation persistence,
// history management, and extended features like storage and reasoning.
type ConvChatRoute struct {
	authService       *auth.AuthService
	convCompletionAPI *ConvCompletionAPI
	convMCPAPI        *ConvMCPAPI
}

// NewConvChatRoute creates a new conversation-aware chat route handler
func NewConvChatRoute(
	authService *auth.AuthService,
	convCompletionAPI *ConvCompletionAPI,
	convMCPAPI *ConvMCPAPI,
) *ConvChatRoute {
	return &ConvChatRoute{
		authService:       authService,
		convCompletionAPI: convCompletionAPI,
		convMCPAPI:        convMCPAPI,
	}
}

// RegisterRouter registers the conversation-aware chat completion routes
// This creates the /v1/conv/completions endpoint with authentication middleware
func (convChatRoute *ConvChatRoute) RegisterRouter(router gin.IRouter) {
	// Register /v1/conv routes with authentication middleware
	convChatRouter := router.Group("/conv",
		convChatRoute.authService.AppUserAuthMiddleware(),
		convChatRoute.authService.RegisteredUserMiddleware(),
	)
	convChatRoute.convCompletionAPI.RegisterRouter(convChatRouter)

	// Register MCP routes separately (without RegisteredUserMiddleware to avoid content type conflicts)
	convChatRoute.convMCPAPI.RegisterRouter(convChatRouter)
}
