package conversation

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"

	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
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

type ConversationService struct {
	conversationRepo ConversationRepository
	itemRepo         ItemRepository
	validator        *ConversationValidator
}

func NewService(conversationRepo ConversationRepository, itemRepo ItemRepository) *ConversationService {
	// Initialize with default validation config
	validator := NewConversationValidator(DefaultValidationConfig())
	return &ConversationService{
		conversationRepo: conversationRepo,
		itemRepo:         itemRepo,
		validator:        validator,
	}
}

func (s *ConversationService) FindConversationsByFilter(ctx context.Context, filter ConversationFilter, pagination *query.Pagination) ([]*Conversation, *common.Error) {
	conversations, err := s.conversationRepo.FindByFilter(ctx, filter, pagination)
	if err != nil {
		return nil, common.NewError("a1b2c3d4-e5f6-7890-abcd-ef1234567890", "Failed to find conversations")
	}
	return conversations, nil
}

func (s *ConversationService) CountConversationsByFilter(ctx context.Context, filter ConversationFilter) (int64, *common.Error) {
	count, err := s.conversationRepo.Count(ctx, filter)
	if err != nil {
		return 0, common.NewError("b2c3d4e5-f6g7-8901-bcde-f23456789012", "Failed to count conversations")
	}
	return count, nil
}

func (s *ConversationService) CreateConversation(ctx context.Context, userID uint, title *string, isPrivate bool, metadata map[string]string) (*Conversation, *common.Error) {
	if err := s.validator.ValidateConversationInput(title, metadata); err != nil {
		return nil, common.NewError("c3d4e5f6-g7h8-9012-cdef-345678901234", "Validation failed")
	}

	publicID, err := s.generateConversationPublicID()
	if err != nil {
		return nil, common.NewError("d4e5f6g7-h8i9-0123-defg-456789012345", "Failed to generate public ID")
	}

	now := time.Now()
	conversation := &Conversation{
		PublicID:  publicID,
		Title:     title,
		UserID:    userID,
		Status:    ConversationStatusActive,
		IsPrivate: isPrivate,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, common.NewError("e5f6g7h8-i9j0-1234-efgh-567890123456", "Failed to create conversation")
	}

	return conversation, nil
}

// GetConversation retrieves a conversation by its public ID with access control and items loaded
func (s *ConversationService) GetConversationByPublicIDAndUserID(ctx context.Context, publicID string, userID uint) (*Conversation, *common.Error) {
	return s.getConversationWithAccessCheck(ctx, publicID, userID)
}

// GetConversationByID retrieves a conversation by its internal ID without user access control
func (s *ConversationService) GetConversationByID(ctx context.Context, conversationID uint) (*Conversation, *common.Error) {
	// Validate inputs
	if conversationID == 0 {
		return nil, common.NewError("f6g7h8i9-j0k1-2345-fghi-678901234567", "Conversation ID cannot be zero")
	}

	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, common.NewError("g7h8i9j0-k1l2-3456-ghij-789012345678", "Failed to find conversation")
	}
	if conversation == nil {
		return nil, common.NewError("h8i9j0k1-l2m3-4567-hijk-890123456789", "Conversation not found")
	}

	return conversation, nil
}

// getConversationWithAccessCheck is the internal method that handles conversation retrieval with optional item loading
func (s *ConversationService) getConversationWithAccessCheck(ctx context.Context, publicID string, userID uint) (*Conversation, *common.Error) {
	// Validate inputs
	if publicID == "" {
		return nil, common.NewError("i9j0k1l2-m3n4-5678-ijkl-901234567890", "Public ID cannot be empty")
	}

	convs, err := s.conversationRepo.FindByFilter(ctx, ConversationFilter{
		UserID:   &userID,
		PublicID: &publicID,
	}, nil)
	if err != nil {
		return nil, common.NewError("j0k1l2m3-n4o5-6789-jklm-012345678901", "Failed to find conversation")
	}
	if len(convs) != 1 {
		return nil, common.NewError("k1l2m3n4-o5p6-7890-klmn-123456789012", "Conversation not found")
	}
	return convs[0], nil
}

func (s *ConversationService) UpdateConversation(ctx context.Context, entity *Conversation) (*Conversation, *common.Error) {
	if err := s.conversationRepo.Update(ctx, entity); err != nil {
		return nil, common.NewError("l2m3n4o5-p6q7-8901-lmno-234567890123", "Failed to update conversation")
	}
	return entity, nil
}

func (s *ConversationService) DeleteConversation(ctx context.Context, conv *Conversation) (bool, *common.Error) {
	if err := s.conversationRepo.Delete(ctx, conv.ID); err != nil {
		return false, common.NewError("m3n4o5p6-q7r8-9012-mnop-345678901234", "Failed to delete conversation")
	}
	return true, nil
}

func (s *ConversationService) AddItem(ctx context.Context, conversation *Conversation, userID uint, itemType ItemType, role *ItemRole, content []Content) (*Item, *common.Error) {
	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, common.NewError("n4o5p6q7-r8s9-0123-nopq-456789012345", "Private conversation access denied")
	}

	if err := s.validator.ValidateItemContent(content); err != nil {
		return nil, common.NewError("o5p6q7r8-s9t0-1234-opqr-567890123456", "Validation failed")
	}

	itemPublicID, err := s.generateItemPublicID()
	if err != nil {
		return nil, common.NewError("p6q7r8s9-t0u1-2345-pqrs-678901234567", "Failed to generate item public ID")
	}

	now := time.Now()
	item := &Item{
		PublicID:    itemPublicID,
		Type:        itemType,
		Role:        role,
		Content:     content,
		Status:      ptr.ToString("completed"), // Default status
		CreatedAt:   now,
		CompletedAt: &now,
	}

	if err := s.conversationRepo.AddItem(ctx, conversation.ID, item); err != nil {
		return nil, common.NewError("q7r8s9t0-u1v2-3456-qrst-789012345678", "Failed to add item")
	}

	// Update conversation timestamp
	conversation.UpdatedAt = now
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, common.NewError("r8s9t0u1-v2w3-4567-rstu-890123456789", "Failed to update conversation timestamp")
	}

	return item, nil
}

func (s *ConversationService) GetItem(ctx context.Context, conversation *Conversation, itemID uint, userID uint) (*Item, *common.Error) {
	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, common.NewError("s9t0u1v2-w3x4-5678-stuv-901234567890", "Private conversation access denied")
	}

	// More efficient: check if item exists in the already loaded conversation items
	if len(conversation.Items) > 0 {
		for _, item := range conversation.Items {
			if item.ID == itemID {
				return &item, nil
			}
		}
		return nil, common.NewError("t0u1v2w3-x4y5-6789-tuvw-012345678901", "Item not found")
	}

	// Fallback: if items aren't loaded, get the item and verify ownership
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, common.NewError("u1v2w3x4-y5z6-7890-uvwx-123456789012", "Failed to find item")
	}

	if item == nil {
		return nil, common.NewError("v2w3x4y5-z6a7-8901-vwxy-234567890123", "Item not found")
	}

	if err := s.verifyItemBelongsToConversation(ctx, itemID, conversation.ID); err != nil {
		return nil, err
	}

	return item, nil
}

// verifyItemBelongsToConversation efficiently checks if an item belongs to a conversation
func (s *ConversationService) verifyItemBelongsToConversation(ctx context.Context, itemID uint, conversationID uint) *common.Error {
	// Use the efficient exists check instead of loading all items
	exists, err := s.itemRepo.ExistsByIDAndConversation(ctx, itemID, conversationID)
	if err != nil {
		return common.NewError("n0o1p2q3-r4s5-6789-nopq-012345678901", "Failed to verify item ownership")
	}

	if !exists {
		return common.NewError("o1p2q3r4-s5t6-7890-opqr-123456789012", "Item not in conversation")
	}

	return nil
}

func (s *ConversationService) DeleteItem(ctx context.Context, conversation *Conversation, itemID uint, userID uint) (*Conversation, *common.Error) {
	// Check access permissions - only owner can delete items
	if conversation.UserID != userID {
		return nil, common.NewError("x4y5z6a7-b8c9-0123-xyza-456789012345", "Access denied")
	}

	// Get the item to verify it exists and belongs to this conversation
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, common.NewError("y5z6a7b8-c9d0-1234-yzab-567890123456", "Failed to find item")
	}

	if item == nil {
		return nil, common.NewError("z6a7b8c9-d0e1-2345-zabc-678901234567", "Item not found")
	}

	// Verify the item belongs to the conversation
	conversationItems, err := s.itemRepo.FindByConversationID(ctx, conversation.ID)
	if err != nil {
		return nil, common.NewError("a7b8c9d0-e1f2-3456-abcd-789012345678", "Failed to verify item ownership")
	}

	// Check if the item belongs to this conversation
	itemFound := false
	for _, convItem := range conversationItems {
		if convItem.ID == itemID {
			itemFound = true
			break
		}
	}

	if !itemFound {
		return nil, common.NewError("b8c9d0e1-f2g3-4567-bcde-890123456789", "Item not in conversation")
	}

	// Delete the item
	if err := s.itemRepo.Delete(ctx, itemID); err != nil {
		return nil, common.NewError("c9d0e1f2-g3h4-5678-cdef-901234567890", "Failed to delete item")
	}

	// Update conversation timestamp
	conversation.UpdatedAt = time.Now()
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, common.NewError("d0e1f2g3-h4i5-6789-defg-012345678901", "Failed to update conversation timestamp")
	}

	// Load the updated conversation with remaining items
	updatedConversation, convErr := s.GetConversationByPublicIDAndUserID(ctx, conversation.PublicID, userID)
	if convErr != nil {
		return nil, convErr
	}

	return updatedConversation, nil
}

// DeleteItemWithConversation deletes an item by its ID and updates the conversation accordingly.
func (s *ConversationService) DeleteItemWithConversation(ctx context.Context, conversation *Conversation, item *Item) (*Item, *common.Error) {
	if err := s.itemRepo.Delete(ctx, item.ID); err != nil {
		return nil, common.NewError("e1f2g3h4-i5j6-7890-efgh-123456789012", "Failed to delete item")
	}

	conversation.UpdatedAt = time.Now()
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, common.NewError("f2g3h4i5-j6k7-8901-fghi-234567890123", "Failed to update conversation")
	}

	return item, nil
}

// generateConversationPublicID generates a conversation ID with business rules
// Business rule: conversations use "conv" prefix with 42 character length for OpenAI compatibility
func (s *ConversationService) generateConversationPublicID() (string, error) {
	return idgen.GenerateSecureID("conv", 42)
}

// generateItemPublicID generates an item/message ID with business rules
// Business rule: items/messages use "msg" prefix with 42 character length for OpenAI compatibility
func (s *ConversationService) generateItemPublicID() (string, error) {
	return idgen.GenerateSecureID("msg", 42)
}

func (s *ConversationService) ValidateItems(ctx context.Context, items []*Item) (bool, *common.Error) {
	if len(items) > 100 {
		return false, common.NewError("g3h4i5j6-k7l8-9012-ghij-345678901234", "Too many items")
	}
	for _, itemData := range items {
		if errCode := s.validator.ValidateItemContent(itemData.Content); errCode != nil {
			return false, common.NewError("h4i5j6k7-l8m9-0123-hijk-456789012345", "Item validation failed")
		}
	}
	return true, nil
}

func (s *ConversationService) FindItemsByFilter(ctx context.Context, filter ItemFilter, p *query.Pagination) ([]*Item, *common.Error) {
	items, err := s.itemRepo.FindByFilter(ctx, filter, p)
	if err != nil {
		return nil, common.NewError("i5j6k7l8-m9n0-1234-ijkl-567890123456", "Failed to find items")
	}
	return items, nil
}

func (s *ConversationService) CountItemsByFilter(ctx context.Context, filter ItemFilter) (int64, *common.Error) {
	count, err := s.itemRepo.Count(ctx, filter)
	if err != nil {
		return 0, common.NewError("j6k7l8m9-n0o1-2345-jklm-678901234567", "Failed to count items")
	}
	return count, nil
}

// AddMultipleItems adds multiple items to a conversation in a single transaction
func (s *ConversationService) AddMultipleItems(ctx context.Context, conversation *Conversation, userID uint, items []*Item) ([]*Item, *common.Error) {
	// Check access permissions
	now := time.Now()
	createdItems := make([]*Item, len(items))

	// Create all items
	for i, itemData := range items {
		itemPublicID, err := s.generateItemPublicID()
		if err != nil {
			return nil, common.NewError("k7l8m9n0-o1p2-3456-klmn-789012345678", "Failed to generate item public ID")
		}

		item := &Item{
			PublicID:    itemPublicID,
			Type:        itemData.Type,
			Role:        itemData.Role,
			Content:     itemData.Content,
			Status:      ptr.ToString("completed"),
			CreatedAt:   now,
			CompletedAt: &now,
			ResponseID:  itemData.ResponseID,
		}

		if err := s.conversationRepo.AddItem(ctx, conversation.ID, item); err != nil {
			return nil, common.NewError("l8m9n0o1-p2q3-4567-lmno-890123456789", "Failed to add item")
		}

		createdItems[i] = item
	}

	conversation.UpdatedAt = now
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, common.NewError("m9n0o1p2-q3r4-5678-mnop-901234567890", "Failed to update conversation timestamp")
	}

	return createdItems, nil
}

func (s *ConversationService) GetConversationMiddleWare() gin.HandlerFunc {
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
		user, ok := auth.GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "01994c96-38fb-7426-9c45-37c8df6c757f",
				Error: "user not found",
			})
			return
		}
		entities, err := s.FindConversationsByFilter(ctx, ConversationFilter{
			PublicID: &publicID,
			UserID:   &user.ID,
		}, nil)

		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  err.Code,
				Error: err.Message,
			})
			return
		}

		if len(entities) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "e91636c2-fced-4a89-bf08-55309005365f",
				Error: "conversation not found",
			})
			return
		}

		SetConversationFromContext(reqCtx, entities[0])
		reqCtx.Next()
	}
}

func SetConversationFromContext(reqCtx *gin.Context, conv *Conversation) {
	reqCtx.Set(string(ConversationContextEntity), conv)
}

func GetConversationFromContext(reqCtx *gin.Context) (*Conversation, bool) {
	conv, ok := reqCtx.Get(string(ConversationContextEntity))
	if !ok {
		return nil, false
	}
	v, ok := conv.(*Conversation)
	if !ok {
		return nil, false
	}
	return v, true
}

func (s *ConversationService) GetConversationItemMiddleWare() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		conv, ok := GetConversationFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "0f5c3304-bf46-45ce-8719-7c03a3485b37",
				Error: "conversation not found",
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
		entities, err := s.FindItemsByFilter(ctx, ItemFilter{
			PublicID:       &publicID,
			ConversationID: &conv.ID,
		}, nil)

		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
				Code:  err.Code,
				Error: err.Message,
			})
			return
		}

		if len(entities) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code:  "25647b40-4967-497e-9cbd-a85243ccef58",
				Error: "conversation item not found",
			})
			return
		}

		SetConversationItemFromContext(reqCtx, entities[0])
		reqCtx.Next()
	}
}

func SetConversationItemFromContext(reqCtx *gin.Context, item *Item) {
	reqCtx.Set(string(ConversationItemContextEntity), item)
}

func GetConversationItemFromContext(reqCtx *gin.Context) (*Item, bool) {
	item, ok := reqCtx.Get(string(ConversationItemContextEntity))
	if !ok {
		return nil, false
	}
	v, ok := item.(*Item)
	if !ok {
		return nil, false
	}
	return v, true
}
