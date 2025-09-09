package conversation

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/handlers/userhandler"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ConversationContextKey string

const (
	ConversationContextKeyPublicID ConversationContextKey = "conv_public_id"
	ConversationContextEntity      ConversationContextKey = "ConversationContextEntity"
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
	Role    conversation.ItemRole        `json:"role,omitempty"`
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

// TODO: OpenAI doesn't provide title, we will need to go back and fix the model
// UpdateConversationRequest represents the request body for updating a conversation
type UpdateConversationRequest struct {
	Title    *string            `json:"title"`
	Metadata *map[string]string `json:"metadata"`
}

// CreateItemsRequest represents the request body for creating items
type CreateItemsRequest struct {
	Items []ConversationItemRequest `json:"items" binding:"required"`
}

func (h *ConversationHandler) ListConversations(reqCtx *gin.Context) (int, responses.ListResponse[*conversation.Conversation], error) {
	ctx := reqCtx.Request.Context()
	user, ok := userhandler.GetUserFromContext(reqCtx)
	if !ok {
		return http.StatusBadRequest, responses.ListResponse[*conversation.Conversation]{
			Status: "50a50c92-a0b9-481f-a7fe-4c0bee16d17f",
		}, nil
	}
	userID := user.ID
	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(lastID string) (*uint, error) {
		convs, err := h.conversationService.FindConversationsByFilter(ctx, conversation.ConversationFilter{
			UserID:   &userID,
			PublicID: &lastID,
		}, nil)
		if err != nil {
			return nil, err
		}
		if len(convs) != 1 {
			return nil, fmt.Errorf("invalid conversation")
		}
		return &convs[0].ID, nil
	})
	if err != nil {
		return http.StatusBadRequest, responses.ListResponse[*conversation.Conversation]{
			Status: "c404de95-8895-41b5-8bb1-e9c260155315",
		}, err
	}
	filter := conversation.ConversationFilter{
		UserID: &userID,
	}
	conversations, err := h.conversationService.FindConversationsByFilter(ctx, filter, pagination)
	if err != nil {
		return http.StatusInternalServerError, responses.ListResponse[*conversation.Conversation]{
			Status: "adb7f0d3-127a-447d-abcf-0dfb50afd548",
		}, err
	}
	count, err := h.conversationService.CountConversationsByFilter(ctx, filter)
	if err != nil {
		return http.StatusInternalServerError, responses.ListResponse[*conversation.Conversation]{
			Status: "0eba79b1-5a32-4552-b66c-419feeabb790",
		}, err
	}
	var firstId *string
	var lastId *string
	hasMore := false
	if len(conversations) > 0 {
		firstId = &conversations[0].PublicID
		lastId = &conversations[len(conversations)-1].PublicID
		moreRecords, err := h.conversationService.FindConversationsByFilter(ctx, filter, &query.Pagination{
			Order: pagination.Order,
			Limit: ptr.ToInt(1),
			After: &conversations[len(conversations)-1].ID,
		})
		if err != nil {
			return http.StatusInternalServerError, responses.ListResponse[*conversation.Conversation]{
				Status: "89fa4638-c897-48c7-88f4-0e4fed0a9ce6",
			}, err
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}

	return http.StatusOK, responses.ListResponse[*conversation.Conversation]{
		Status:  responses.ResponseCodeOk,
		Results: conversations,
		Total:   count,
		HasMore: hasMore,
		FirstID: firstId,
		LastID:  lastId,
	}, nil
}

// CreateConversation handles conversation creation
func (h *ConversationHandler) CreateConversationByUserID(ctx *gin.Context, userId uint) (int, responses.GeneralResponse[*conversation.Conversation], error) {
	// Parse request
	var request CreateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return http.StatusBadRequest, responses.GeneralResponse[*conversation.Conversation]{
			Status: "b2c3d4e5-f6g7-8901-bcde-f23456789012",
		}, err
	}

	if len(request.Items) > 20 {
		return http.StatusBadRequest, responses.GeneralResponse[*conversation.Conversation]{
			Status: "c8ac11af-5972-41b6-a5fb-fde39b9d0a0f",
		}, nil
	}

	itemsToCreate := make([]*conversation.Item, len(request.Items))

	for i, itemReq := range request.Items {
		// Convert request to domain types
		ok := conversation.ValidateItemType(string(itemReq.Type))
		if !ok {
			return http.StatusBadRequest, responses.GeneralResponse[*conversation.Conversation]{
				Status: "ac4d231e-c085-432f-94df-ef2440c1e01a",
			}, nil
		}
		itemType := conversation.ItemType(itemReq.Type)
		var role *conversation.ItemRole
		if itemReq.Role != "" {
			ok := conversation.ValidateItemRole(string(itemReq.Role))
			if !ok {
				return http.StatusBadRequest, responses.GeneralResponse[*conversation.Conversation]{
					Status: "4a9fc6d5-6c34-4f9f-a2ea-becfab1ecbcb",
				}, nil
			}
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

	ok, errorCode := h.conversationService.ValidateItems(ctx, itemsToCreate)
	if !ok {
		return http.StatusBadRequest, responses.GeneralResponse[*conversation.Conversation]{
			Status: *errorCode,
		}, nil
	}

	// Create conversation
	conv, err := h.conversationService.CreateConversation(ctx, userId, nil, true, request.Metadata)
	if err != nil {
		return http.StatusBadRequest, responses.GeneralResponse[*conversation.Conversation]{
			Status: "0f58924c-3c32-4ab1-b39b-dd74d34b5b2c",
		}, nil
	}

	// Add items if provided using batch operation
	if len(request.Items) > 0 {

		_, err := h.conversationService.AddMultipleItems(ctx, conv, userId, itemsToCreate)
		if err != nil {
			return http.StatusInternalServerError, responses.GeneralResponse[*conversation.Conversation]{
				Status: "af56ad8e-e88a-4ca2-8a1d-46c3ade25df0",
			}, nil
		}

		// Reload conversation with items
		conv, err = h.conversationService.GetConversationByPublicIDAndUserID(ctx, conv.PublicID, userId)
		if err != nil {
			return http.StatusBadRequest, responses.GeneralResponse[*conversation.Conversation]{
				Status: "d4e5f6g7-h8i9-0123-defg-456789012345",
			}, nil
		}
	}

	return http.StatusOK, responses.GeneralResponse[*conversation.Conversation]{
		Status: responses.ResponseCodeOk,
		Result: conv,
	}, nil
}

func (h *ConversationHandler) GetConversationMiddleWare() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		publicID := reqCtx.Param(string(ConversationContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "f5742805-2c6e-45a8-b6a8-95091b9d46f0",
				Error: "missing conversation public ID",
			})
			return
		}
		user, ok := userhandler.GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "f5742805-2c6e-45a8-b6a8-95091b9d46f0",
			})
			return
		}
		entities, err := h.conversationService.FindConversationsByFilter(ctx, conversation.ConversationFilter{
			PublicID: &publicID,
			UserID:   &user.ID,
		}, nil)

		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:          "1fe94ab8-ba2c-4356-a446-f091c256e260",
				ErrorInstance: err,
			})
			return
		}

		if len(entities) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "e91636c2-fced-4a89-bf08-55309005365f",
			})
			return
		}

		SetConversationFromContext(reqCtx, entities[0])
		reqCtx.Next()
	}
}

func SetConversationFromContext(reqCtx *gin.Context, conv *conversation.Conversation) {
	reqCtx.Set(string(ConversationContextEntity), conv)
}

func GetConversationFromContext(reqCtx *gin.Context) (*conversation.Conversation, bool) {
	conv, ok := reqCtx.Get(string(ConversationContextEntity))
	if !ok {
		return nil, false
	}
	return conv.(*conversation.Conversation), true
}

// UpdateConversation handles conversation updates
func (h *ConversationHandler) UpdateConversation(reqCtx *gin.Context) (httpStatus int, result responses.GeneralResponse[conversation.Conversation], err error) {
	ctx := reqCtx.Request.Context()
	conv, ok := GetConversationFromContext(reqCtx)
	if !ok {
		return
	}

	var request UpdateConversationRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		result.Status = "6ebfd85f-a462-4176-8ea7-d9367330b699"
		return http.StatusBadRequest, result, err
	}

	if request.Title != nil {
		conv.Title = request.Title
	}
	if request.Metadata != nil {
		conv.Metadata = *request.Metadata
	}

	conv, err = h.conversationService.UpdateConversation(ctx, conv)
	if err != nil {
		result.Status = "6ebfd85f-a462-4176-8ea7-d9367330b699"
		return http.StatusInternalServerError, result, err
	}
	result.Result = *conv
	result.Status = responses.ResponseCodeOk
	return http.StatusOK, result, nil
}

// DeleteConversation handles conversation deletion
func (h *ConversationHandler) DeleteConversation(reqCtx *gin.Context) (httpStatusCode int, result responses.GeneralResponse[conversation.Conversation], err error) {
	ctx := reqCtx.Request.Context()
	conv, ok := GetConversationFromContext(reqCtx)
	if !ok {
		result.Status = "db18fa07-a469-4564-8110-c0275d460653"
		return http.StatusNotFound, result, nil
	}

	err = h.conversationService.DeleteConversation(ctx, conv)
	if err != nil {
		result.Status = "691463d7-79a9-48e9-aa70-3c0787dbc840"
		return http.StatusInternalServerError, result, err
	}

	result.Result = *conv
	result.Status = responses.ResponseCodeOk
	return http.StatusOK, result, nil
}

// CreateItems handles item creation
func (h *ConversationHandler) CreateItems(ctx *gin.Context) {
	// Authenticate user
	user, err := h.authenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "v2w3x4y5-z6a7-8901-vwxy-234567890123",
			Error: "Invalid or missing API key",
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
	// Authenticate user
	user, err := h.authenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b8c9d0e1-f2g3-4567-bcde-890123456789",
			Error: "Invalid or missing API key",
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
	// Authenticate user
	user, err := h.authenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f2g3h4i5-j6k7-8901-fghi-234567890123",
			Error: "Invalid or missing API key",
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
	// Authenticate user
	user, err := h.authenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "k7l8m9n0-o1p2-3456-klmn-789012345678",
			Error: "Invalid or missing API key",
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
		CreatedAt: entity.CreatedAt,
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
		CreatedAt: entity.CreatedAt,
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
