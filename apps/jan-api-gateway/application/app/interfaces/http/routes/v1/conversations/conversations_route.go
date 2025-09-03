package conversations

import (
	"github.com/gin-gonic/gin"
	_ "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation" // Import for Swagger types
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
	_ "menlo.ai/jan-api-gateway/app/interfaces/http/responses" // Import for Swagger types
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

	// OpenAI-compatible endpoints with Swagger documentation
	conversationsRouter.POST("", api.createConversation)
	conversationsRouter.GET("/:conversation_id", api.getConversation)
	conversationsRouter.PATCH("/:conversation_id", api.updateConversation)
	conversationsRouter.DELETE("/:conversation_id", api.deleteConversation)
	conversationsRouter.POST("/:conversation_id/items", api.createItems)
	conversationsRouter.GET("/:conversation_id/items", api.listItems)
	conversationsRouter.GET("/:conversation_id/items/:item_id", api.getItem)
	conversationsRouter.DELETE("/:conversation_id/items/:item_id", api.deleteItem)
}

// createConversation handles conversation creation
// @Summary Create a conversation
// @Description Creates a new conversation for the authenticated user
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.CreateConversationRequest true "Create conversation request"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Created conversation"
// @Failure 400 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Invalid request"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations [post]
func (api *ConversationAPI) createConversation(ctx *gin.Context) {
	api.handler.CreateConversation(ctx)
}

// getConversation handles conversation retrieval
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Conversation details"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [get]
func (api *ConversationAPI) getConversation(ctx *gin.Context) {
	api.handler.GetConversation(ctx)
}

// updateConversation handles conversation updates
// @Summary Update a conversation
// @Description Updates conversation metadata
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.UpdateConversationRequest true "Update conversation request"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Updated conversation"
// @Failure 400 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Invalid request"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [patch]
func (api *ConversationAPI) updateConversation(ctx *gin.Context) {
	api.handler.UpdateConversation(ctx)
}

// deleteConversation handles conversation deletion
// @Summary Delete a conversation
// @Description Deletes a conversation and all its items
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.DeletedConversationResponse "Deleted conversation"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [delete]
func (api *ConversationAPI) deleteConversation(ctx *gin.Context) {
	api.handler.DeleteConversation(ctx)
}

// createItems handles item creation
// @Summary Create items in a conversation
// @Description Adds multiple items to a conversation
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.CreateItemsRequest true "Create items request"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationItemListResponse "Created items"
// @Failure 400 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Invalid request"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) createItems(ctx *gin.Context) {
	api.handler.CreateItems(ctx)
}

// listItems handles item listing with optional pagination
// @Summary List items in a conversation
// @Description Lists all items in a conversation
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "Number of items to return (1-100)"
// @Param cursor query string false "Cursor for pagination"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationItemListResponse "List of items"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) listItems(ctx *gin.Context) {
	api.handler.ListItems(ctx)
}

// getItem handles single item retrieval
// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationItemResponse "Item details"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) getItem(ctx *gin.Context) {
	api.handler.GetItem(ctx)
}

// deleteItem handles item deletion
// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Updated conversation"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [delete]
func (api *ConversationAPI) deleteItem(ctx *gin.Context) {
	api.handler.DeleteItem(ctx)
}
