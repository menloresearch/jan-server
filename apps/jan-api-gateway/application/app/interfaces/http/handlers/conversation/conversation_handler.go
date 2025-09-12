package conversation

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
)

// ConversationHandler handles HTTP requests for conversations
type ConversationHandler struct {
	conversationService *conversation.ConversationService
	userService         *user.UserService
	apiKeyService       *apikey.ApiKeyService
}

// NewConversationHandler creates a new conversation handler
func NewConversationHandler(
	conversationService *conversation.ConversationService,
	userService *user.UserService,
	apiKeyService *apikey.ApiKeyService,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		userService:         userService,
		apiKeyService:       apiKeyService,
	}
}

// AuthenticatedUser represents an authenticated user context
type AuthenticatedUser struct {
	ID uint
}

// CreateConversationRequest represents the input for creating a conversation
type CreateConversationRequest struct {
	Metadata map[string]string         `json:"metadata,omitempty"`
	Items    []ConversationItemRequest `json:"items,omitempty"`
}

// ConversationItemRequest represents an item in the conversation request
type ConversationItemRequest struct {
	Type    string                       `json:"type" binding:"required"`
	Role    string                       `json:"role,omitempty"`
	Content []ConversationContentRequest `json:"content" binding:"required"`
}

// ConversationContentRequest represents content in the request
type ConversationContentRequest struct {
	Type string `json:"type" binding:"required"`
	Text string `json:"text,omitempty"`
}

// ConversationResponse represents the response structure
type ConversationResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata"`
}

// DeletedConversationResponse represents the deleted conversation response
type DeletedConversationResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// ConversationItemResponse represents an item in the response
type ConversationItemResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	Type      string            `json:"type"`
	Role      *string           `json:"role,omitempty"`
	Status    *string           `json:"status,omitempty"`
	CreatedAt int64             `json:"created_at"`
	Content   []ContentResponse `json:"content,omitempty"`
}

// ContentResponse represents content in the response
type ContentResponse struct {
	Type       string                `json:"type"`
	Text       *TextResponse         `json:"text,omitempty"`
	InputText  *string               `json:"input_text,omitempty"`
	OutputText *OutputTextResponse   `json:"output_text,omitempty"`
	Image      *ImageContentResponse `json:"image,omitempty"`
	File       *FileContentResponse  `json:"file,omitempty"`
}

// TextResponse represents text content in the response
type TextResponse struct {
	Value string `json:"value"`
}

// OutputTextResponse represents AI output text with annotations
type OutputTextResponse struct {
	Text        string               `json:"text"`
	Annotations []AnnotationResponse `json:"annotations"`
}

// ImageContentResponse represents image content
type ImageContentResponse struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// FileContentResponse represents file content
type FileContentResponse struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// AnnotationResponse represents annotation in the response
type AnnotationResponse struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	FileID     string `json:"file_id,omitempty"`
	URL        string `json:"url,omitempty"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Index      int    `json:"index,omitempty"`
}

// ConversationItemListResponse represents the response for item lists
type ConversationItemListResponse struct {
	Object  string                     `json:"object"`
	Data    []ConversationItemResponse `json:"data"`
	HasMore bool                       `json:"has_more"`
	FirstID string                     `json:"first_id"`
	LastID  string                     `json:"last_id"`
}

// UpdateConversationRequest represents the request body for updating a conversation
type UpdateConversationRequest struct {
	Metadata map[string]string `json:"metadata" binding:"required"`
}

// CreateItemsRequest represents the request body for creating items
type CreateItemsRequest struct {
	Items []ConversationItemRequest `json:"items" binding:"required"`
}

// getUserFromContext gets the authenticated user from the request context
func (h *ConversationHandler) getUserFromContext(ctx *gin.Context) (*AuthenticatedUser, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}

	return &AuthenticatedUser{ID: user.ID}, nil
}

// CreateConversation handles conversation creation
func (h *ConversationHandler) CreateConversation(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Error: "User not authenticated",
		})
		return
	}

	// Parse request
	var request CreateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b2c3d4e5-f6g7-8901-bcde-f23456789012",
			Error: "Invalid request format",
		})
		return
	}

	// Create conversation
	conv, err := h.conversationService.CreateConversation(ctx, user.ID, nil, true, request.Metadata)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "c3d4e5f6-g7h8-9012-cdef-345678901234",
			Error: fmt.Sprintf("failed to create conversation: %v", err),
		})
		return
	}

	if len(request.Items) > 20 {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "e5f6g7h8-i9j0-1234-efgh-567890123456",
			Error: "Cannot create more than 20 items in a single conversation creation request",
		})
		return
	}
	// Add items if provided using batch operation
	if len(request.Items) > 0 {
		// Convert all items at once for batch processing
		itemsToCreate := make([]*conversation.Item, len(request.Items))

		for i, itemReq := range request.Items {
			// Convert request to domain types
			itemType := conversation.ItemType(itemReq.Type)
			var role *conversation.ItemRole
			if itemReq.Role != "" {
				r := conversation.ItemRole(itemReq.Role)
				role = &r
			}

			// Convert content
			content := make([]conversation.Content, len(itemReq.Content))
			for j, c := range itemReq.Content {
				content[j] = conversation.Content{
					Type: c.Type,
					Text: &conversation.Text{
						Value: c.Text,
					},
				}
			}

			itemsToCreate[i] = &conversation.Item{
				Type:    itemType,
				Role:    role,
				Content: content,
			}
		}

		// Single batch operation instead of N individual operations
		_, err := h.conversationService.AddMultipleItems(ctx, conv, user.ID, itemsToCreate)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "d4e5f6g7-h8i9-0123-defg-456789012345",
				Error: fmt.Sprintf("failed to add items: %v", err),
			})
			return
		}

		// Reload conversation with items
		conv, err = h.conversationService.GetConversationByPublicIDAndUserID(ctx, conv.PublicID, user.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  "e5f6g7h8-i9j0-1234-efgh-567890123456",
				Error: fmt.Sprintf("failed to reload conversation: %v", err),
			})
			return
		}
	}

	response := h.domainToConversationResponse(conv)
	ctx.JSON(http.StatusOK, response)
}

// GetConversation handles conversation retrieval
func (h *ConversationHandler) GetConversation(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f6g7h8i9-j0k1-2345-fghi-678901234567",
			Error: "User not authenticated",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "g7h8i9j0-k1l2-3456-ghij-789012345678",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "h8i9j0k1-l2m3-4567-hijk-890123456789",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "i9j0k1l2-m3n4-5678-ijkl-901234567890",
			Error: err.Error(),
		})
		return
	}

	response := h.domainToConversationResponse(conv)
	ctx.JSON(http.StatusOK, response)
}

// UpdateConversation handles conversation updates
func (h *ConversationHandler) UpdateConversation(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "j0k1l2m3-n4o5-6789-jklm-012345678901",
			Error: "User not authenticated",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Parse request
	var request UpdateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "k1l2m3n4-o5p6-7890-klmn-123456789012",
			Error: "Invalid request format",
		})
		return
	}

	conv, err := h.conversationService.UpdateAndAuthorizeConversation(ctx, conversationID, user.ID, nil, request.Metadata)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "l2m3n4o5-p6q7-8901-lmno-234567890123",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "m3n4o5p6-q7r8-9012-mnop-345678901234",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "n4o5p6q7-r8s9-0123-nopq-456789012345",
			Error: err.Error(),
		})
		return
	}

	response := h.domainToConversationResponse(conv)
	ctx.JSON(http.StatusOK, response)
}

// DeleteConversation handles conversation deletion
func (h *ConversationHandler) DeleteConversation(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "o5p6q7r8-s9t0-1234-opqr-567890123456",
			Error: "User not authenticated",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Get conversation first to get the public ID for response
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "p6q7r8s9-t0u1-2345-pqrs-678901234567",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "q7r8s9t0-u1v2-3456-qrst-789012345678",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "r8s9t0u1-v2w3-4567-rstu-890123456789",
			Error: err.Error(),
		})
		return
	}

	// Get conversation first to pass to DeleteConversation
	convToDelete, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "s9t0u1v2-w3x4-5678-stuv-901234567890",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "t0u1v2w3-x4y5-6789-tuvw-012345678901",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "u1v2w3x4-y5z6-7890-uvwx-123456789012",
			Error: err.Error(),
		})
		return
	}

	err = h.conversationService.DeleteConversation(ctx, convToDelete)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "s9t0u1v2-w3x4-5678-stuv-901234567890",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "t0u1v2w3-x4y5-6789-tuvw-012345678901",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "u1v2w3x4-y5z6-7890-uvwx-123456789012",
			Error: err.Error(),
		})
		return
	}

	response := h.domainToDeletedConversationResponse(conv)
	ctx.JSON(http.StatusOK, response)
}

// CreateItems handles item creation
func (h *ConversationHandler) CreateItems(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "v2w3x4y5-z6a7-8901-vwxy-234567890123",
			Error: "User not authenticated",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Parse request
	var request CreateItemsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "w3x4y5z6-a7b8-9012-wxyz-345678901234",
			Error: "Invalid request format",
		})
		return
	}

	// Get conversation first to avoid N+1 queries
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "x4y5z6a7-b8c9-0123-xyza-456789012345",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "y5z6a7b8-c9d0-1234-yzab-567890123456",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "z6a7b8c9-d0e1-2345-zabc-678901234567",
			Error: err.Error(),
		})
		return
	}

	// Convert all items at once for batch processing
	itemsToCreate := make([]*conversation.Item, len(request.Items))

	for i, itemReq := range request.Items {
		// Convert request to domain types
		itemType := conversation.ItemType(itemReq.Type)
		var role *conversation.ItemRole
		if itemReq.Role != "" {
			r := conversation.ItemRole(itemReq.Role)
			role = &r
		}

		// Convert content
		content := make([]conversation.Content, len(itemReq.Content))
		for j, c := range itemReq.Content {
			content[j] = conversation.Content{
				Type: c.Type,
				Text: &conversation.Text{
					Value: c.Text,
				},
			}
		}

		itemsToCreate[i] = &conversation.Item{
			Type:    itemType,
			Role:    role,
			Content: content,
		}
	}

	// Single batch operation instead of N individual operations
	createdItems, err := h.conversationService.AddMultipleItems(ctx, conv, user.ID, itemsToCreate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "a7b8c9d0-e1f2-3456-abcd-789012345678",
			Error: err.Error(),
		})
		return
	}

	response := h.domainToConversationItemListResponse(createdItems)
	ctx.JSON(http.StatusOK, response)
}

// ListItems handles item listing
func (h *ConversationHandler) ListItems(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b8c9d0e1-f2g3-4567-bcde-890123456789",
			Error: "User not authenticated",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Get conversation first to check access
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "c9d0e1f2-g3h4-5678-cdef-901234567890",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "d0e1f2g3-h4i5-6789-defg-012345678901",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "e1f2g3h4-i5j6-7890-efgh-123456789012",
			Error: err.Error(),
		})
		return
	}

	// Convert items to pointers for consistency
	items := make([]*conversation.Item, len(conv.Items))
	for i := range conv.Items {
		items[i] = &conv.Items[i]
	}

	response := h.domainToConversationItemListResponse(items)
	ctx.JSON(http.StatusOK, response)
}

// GetItem handles single item retrieval
func (h *ConversationHandler) GetItem(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f2g3h4i5-j6k7-8901-fghi-234567890123",
			Error: "User not authenticated",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemID := ctx.Param("item_id")

	// Get conversation first to avoid N+1 queries
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "g3h4i5j6-k7l8-9012-ghij-345678901234",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "h4i5j6k7-l8m9-0123-hijk-456789012345",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "i5j6k7l8-m9n0-1234-ijkl-567890123456",
			Error: err.Error(),
		})
		return
	}

	// Find item by public ID
	var foundItem *conversation.Item
	for _, item := range conv.Items {
		if item.PublicID == itemID {
			foundItem = &item
			break
		}
	}

	if foundItem == nil {
		ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "j6k7l8m9-n0o1-2345-jklm-678901234567",
			Error: "Item not found",
		})
		return
	}

	response := h.domainToConversationItemResponse(*foundItem)
	ctx.JSON(http.StatusOK, response)
}

// DeleteItem handles item deletion
func (h *ConversationHandler) DeleteItem(ctx *gin.Context) {
	// Get authenticated user from context
	user, err := h.getUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "k7l8m9n0-o1p2-3456-klmn-789012345678",
			Error: "User not authenticated",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemID := ctx.Param("item_id")

	// Get conversation first to avoid N+1 queries (without loading all items)
	conv, err := h.conversationService.GetConversationWithoutItems(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "l8m9n0o1-p2q3-4567-lmno-890123456789",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "m9n0o1p2-q3r4-5678-mnop-901234567890",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "n0o1p2q3-r4s5-6789-nopq-012345678901",
			Error: err.Error(),
		})
		return
	}

	// Use efficient deletion with item public ID instead of loading all items
	updatedConversation, err := h.conversationService.DeleteItemByPublicID(ctx, conv, itemID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrAccessDenied) {
			ctx.JSON(http.StatusForbidden, responses.ErrorResponse{
				Code:  "o1p2q3r4-s5t6-7890-opqr-123456789012",
				Error: err.Error(),
			})
			return
		}
		if errors.Is(err, conversation.ErrItemNotFound) || errors.Is(err, conversation.ErrItemNotInConversation) {
			ctx.JSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "p2q3r4s5-t6u7-8901-pqrs-234567890123",
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "q3r4s5t6-u7v8-9012-qrst-345678901234",
			Error: err.Error(),
		})
		return
	}

	response := h.domainToConversationResponse(updatedConversation)
	ctx.JSON(http.StatusOK, response)
}

// Domain to response conversion methods

func (h *ConversationHandler) domainToConversationResponse(entity *conversation.Conversation) *ConversationResponse {
	metadata := entity.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return &ConversationResponse{
		ID:        entity.PublicID,
		Object:    "conversation",
		CreatedAt: entity.CreatedAt.Unix(),
		Metadata:  metadata,
	}
}

func (h *ConversationHandler) domainToDeletedConversationResponse(entity *conversation.Conversation) *DeletedConversationResponse {
	return &DeletedConversationResponse{
		ID:      entity.PublicID,
		Object:  "conversation.deleted",
		Deleted: true,
	}
}

func (h *ConversationHandler) domainToConversationItemListResponse(items []*conversation.Item) *ConversationItemListResponse {
	data := make([]ConversationItemResponse, len(items))
	for i, item := range items {
		data[i] = *h.domainToConversationItemResponse(*item)
	}

	result := &ConversationItemListResponse{
		Object:  "list",
		Data:    data,
		HasMore: false, // TODO: Implement proper pagination
	}

	if len(data) > 0 {
		result.FirstID = data[0].ID
		result.LastID = data[len(data)-1].ID
	}

	return result
}

func (h *ConversationHandler) domainToConversationItemResponse(entity conversation.Item) *ConversationItemResponse {
	response := &ConversationItemResponse{
		ID:        entity.PublicID,
		Object:    "conversation.item",
		Type:      string(entity.Type),
		Status:    entity.Status,
		CreatedAt: entity.CreatedAt.Unix(),
		Content:   h.domainToContentResponse(entity.Content),
	}

	if entity.Role != nil {
		role := string(*entity.Role)
		response.Role = &role
	}

	return response
}

func (h *ConversationHandler) domainToContentResponse(content []conversation.Content) []ContentResponse {
	if len(content) == 0 {
		return nil
	}

	result := make([]ContentResponse, len(content))
	for i, c := range content {
		contentResp := ContentResponse{
			Type: c.Type,
		}

		// Handle different content types
		switch c.Type {
		case "text":
			if c.Text != nil {
				contentResp.Text = &TextResponse{
					Value: c.Text.Value,
				}
			}
		case "input_text":
			if c.InputText != nil {
				contentResp.InputText = c.InputText
			}
		case "output_text":
			if c.OutputText != nil {
				contentResp.OutputText = &OutputTextResponse{
					Text:        c.OutputText.Text,
					Annotations: h.domainToAnnotationResponse(c.OutputText.Annotations),
				}
			}
		case "image":
			if c.Image != nil {
				contentResp.Image = &ImageContentResponse{
					URL:    c.Image.URL,
					FileID: c.Image.FileID,
					Detail: c.Image.Detail,
				}
			}
		case "file":
			if c.File != nil {
				contentResp.File = &FileContentResponse{
					FileID:   c.File.FileID,
					Name:     c.File.Name,
					MimeType: c.File.MimeType,
					Size:     c.File.Size,
				}
			}
		}

		result[i] = contentResp
	}
	return result
}

func (h *ConversationHandler) domainToAnnotationResponse(annotations []conversation.Annotation) []AnnotationResponse {
	if len(annotations) == 0 {
		return nil
	}

	result := make([]AnnotationResponse, len(annotations))
	for i, a := range annotations {
		result[i] = AnnotationResponse{
			Type:       a.Type,
			Text:       a.Text,
			FileID:     a.FileID,
			URL:        a.URL,
			StartIndex: a.StartIndex,
			EndIndex:   a.EndIndex,
			Index:      a.Index,
		}
	}
	return result
}
