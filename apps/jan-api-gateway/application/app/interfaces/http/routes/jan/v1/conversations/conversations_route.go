package conversations

import (
	"errors"
	"fmt"
	"net/http"

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

	// OpenAI-compatible endpoints only
	conversationsRouter.POST("", api.CreateConversation)
	conversationsRouter.GET("/:conversation_id", api.GetConversation)
	conversationsRouter.POST("/:conversation_id", api.UpdateConversation) // OpenAI uses POST for updates
	conversationsRouter.DELETE("/:conversation_id", api.DeleteConversation)
	conversationsRouter.POST("/:conversation_id/items", api.CreateItems)           // OpenAI creates multiple items
	conversationsRouter.GET("/:conversation_id/items", api.ListItems)              // OpenAI list items
	conversationsRouter.GET("/:conversation_id/items/:item_id", api.GetItem)       // OpenAI get single item
	conversationsRouter.DELETE("/:conversation_id/items/:item_id", api.DeleteItem) // OpenAI delete single item
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
	// Initialize metadata as empty map if nil (required by OpenAI spec)
	metadata := entity.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return ConversationResponse{
		ID:        entity.PublicID,
		Object:    "conversation",
		CreatedAt: entity.CreatedAt.Unix(),
		Metadata:  metadata,
	}
}

func domainToDeletedConversationResponse(entity *conversation.Conversation) DeletedConversationResponse {
	return DeletedConversationResponse{
		ID:      entity.PublicID,
		Object:  "conversation.deleted",
		Deleted: true,
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

// CreateConversationRequest represents the request body for creating a conversation (matches OpenAI schema)
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

// ConversationResponse represents the response body for conversation operations (matches OpenAI ConversationResource)
type ConversationResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata"`
}

// DeletedConversationResponse represents the response body for deleted conversations (matches OpenAI DeletedConversationResource)
type DeletedConversationResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
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

// UpdateConversationRequest represents the request body for updating a conversation (matches OpenAI UpdateConversationBody)
type UpdateConversationRequest struct {
	Metadata map[string]string `json:"metadata" binding:"required"`
}

// CreateItemsRequest represents the request body for creating items (matches OpenAI schema)
type CreateItemsRequest struct {
	Items []ConversationItemRequest `json:"items" binding:"required"`
}

// ConversationItemListResponse represents the response for item lists (matches OpenAI ConversationItemList)
type ConversationItemListResponse struct {
	Object  string                     `json:"object"`
	Data    []ConversationItemResponse `json:"data"`
	HasMore bool                       `json:"has_more"`
	FirstID string                     `json:"first_id"`
	LastID  string                     `json:"last_id"`
}

// CreateConversation creates a new conversation
// @Summary Create a conversation
// @Description Creates a new conversation for the authenticated user
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateConversationRequest true "Create conversation request"
// @Success 200 {object} ConversationResponse "Created conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations [post]
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

	ctx.JSON(http.StatusOK, domainToConversationResponse(conv))
}

// GetConversation retrieves a specific conversation
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} ConversationResponse "Conversation details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [get]
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

// UpdateConversation updates a conversation's metadata
// @Summary Update a conversation
// @Description Updates a conversation's metadata
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body UpdateConversationRequest true "Update conversation request"
// @Success 200 {object} ConversationResponse "Updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [post]
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

	conversationID := ctx.Param("conversation_id")

	var request UpdateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "o5p6q7r8-s9t0-1234-opqr-567890123456",
			Error: err.Error(),
		})
		return
	}

	conv, err := api.conversationService.UpdateConversation(ctx, conversationID, user.ID, nil, request.Metadata)
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
// @Success 200 {object} DeletedConversationResponse "Conversation deleted successfully"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [delete]
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

	// Get conversation first to populate response
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "u1v2w3x4-y5z6-7890-uvwx-123456789012",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "v2w3x4y5-z6a7-8901-vwxy-234567890123",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "w3x4y5z6-a7b8-9012-wxyz-345678901234",
			Error: err.Error(),
		})
		return
	}

	err = api.conversationService.DeleteConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "x4y5z6a7-b8c9-0123-xyza-456789012345",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "y5z6a7b8-c9d0-1234-yzab-567890123456",
				Error: "Access denied: not the owner of this conversation",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "z6a7b8c9-d0e1-2345-zabc-678901234567",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, domainToDeletedConversationResponse(conv))
}

// CreateItems creates items in a conversation
// @Summary Create items
// @Description Creates multiple items in a conversation
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body CreateItemsRequest true "Create items request"
// @Success 200 {object} ConversationItemListResponse "Created items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) CreateItems(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "create-items-unauthorized",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "create-items-user-not-found",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	var request CreateItemsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "create-items-invalid-request",
			Error: err.Error(),
		})
		return
	}

	var createdItems []conversation.Item

	// Create each item
	for _, itemRequest := range request.Items {
		itemType := conversation.ItemType(itemRequest.Type)
		var role *conversation.ItemRole
		if itemRequest.Role != "" {
			r := conversation.ItemRole(itemRequest.Role)
			role = &r
		}

		// Convert string content to Content slice
		content := []conversation.Content{{
			Type: "text",
			Text: &conversation.Text{
				Value: itemRequest.Content,
			},
		}}

		item, err := api.conversationService.AddItem(ctx, conversationID, user.ID, itemType, role, content)
		if err != nil {
			if errors.Is(err, conversation.ErrConversationNotFound) {
				ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
					Code:  "create-items-conversation-not-found",
					Error: "Conversation not found",
				})
				return
			}
			if errors.Is(err, conversation.ErrAccessDenied) {
				ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
					Code:  "create-items-access-denied",
					Error: "Access denied: not the owner of this conversation",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "create-items-internal-error",
				Error: err.Error(),
			})
			return
		}

		createdItems = append(createdItems, *item)
	}

	// Convert to response format (matching OpenAI ConversationItemList)
	data := make([]ConversationItemResponse, len(createdItems))
	for i, item := range createdItems {
		data[i] = domainToConversationItemResponse(item)
	}

	response := ConversationItemListResponse{
		Object:  "list",
		Data:    data,
		HasMore: false,
		FirstID: "",
		LastID:  "",
	}

	if len(data) > 0 {
		response.FirstID = data[0].ID
		response.LastID = data[len(data)-1].ID
	}

	ctx.JSON(http.StatusOK, response)
}

// ListItems lists items in a conversation
// @Summary List items
// @Description List all items for a conversation with the given ID
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "A limit on the number of objects to be returned" default(20)
// @Param order query string false "The order to return the input items in" Enums(asc, desc) default(desc)
// @Param after query string false "An item ID to list items after, used in pagination"
// @Success 200 {object} ConversationItemListResponse "List of items"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) ListItems(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "list-items-unauthorized",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "list-items-user-not-found",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Get conversation first to check access
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "list-items-conversation-not-found",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "list-items-private-conversation",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "list-items-internal-error",
			Error: err.Error(),
		})
		return
	}

	// Convert items to response format
	data := make([]ConversationItemResponse, len(conv.Items))
	for i, item := range conv.Items {
		data[i] = domainToConversationItemResponse(item)
	}

	response := ConversationItemListResponse{
		Object:  "list",
		Data:    data,
		HasMore: false, // For now, we don't implement pagination
		FirstID: "",
		LastID:  "",
	}

	if len(data) > 0 {
		response.FirstID = data[0].ID
		response.LastID = data[len(data)-1].ID
	}

	ctx.JSON(http.StatusOK, response)
}

// GetItem retrieves a specific item from a conversation
// @Summary Retrieve an item
// @Description Get a single item from a conversation with the given IDs
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} ConversationItemResponse "Item details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) GetItem(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "get-item-unauthorized",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "get-item-user-not-found",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemIDStr := ctx.Param("item_id")

	// For OpenAI compatibility, item_id should be a string like "msg_abc"
	// We need to extract the numeric ID or implement string-based IDs
	// For now, let's try to find the item by string ID directly
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "get-item-conversation-not-found",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "get-item-private-conversation",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "get-item-internal-error",
			Error: err.Error(),
		})
		return
	}

	// Find item by ID (our current implementation uses "item_<id>" format)
	var foundItem *conversation.Item
	for _, item := range conv.Items {
		itemResponseID := fmt.Sprintf("item_%d", item.ID)
		if itemResponseID == itemIDStr {
			foundItem = &item
			break
		}
	}

	if foundItem == nil {
		ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "get-item-not-found",
			Error: "Item not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, domainToConversationItemResponse(*foundItem))
}

// DeleteItem deletes a specific item from a conversation
// @Summary Delete an item
// @Description Delete an item from a conversation with the given IDs
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
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [delete]
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

	// Extract numeric ID from string ID (e.g., "item_123" -> 123)
	var itemID uint
	if _, err := fmt.Sscanf(itemIDStr, "item_%d", &itemID); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "delete-item-invalid-id",
			Error: "Invalid item ID format",
		})
		return
	}

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
