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
		CreatedAt: entity.CreatedAt, // Already Unix timestamp
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
		ID:        entity.PublicID, // Use the generated public ID
		Object:    "conversation.item",
		Type:      string(entity.Type),
		Status:    entity.Status,
		CreatedAt: entity.CreatedAt, // Already Unix timestamp
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

	if len(request.Items) > 0 {
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

			_, err := api.conversationService.AddItem(ctx, conv, user.ID, itemType, role, content)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
					Code:  "e5f6g7h8-i9j0-1234-efgh-567890123456",
					Error: fmt.Sprintf("Failed to add item: %s", err.Error()),
				})
				return
			}
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
// @Router /v1/conversations/{conversation_id} [post]
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
	conv, err := api.conversationService.UpdateAndAuthorizeConversation(ctx, conversationID, user.ID, nil, request.Metadata)
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

	// Get conversation before deleting for response
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "0198f4cf-83f2-7570-851b-270114ede87e",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "0198f4cf-9201-7633-855d-d7178a0f0e64",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "0198f4cf-addd-76bc-aa7c-5c7e39aa44b9",
			Error: err.Error(),
		})
		return
	}

	// Now delete the conversation
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
// @Router /v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) CreateItems(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "0198f4cf-da13-763d-9bad-b05397092d3c",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0198f4cf-e893-729e-9a7b-651f49f695bf",
			Error: "User not found",
		})
		return
	}

	var request CreateItemsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0198f4cf-f89d-7606-9b44-c52eeede2f60",
			Error: err.Error(),
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Get conversation first to avoid N+1 queries
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "create-items-conversation-not-found",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "create-items-private-conversation",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "create-items-get-conversation-error",
			Error: err.Error(),
		})
		return
	}

	// Convert request items to conversation content format and add them
	var createdItems []conversation.Item
	if len(request.Items) > 0 {
		for _, reqItem := range request.Items {
			// Convert string content to []conversation.Content
			content := []conversation.Content{{
				Type: "text",
				Text: &conversation.Text{
					Value: reqItem.Content,
				},
			}}

			// Convert role string to ItemRole
			var role *conversation.ItemRole
			if reqItem.Role != "" {
				itemRole := conversation.ItemRole(reqItem.Role)
				role = &itemRole
			}

			// Convert type string to ItemType
			itemType := conversation.ItemType(reqItem.Type)

			item, err := api.conversationService.AddItem(ctx, conv, user.ID, itemType, role, content)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
					Code:  "create-items-add-item-error",
					Error: err.Error(),
				})
				return
			}

			createdItems = append(createdItems, *item)
		}
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
// @Router /v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) ListItems(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "0198f4d0-3c34-73eb-9f64-9077238d2eda",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0198f4d0-4b8b-720e-b26a-4b7cc32af275",
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
				Code:  "0198f4d0-5a23-755e-8929-39abb1e96136",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "0198f4d0-69a5-722c-b604-c86573a7e151",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "0198f4d0-939d-751a-8bbc-3d962fa9369c",
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
// @Router /v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) GetItem(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "0198f4d0-af1f-774e-80c7-2aa9eeeb8e89",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0198f4d0-bc70-7267-bec6-4631b7d5dc18",
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
				Code:  "0198f4d0-d02c-774c-87e2-da1ff3befd73",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "0198f4d0-df55-725f-bd87-419872dc9561",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "0198f4d0-ed65-71d8-8eb1-bd01f04aacfa",
			Error: err.Error(),
		})
		return
	}

	// Find item by public ID
	var foundItem *conversation.Item
	for _, item := range conv.Items {
		if item.PublicID == itemIDStr {
			foundItem = &item
			break
		}
	}

	if foundItem == nil {
		ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "0198f4d0-fcfe-752b-9abb-6450a1332889",
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
// @Router /v1/conversations/{conversation_id}/items/{item_id} [delete]
func (api *ConversationAPI) DeleteItem(ctx *gin.Context) {
	ownerID, err := api.validateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "0198f4d1-0f04-74bd-8617-cf900a139792",
			Error: "Invalid or missing API key",
		})
		return
	}

	user, err := api.userService.FindByID(ctx, *ownerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0198f4d1-1ef8-70d7-b7a0-9e22eda8a55e",
			Error: "User not found",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemPublicID := ctx.Param("item_id")

	// Get conversation first to avoid N+1 queries
	conv, err := api.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "delete-item-conversation-not-found",
				Error: "Conversation not found",
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "delete-item-private-conversation",
				Error: "Access denied: conversation is private",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "delete-item-get-conversation-error",
			Error: err.Error(),
		})
		return
	}

	// Find item by public ID to get internal ID
	var itemID uint
	found := false
	for _, item := range conv.Items {
		if item.PublicID == itemPublicID {
			itemID = item.ID
			found = true
			break
		}
	}

	if !found {
		ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "delete-item-not-found",
			Error: "Item not found",
		})
		return
	}

	updatedConversation, err := api.conversationService.DeleteItem(ctx, conv, itemID, user.ID)
	if err != nil {
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
