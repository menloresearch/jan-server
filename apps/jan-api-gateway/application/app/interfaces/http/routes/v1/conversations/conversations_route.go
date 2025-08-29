package conversations

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

type ConversationAPI struct {
	conversationService *conversation.ConversationService
	userService         *user.UserService
	apiKeyService       *apikey.ApiKeyService
}

func NewConversationAPI(conversationService *conversation.ConversationService, userService *user.UserService, apiKeyService *apikey.ApiKeyService) *ConversationAPI {
	return &ConversationAPI{
		conversationService: conversationService,
		userService:         userService,
		apiKeyService:       apiKeyService,
	}
}

func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations")

	conversationsRouter.POST("", api.CreateConversation)
	conversationsRouter.GET("/:conversation_id", api.GetConversation)
	conversationsRouter.PATCH("/:conversation_id", api.UpdateConversation)
	conversationsRouter.DELETE("/:conversation_id", api.DeleteConversation)
	conversationsRouter.GET("/:conversation_id/items/:item_id", api.GetItem)
	conversationsRouter.DELETE("/:conversation_id/items/:item_id", api.DeleteItem)
}

// validateAPIKey validates the API key from request and returns the owner ID
func (api *ConversationAPI) validateAPIKey(ctx *gin.Context) (*uint, error) {
	apiKey, ok := requests.GetTokenFromBearer(ctx)
	if !ok {
		return nil, fmt.Errorf("invalid or missing API key")
	}

	keyHash := api.apiKeyService.HashKey(ctx, apiKey)
	apiKeyEntity, err := api.apiKeyService.FindByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or missing API key")
	}

	if !apiKeyEntity.IsValid() {
		return nil, fmt.Errorf("invalid or expired API key")
	}

	if apiKeyEntity.OwnerID == nil {
		return nil, fmt.Errorf("API key has no associated owner")
	}

	return apiKeyEntity.OwnerID, nil
}

func domainToConversationResponse(entity *conversation.Conversation) ConversationResponse {
	return ConversationResponse{
		ID:       entity.PublicID,
		Object:   "conversation",
		Created:  entity.CreatedAt.Unix(),
		Metadata: entity.Metadata,
		Items:    domainToConversationItemsResponse(entity.Items),
	}
}

func domainToConversationItemsResponse(items []conversation.Item) []ConversationItemResponse {
	if len(items) == 0 {
		return nil
	}

	response := make([]ConversationItemResponse, len(items))
	for i, item := range items {
		response[i] = domainToConversationItemResponse(item)
	}
	return response
}

func domainToConversationItemResponse(entity conversation.Item) ConversationItemResponse {
	response := ConversationItemResponse{
		ID:        fmt.Sprintf("item_%d", entity.ID),
		Object:    "conversation.item",
		Type:      string(entity.Type),
		Status:    entity.Status,
		CreatedAt: entity.CreatedAt.Unix(),
		Content:   domainToContentResponse(entity.Content),
	}

	if entity.Role != nil {
		role := string(*entity.Role)
		response.Role = &role
	}

	return response
}

func domainToContentResponse(content []conversation.Content) []ContentResponse {
	if len(content) == 0 {
		return nil
	}

	response := make([]ContentResponse, len(content))
	for i, c := range content {
		response[i] = ContentResponse{
			Type: c.Type,
		}
		if c.Text != nil {
			response[i].Text = &TextResponse{
				Value: c.Text.Value,
			}
		}
	}
	return response
}

// CreateConversationRequest represents the request body for creating a conversation
type CreateConversationRequest struct {
	Metadata map[string]string         `json:"metadata,omitempty"`
	Items    []ConversationItemRequest `json:"items,omitempty"`
}

// ConversationItemRequest represents an item in the conversation request
type ConversationItemRequest struct {
	Type    string `json:"type" binding:"required"`
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ConversationResponse represents the response body for conversation operations
type ConversationResponse struct {
	ID       string                     `json:"id"`
	Object   string                     `json:"object"`
	Created  int64                      `json:"created"`
	Metadata map[string]string          `json:"metadata,omitempty"`
	Items    []ConversationItemResponse `json:"items,omitempty"`
}

// ConversationItemResponse represents an item in the conversation response
type ConversationItemResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	Type      string            `json:"type"`
	Role      *string           `json:"role,omitempty"`
	Content   []ContentResponse `json:"content,omitempty"`
	Status    *string           `json:"status,omitempty"`
	CreatedAt int64             `json:"created_at"`
}

// ContentResponse represents content in the response
type ContentResponse struct {
	Type string        `json:"type"`
	Text *TextResponse `json:"text,omitempty"`
}

// TextResponse represents text content in the response
type TextResponse struct {
	Value string `json:"value"`
}

// UpdateConversationRequest represents the request body for updating a conversation
type UpdateConversationRequest struct {
	Title    *string           `json:"title,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
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
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b2c3d4e5-f6g7-8901-bcde-f23456789012",
			Error: "User not found",
		})
		return
	}

	var request CreateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "c3d4e5f6-g7h8-9012-cdef-345678901234",
			Error: err.Error(),
		})
		return
	}

	// Create conversation with default settings (private, no title)
	conv, err := api.conversationService.CreateConversation(ctx, user.ID, nil, true, request.Metadata)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "d4e5f6g7-h8i9-0123-defg-456789012345",
			Error: err.Error(),
		})
		return
	}

	// Add items to the conversation if provided
	for _, item := range request.Items {
		itemType := conversation.ItemType(item.Type)
		var role *conversation.ItemRole
		if item.Role != "" {
			r := conversation.ItemRole(item.Role)
			role = &r
		}

		// Convert string content to Content slice
		content := []conversation.Content{{
			Type: "text",
			Text: &conversation.Text{
				Value: item.Content,
			},
		}}

		_, err := api.conversationService.AddItem(ctx, conv.PublicID, user.ID, itemType, role, content)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "e5f6g7h8-i9j0-1234-efgh-567890123456",
				Error: fmt.Sprintf("Failed to add item: %s", err.Error()),
			})
			return
		}
	}

	ctx.JSON(http.StatusCreated, domainToConversationResponse(conv))
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
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "h8i9j0k1-l2m3-4567-hijk-890123456789",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "i9j0k1l2-m3n4-5678-ijkl-901234567890",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "j0k1l2m3-n4o5-6789-jklm-012345678901",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "k1l2m3n4-o5p6-7890-klmn-123456789012",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "l2m3n4o5-p6q7-8901-lmno-234567890123",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, domainToConversationResponse(conv))
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
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "m3n4o5p6-q7r8-9012-mnop-345678901234",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "n4o5p6q7-r8s9-0123-nopq-456789012345",
			Error: "User not found",
		})
		return
	}

	var request UpdateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "o5p6q7r8-s9t0-1234-opqr-567890123456",
			Error: err.Error(),
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	conv, err := api.conversationService.UpdateConversation(ctx, conversationID, user.ID, request.Title, request.Metadata)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "p6q7r8s9-t0u1-2345-pqrs-678901234567",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "q7r8s9t0-u1v2-3456-qrst-789012345678",
				Error: "Access denied: not the owner of this conversation",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "r8s9t0u1-v2w3-4567-rstu-890123456789",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, domainToConversationResponse(conv))
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
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "s9t0u1v2-w3x4-5678-stuv-901234567890",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "t0u1v2w3-x4y5-6789-tuvw-012345678901",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	err = api.conversationService.DeleteConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "u1v2w3x4-y5z6-7890-uvwx-123456789012",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "v2w3x4y5-z6a7-8901-vwxy-234567890123",
				Error: "Access denied: not the owner of this conversation",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "w3x4y5z6-a7b8-9012-wxyz-345678901234",
			Error: err.Error(),
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetItem retrieves a specific item from a conversation
// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation by conversation ID and item ID
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Param include query array false "Additional fields to include in the response"
// @Success 200 {object} ConversationItemResponse "Item details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) GetItem(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "item-unauthorized",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "item-user-not-found",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemIDStr := ctx.Param("item_id")

	// Convert item_id from string to uint
	itemIDParsed, err := strconv.ParseUint(itemIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "item-invalid-id",
			Error: "Invalid item ID format",
		})
		return
	}
	itemID := uint(itemIDParsed)

	item, err := api.conversationService.GetItem(ctx, conversationID, itemID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "conversation-not-found",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "conversation-private",
				Error: "Access denied: conversation is private",
			})
			return
		}
		if errors.Is(err, conversation.ErrItemNotFound) || errors.Is(err, conversation.ErrItemNotInConversation) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "item-not-found",
				Error: "Item not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "item-internal-error",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, domainToConversationItemResponse(*item))
}

// DeleteItem deletes a specific item from a conversation
// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation by conversation ID and item ID. Returns the updated conversation.
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} ConversationResponse "Updated conversation after item deletion"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [delete]
func (api *ConversationAPI) DeleteItem(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "delete-item-unauthorized",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "delete-item-user-not-found",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemIDStr := ctx.Param("item_id")

	// Convert item_id from string to uint
	itemIDParsed, err := strconv.ParseUint(itemIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "delete-item-invalid-id",
			Error: "Invalid item ID format",
		})
		return
	}
	itemID := uint(itemIDParsed)

	updatedConversation, err := api.conversationService.DeleteItem(ctx, conversationID, itemID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "delete-item-conversation-not-found",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "delete-item-access-denied",
				Error: "Access denied: not the owner of this conversation",
			})
			return
		}
		if errors.Is(err, conversation.ErrItemNotFound) || errors.Is(err, conversation.ErrItemNotInConversation) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "delete-item-not-found",
				Error: "Item not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "delete-item-internal-error",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, domainToConversationResponse(updatedConversation))
}
