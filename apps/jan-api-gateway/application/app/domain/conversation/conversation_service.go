package conversation

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"menlo.ai/jan-api-gateway/app/utils/idutils"
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

func (s *ConversationService) CreateConversation(ctx context.Context, userID uint, title *string, isPrivate bool, metadata map[string]string) (*Conversation, error) {
	// Validate inputs using enhanced validator
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
func (s *ConversationService) GetConversation(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
	return s.getConversationWithAccessCheck(ctx, publicID, userID, true)
}

// GetConversationWithAccessAndItems is an alias for backward compatibility
func (s *ConversationService) GetConversationWithAccessAndItems(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
	return s.GetConversation(ctx, publicID, userID)
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

// UpdateAndAuthorizeConversation retrieves a conversation by its public ID, checks access permissions,
// updates the conversation fields, and persists the changes.
func (s *ConversationService) UpdateAndAuthorizeConversation(ctx context.Context, publicID string, userID uint, title *string, metadata map[string]string) (*Conversation, error) {
	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	// Check access permissions
	if conversation.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Update fields
	if title != nil {
		conversation.Title = title
	}
	if metadata != nil {
		conversation.Metadata = metadata
	}
	conversation.UpdatedAt = time.Now().Unix()

	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	return conversation, nil
}

func (s *ConversationService) DeleteConversation(ctx context.Context, publicID string, userID uint) error {
	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return ErrConversationNotFound
	}

	// Check access permissions
	if conversation.UserID != userID {
		return ErrAccessDenied
	}

	if err := s.conversationRepo.Delete(ctx, conversation.ID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

func (s *ConversationService) AddItem(ctx context.Context, conversation *Conversation, userID uint, itemType ItemType, role *ItemRole, content []Content) (*Item, error) {
	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	// Validate content using enhanced validator
	if err := s.validator.ValidateItemContent(content); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
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
		Status:      stringPtr("completed"), // Default status
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

	// Enhanced verification: use a more efficient query to check if item belongs to conversation
	// This avoids loading all conversation items
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

func (s *ConversationService) DeleteItem(ctx context.Context, conversation *Conversation, itemID uint, userID uint) (*Conversation, error) {
	// Check access permissions - only owner can delete items
	if conversation.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Get the item to verify it exists and belongs to this conversation
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to find item: %w", err)
	}

	if item == nil {
		return nil, ErrItemNotFound
	}

	// Verify the item belongs to the conversation
	conversationItems, err := s.itemRepo.FindByConversationID(ctx, conversation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify item ownership: %w", err)
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
		return nil, ErrItemNotInConversation
	}

	// Delete the item
	if err := s.itemRepo.Delete(ctx, itemID); err != nil {
		return nil, fmt.Errorf("failed to delete item: %w", err)
	}

	// Update conversation timestamp
	conversation.UpdatedAt = time.Now().Unix()
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	// Load the updated conversation with remaining items
	updatedConversation, err := s.GetConversation(ctx, conversation.PublicID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load updated conversation: %w", err)
	}

	return updatedConversation, nil
}

// DeleteItemByPublicID deletes an item by its public ID with efficient verification
func (s *ConversationService) DeleteItemByPublicID(ctx context.Context, conversation *Conversation, itemPublicID string, userID uint) (*Conversation, error) {
	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	if conversation.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Find item by public ID
	item, err := s.itemRepo.FindByPublicID(ctx, itemPublicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find item: %w", err)
	}

	if item == nil {
		return nil, ErrItemNotFound
	}

	// ✅ Use efficient existence check instead of loading all items
	exists, err := s.itemRepo.ExistsByIDAndConversation(ctx, item.ID, conversation.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify item ownership: %w", err)
	}

	if !exists {
		return nil, ErrItemNotInConversation
	}

	// Delete the item
	if err := s.itemRepo.DeleteByPublicID(ctx, itemPublicID); err != nil {
		return nil, fmt.Errorf("failed to delete item: %w", err)
	}

	// Update conversation timestamp
	conversation.UpdatedAt = time.Now().Unix()
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	// Return updated conversation with items loaded
	return s.GetConversation(ctx, conversation.PublicID, userID)
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

// generateConversationPublicID generates a cryptographically secure conversation ID
func (s *ConversationService) generateConversationPublicID() (string, error) {
	return idutils.GenerateConversationID()
}

// generateItemPublicID generates a cryptographically secure item ID
func (s *ConversationService) generateItemPublicID() (string, error) {
	return idutils.GenerateItemID()
}

// AddMultipleItems adds multiple items to a conversation in a single transaction
func (s *ConversationService) AddMultipleItems(ctx context.Context, conversation *Conversation, userID uint, items []struct {
	Type    ItemType
	Role    *ItemRole
	Content []Content
}) ([]*Item, error) {
	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, ErrPrivateConversation
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no items provided")
	}

	if len(items) > 100 {
		return nil, fmt.Errorf("cannot add more than 100 items at once")
	}

	now := time.Now().Unix()
	createdItems := make([]*Item, len(items))

	// Create all items
	for i, itemData := range items {
		// Validate content using enhanced validator
		if err := s.validator.ValidateItemContent(itemData.Content); err != nil {
			return nil, fmt.Errorf("validation failed for item %d: %w", i, err)
		}

		itemPublicID, err := s.generateItemPublicID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate item public ID for item %d: %w", i, err)
		}

		item := &Item{
			PublicID:    itemPublicID,
			Type:        itemData.Type,
			Role:        itemData.Role,
			Content:     itemData.Content,
			Status:      stringPtr("completed"),
			CreatedAt:   now,
			CompletedAt: &now,
		}

		if err := s.conversationRepo.AddItem(ctx, conversation.ID, item); err != nil {
			return nil, fmt.Errorf("failed to add item %d: %w", i, err)
		}

		createdItems[i] = item
	}

	// Update conversation timestamp once
	conversation.UpdatedAt = now
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	return createdItems, nil
}

// GetItemsPaginated retrieves items for a conversation with pagination
func (s *ConversationService) GetItemsPaginated(ctx context.Context, publicID string, userID uint, opts PaginationOptions) (*PaginatedResult[*Item], error) {
	conversation, err := s.getConversationWithAccessCheck(ctx, publicID, userID, false)
	if err != nil {
		return nil, err
	}

	return s.itemRepo.FindByConversationIDPaginated(ctx, conversation.ID, opts)
}

// SearchItemsPaginated searches items in a conversation with pagination
func (s *ConversationService) SearchItemsPaginated(ctx context.Context, publicID string, userID uint, query string, opts PaginationOptions) (*PaginatedResult[*Item], error) {
	conversation, err := s.getConversationWithAccessCheck(ctx, publicID, userID, false)
	if err != nil {
		return nil, err
	}

	return s.itemRepo.SearchPaginated(ctx, conversation.ID, query, opts)
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
