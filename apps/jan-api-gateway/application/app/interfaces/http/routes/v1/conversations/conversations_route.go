package conversations

import (
	"github.com/gin-gonic/gin"
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
)

// ConversationAPI handles route registration for V1 conversations
type ConversationAPI struct {
	handler *conversationHandler.ConversationHandler
}

// NewConversationAPI creates a new conversation API instance
func NewConversationAPI(handler *conversationHandler.ConversationHandler) *ConversationAPI {
	return &ConversationAPI{
		handler: handler,
	}
}

// RegisterRouter registers OpenAI-compatible conversation routes
func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations")

	// OpenAI-compatible endpoints only
	conversationsRouter.POST("", api.handler.CreateConversation)
	conversationsRouter.GET("/:conversation_id", api.handler.GetConversation)
	conversationsRouter.POST("/:conversation_id", api.handler.UpdateConversation)
	conversationsRouter.DELETE("/:conversation_id", api.handler.DeleteConversation)
	conversationsRouter.POST("/:conversation_id/items", api.handler.CreateItems)
	conversationsRouter.GET("/:conversation_id/items", api.handler.ListItems)
	conversationsRouter.GET("/:conversation_id/items/:item_id", api.handler.GetItem)
	conversationsRouter.DELETE("/:conversation_id/items/:item_id", api.handler.DeleteItem)
}
