package conversations

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"

	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// ConversationAPI handles route registration for V1 conversations
type ConversationAPI struct {
	conversationService *conversation.ConversationService
	authService         *auth.AuthService
}

// NewConversationAPI creates a new conversation API instance
func NewConversationAPI(
	conversationService *conversation.ConversationService,
	authService *auth.AuthService) *ConversationAPI {
	return &ConversationAPI{
		conversationService,
		authService,
	}
}

// RegisterRouter registers OpenAI-compatible conversation routes
func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations",
		api.authService.AppUserAuthMiddleware(),
		api.authService.RegisteredUserMiddleware(),
	)

	conversationsRouter.POST("", api.createConversation)
	conversationsRouter.GET("", api.listConversations)

	conversationMiddleWare := api.conversationService.GetConversationMiddleWare()
	conversationsRouter.GET(fmt.Sprintf("/:%s", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.getConversation)
	conversationsRouter.PATCH(fmt.Sprintf("/:%s", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.updateConversation)
	conversationsRouter.DELETE(fmt.Sprintf("/:%s", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.deleteConversation)
	conversationsRouter.POST(fmt.Sprintf("/:%s/items", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.createItems)
	conversationsRouter.GET(fmt.Sprintf("/:%s/items", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.listItems)

	conversationItemMiddleWare := api.conversationService.GetConversationItemMiddleWare()
	conversationsRouter.GET(
		fmt.Sprintf(
			"/:%s/items/:%s",
			conversation.ConversationContextKeyPublicID,
			conversation.ConversationItemContextKeyPublicID,
		),
		conversationMiddleWare,
		conversationItemMiddleWare,
		api.getItem,
	)
	conversationsRouter.DELETE(
		fmt.Sprintf(
			"/:%s/items/:%s",
			conversation.ConversationContextKeyPublicID,
			conversation.ConversationItemContextKeyPublicID,
		),
		conversationMiddleWare,
		conversationItemMiddleWare,
		api.deleteItem,
	)
}

// ListConversations
// @Summary List Conversations
// @Description Retrieves a paginated list of conversations for the authenticated user.
// @Tags Conversations
// @Security BearerAuth
// @Param limit query int false "The maximum number of items to return" default(20)
// @Param after query string false "A cursor for use in pagination. The ID of the last object from the previous page"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} ListResponse[ConversationResponse] "Successfully retrieved the list of conversations"
// @Failure 400 {object} responses.ErrorResponse "Bad Request - Invalid pagination parameters"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /v1/conversations [get]
func (api *ConversationAPI) listConversations(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, _ := auth.GetUserFromContext(reqCtx)
	userID := user.ID

	result, err := api.doListConversations(ctx, userID, reqCtx)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doListConversations performs the business logic for listing conversations
func (api *ConversationAPI) doListConversations(ctx context.Context, userID uint, reqCtx *gin.Context) (*ListResponse[*ConversationResponse], *common.Error) {
	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(lastID string) (*uint, error) {
		convs, convErr := api.conversationService.FindConversationsByFilter(ctx, conversation.ConversationFilter{
			UserID:   &userID,
			PublicID: &lastID,
		}, nil)
		if !convErr.IsEmpty() {
			return nil, convErr
		}
		if len(convs) != 1 {
			return nil, fmt.Errorf("invalid conversation")
		}
		return &convs[0].ID, nil
	})
	if err != nil {
		return nil, common.NewError("5f89e23d-d4a0-45ce-ba43-ae2a9be0ca64", "Invalid pagination parameters")
	}

	filter := conversation.ConversationFilter{
		UserID: &userID,
	}
	conversations, convErr := api.conversationService.FindConversationsByFilter(ctx, filter, pagination)
	if !convErr.IsEmpty() {
		return nil, convErr
	}
	count, countErr := api.conversationService.CountConversationsByFilter(ctx, filter)
	if !countErr.IsEmpty() {
		return nil, countErr
	}
	var firstId *string
	var lastId *string
	hasMore := false
	if len(conversations) > 0 {
		firstId = &conversations[0].PublicID
		lastId = &conversations[len(conversations)-1].PublicID
		moreRecords, moreErr := api.conversationService.FindConversationsByFilter(ctx, filter, &query.Pagination{
			Order: pagination.Order,
			Limit: ptr.ToInt(1),
			After: &conversations[len(conversations)-1].ID,
		})
		if !moreErr.IsEmpty() {
			return nil, moreErr
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}

	return &ListResponse[*ConversationResponse]{
		Object:  "list",
		FirstID: firstId,
		LastID:  lastId,
		Total:   count,
		HasMore: hasMore,
		Data:    functional.Map(conversations, domainToConversationResponse),
	}, common.EmptyError
}

// ListResponse represents a paginated list response
type ListResponse[T any] struct {
	Object  string  `json:"object"`
	Data    []T     `json:"data"`
	FirstID *string `json:"first_id,omitempty"`
	LastID  *string `json:"last_id,omitempty"`
	HasMore bool    `json:"has_more"`
	Total   int64   `json:"total"`
}

// ConversationResponse represents the response structure
type ConversationResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata"`
}

// createConversation handles conversation creation
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
func (api *ConversationAPI) createConversation(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, _ := auth.GetUserFromContext(reqCtx)
	userId := user.ID

	var request CreateConversationRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "e5c96a9e-7ff9-4408-9514-9d206ca85b33",
			Error: "Invalid request payload",
		})
		return
	}

	result, err := api.doCreateConversation(ctx, userId, request)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doCreateConversation performs the business logic for creating a conversation
func (api *ConversationAPI) doCreateConversation(ctx context.Context, userId uint, request CreateConversationRequest) (*ConversationResponse, *common.Error) {
	if len(request.Items) > 20 {
		return nil, common.NewError("0e5b8426-b1d2-4114-ac81-d3982dc497cf", "Too many items")
	}

	itemsToCreate := make([]*conversation.Item, len(request.Items))

	for i, itemReq := range request.Items {
		item, ok := NewItemFromConversationItemRequest(itemReq)
		if !ok {
			return nil, common.NewError("1fe8d03b-9e1e-4e52-b5b5-77a25954fc43", "Invalid item format")
		}
		itemsToCreate[i] = item
	}

	ok, err := api.conversationService.ValidateItems(ctx, itemsToCreate)
	if !ok {
		return nil, err
	}

	// Create conversation
	conv, err := api.conversationService.CreateConversation(ctx, userId, &request.Title, true, request.Metadata)
	if !err.IsEmpty() {
		return nil, err
	}

	// Add items if provided using batch operation
	if len(request.Items) > 0 {
		_, err := api.conversationService.AddMultipleItems(ctx, conv, userId, itemsToCreate)
		if !err.IsEmpty() {
			return nil, err
		}

		// Reload conversation with items
		conv, err = api.conversationService.GetConversationByPublicIDAndUserID(ctx, conv.PublicID, userId)
		if !err.IsEmpty() {
			return nil, err
		}
	}

	return domainToConversationResponse(conv), common.EmptyError
}

// getConversation handles conversation retrieval
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} ConversationResponse "Conversation details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [get]
func (api *ConversationAPI) getConversation(reqCtx *gin.Context) {
	conv, ok := conversation.GetConversationFromContext(reqCtx)
	if !ok {
		return
	}

	result, err := api.doGetConversation(conv)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doGetConversation performs the business logic for getting a conversation
func (api *ConversationAPI) doGetConversation(conv *conversation.Conversation) (*ConversationResponse, *common.Error) {
	return domainToConversationResponse(conv), common.EmptyError
}

type UpdateConversationRequest struct {
	Title    *string            `json:"title"`
	Metadata *map[string]string `json:"metadata"`
}

// updateConversation handles conversation updates
// @Summary Update a conversation
// @Description Updates conversation metadata
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
// @Router /v1/conversations/{conversation_id} [patch]
func (api *ConversationAPI) updateConversation(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, ok := conversation.GetConversationFromContext(reqCtx)
	if !ok {
		return
	}

	var request UpdateConversationRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "4183e285-08ef-4a79-8a68-d53cddd0c0e2",
			Error: "Invalid request payload",
		})
		return
	}

	result, err := api.doUpdateConversation(ctx, conv, request)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doUpdateConversation performs the business logic for updating a conversation
func (api *ConversationAPI) doUpdateConversation(ctx context.Context, conv *conversation.Conversation, request UpdateConversationRequest) (*ConversationResponse, *common.Error) {
	if request.Title != nil {
		conv.Title = request.Title
	}
	if request.Metadata != nil {
		conv.Metadata = *request.Metadata
	}

	conv, err := api.conversationService.UpdateConversation(ctx, conv)
	if !err.IsEmpty() {
		return nil, err
	}

	return domainToConversationResponse(conv), common.EmptyError
}

// DeletedConversationResponse represents the deleted conversation response
type DeletedConversationResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// deleteConversation handles conversation deletion
// @Summary Delete a conversation
// @Description Deletes a conversation and all its items
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} DeletedConversationResponse "Deleted conversation"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [delete]
func (api *ConversationAPI) deleteConversation(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, ok := conversation.GetConversationFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code: "a4fb6e9b-00c8-423c-9836-a83080e34d28",
		})
		return
	}

	result, err := api.doDeleteConversation(ctx, conv)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doDeleteConversation performs the business logic for deleting a conversation
func (api *ConversationAPI) doDeleteConversation(ctx context.Context, conv *conversation.Conversation) (*DeletedConversationResponse, *common.Error) {
	success, err := api.conversationService.DeleteConversation(ctx, conv)
	if !success {
		return nil, err
	}
	return domainToDeletedConversationResponse(conv), common.EmptyError
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

// createItems handles item creation
// @Summary Create items in a conversation
// @Description Adds multiple items to a conversation
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body CreateItemsRequest true "Create items request"
// @Success 200 {object} ListResponse[ConversationItemResponse] "Created items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) createItems(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, _ := conversation.GetConversationFromContext(reqCtx)

	var request CreateItemsRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "a4fb6e9b-00c8-423c-9836-a83080e34d28",
			Error: "Invalid request payload",
		})
		return
	}

	result, err := api.doCreateItems(ctx, conv, request)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doCreateItems performs the business logic for creating items
func (api *ConversationAPI) doCreateItems(ctx context.Context, conv *conversation.Conversation, request CreateItemsRequest) (*ListResponse[*ConversationItemResponse], *common.Error) {
	itemsToCreate := make([]*conversation.Item, len(request.Items))
	for i, itemReq := range request.Items {
		item, ok := NewItemFromConversationItemRequest(itemReq)
		if !ok {
			return nil, common.NewError("a4fb6e9b-00c8-423c-9836-a83080e34d28", "Invalid item format")
		}
		itemsToCreate[i] = item
	}

	ok, err := api.conversationService.ValidateItems(ctx, itemsToCreate)
	if !ok {
		return nil, err
	}

	createdItems, err := api.conversationService.AddMultipleItems(ctx, conv, conv.UserID, itemsToCreate)
	if !err.IsEmpty() {
		return nil, err
	}
	var firstId *string
	var lastId *string
	if len(createdItems) > 0 {
		firstId = &createdItems[0].PublicID
		lastId = &createdItems[len(createdItems)-1].PublicID
	}

	return &ListResponse[*ConversationItemResponse]{
		Object:  "list",
		Data:    functional.Map(createdItems, domainToConversationItemResponse),
		FirstID: firstId,
		LastID:  lastId,
		HasMore: false,
		Total:   int64(len(createdItems)),
	}, common.EmptyError
}

// listItems handles item listing with optional pagination
// @Summary List items in a conversation
// @Description Lists all items in a conversation
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "Number of items to return (1-100)"
// @Param cursor query string false "Cursor for pagination"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} ListResponse[ConversationItemResponse] "List of items"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) listItems(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, _ := conversation.GetConversationFromContext(reqCtx)

	result, err := api.doListItems(ctx, conv, reqCtx)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doListItems performs the business logic for listing items
func (api *ConversationAPI) doListItems(ctx context.Context, conv *conversation.Conversation, reqCtx *gin.Context) (*ListResponse[*ConversationItemResponse], *common.Error) {
	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(lastID string) (*uint, error) {
		items, err := api.conversationService.FindItemsByFilter(ctx, conversation.ItemFilter{
			PublicID:       &lastID,
			ConversationID: &conv.ID,
		}, nil)
		if !err.IsEmpty() {
			return nil, fmt.Errorf("%s: %s", err.Code, err.Message)
		}
		if len(items) != 1 {
			return nil, fmt.Errorf("invalid conversation")
		}
		return &items[0].ID, nil
	})
	if err != nil {
		return nil, common.NewError("e9144b73-6fc1-4b16-b9c7-460d8a4ecf6b", "Invalid pagination parameters")
	}

	filter := conversation.ItemFilter{
		ConversationID: &conv.ID,
	}
	itemEntities, filterErr := api.conversationService.FindItemsByFilter(ctx, filter, pagination)
	if !filterErr.IsEmpty() {
		return nil, filterErr
	}

	var firstId *string
	var lastId *string
	hasMore := false
	if len(itemEntities) > 0 {
		firstId = &itemEntities[0].PublicID
		lastId = &itemEntities[len(itemEntities)-1].PublicID
		moreRecords, moreErr := api.conversationService.FindItemsByFilter(ctx, filter, &query.Pagination{
			Order: pagination.Order,
			Limit: ptr.ToInt(1),
			After: &itemEntities[len(itemEntities)-1].ID,
		})
		if !moreErr.IsEmpty() {
			return nil, moreErr
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}

	return &ListResponse[*ConversationItemResponse]{
		Object:  "list",
		Data:    functional.Map(itemEntities, domainToConversationItemResponse),
		FirstID: firstId,
		LastID:  lastId,
		HasMore: hasMore,
		Total:   int64(len(itemEntities)),
	}, common.EmptyError
}

// getItem handles single item retrieval
// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} ConversationItemResponse "Item details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) getItem(reqCtx *gin.Context) {
	item, ok := conversation.GetConversationItemFromContext(reqCtx)
	if !ok {
		return
	}

	result, err := api.doGetItem(item)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doGetItem performs the business logic for getting an item
func (api *ConversationAPI) doGetItem(item *conversation.Item) (*ConversationItemResponse, *common.Error) {
	return domainToConversationItemResponse(item), common.EmptyError
}

// deleteItem handles item deletion
// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} ConversationResponse "Updated conversation"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [delete]
func (api *ConversationAPI) deleteItem(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, ok := conversation.GetConversationFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "8fcd7439-a81c-48d3-9208-33afaa7146ac",
		})
		return
	}
	item, ok := conversation.GetConversationItemFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code: "8a03dd04-0a8d-40b5-8664-01ddfb8bcb48",
		})
		return
	}

	result, err := api.doDeleteItem(ctx, conv, item)
	if !err.IsEmpty() {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  err.Code,
			Error: err.Message,
		})
		return
	}

	reqCtx.JSON(http.StatusOK, result)
}

// doDeleteItem performs the business logic for deleting an item
func (api *ConversationAPI) doDeleteItem(ctx context.Context, conv *conversation.Conversation, item *conversation.Item) (*ConversationItemResponse, *common.Error) {
	// Use efficient deletion with item public ID instead of loading all items
	itemDeleted, err := api.conversationService.DeleteItemWithConversation(ctx, conv, item)
	if !err.IsEmpty() {
		return nil, err
	}
	return domainToConversationItemResponse(itemDeleted), common.EmptyError
}

func domainToConversationResponse(entity *conversation.Conversation) *ConversationResponse {
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

func domainToDeletedConversationResponse(entity *conversation.Conversation) *DeletedConversationResponse {
	return &DeletedConversationResponse{
		ID:      entity.PublicID,
		Object:  "conversation.deleted",
		Deleted: true,
	}
}

func domainToConversationItemResponse(entity *conversation.Item) *ConversationItemResponse {
	response := &ConversationItemResponse{
		ID:        entity.PublicID,
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
					Annotations: domainToAnnotationResponse(c.OutputText.Annotations),
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

func domainToAnnotationResponse(annotations []conversation.Annotation) []AnnotationResponse {
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

// CreateConversationRequest represents the input for creating a conversation
type CreateConversationRequest struct {
	Title    string                    `json:"title"`
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

func NewItemFromConversationItemRequest(itemReq ConversationItemRequest) (*conversation.Item, bool) {
	ok := conversation.ValidateItemType(string(itemReq.Type))
	if !ok {
		return nil, false
	}
	itemType := conversation.ItemType(itemReq.Type)

	var role *conversation.ItemRole
	if itemReq.Role != "" {
		ok := conversation.ValidateItemRole(string(itemReq.Role))
		if !ok {
			return nil, false
		}
		r := conversation.ItemRole(itemReq.Role)
		role = &r
	}

	content := make([]conversation.Content, len(itemReq.Content))
	for j, c := range itemReq.Content {
		content[j] = conversation.Content{
			Type: c.Type,
			Text: &conversation.Text{
				Value: c.Text,
			},
		}
	}

	return &conversation.Item{
		Type:    itemType,
		Role:    role,
		Content: content,
	}, true
}

type CreateItemsRequest struct {
	Items []ConversationItemRequest `json:"items" binding:"required"`
}
