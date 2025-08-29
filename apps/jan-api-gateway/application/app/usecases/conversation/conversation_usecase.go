package conversation

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/interfaces/http/requests"
)

// ConversationUseCase encapsulates all conversation-related business logic
type ConversationUseCase struct {
	conversationService *conversation.ConversationService
	userService         *user.UserService
	apiKeyService       *apikey.ApiKeyService
}

// NewConversationUseCase creates a new conversation use case
func NewConversationUseCase(
	conversationService *conversation.ConversationService,
	userService *user.UserService,
	apiKeyService *apikey.ApiKeyService,
) *ConversationUseCase {
	return &ConversationUseCase{
		conversationService: conversationService,
		userService:         userService,
		apiKeyService:       apiKeyService,
	}
}

// AuthenticatedUser represents an authenticated user context
type AuthenticatedUser struct {
	ID uint
}

// AuthenticateAPIKey validates API key and returns authenticated user
func (uc *ConversationUseCase) AuthenticateAPIKey(ctx *gin.Context) (*AuthenticatedUser, error) {
	apiKey, ok := requests.GetTokenFromBearer(ctx)
	if !ok {
		return nil, fmt.Errorf("invalid or missing API key")
	}

	apiKeyEntity, err := uc.apiKeyService.FindByKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	if apiKeyEntity == nil {
		return nil, fmt.Errorf("API key not found")
	}

	user, err := uc.userService.FindByID(ctx, *apiKeyEntity.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &AuthenticatedUser{ID: user.ID}, nil
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

// CreateConversation creates a new conversation

func (uc *ConversationUseCase) CreateConversation(ctx context.Context, user *AuthenticatedUser, req CreateConversationRequest) (*ConversationResponse, int, error) {
	// Create conversation
	conv, err := uc.conversationService.CreateConversation(ctx, user.ID, nil, true, req.Metadata)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create conversation: %w", err)
	}

	// Add items if provided using batch operation
	if len(req.Items) > 0 {
		// Convert all items at once for batch processing
		itemsToCreate := make([]struct {
			Type    conversation.ItemType
			Role    *conversation.ItemRole
			Content []conversation.Content
		}, len(req.Items))

		for i, itemReq := range req.Items {
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

			itemsToCreate[i] = struct {
				Type    conversation.ItemType
				Role    *conversation.ItemRole
				Content []conversation.Content
			}{
				Type:    itemType,
				Role:    role,
				Content: content,
			}
		}

		// ✅ Single batch operation instead of N individual operations
		_, err := uc.conversationService.AddMultipleItems(ctx, conv, user.ID, itemsToCreate)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to add items: %w", err)
		}

		// Reload conversation with items
		conv, err = uc.conversationService.GetConversation(ctx, conv.PublicID, user.ID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to reload conversation: %w", err)
		}
	}

	return uc.domainToConversationResponse(conv), http.StatusOK, nil
}

// GetConversation retrieves a conversation
func (uc *ConversationUseCase) GetConversation(ctx context.Context, user *AuthenticatedUser, conversationID string) (*ConversationResponse, int, error) {
	conv, err := uc.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return uc.domainToConversationResponse(conv), http.StatusOK, nil
}

// UpdateConversation updates a conversation
func (uc *ConversationUseCase) UpdateConversation(ctx context.Context, user *AuthenticatedUser, conversationID string, req UpdateConversationRequest) (*ConversationResponse, int, error) {
	conv, err := uc.conversationService.UpdateAndAuthorizeConversation(ctx, conversationID, user.ID, nil, req.Metadata)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return uc.domainToConversationResponse(conv), http.StatusOK, nil
}

// DeleteConversation deletes a conversation
func (uc *ConversationUseCase) DeleteConversation(ctx context.Context, user *AuthenticatedUser, conversationID string) (*DeletedConversationResponse, int, error) {
	// Get conversation first to get the public ID for response
	conv, err := uc.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}

	err = uc.conversationService.DeleteConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrAccessDenied) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return uc.domainToDeletedConversationResponse(conv), http.StatusOK, nil
}

// CreateItems creates multiple items in a conversation using batch operations
func (uc *ConversationUseCase) CreateItems(ctx context.Context, user *AuthenticatedUser, conversationID string, req CreateItemsRequest) (*ConversationItemListResponse, int, error) {
	// Get conversation first to avoid N+1 queries
	conv, err := uc.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}

	// Convert all items at once for batch processing
	itemsToCreate := make([]struct {
		Type    conversation.ItemType
		Role    *conversation.ItemRole
		Content []conversation.Content
	}, len(req.Items))

	for i, itemReq := range req.Items {
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

		itemsToCreate[i] = struct {
			Type    conversation.ItemType
			Role    *conversation.ItemRole
			Content []conversation.Content
		}{
			Type:    itemType,
			Role:    role,
			Content: content,
		}
	}

	// ✅ Single batch operation instead of N individual operations
	createdItems, err := uc.conversationService.AddMultipleItems(ctx, conv, user.ID, itemsToCreate)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return uc.domainToConversationItemListResponse(createdItems), http.StatusOK, nil
}

// ListItems lists all items in a conversation
func (uc *ConversationUseCase) ListItems(ctx context.Context, user *AuthenticatedUser, conversationID string) (*ConversationItemListResponse, int, error) {
	// Get conversation first to check access
	conv, err := uc.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}

	// Convert items to pointers for consistency
	items := make([]*conversation.Item, len(conv.Items))
	for i := range conv.Items {
		items[i] = &conv.Items[i]
	}

	return uc.domainToConversationItemListResponse(items), http.StatusOK, nil
}

// GetItem retrieves a specific item from a conversation
func (uc *ConversationUseCase) GetItem(ctx context.Context, user *AuthenticatedUser, conversationID, itemID string) (*ConversationItemResponse, int, error) {
	// Get conversation first to avoid N+1 queries
	conv, err := uc.conversationService.GetConversation(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
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
		return nil, http.StatusNotFound, conversation.ErrItemNotFound
	}

	return uc.domainToConversationItemResponse(*foundItem), http.StatusOK, nil
}

// DeleteItem deletes a specific item from a conversation using efficient verification
func (uc *ConversationUseCase) DeleteItem(ctx context.Context, user *AuthenticatedUser, conversationID, itemPublicID string) (*ConversationResponse, int, error) {
	// Get conversation first to avoid N+1 queries (without loading all items)
	conv, err := uc.conversationService.GetConversationWithoutItems(ctx, conversationID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrConversationNotFound) {
			return nil, http.StatusNotFound, err
		}
		if errors.Is(err, conversation.ErrPrivateConversation) {
			return nil, http.StatusForbidden, err
		}
		return nil, http.StatusInternalServerError, err
	}

	// ✅ Use efficient deletion with item public ID instead of loading all items
	updatedConversation, err := uc.conversationService.DeleteItemByPublicID(ctx, conv, itemPublicID, user.ID)
	if err != nil {
		if errors.Is(err, conversation.ErrAccessDenied) {
			return nil, http.StatusForbidden, err
		}
		if errors.Is(err, conversation.ErrItemNotFound) || errors.Is(err, conversation.ErrItemNotInConversation) {
			return nil, http.StatusNotFound, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return uc.domainToConversationResponse(updatedConversation), http.StatusOK, nil
}

// Helper methods for domain to response conversion
func (uc *ConversationUseCase) domainToConversationResponse(entity *conversation.Conversation) *ConversationResponse {
	// Initialize metadata as empty map if nil (required by OpenAI spec)
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

func (uc *ConversationUseCase) domainToDeletedConversationResponse(entity *conversation.Conversation) *DeletedConversationResponse {
	return &DeletedConversationResponse{
		ID:      entity.PublicID,
		Object:  "conversation.deleted",
		Deleted: true,
	}
}

func (uc *ConversationUseCase) domainToConversationItemListResponse(items []*conversation.Item) *ConversationItemListResponse {
	response := make([]ConversationItemResponse, len(items))
	for i, item := range items {
		response[i] = *uc.domainToConversationItemResponse(*item)
	}

	result := &ConversationItemListResponse{
		Object:  "list",
		Data:    response,
		HasMore: false, // For now, we don't implement pagination here
	}

	if len(response) > 0 {
		result.FirstID = response[0].ID
		result.LastID = response[len(response)-1].ID
	}

	return result
}

func (uc *ConversationUseCase) domainToConversationItemResponse(entity conversation.Item) *ConversationItemResponse {
	response := &ConversationItemResponse{
		ID:        entity.PublicID,
		Object:    "conversation.item",
		Type:      string(entity.Type),
		Status:    entity.Status,
		CreatedAt: entity.CreatedAt,
		Content:   uc.domainToContentResponse(entity.Content),
	}

	if entity.Role != nil {
		role := string(*entity.Role)
		response.Role = &role
	}

	return response
}

func (uc *ConversationUseCase) domainToContentResponse(content []conversation.Content) []ContentResponse {
	if len(content) == 0 {
		return nil
	}

	response := make([]ContentResponse, len(content))
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
					Annotations: uc.domainToAnnotationResponse(c.OutputText.Annotations),
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

		response[i] = contentResp
	}
	return response
}

func (uc *ConversationUseCase) domainToAnnotationResponse(annotations []conversation.Annotation) []AnnotationResponse {
	if len(annotations) == 0 {
		return nil
	}

	response := make([]AnnotationResponse, len(annotations))
	for i, a := range annotations {
		response[i] = AnnotationResponse{
			Type:       a.Type,
			Text:       a.Text,
			FileID:     a.FileID,
			URL:        a.URL,
			StartIndex: a.StartIndex,
			EndIndex:   a.EndIndex,
			Index:      a.Index,
		}
	}
	return response
}
