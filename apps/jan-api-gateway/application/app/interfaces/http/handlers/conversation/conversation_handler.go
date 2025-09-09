package conversation

import (
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

type ConversationItemContextKey string

const (
	ConversationItemContextKeyPublicID ConversationItemContextKey = "conv_item_public_id"
	ConversationItemContextEntity      ConversationItemContextKey = "ConversationItemContextEntity"
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

func (h *ConversationHandler) ListConversations(reqCtx *gin.Context) (httpStatusCode int, result responses.ListResponse[*conversation.Conversation], err error) {
	ctx := reqCtx.Request.Context()
	user, ok := userhandler.GetUserFromContext(reqCtx)
	if !ok {
		result.Status = "50a50c92-a0b9-481f-a7fe-4c0bee16d17f"
		return http.StatusBadRequest, result, nil
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
		result.Status = "afa523f5-2185-4d43-98cc-4d7bfd2aa9a3"
		return http.StatusBadRequest, result, nil
	}
	filter := conversation.ConversationFilter{
		UserID: &userID,
	}
	conversations, err := h.conversationService.FindConversationsByFilter(ctx, filter, pagination)
	if err != nil {
		result.Status = "6c941a53-832c-4bf7-996b-ac5c224b3bb4"
		return http.StatusBadRequest, result, nil
	}
	count, err := h.conversationService.CountConversationsByFilter(ctx, filter)
	if err != nil {
		result.Status = "4fdf9af4-4a80-49fd-b103-b4bcfc59d042"
		return http.StatusBadRequest, result, nil
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
			result.Status = "113f1d21-2d52-4745-ba12-e4f26023c843"
			return http.StatusBadRequest, result, nil
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
func (h *ConversationHandler) CreateConversationByUserID(ctx *gin.Context, userId uint) (httpStatusCode int, result responses.GeneralResponse[*conversation.Conversation], err error) {
	// Parse request
	var request CreateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Status = "b2c3d4e5-f6g7-8901-bcde-f23456789012"
		return http.StatusBadRequest, result, err
	}

	if len(request.Items) > 20 {
		result.Status = "94b83bca-4bbf-458e-a2ce-181a30c3e418"
		return http.StatusBadRequest, result, err
	}

	itemsToCreate := make([]*conversation.Item, len(request.Items))

	for i, itemReq := range request.Items {
		item, ok := NewItemFromConversationItemRequest(itemReq)
		if !ok {
			result.Status = "b3a5d6bc-3a61-44c2-867e-eddf90788408"
			return http.StatusBadRequest, result, err
		}
		itemsToCreate[i] = item
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
		result.Status = "f3b618dc-99a4-41bd-a385-f68ad4073c37"
		return http.StatusInternalServerError, result, err
	}

	// Add items if provided using batch operation
	if len(request.Items) > 0 {
		_, err := h.conversationService.AddMultipleItems(ctx, conv, userId, itemsToCreate)
		if err != nil {
			result.Status = "82fe6b60-f1fb-49f3-8e68-3bfcf1b5355c"
			return http.StatusInternalServerError, result, err
		}

		// Reload conversation with items
		conv, err = h.conversationService.GetConversationByPublicIDAndUserID(ctx, conv.PublicID, userId)
		if err != nil {
			result.Status = "9e0455c9-64af-43ff-aebb-c12921de80d8"
			return http.StatusInternalServerError, result, err
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
func (h *ConversationHandler) CreateItems(reqCtx *gin.Context) (httpStatusCode int, result responses.ListResponse[*conversation.Item], err error) {
	ctx := reqCtx.Request.Context()
	conv, _ := GetConversationFromContext(reqCtx)

	// Parse request
	var request CreateItemsRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		result.Status = "051bc720-4844-4883-8262-c8f1365ed68e"
		return http.StatusBadRequest, result, err
	}

	// Convert all items at once for batch processing
	itemsToCreate := make([]*conversation.Item, len(request.Items))
	for i, itemReq := range request.Items {
		item, ok := NewItemFromConversationItemRequest(itemReq)
		if !ok {
			result.Status = "345b87d3-8d96-4d68-8c3c-408875c30c31"
			return http.StatusBadRequest, result, nil
		}

		itemsToCreate[i] = item
	}

	ok, errorCode := h.conversationService.ValidateItems(ctx, itemsToCreate)
	if !ok {
		if errorCode != nil {
			result.Status = *errorCode
		} else {
			result.Status = "41b80303-0e55-4a24-a079-d2d9340d713b"
		}
		return http.StatusBadRequest, result, nil
	}
	// Single batch operation instead of N individual operations
	createdItems, err := h.conversationService.AddMultipleItems(ctx, conv, conv.UserID, itemsToCreate)
	if err != nil {
		result.Status = "00cbd035-afcb-4095-9d8e-660c6dee8858"
		return http.StatusBadRequest, result, err
	}

	result.Status = responses.ResponseCodeOk
	result.Results = createdItems
	result.HasMore = false
	result.Total = int64(len(createdItems))
	if len(createdItems) > 1 {
		result.FirstID = &createdItems[0].PublicID
		result.LastID = &createdItems[len(createdItems)-1].PublicID
	}
	return http.StatusOK, result, err
}

// ListItems handles item listing
func (h *ConversationHandler) ListItems(reqCtx *gin.Context) (httpStatusCode int, result responses.ListResponse[*conversation.Item], err error) {
	ctx := reqCtx.Request.Context()
	conv, _ := GetConversationFromContext(reqCtx)

	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(lastID string) (*uint, error) {
		items, err := h.conversationService.FindItemsByFilter(ctx, conversation.ItemFilter{
			PublicID:       &lastID,
			ConversationID: &conv.ID,
		}, nil)
		if err != nil {
			return nil, err
		}
		if len(items) != 1 {
			return nil, fmt.Errorf("invalid conversation")
		}
		return &items[0].ID, nil
	})
	if err != nil {
		result.Status = "669ea1d6-f44b-4eb8-978d-fd4fdecf0d13"
		return http.StatusBadRequest, result, nil
	}

	filter := conversation.ItemFilter{
		ConversationID: &conv.ID,
	}
	itemEntities, err := h.conversationService.FindItemsByFilter(ctx, filter, pagination)
	if err != nil {
		result.Status = "352314fb-a912-422c-af3a-305ddbb033b9"
		return http.StatusInternalServerError, result, err
	}
	result.Status = responses.ResponseCodeOk
	result.Results = itemEntities

	if len(itemEntities) > 0 {
		result.FirstID = &itemEntities[0].PublicID
		result.LastID = &itemEntities[len(itemEntities)-1].PublicID
		moreRecords, err := h.conversationService.FindItemsByFilter(ctx, filter, &query.Pagination{
			Order: pagination.Order,
			Limit: ptr.ToInt(1),
			After: &itemEntities[len(itemEntities)-1].ID,
		})
		if err != nil {
			result.Status = "89fa4638-c897-48c7-88f4-0e4fed0a9ce6"
			return http.StatusInternalServerError, result, err
		}
		if len(moreRecords) != 0 {
			result.HasMore = true
		}
	}
	return http.StatusOK, result, nil
}

func (h *ConversationHandler) GetConversationItemMiddleWare() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		conv, ok := GetConversationFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "0f5c3304-bf46-45ce-8719-7c03a3485b37",
			})
			return
		}
		publicID := reqCtx.Param(string(ConversationItemContextKeyPublicID))
		if publicID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "f5b144fe-090e-4251-bed0-66e27c37c328",
				Error: "missing conversation item public ID",
			})
			return
		}

		entities, err := h.conversationService.FindItemsByFilter(ctx, conversation.ItemFilter{
			PublicID:       &publicID,
			ConversationID: &conv.ID,
		}, nil)

		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:          "bff3c8bf-c259-46a1-8ff0-7c2b2dbfe1b2",
				ErrorInstance: err,
			})
			return
		}

		if len(entities) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "25647b40-4967-497e-9cbd-a85243ccef58",
			})
			return
		}

		SetConversationItemFromContext(reqCtx, entities[0])
		reqCtx.Next()
	}
}

func SetConversationItemFromContext(reqCtx *gin.Context, item *conversation.Item) {
	reqCtx.Set(string(ConversationItemContextEntity), item)
}

func GetConversationItemFromContext(reqCtx *gin.Context) (*conversation.Item, bool) {
	item, ok := reqCtx.Get(string(ConversationItemContextEntity))
	if !ok {
		return nil, false
	}
	return item.(*conversation.Item), true
}

// DeleteItem handles item deletion
func (h *ConversationHandler) DeleteItem(reqCtx *gin.Context) (httpStatusCode int, result responses.GeneralResponse[*conversation.Item], err error) {
	ctx := reqCtx.Request.Context()
	conv, ok := GetConversationFromContext(reqCtx)
	if !ok {
		result.Status = "49c615dc-b033-42e7-bf02-4c3126057b9a"
		return http.StatusNotFound, result, nil
	}
	item, ok := GetConversationItemFromContext(reqCtx)
	if !ok {
		result.Status = "e6ddf250-d5a7-493b-95c0-dfe15f3889d3"
		return http.StatusNotFound, result, nil
	}

	// Use efficient deletion with item public ID instead of loading all items
	itemDeleted, err := h.conversationService.DeleteItemWithConversation(ctx, conv, item)
	if err != nil {
		result.Status = "6ccd10c8-290b-43f1-8af5-c3251dee9bc0"
		return http.StatusInternalServerError, result, err
	}

	result.Result = itemDeleted
	result.Status = responses.ResponseCodeOk
	return http.StatusOK, result, nil
}
