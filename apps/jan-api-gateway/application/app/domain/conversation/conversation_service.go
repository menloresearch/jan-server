package conversation

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// Custom errors
var (
	ErrConversationNotFound  = errors.New("conversation not found")
	ErrAccessDenied          = errors.New("access denied: not the owner of this conversation")
	ErrPrivateConversation   = errors.New("access denied: conversation is private")
	ErrItemNotFound          = errors.New("item not found")
	ErrItemNotInConversation = errors.New("item not found in conversation")
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

func NewServiceWithValidator(conversationRepo ConversationRepository, itemRepo ItemRepository, validator *ConversationValidator) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		itemRepo:         itemRepo,
		validator:        validator,
	}
}

func (s *ConversationService) FindConversationsByFilter(ctx context.Context, filter ConversationFilter, pagination *query.Pagination) ([]*Conversation, error) {
	return s.conversationRepo.FindByFilter(ctx, filter, pagination)
}

func (s *ConversationService) CountConversationsByFilter(ctx context.Context, filter ConversationFilter) (int64, error) {
	return s.conversationRepo.Count(ctx, filter)
}

func (s *ConversationService) CreateConversation(ctx context.Context, userID uint, title *string, isPrivate bool, metadata map[string]string) (*Conversation, error) {
	if err := s.validator.ValidateConversationInput(title, metadata); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	publicID, err := s.generateConversationPublicID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate public ID: %w", err)
	}

	now := time.Now().Unix()
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
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conversation, nil
}

// GetConversation retrieves a conversation by its public ID with access control and items loaded
func (s *ConversationService) GetConversationByPublicIDAndUserID(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
	return s.getConversationWithAccessCheck(ctx, publicID, userID, true)
}

// GetConversationWithAccessAndItems is an alias for backward compatibility
func (s *ConversationService) GetConversationWithAccessAndItems(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
	return s.GetConversationByPublicIDAndUserID(ctx, publicID, userID)
}

// GetConversationWithoutItems retrieves a conversation without loading items for performance
func (s *ConversationService) GetConversationWithoutItems(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
	return s.getConversationWithAccessCheck(ctx, publicID, userID, false)
}

// getConversationWithAccessCheck is the internal method that handles conversation retrieval with optional item loading
func (s *ConversationService) getConversationWithAccessCheck(ctx context.Context, publicID string, userID uint, loadItems bool) (*Conversation, error) {
	// Validate inputs
	if publicID == "" {
		return nil, fmt.Errorf("public ID cannot be empty")
	}

	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	// Load items if requested
	if loadItems {
		items, err := s.itemRepo.FindByConversationID(ctx, conversation.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load items: %w", err)
		}

		// Convert []*Item to []Item
		itemSlice := make([]Item, len(items))
		for i, item := range items {
			itemSlice[i] = *item
		}
		conversation.Items = itemSlice
	}

	return conversation, nil
}

func (s *ConversationService) UpdateConversation(ctx context.Context, entity *Conversation) (*Conversation, error) {
	if err := s.conversationRepo.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}
	return entity, nil
}

func (s *ConversationService) DeleteConversation(ctx context.Context, conv *Conversation) error {
	if err := s.conversationRepo.Delete(ctx, conv.ID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	return nil
}

func (s *ConversationService) AddItem(ctx context.Context, conversation *Conversation, userID uint, itemType ItemType, role *ItemRole, content []Content) (*Item, error) {
	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	if errodCode := s.validator.ValidateItemContent(content); errodCode != nil {
		return nil, fmt.Errorf("validation failed: %s", *errodCode)
	}

	itemPublicID, err := s.generateItemPublicID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate item public ID: %w", err)
	}

	now := time.Now().Unix()
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
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	// Update conversation timestamp
	conversation.UpdatedAt = now
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	return item, nil
}

func (s *ConversationService) GetItem(ctx context.Context, conversation *Conversation, itemID uint, userID uint) (*Item, error) {
	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	// More efficient: check if item exists in the already loaded conversation items
	if len(conversation.Items) > 0 {
		for _, item := range conversation.Items {
			if item.ID == itemID {
				return &item, nil
			}
		}
		return nil, ErrItemNotFound
	}

	// Fallback: if items aren't loaded, get the item and verify ownership
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to find item: %w", err)
	}

	if item == nil {
		return nil, ErrItemNotFound
	}

	if err := s.verifyItemBelongsToConversation(ctx, itemID, conversation.ID); err != nil {
		return nil, err
	}

	return item, nil
}

// verifyItemBelongsToConversation efficiently checks if an item belongs to a conversation
func (s *ConversationService) verifyItemBelongsToConversation(ctx context.Context, itemID uint, conversationID uint) error {
	// Use the efficient exists check instead of loading all items
	exists, err := s.itemRepo.ExistsByIDAndConversation(ctx, itemID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to verify item ownership: %w", err)
	}

	if !exists {
		return ErrItemNotInConversation
	}

	return nil
}

// DeleteItemWithConversation deletes an item by its ID and updates the conversation accordingly.
func (s *ConversationService) DeleteItemWithConversation(ctx context.Context, conversation *Conversation, item *Item) (*Item, error) {
	if err := s.itemRepo.Delete(ctx, item.ID); err != nil {
		return nil, err
	}

	conversation.UpdatedAt = time.Now().Unix()
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	return item, nil
}

func (s *ConversationService) SearchItems(ctx context.Context, publicID string, userID uint, query string) ([]*Item, error) {
	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	items, err := s.itemRepo.Search(ctx, conversation.ID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search items: %w", err)
	}

	return items, nil
}

// generateConversationPublicID generates a conversation ID with business rules
// Business rule: conversations use "conv" prefix with 16 character length for OpenAI compatibility
func (s *ConversationService) generateConversationPublicID() (string, error) {
	return idgen.GenerateSecureID("conv", 16)
}

// generateItemPublicID generates an item/message ID with business rules
// Business rule: items/messages use "msg" prefix with 16 character length for OpenAI compatibility
func (s *ConversationService) generateItemPublicID() (string, error) {
	return idgen.GenerateSecureID("msg", 16)
}

func (s *ConversationService) ValidateItems(ctx context.Context, items []*Item) (bool, *string) {
	if len(items) > 100 {
		return false, ptr.ToString("0502c02c-ea2d-429e-933c-1243d4e2bcb2")
	}
	for _, itemData := range items {
		if errCode := s.validator.ValidateItemContent(itemData.Content); errCode != nil {
			return false, errCode
		}
	}
	return true, nil
}

func (s *ConversationService) FindItemsByFilter(ctx context.Context, filter ItemFilter, p *query.Pagination) ([]*Item, error) {
	return s.itemRepo.FindByFilter(ctx, filter, p)
}

func (s *ConversationService) CountItemsByFilter(ctx context.Context, filter ItemFilter) (int64, error) {
	return s.itemRepo.Count(ctx, filter)
}

// AddMultipleItems adds multiple items to a conversation in a single transaction
func (s *ConversationService) AddMultipleItems(ctx context.Context, conversation *Conversation, userID uint, items []*Item) ([]*Item, error) {
	// Check access permissions
	// TODO: Validate before persisting
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	now := time.Now().Unix()
	createdItems := make([]*Item, len(items))

	// Create all items
	for i, itemData := range items {
		itemPublicID, err := s.generateItemPublicID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate item public ID for item %d: %w", i, err)
		}

		item := &Item{
			PublicID:    itemPublicID,
			Type:        itemData.Type,
			Role:        itemData.Role,
			Content:     itemData.Content,
			Status:      ptr.ToString("completed"),
			CreatedAt:   now,
			CompletedAt: &now,
		}

		if err := s.conversationRepo.AddItem(ctx, conversation.ID, item); err != nil {
			return nil, fmt.Errorf("failed to add item %d: %w", i, err)
		}

		createdItems[i] = item
	}

	conversation.UpdatedAt = now
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	return createdItems, nil
}
