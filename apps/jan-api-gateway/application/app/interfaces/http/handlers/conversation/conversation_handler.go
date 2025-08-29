package conversation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	conversationUseCase "menlo.ai/jan-api-gateway/app/usecases/conversation"
)

// ConversationHandler handles HTTP requests for conversations using the use case layer
type ConversationHandler struct {
	useCase *conversationUseCase.ConversationUseCase
}

// NewConversationHandler creates a new conversation handler
func NewConversationHandler(useCase *conversationUseCase.ConversationUseCase) *ConversationHandler {
	return &ConversationHandler{
		useCase: useCase,
	}
}

// CreateConversation handles conversation creation
func (h *ConversationHandler) CreateConversation(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Error: "Invalid or missing API key",
		})
		return
	}

	// Parse request
	var request conversationUseCase.CreateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "b2c3d4e5-f6g7-8901-bcde-f23456789012",
			Error: "Invalid request format",
		})
		return
	}

	// Execute use case
	response, statusCode, err := h.useCase.CreateConversation(ctx, user, request)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "c3d4e5f6-g7h8-9012-cdef-345678901234",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}

// GetConversation handles conversation retrieval
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} conversationUseCase.ConversationResponse "Conversation details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [get]
func (h *ConversationHandler) GetConversation(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "d4e5f6g7-h8i9-0123-defg-456789012345",
			Error: "Invalid or missing API key",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Execute use case
	response, statusCode, err := h.useCase.GetConversation(ctx, user, conversationID)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "e5f6g7h8-i9j0-1234-efgh-567890123456",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}

// UpdateConversation handles conversation updates
// @Summary Update a conversation
// @Description Updates conversation metadata
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body conversationUseCase.UpdateConversationRequest true "Update conversation request"
// @Success 200 {object} conversationUseCase.ConversationResponse "Updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [patch]
func (h *ConversationHandler) UpdateConversation(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f6g7h8i9-j0k1-2345-fghi-678901234567",
			Error: "Invalid or missing API key",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Parse request
	var request conversationUseCase.UpdateConversationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "g7h8i9j0-k1l2-3456-ghij-789012345678",
			Error: "Invalid request format",
		})
		return
	}

	// Execute use case
	response, statusCode, err := h.useCase.UpdateConversation(ctx, user, conversationID, request)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "h8i9j0k1-l2m3-4567-hijk-890123456789",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}

// DeleteConversation handles conversation deletion
// @Summary Delete a conversation
// @Description Deletes a conversation and all its items
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} conversationUseCase.DeletedConversationResponse "Deleted conversation"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [delete]
func (h *ConversationHandler) DeleteConversation(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "i9j0k1l2-m3n4-5678-ijkl-901234567890",
			Error: "Invalid or missing API key",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Execute use case
	response, statusCode, err := h.useCase.DeleteConversation(ctx, user, conversationID)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "j0k1l2m3-n4o5-6789-jklm-012345678901",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}

// CreateItems handles item creation
// @Summary Create items in a conversation
// @Description Adds multiple items to a conversation
// @Tags Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body conversationUseCase.CreateItemsRequest true "Create items request"
// @Success 200 {object} conversationUseCase.ConversationItemListResponse "Created items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items [post]
func (h *ConversationHandler) CreateItems(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "k1l2m3n4-o5p6-7890-klmn-123456789012",
			Error: "Invalid or missing API key",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// Parse request
	var request conversationUseCase.CreateItemsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "l2m3n4o5-p6q7-8901-lmno-234567890123",
			Error: "Invalid request format",
		})
		return
	}

	// Execute use case
	response, statusCode, err := h.useCase.CreateItems(ctx, user, conversationID, request)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "m3n4o5p6-q7r8-9012-mnop-345678901234",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}

// ListItems handles item listing with optional pagination
// @Summary List items in a conversation
// @Description Lists all items in a conversation
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "Number of items to return (1-100)"
// @Param cursor query string false "Cursor for pagination"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} conversationUseCase.ConversationItemListResponse "List of items"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items [get]
func (h *ConversationHandler) ListItems(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "n4o5p6q7-r8s9-0123-nopq-456789012345",
			Error: "Invalid or missing API key",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")

	// TODO: Add pagination support when needed
	// For now, use the simple list method
	response, statusCode, err := h.useCase.ListItems(ctx, user, conversationID)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "o5p6q7r8-s9t0-1234-opqr-567890123456",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}

// GetItem handles single item retrieval
// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} conversationUseCase.ConversationItemResponse "Item details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [get]
func (h *ConversationHandler) GetItem(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "p6q7r8s9-t0u1-2345-pqrs-678901234567",
			Error: "Invalid or missing API key",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemID := ctx.Param("item_id")

	// Execute use case
	response, statusCode, err := h.useCase.GetItem(ctx, user, conversationID, itemID)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "q7r8s9t0-u1v2-3456-qrst-789012345678",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}

// DeleteItem handles item deletion
// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation
// @Tags Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} conversationUseCase.ConversationResponse "Updated conversation"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [delete]
func (h *ConversationHandler) DeleteItem(ctx *gin.Context) {
	// Authenticate user
	user, err := h.useCase.AuthenticateAPIKey(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "r8s9t0u1-v2w3-4567-rstu-890123456789",
			Error: "Invalid or missing API key",
		})
		return
	}

	conversationID := ctx.Param("conversation_id")
	itemID := ctx.Param("item_id")

	// Execute use case
	response, statusCode, err := h.useCase.DeleteItem(ctx, user, conversationID, itemID)
	if err != nil {
		ctx.JSON(statusCode, responses.ErrorResponse{
			Code:  "s9t0u1v2-w3x4-5678-stuv-901234567890",
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(statusCode, response)
}
