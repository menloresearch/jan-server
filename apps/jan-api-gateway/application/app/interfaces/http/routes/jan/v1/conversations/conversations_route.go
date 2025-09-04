package conversations

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/user"
	_ "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation" // Import for Swagger types
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	_ "menlo.ai/jan-api-gateway/app/interfaces/http/responses" // Import for Swagger types
)

// ConversationAPI handles route registration for Jan V1 conversations
type ConversationAPI struct {
	handler     *conversationHandler.ConversationHandler
	userService *user.UserService
}

// NewConversationAPI creates a new conversation API instance
func NewConversationAPI(handler *conversationHandler.ConversationHandler, userService *user.UserService) *ConversationAPI {
	return &ConversationAPI{
		handler:     handler,
		userService: userService,
	}
}


// getAuthenticatedUser gets the authenticated user from JAN route context
func (api *ConversationAPI) getAuthenticatedUser(ctx *gin.Context) (*conversationHandler.AuthenticatedUser, error) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid or missing authentication")
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return &conversationHandler.AuthenticatedUser{ID: user.ID}, nil
}

// RegisterRouter registers Jan-specific conversation routes (currently identical to OpenAI)
func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations")

	// Jan API endpoints (same as OpenAI for now, but can be extended)
	conversationsRouter.POST("", middleware.OptionalAuthMiddleware(), api.createConversation)
	conversationsRouter.GET("/:conversation_id", middleware.OptionalAuthMiddleware(), api.getConversation)
	conversationsRouter.PATCH("/:conversation_id", middleware.OptionalAuthMiddleware(), api.updateConversation)
	conversationsRouter.DELETE("/:conversation_id", middleware.OptionalAuthMiddleware(), api.deleteConversation)
	conversationsRouter.POST("/:conversation_id/items", middleware.OptionalAuthMiddleware(), api.createItems)
	conversationsRouter.GET("/:conversation_id/items", middleware.OptionalAuthMiddleware(), api.listItems)
	conversationsRouter.GET("/:conversation_id/items/:item_id", middleware.OptionalAuthMiddleware(), api.getItem)
	conversationsRouter.DELETE("/:conversation_id/items/:item_id", middleware.OptionalAuthMiddleware(), api.deleteItem)

	// Future Jan-specific extensions can be added here:
	// conversationsRouter.GET("", api.handler.ListConversations)  // Jan-specific: list all conversations
	// conversationsRouter.GET("/:conversation_id/items/search", api.handler.SearchItems)  // Jan-specific: search items
}

// createConversation handles conversation creation
// @Summary Create a conversation
// @Description Creates a new conversation for the authenticated user
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.CreateConversationRequest true "Create conversation request"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Created conversation"
// @Failure 400 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Invalid request"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations [post]
func (api *ConversationAPI) createConversation(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.CreateConversation(ctx, user)
}

// getConversation handles conversation retrieval
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Conversation details"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [get]
func (api *ConversationAPI) getConversation(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f6g7h8i9-j0k1-2345-fghi-678901234567",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.GetConversation(ctx, user)
}

// updateConversation handles conversation updates
// @Summary Update a conversation
// @Description Updates conversation metadata
// @Tags Jan, Jan-Conversations
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
// @Router /jan/v1/conversations/{conversation_id} [patch]
func (api *ConversationAPI) updateConversation(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "j0k1l2m3-n4o5-6789-jklm-012345678901",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.UpdateConversation(ctx, user)
}

// deleteConversation handles conversation deletion
// @Summary Delete a conversation
// @Description Deletes a conversation and all its items
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.DeletedConversationResponse "Deleted conversation"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [delete]
func (api *ConversationAPI) deleteConversation(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "o5p6q7r8-s9t0-1234-opqr-567890123456",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.DeleteConversation(ctx, user)
}

// createItems handles item creation
// @Summary Create items in a conversation
// @Description Adds multiple items to a conversation
// @Tags Jan, Jan-Conversations
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
// @Router /jan/v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) createItems(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "v2w3x4y5-z6a7-8901-vwxy-234567890123",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.CreateItems(ctx, user)
}

// listItems handles item listing with optional pagination
// @Summary List items in a conversation
// @Description Lists all items in a conversation
// @Tags Jan, Jan-Conversations
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
// @Router /jan/v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) listItems(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b8c9d0e1-f2g3-4567-bcde-890123456789",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.ListItems(ctx, user)
}

// getItem handles single item retrieval
// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationItemResponse "Item details"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) getItem(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f2g3h4i5-j6k7-8901-fghi-234567890123",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.GetItem(ctx, user)
}

// deleteItem handles item deletion
// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Updated conversation"
// @Failure 401 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Access denied"
// @Failure 404 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} menlo_ai_jan-api-gateway_app_interfaces_http_responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [delete]
func (api *ConversationAPI) deleteItem(ctx *gin.Context) {
	user, err := api.getAuthenticatedUser(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "k7l8m9n0-o1p2-3456-klmn-789012345678",
			Error: "Invalid or missing authentication",
		})
		return
	}
	api.handler.DeleteItem(ctx, user)
}
