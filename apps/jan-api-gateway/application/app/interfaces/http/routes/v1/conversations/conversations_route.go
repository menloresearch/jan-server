package conversations

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/workspace"

	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses/openai"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// ConversationAPI handles route registration for V1 conversations
type ConversationAPI struct {
	conversationService *conversation.ConversationService
	authService         *auth.AuthService
	workspaceService    *workspace.WorkspaceService
}

// Request structs
type CreateConversationRequest struct {
	Title       string                    `json:"title"`
	Metadata    map[string]string         `json:"metadata,omitempty"`
	Items       []ConversationItemRequest `json:"items,omitempty"`
	WorkspaceID string                    `json:"workspace_id,omitempty"`
}

type UpdateConversationRequest struct {
	Title    *string            `json:"title"`
	Metadata *map[string]string `json:"metadata"`
}

type UpdateConversationWorkspaceRequest struct {
	WorkspaceID *string `json:"workspace_id,omitempty"`
}

type ConversationItemRequest struct {
	Type    string                       `json:"type" binding:"required"`
	Role    conversation.ItemRole        `json:"role,omitempty"`
	Content []ConversationContentRequest `json:"content" binding:"required"`
}

type ConversationContentRequest struct {
	Type string `json:"type" binding:"required"`
	Text string `json:"text,omitempty"`
}

type CreateItemsRequest struct {
	Items []ConversationItemRequest `json:"items" binding:"required"`
}

// Response structs
type ExtendedConversationResponse struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	Object            string            `json:"object"`
	WorkspacePublicID string            `json:"workspace_id,omitempty"`
	CreatedAt         int64             `json:"created_at"`
	Metadata          map[string]string `json:"metadata"`
}

type DeletedConversationResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

type ConversationItemResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	Type      string            `json:"type"`
	Role      *string           `json:"role,omitempty"`
	Status    *string           `json:"status,omitempty"`
	CreatedAt int64             `json:"created_at"`
	Content   []ContentResponse `json:"content,omitempty"`
}

type ContentResponse struct {
	Type             string                `json:"type"`
	FinishReason     *string               `json:"finish_reason,omitempty"`
	Text             *TextResponse         `json:"text,omitempty"`
	InputText        *string               `json:"input_text,omitempty"`
	OutputText       *OutputTextResponse   `json:"output_text,omitempty"`
	ReasoningContent *string               `json:"reasoning_content,omitempty"`
	Image            *ImageContentResponse `json:"image,omitempty"`
	File             *FileContentResponse  `json:"file,omitempty"`
}

type TextResponse struct {
	Value string `json:"value"`
}

type OutputTextResponse struct {
	Text        string               `json:"text"`
	Annotations []AnnotationResponse `json:"annotations"`
}

type ImageContentResponse struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type FileContentResponse struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type AnnotationResponse struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	FileID     string `json:"file_id,omitempty"`
	URL        string `json:"url,omitempty"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Index      int    `json:"index,omitempty"`
}

// NewConversationAPI creates a new conversation API instance
func NewConversationAPI(
	conversationService *conversation.ConversationService,
	authService *auth.AuthService,
	workspaceService *workspace.WorkspaceService) *ConversationAPI {
	return &ConversationAPI{
		conversationService,
		authService,
		workspaceService,
	}
}

// RegisterRouter registers OpenAI-compatible conversation routes
func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations",
		api.authService.AppUserAuthMiddleware(),
		api.authService.RegisteredUserMiddleware(),
	)

	conversationsRouter.POST("", api.CreateConversationHandler)
	conversationsRouter.GET("", api.ListConversationsHandler)

	conversationMiddleWare := api.conversationService.GetConversationMiddleWare()
	conversationsRouter.PATCH(
		fmt.Sprintf("/:%s/workspace", conversation.ConversationContextKeyPublicID),
		conversationMiddleWare,
		api.UpdateConversationWorkspaceHandler,
	)
	conversationsRouter.GET(fmt.Sprintf("/:%s", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.GetConversationHandler)
	conversationsRouter.PATCH(fmt.Sprintf("/:%s", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.UpdateConversationHandler)
	conversationsRouter.DELETE(fmt.Sprintf("/:%s", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.DeleteConversationHandler)
	conversationsRouter.POST(fmt.Sprintf("/:%s/items", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.CreateItemsHandler)
	conversationsRouter.GET(fmt.Sprintf("/:%s/items", conversation.ConversationContextKeyPublicID), conversationMiddleWare, api.ListItemsHandler)

	conversationItemMiddleWare := api.conversationService.GetConversationItemMiddleWare()
	conversationsRouter.GET(
		fmt.Sprintf(
			"/:%s/items/:%s",
			conversation.ConversationContextKeyPublicID,
			conversation.ConversationItemContextKeyPublicID,
		),
		conversationMiddleWare,
		conversationItemMiddleWare,
		api.GetItemHandler,
	)
	conversationsRouter.DELETE(
		fmt.Sprintf(
			"/:%s/items/:%s",
			conversation.ConversationContextKeyPublicID,
			conversation.ConversationItemContextKeyPublicID,
		),
		conversationMiddleWare,
		conversationItemMiddleWare,
		api.DeleteItemHandler,
	)
}

// @Summary List Conversations
// @Description Retrieves a paginated list of conversations for the authenticated user with OpenAI-compatible response format.
// @Tags Conversations API
// @Security BearerAuth
// @Param limit query int false "The maximum number of items to return" default(20)
// @Param after query string false "A cursor for use in pagination. The ID of the last object from the previous page"
// @Param order query string false "Order of items (asc/desc)"
// @Param workspace_id query string false "Filter conversations by workspace public ID"
// @Success 200 {object} object "Successfully retrieved the list of conversations"
// @Failure 400 {object} responses.ErrorResponse "Bad Request - Invalid pagination parameters"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /v1/conversations [get]
func (api *ConversationAPI) ListConversationsHandler(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, _ := auth.GetUserFromContext(reqCtx)
	userID := user.ID

	workspaceIDParam := strings.TrimSpace(reqCtx.Query("workspace_id"))

	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(lastID string) (*uint, error) {
		filter := conversation.ConversationFilter{
			UserID:   &userID,
			PublicID: &lastID,
		}
		if workspaceIDParam != "" {
			filter.WorkspacePublicID = &workspaceIDParam
		}
		convs, convErr := api.conversationService.FindConversationsByFilter(ctx, filter, nil)
		if convErr != nil {
			return nil, convErr
		}
		if len(convs) != 1 {
			return nil, fmt.Errorf("invalid conversation")
		}
		return &convs[0].ID, nil
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "5f89e23d-d4a0-45ce-ba43-ae2a9be0ca64",
			Error: "Invalid pagination parameters",
		})
		return
	}

	filter := conversation.ConversationFilter{
		UserID: &userID,
	}
	if workspaceIDParam != "" {
		filter.WorkspacePublicID = &workspaceIDParam
	}
	conversations, convErr := api.conversationService.FindConversationsByFilter(ctx, filter, pagination)
	if convErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "019952d5-6876-7323-96fa-89784b7d082e",
			ErrorInstance: convErr.GetError(),
		})
		return
	}
	count, countErr := api.conversationService.CountConversationsByFilter(ctx, filter)
	if countErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "019952d5-1afc-7229-bb57-6928f54d5171",
			ErrorInstance: convErr.GetError(),
		})
		return
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
		if moreErr != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:          "019952d5-983a-73a8-8439-290ae4b4ee51",
				ErrorInstance: convErr.GetError(),
			})
			return
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}

	result := functional.Map(conversations, domainToExtendedConversationResponse)

	response := openai.ListResponse[*ExtendedConversationResponse]{
		Object:  "list",
		FirstID: firstId,
		LastID:  lastId,
		Total:   count,
		HasMore: hasMore,
		Data:    result,
	}

	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Create a conversation
// @Description Creates a new conversation for the authenticated user with optional items
// @Tags Conversations API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateConversationRequest true "Create conversation request"
// @Success 200 {object} ExtendedConversationResponse "Created conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - Bad payload, too many items, or invalid item format"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations [post]
func (api *ConversationAPI) CreateConversationHandler(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	user, _ := auth.GetUserFromContext(reqCtx)
	userId := user.ID

	var request CreateConversationRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "e5c96a9e-7ff9-4408-9514-9d206ca85b33",
			ErrorInstance: err,
		})
		return
	}

	if len(request.Items) > 20 {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "0e5b8426-b1d2-4114-ac81-d3982dc497cf",
			Error: "Too many items",
		})
		return
	}

	itemsToCreate := make([]*conversation.Item, len(request.Items))

	for i, itemReq := range request.Items {
		item, ok := NewItemFromConversationItemRequest(itemReq)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "1fe8d03b-9e1e-4e52-b5b5-77a25954fc43",
				Error: "Invalid item format",
			})
			return
		}
		itemsToCreate[i] = item
	}

	err := api.conversationService.ValidateItems(ctx, itemsToCreate)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952d0-1dc9-746e-82ff-dd42b1e7930f",
			ErrorInstance: err.GetError(),
		})
		return
	}

	// validate workspace ID belongs to user and fetch the workspace public ID
	workspace, err := api.workspaceService.GetWorkspaceByPublicIDAndUserID(ctx, request.WorkspaceID, userId)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952d0-1dc9-746e-82ff-dd42b1e7930f",
			ErrorInstance: err.GetError(),
		})
		return
	}

	workspacePublicID := workspace.PublicID

	// Create conversation
	conv, err := api.conversationService.CreateConversation(ctx, userId, &request.Title, true, request.Metadata, &workspacePublicID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952d0-3e32-76ba-a97f-711223df2c84",
			ErrorInstance: err.GetError(),
		})
		return
	}

	// Add items if provided using batch operation
	if len(request.Items) > 0 {
		_, err := api.conversationService.AddMultipleItems(ctx, conv, userId, itemsToCreate)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:          "019952d0-6d70-7419-81ae-828d8009ee56",
				ErrorInstance: err.GetError(),
			})
			return
		}
	}

	response := domainToExtendedConversationResponse(conv)
	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Get a conversation
// @Description Retrieves a conversation by its ID with full metadata and title
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} ExtendedConversationResponse "Conversation details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [get]
func (api *ConversationAPI) GetConversationHandler(reqCtx *gin.Context) {
	conv, ok := conversation.GetConversationFromContext(reqCtx)
	if !ok {
		return
	}
	response := domainToExtendedConversationResponse(conv)
	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Update a conversation
// @Description Updates conversation title and/or metadata
// @Tags Conversations API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body UpdateConversationRequest true "Update conversation request"
// @Success 200 {object} ExtendedConversationResponse "Updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload or update failed"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [patch]
func (api *ConversationAPI) UpdateConversationHandler(reqCtx *gin.Context) {
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

	if request.Title != nil {
		conv.Title = request.Title
	}
	if request.Metadata != nil {
		conv.Metadata = *request.Metadata
	}

	conv, err := api.conversationService.UpdateConversation(ctx, conv)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952d0-a754-73bc-adbc-781ac31e12d7",
			ErrorInstance: err,
		})
		return
	}

	response := domainToExtendedConversationResponse(conv)
	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Delete a conversation
// @Description Deletes a conversation and all its items permanently
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} DeletedConversationResponse "Deleted conversation"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [delete]
func (api *ConversationAPI) DeleteConversationHandler(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, ok := conversation.GetConversationFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "a4fb6e9b-00c8-423c-9836-a83080e34d28",
			Error: "Conversation not found",
		})
		return
	}

	success, err := api.conversationService.DeleteConversation(ctx, conv)
	if !success {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952c3-9836-75ea-9785-a8d035a7c136",
			ErrorInstance: err.GetError(),
		})
	}
	response := domainToDeletedConversationResponse(conv)

	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Update conversation workspace
// @Description Moves a conversation to another workspace or removes it from a workspace
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body UpdateConversationWorkspaceRequest true "Workspace assignment payload"
// @Success 200 {object} ExtendedConversationResponse "Updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Workspace or conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/workspace [patch]
func (api *ConversationAPI) UpdateConversationWorkspaceHandler(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, ok := conversation.GetConversationFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "a4fb6e9b-00c8-423c-9836-a83080e34d28",
			Error: "conversation not found",
		})
		return
	}

	var request UpdateConversationWorkspaceRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "7733200c-9a0b-43e1-8807-6b5fa2799f77",
			Error: "Invalid request payload",
		})
		return
	}

	var workspacePublicID *string
	if request.WorkspaceID != nil && strings.TrimSpace(*request.WorkspaceID) != "" {
		trimmedWorkspaceID := strings.TrimSpace(*request.WorkspaceID)
		workspaceEntity, err := api.workspaceService.GetWorkspaceByPublicIDAndUserID(ctx, trimmedWorkspaceID, conv.UserID)
		if err != nil {
			status := http.StatusInternalServerError
			if err.GetCode() == "c8bc424c-5b20-4cf9-8ca1-7d9ad1b098c8" {
				status = http.StatusNotFound
			}
			reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
				Code:  err.GetCode(),
				Error: err.Error(),
			})
			return
		}
		publicID := workspaceEntity.PublicID
		workspacePublicID = &publicID
	} else {
		workspacePublicID = nil
	}

	updatedConv, err := api.conversationService.UpdateConversationWorkspace(ctx, conv, workspacePublicID)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  "4c46036b-1f5b-4251-80c9-3d20292a0194",
			Error: err.Error(),
		})
		return
	}

	response := domainToExtendedConversationResponse(updatedConv)
	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Create items in a conversation
// @Description Adds multiple items to a conversation with OpenAI-compatible format
// @Tags Conversations API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body CreateItemsRequest true "Create items request"
// @Success 200 {object} openai.ListResponse[ConversationItemResponse] "Created items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload or invalid item format"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) CreateItemsHandler(reqCtx *gin.Context) {
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

	itemsToCreate := make([]*conversation.Item, len(request.Items))
	for i, itemReq := range request.Items {
		item, ok := NewItemFromConversationItemRequest(itemReq)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "a4fb6e9b-00c8-423c-9836-a83080e34d28",
				Error: "Invalid item format",
			})
			return
		}
		itemsToCreate[i] = item
	}

	err := api.conversationService.ValidateItems(ctx, itemsToCreate)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952d1-265e-738b-bada-7918c32a61d2",
			ErrorInstance: err.GetError(),
		})
		return
	}

	createdItems, err := api.conversationService.AddMultipleItems(ctx, conv, conv.UserID, itemsToCreate)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952d1-68dc-73f3-84e3-0f53d8fce318",
			ErrorInstance: err.GetError(),
		})
		return
	}

	var firstId *string
	var lastId *string
	if len(createdItems) > 0 {
		firstId = &createdItems[0].PublicID
		lastId = &createdItems[len(createdItems)-1].PublicID
	}

	response := &openai.ListResponse[*ConversationItemResponse]{
		Object:  "list",
		Data:    functional.Map(createdItems, domainToConversationItemResponse),
		FirstID: firstId,
		LastID:  lastId,
		HasMore: false,
		Total:   int64(len(createdItems)),
	}

	reqCtx.JSON(http.StatusOK, response)
}

// @Summary List items in a conversation
// @Description Lists all items in a conversation with OpenAI-compatible pagination
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "Number of items to return (1-100)"
// @Param after query string false "Cursor for pagination - ID of the last item from previous page"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} openai.ListResponse[ConversationItemResponse] "List of items"
// @Failure 400 {object} responses.ErrorResponse "Bad Request - Invalid pagination parameters"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) ListItemsHandler(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	conv, _ := conversation.GetConversationFromContext(reqCtx)

	pagination, err := query.GetCursorPaginationFromQuery(reqCtx, func(lastID string) (*uint, error) {
		items, err := api.conversationService.FindItemsByFilter(ctx, conversation.ItemFilter{
			PublicID:       &lastID,
			ConversationID: &conv.ID,
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", err.GetCode(), err.Error())
		}
		if len(items) != 1 {
			return nil, fmt.Errorf("invalid conversation")
		}
		return &items[0].ID, nil
	})
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "e9144b73-6fc1-4b16-b9c7-460d8a4ecf6b",
			Error: "Invalid pagination parameters",
		})
		return
	}

	filter := conversation.ItemFilter{
		ConversationID: &conv.ID,
	}
	itemEntities, filterErr := api.conversationService.FindItemsByFilter(ctx, filter, pagination)
	if filterErr != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:          "019952d1-a6d2-76ff-9c10-3e9264056f90",
			ErrorInstance: filterErr.GetError(),
		})
		return
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
		if moreErr != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:          "019952d1-e914-7466-b527-49e498129426",
				ErrorInstance: moreErr.GetError(),
			})
			return
		}
		if len(moreRecords) != 0 {
			hasMore = true
		}
	}

	response := &openai.ListResponse[*ConversationItemResponse]{
		Object:  "list",
		Data:    functional.Map(itemEntities, domainToConversationItemResponse),
		FirstID: firstId,
		LastID:  lastId,
		HasMore: hasMore,
		Total:   int64(len(itemEntities)),
	}

	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation with full content details
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} ConversationItemResponse "Item details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) GetItemHandler(reqCtx *gin.Context) {
	item, ok := conversation.GetConversationItemFromContext(reqCtx)
	if !ok {
		return
	}

	response := domainToConversationItemResponse(item)
	reqCtx.JSON(http.StatusOK, response)
}

// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation and returns the deleted item details
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} ConversationItemResponse "Deleted item details"
// @Failure 400 {object} responses.ErrorResponse "Bad Request - Deletion failed"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items/{item_id} [delete]
func (api *ConversationAPI) DeleteItemHandler(reqCtx *gin.Context) {
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

	// Use efficient deletion with item public ID instead of loading all items
	_, err := api.conversationService.DeleteItemWithConversation(ctx, conv, item)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:          "019952d2-0f0f-730c-9abc-3fecc1db55c2",
			ErrorInstance: err,
		})
		return
	}

	// OpenAI: Returns the updated Conversation object.
	response := domainToExtendedConversationResponse(conv)
	reqCtx.JSON(http.StatusOK, response)
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

func domainToExtendedConversationResponse(entity *conversation.Conversation) *ExtendedConversationResponse {
	metadata := entity.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}
	return &ExtendedConversationResponse{
		ID:                entity.PublicID,
		Object:            "conversation",
		Title:             ptr.FromString(entity.Title),
		WorkspacePublicID: ptr.FromString(entity.WorkspacePublicID),
		CreatedAt:         entity.CreatedAt.Unix(),
		Metadata:          metadata,
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
		Status:    conversation.ItemStatusToStringPtr(entity.Status),
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

		// Handle finish reason (available for all content types)
		if c.FinishReason != nil {
			contentResp.FinishReason = c.FinishReason
		}

		// Handle reasoning content (available for all content types)
		if c.ReasoningContent != nil {
			contentResp.ReasoningContent = c.ReasoningContent
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
