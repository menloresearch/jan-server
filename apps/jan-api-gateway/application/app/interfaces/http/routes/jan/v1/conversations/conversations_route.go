package conversations

import (
	"github.com/gin-gonic/gin"
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
)

// ConversationAPI handles route registration for Jan V1 conversations
type ConversationAPI struct {
	handler *conversationHandler.ConversationHandler
}

// NewConversationAPI creates a new conversation API instance
func NewConversationAPI(handler *conversationHandler.ConversationHandler) *ConversationAPI {
	return &ConversationAPI{
		handler: handler,
	}
}

// RegisterRouter registers Jan-specific conversation routes (currently identical to OpenAI)
func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations")

	// Jan API endpoints (same as OpenAI for now, but can be extended)
	conversationsRouter.POST("", api.handler.CreateConversation)
	conversationsRouter.GET("/:conversation_id", api.handler.GetConversation)
	conversationsRouter.POST("/:conversation_id", api.handler.UpdateConversation)
	conversationsRouter.DELETE("/:conversation_id", api.handler.DeleteConversation)
	conversationsRouter.POST("/:conversation_id/items", api.handler.CreateItems)
	conversationsRouter.GET("/:conversation_id/items", api.handler.ListItems)
	conversationsRouter.GET("/:conversation_id/items/:item_id", api.handler.GetItem)
	conversationsRouter.DELETE("/:conversation_id/items/:item_id", api.handler.DeleteItem)

	// Future Jan-specific extensions can be added here:
	// conversationsRouter.GET("", api.handler.ListConversations)  // Jan-specific: list all conversations
	// conversationsRouter.GET("/:conversation_id/items/search", api.handler.SearchItems)  // Jan-specific: search items
}
