package conversations

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	conversationHandler "menlo.ai/jan-api-gateway/app/interfaces/http/handlers/conversation"
	"menlo.ai/jan-api-gateway/app/interfaces/http/handlers/userhandler"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

// ConversationAPI handles route registration for V1 conversations
type ConversationAPI struct {
	handler     *conversationHandler.ConversationHandler
	userHandler *userhandler.UserHandler
}

// NewConversationAPI creates a new conversation API instance
func NewConversationAPI(handler *conversationHandler.ConversationHandler, userHandler *userhandler.UserHandler) *ConversationAPI {
	return &ConversationAPI{
		handler:     handler,
		userHandler: userHandler,
	}
}

// RegisterRouter registers OpenAI-compatible conversation routes
func (api *ConversationAPI) RegisterRouter(router *gin.RouterGroup) {
	conversationsRouter := router.Group("/conversations", api.userHandler.RegisteredApiKeyUserMiddleware())

	// OpenAI-compatible endpoints with Swagger documentation
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
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.CreateConversationRequest true "Create conversation request"
// @Success 200 {object} ConversationResponse "Created conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations [post]
func (api *ConversationAPI) createConversation(reqCtx *gin.Context) {
	user, ok := userhandler.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code: "13bd47d2-2674-4a44-9df6-b88f5f0ef590",
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

	reqCtx.JSON(httpCode, domainToConversationResponse(response.Result))
}

// getConversation handles conversation retrieval
// @Summary Get a conversation
// @Description Retrieves a conversation by its ID
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationResponse "Conversation details"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [get]
func (api *ConversationAPI) getConversation(reqCtx *gin.Context) {
	conv, ok := conversationHandler.GetConversationFromContext(reqCtx)
	if !ok {
		return
	}
	reqCtx.JSON(http.StatusOK, domainToConversationResponse(conv))
}

// updateConversation handles conversation updates
// @Summary Update a conversation
// @Description Updates conversation metadata
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.UpdateConversationRequest true "Update conversation request"
// @Success 200 {object} ConversationResponse "Updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id} [patch]
func (api *ConversationAPI) updateConversation(reqCtx *gin.Context) {
	httpStatus, result, err := api.handler.UpdateConversation(reqCtx)
	if httpStatus != http.StatusOK {
		reqCtx.AbortWithStatusJSON(httpStatus, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
	}
	reqCtx.JSON(httpStatus, domainToConversationResponse(&result.Result))
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
// @Tags Platform, Platform-Conversations
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
	status, result, err := api.handler.DeleteConversation(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(status, domainToDeletedConversationResponse(&result.Result))
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
	Object  string                      `json:"object"`
	Data    []*ConversationItemResponse `json:"data"`
	HasMore bool                        `json:"has_more"`
	FirstID *string                     `json:"first_id,omitempty"`
	LastID  *string                     `json:"last_id,omitempty"`
}

// createItems handles item creation
// @Summary Create items in a conversation
// @Description Adds multiple items to a conversation
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param request body menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.CreateItemsRequest true "Create items request"
// @Success 200 {object} menlo_ai_jan-api-gateway_app_interfaces_http_handlers_conversation.ConversationItemListResponse "Created items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [post]
func (api *ConversationAPI) createItems(reqCtx *gin.Context) {
	status, result, err := api.handler.CreateItems(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(status, ConversationItemListResponse{
		Object:  "list",
		Data:    functional.Map(result.Results, domainToConversationItemResponse),
		FirstID: result.FirstID,
		LastID:  result.LastID,
		HasMore: result.HasMore,
	})
}

// listItems handles item listing with optional pagination
// @Summary List items in a conversation
// @Description Lists all items in a conversation
// @Tags Platform, Platform-Conversations
// @Security BearerAuth
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Param limit query int false "Number of items to return (1-100)"
// @Param cursor query string false "Cursor for pagination"
// @Param order query string false "Order of items (asc/desc)"
// @Success 200 {object} ConversationItemListResponse "List of items"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 403 {object} responses.ErrorResponse "Access denied"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conversation_id}/items [get]
func (api *ConversationAPI) listItems(reqCtx *gin.Context) {
	status, result, err := api.handler.ListItems(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(status, ConversationItemListResponse{
		Object:  "list",
		Data:    functional.Map(result.Results, domainToConversationItemResponse),
		FirstID: result.FirstID,
		LastID:  result.LastID,
		HasMore: result.HasMore,
	})
}

// getItem handles single item retrieval
// @Summary Get an item from a conversation
// @Description Retrieves a specific item from a conversation
// @Tags Platform, Platform-Conversations
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
	item, ok := conversationHandler.GetConversationItemFromContext(reqCtx)
	if !ok {
		return
	}
	reqCtx.JSON(http.StatusOK, domainToConversationItemResponse(item))
}

// deleteItem handles item deletion
// @Summary Delete an item from a conversation
// @Description Deletes a specific item from a conversation
// @Tags Platform, Platform-Conversations
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
	status, result, err := api.handler.DeleteItem(reqCtx)
	if status != http.StatusOK {
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:          result.Status,
			ErrorInstance: err,
		})
		return
	}
	reqCtx.JSON(http.StatusOK, domainToConversationItemResponse(result.Result))
}

func domainToConversationResponse(entity *conversation.Conversation) *ConversationResponse {
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
		CreatedAt: entity.CreatedAt,
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
