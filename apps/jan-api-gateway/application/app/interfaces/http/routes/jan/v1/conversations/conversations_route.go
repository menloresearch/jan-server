package conversations

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
	"menlo.ai/jan-api-gateway/app/interfaces/http/handlers/userhandler"
	"menlo.ai/jan-api-gateway/app/interfaces/http/middleware"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses/jan"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

// ConversationAPI handles route registration for Jan V1 conversations
type ConversationAPI struct {
	handler     *conversationHandler.ConversationHandler
	userHandler *userhandler.UserHandler
}

// NewConversationAPI creates a new conversation API instance
func NewConversationAPI(handler *conversationHandler.ConversationHandler, userHandler *userhandler.UserHandler) *ConversationAPI {
	return &ConversationAPI{
		handler,
		userHandler,
	}
}

// RegisterRouter registers Jan-specific conversation routes
func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations", middleware.AuthMiddleware(), api.userHandler.RegisteredUserMiddleware())

	conversationsRouter.GET("", api.listConversations)
	conversationsRouter.POST("", api.createConversation)

	conversationMiddleWare := api.handler.GetConversationMiddleWare()
	conversationsRouter.GET(fmt.Sprintf("/:%s", conversationHandler.ConversationContextKeyPublicID), conversationMiddleWare, api.getConversation)
	conversationsRouter.PATCH(fmt.Sprintf("/:%s", conversationHandler.ConversationContextKeyPublicID), conversationMiddleWare, api.updateConversation)
	conversationsRouter.DELETE(fmt.Sprintf("/:%s", conversationHandler.ConversationContextKeyPublicID), conversationMiddleWare, api.deleteConversation)
	conversationsRouter.POST(fmt.Sprintf("/:%s/items", conversationHandler.ConversationContextKeyPublicID), conversationMiddleWare, api.createItems)
	conversationsRouter.GET(fmt.Sprintf("/:%s/items", conversationHandler.ConversationContextKeyPublicID), conversationMiddleWare, api.listItems)

	conversationItemMiddleWare := api.handler.GetConversationItemMiddleWare()
	conversationsRouter.GET(fmt.Sprintf("/:%s/items/:%s", conversationHandler.ConversationContextKeyPublicID, conversationHandler.ConversationItemContextKeyPublicID), conversationMiddleWare, conversationItemMiddleWare, api.getItem)
	conversationsRouter.DELETE(fmt.Sprintf("/:%s/items/:%s", conversationHandler.ConversationContextKeyPublicID, conversationHandler.ConversationItemContextKeyPublicID), conversationMiddleWare, conversationItemMiddleWare, api.deleteItem)
	// conversationsRouter.GET("/:conversation_id/items/search", api.handler.SearchItems)  // Jan-specific: search items
}

type ConversationResponse struct {
	ID        string            `json:"id"`
	Title     *string           `json:"title"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}

func convertConversationEntityToResponse(entity *conversation.Conversation) ConversationResponse {
	return ConversationResponse{
		ID:        entity.PublicID,
		Title:     entity.Title,
		Status:    string(entity.Status),
		Metadata:  entity.Metadata,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

type ItemResponse struct {
	ID                string                          `json:"id"`
	Type              conversation.ItemType           `json:"type"`
	Role              *conversation.ItemRole          `json:"role,omitempty"`
	Content           []conversation.Content          `json:"content,omitempty"`
	Status            *string                         `json:"status,omitempty"`
	IncompleteAt      *int64                          `json:"incomplete_at,omitempty"`
	IncompleteDetails *conversation.IncompleteDetails `json:"incomplete_details,omitempty"`
	CompletedAt       *int64                          `json:"completed_at,omitempty"`
	CreatedAt         int64                           `json:"created_at"`
}

func convertItemEntityToResponse(entity *conversation.Item) ItemResponse {
	return ItemResponse{
		ID:                entity.PublicID,
		Type:              entity.Type,
		Role:              entity.Role,
		Content:           entity.Content,
		Status:            entity.Status,
		IncompleteAt:      entity.IncompleteAt,
		IncompleteDetails: entity.IncompleteDetails,
		CompletedAt:       entity.CompletedAt,
		CreatedAt:         entity.CreatedAt,
	}
}

// ListConversations
// @Summary List Conversations
// @Description Retrieves a paginated list of conversations for the authenticated user.
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Param limit query int false "The maximum number of items to return" default(20)
// @Param after query string false "A cursor for use in pagination. The ID of the last object from the previous page"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} jan.JanListResponse[ConversationResponse] "Successfully retrieved the list of conversations"
// @Failure 400 {object} responses.ErrorResponse "Bad Request - Invalid pagination parameters"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or missing API key"
// @Failure 500 {object} responses.ErrorResponse "Internal Server Error"
// @Router /jan/v1/conversations [get]
func (api *ConversationAPI) listConversations(reqCtx *gin.Context) {
	httpCode, response, err := api.handler.ListConversations(reqCtx)
	if httpCode != http.StatusOK {
		reqCtx.AbortWithStatusJSON(httpCode, responses.ErrorResponse{
			Code:          response.Status,
			ErrorInstance: err,
		})
		return
	}

	reqCtx.JSON(httpCode, jan.JanListResponse[ConversationResponse]{
		Status:  responses.ResponseCodeOk,
		FirstID: response.FirstID,
		LastID:  response.LastID,
		Total:   response.Total,
		HasMore: response.HasMore,
		Results: functional.Map(response.Results, convertConversationEntityToResponse),
	})
}

// createConversation handles conversation creation
// @Summary Create a conversation
// @Description Creates a new conversation for the authenticated user
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.CreateConversationRequest true "Create conversation request"
// @Success 200 {object} jan.JanGeneralResponse[ConversationResponse] "Created conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations [post]
func (api *ConversationAPI) createConversation(reqCtx *gin.Context) {
	user, ok := userhandler.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "4a0aeda0-417c-43ac-9b14-9a2f0a98acb1",
		})
		return
	}

	httpCode, response, err := api.handler.CreateConversationByUserID(reqCtx, user.ID)
	if httpCode != http.StatusOK {
		reqCtx.AbortWithStatusJSON(httpCode, responses.ErrorResponse{
			Code:          response.Status,
			ErrorInstance: err,
		})
		return
	}

	reqCtx.JSON(httpCode, jan.JanGeneralResponse[ConversationResponse]{
		Status: responses.ResponseCodeOk,
		Result: convertConversationEntityToResponse(response.Result),
	})
}

// getConversation handles conversation retrieval
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} jan.JanGeneralResponse[ConversationResponse] "Conversation details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [get]
func (api *ConversationAPI) getConversation(reqCtx *gin.Context) {
	conv, ok := conversationHandler.GetConversationFromContext(reqCtx)
	if !ok {
		return
	}
	reqCtx.JSON(http.StatusOK, convertConversationEntityToResponse(conv))
}

// updateConversation handles conversation updates
// @Summary Update a conversation
// @Description Updates conversation metadata
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.UpdateConversationRequest true "Update conversation request"
// @Success 200 {object} jan.JanGeneralResponse[ConversationResponse] "Updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [patch]
func (api *ConversationAPI) updateConversation(reqCtx *gin.Context) {
	httpStatus, result, err := api.handler.UpdateConversation(reqCtx)
	if httpStatus != http.StatusOK {
		reqCtx.AbortWithStatusJSON(httpStatus, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
	}
	reqCtx.JSON(httpStatus, jan.JanGeneralResponse[ConversationResponse]{
		Result: convertConversationEntityToResponse(&result.Result),
		Status: result.Status,
	})
}

// deleteConversation handles conversation deletion
// @Summary Delete a conversation
// @Description Deletes a conversation and all its items
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} jan.JanGeneralResponse[ConversationResponse] "Deleted conversation"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id} [delete]
func (api *ConversationAPI) deleteConversation(reqCtx *gin.Context) {
	status, result, err := api.handler.DeleteConversation(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(status, jan.JanGeneralResponse[ConversationResponse]{
		Status: result.Status,
		Result: convertConversationEntityToResponse(&result.Result),
	})
}

// createItems handles item creation
// @Summary Create items in a conversation
// @Description Adds multiple items to a conversation
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.CreateItemsRequest true "Create items request"
// @Success 200 {object} jan.JanListResponse[ItemResponse] "Created items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) createItems(reqCtx *gin.Context) {
	status, result, err := api.handler.CreateItems(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(status, jan.JanListResponse[ItemResponse]{
		Status:  result.Status,
		Results: functional.Map(result.Results, convertItemEntityToResponse),
		FirstID: result.FirstID,
		LastID:  result.LastID,
		HasMore: result.HasMore,
		Total:   result.Total,
	})
}

// listItems handles item listing with optional pagination
// @Summary List items in a conversation
// @Description Lists all items in a conversation
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "The maximum number of items to return" default(20)
// @Param after query string false "A cursor for use in pagination. The ID of the last object from the previous page"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} jan.JanListResponse[ItemResponse] "List of items"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) listItems(reqCtx *gin.Context) {
	status, result, err := api.handler.ListItems(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(status, jan.JanListResponse[ItemResponse]{
		Status:  result.Status,
		Results: functional.Map(result.Results, convertItemEntityToResponse),
		FirstID: result.FirstID,
		LastID:  result.LastID,
		HasMore: result.HasMore,
		Total:   result.Total,
	})
}

// getItem handles single item retrieval
// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} jan.JanGeneralResponse[ItemResponse] "Item details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [get]
func (api *ConversationAPI) getItem(reqCtx *gin.Context) {
	item, ok := conversationHandler.GetConversationItemFromContext(reqCtx)
	if !ok {
		return
	}
	reqCtx.JSON(http.StatusOK, jan.JanGeneralResponse[ItemResponse]{
		Status: responses.ResponseCodeOk,
		Result: convertItemEntityToResponse(item),
	})
}

// deleteItem handles item deletion
// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation
// @Tags Jan, Jan-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} jan.JanGeneralResponse[ItemResponse] "Updated conversation"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /jan/v1/conversations/{conversation_id}/items/{item_id} [delete]
func (api *ConversationAPI) deleteItem(reqCtx *gin.Context) {
	status, result, err := api.handler.DeleteItem(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(http.StatusOK, jan.JanGeneralResponse[ItemResponse]{
		Status: result.Status,
		Result: convertItemEntityToResponse(result.Result),
	})
}
