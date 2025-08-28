package conversations

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type ConversationAPI struct {
	conversationService *conversation.ConversationService
	userService         *user.UserService
}

func NewConversationAPI(conversationService *conversation.ConversationService, userService *user.UserService) *ConversationAPI {
	return &ConversationAPI{
		conversationService: conversationService,
		userService:         userService,
	}
}

func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations")
	conversationsRouter.Use(middleware.AuthMiddleware())

	conversationsRouter.POST("", api.CreateConversation)
	conversationsRouter.GET("", api.ListConversations)
	conversationsRouter.GET("/:conversation_id", api.GetConversation)
	conversationsRouter.PATCH("/:conversation_id", api.UpdateConversation)
	conversationsRouter.DELETE("/:conversation_id", api.DeleteConversation)
	conversationsRouter.POST("/:conversation_id/items", api.AddItem)
	conversationsRouter.GET("/:conversation_id/items/search", api.SearchItems)
}

// CreateConversationRequest represents the request body for creating a conversation
type CreateConversationRequest struct {
	Title     *string           `json:"title,omitempty"`
	IsPrivate *bool             `json:"is_private,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// UpdateConversationRequest represents the request body for updating a conversation
type UpdateConversationRequest struct {
	Title    *string           `json:"title,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// AddItemRequest represents the request body for adding an item
type AddItemRequest struct {
	Type    conversation.ItemType  `json:"type" binding:"required"`
	Role    *conversation.ItemRole `json:"role,omitempty"`
	Content []conversation.Content `json:"content,omitempty"`
}

// SearchItemsRequest represents query parameters for searching items
type SearchItemsRequest struct {
	Query string `form:"q" binding:"required"`
}

// CreateConversation creates a new conversation
// @Summary Create a conversation
// @Description Creates a new conversation for the authenticated user
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateConversationRequest true "Create conversation request"
// @Success 201 {object} conversation.Conversation "Created conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations [post]
func (api *ConversationAPI) CreateConversation(ctx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "unauthorized",
			Error: "User not authenticated",
		})
		return
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "user_not_found",
			Error: err.Error(),
		})
		return
	}

	var request CreateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "invalid_request",
			Error: err.Error(),
		})
		return
	}

	// Default to private if not specified
	isPrivate := true
	if request.IsPrivate != nil {
		isPrivate = *request.IsPrivate
	}

	conv, err := api.conversationService.CreateConversation(ctx, user.ID, request.Title, isPrivate, request.Metadata)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "create_failed",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, conv)
}

// ListConversations lists conversations for the authenticated user
// @Summary List conversations
// @Description Lists all conversations for the authenticated user
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Number of conversations to return" default(20)
// @Param offset query int false "Number of conversations to skip" default(0)
// @Param status query string false "Filter by conversation status"
// @Param search query string false "Search in conversation titles"
// @Success 200 {array} conversation.Conversation "List of conversations"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations [get]
func (api *ConversationAPI) ListConversations(ctx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "unauthorized",
			Error: "User not authenticated",
		})
		return
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "user_not_found",
			Error: err.Error(),
		})
		return
	}

	// Parse query parameters
	limit := 20
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := ctx.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	filter := conversation.ConversationFilter{}
	if status := ctx.Query("status"); status != "" {
		convStatus := conversation.ConversationStatus(status)
		filter.Status = &convStatus
	}
	if search := ctx.Query("search"); search != "" {
		filter.Search = &search
	}

	conversations, err := api.conversationService.ListConversations(ctx, user.ID, filter, &limit, &offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "list_failed",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, conversations)
}

// GetConversation retrieves a specific conversation
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} conversation.Conversation "Conversation details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [get]
func (api *ConversationAPI) GetConversation(ctx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "unauthorized",
			Error: "User not authenticated",
		})
		return
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "user_not_found",
			Error: err.Error(),
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if err.Error() == "conversation not found" {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "not_found",
				Error: err.Error(),
			})
			return
		}
		if err.Error() == "access denied: conversation is private" {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "access_denied",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "get_failed",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, conv)
}

// UpdateConversation updates a conversation
// @Summary Update a conversation
// @Description Updates a conversation's title and metadata
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body UpdateConversationRequest true "Update conversation request"
// @Success 200 {object} conversation.Conversation "Updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [patch]
func (api *ConversationAPI) UpdateConversation(ctx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "unauthorized",
			Error: "User not authenticated",
		})
		return
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "user_not_found",
			Error: err.Error(),
		})
		return
	}

	var request UpdateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "invalid_request",
			Error: err.Error(),
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	conv, err := api.conversationService.UpdateConversation(ctx, conversationID, user.ID, request.Title, request.Metadata)
	if err != nil {
		if err.Error() == "conversation not found" {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "not_found",
				Error: err.Error(),
			})
			return
		}
		if err.Error() == "access denied: not the owner of this conversation" {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "access_denied",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "update_failed",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, conv)
}

// DeleteConversation deletes a conversation
// @Summary Delete a conversation
// @Description Deletes a conversation and all its messages
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 204 "Conversation deleted successfully"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [delete]
func (api *ConversationAPI) DeleteConversation(ctx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "unauthorized",
			Error: "User not authenticated",
		})
		return
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "user_not_found",
			Error: err.Error(),
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	err = api.conversationService.DeleteConversation(ctx, conversationID, user.ID)
	if err != nil {
		if err.Error() == "conversation not found" {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "not_found",
				Error: err.Error(),
			})
			return
		}
		if err.Error() == "access denied: not the owner of this conversation" {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "access_denied",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "delete_failed",
			Error: err.Error(),
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AddItem adds an item to a conversation
// @Summary Add an item to a conversation
// @Description Adds a new item to an existing conversation
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body AddItemRequest true "Add item request"
// @Success 201 {object} conversation.Item "Created item"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) AddItem(ctx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "unauthorized",
			Error: "User not authenticated",
		})
		return
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "user_not_found",
			Error: err.Error(),
		})
		return
	}

	var request AddItemRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "invalid_request",
			Error: err.Error(),
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	item, err := api.conversationService.AddItem(ctx, conversationID, user.ID, request.Type, request.Role, request.Content)
	if err != nil {
		if err.Error() == "conversation not found" {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "not_found",
				Error: err.Error(),
			})
			return
		}
		if err.Error() == "access denied: conversation is private" {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "access_denied",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "add_item_failed",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, item)
}

// SearchItems searches for items within a conversation
// @Summary Search items in a conversation
// @Description Searches for items containing specific text within a conversation
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param q query string true "Search query"
// @Success 200 {array} conversation.Item "Matching items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/search [get]
func (api *ConversationAPI) SearchItems(ctx *gin.Context) {
	userClaim, err := auth.GetUserClaimFromRequestContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "unauthorized",
			Error: "User not authenticated",
		})
		return
	}

	user, err := api.userService.FindByEmail(ctx, userClaim.Email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "user_not_found",
			Error: err.Error(),
		})
		return
	}

	var request SearchItemsRequest
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "invalid_request",
			Error: err.Error(),
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	items, err := api.conversationService.SearchItems(ctx, conversationID, user.ID, request.Query)
	if err != nil {
		if err.Error() == "conversation not found" {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "not_found",
				Error: err.Error(),
			})
			return
		}
		if err.Error() == "access denied: conversation is private" {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "access_denied",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "search_failed",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, items)
}
