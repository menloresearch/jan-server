package conversation

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"
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
}

func NewService(conversationRepo ConversationRepository, itemRepo ItemRepository) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		itemRepo:         itemRepo,
	}
}

func (s *ConversationService) CreateConversation(ctx context.Context, userID uint, title *string, isPrivate bool, metadata map[string]string) (*Conversation, error) {
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

// GetConversationWithAccessAndItems retrieves a conversation by its public ID,
// checks access permissions based on userID, loads associated items, and
// populates the Items field on the returned Conversation object.
func (s *ConversationService) GetConversationWithAccessAndItems(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
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

	// Load items
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

	return conversation, nil
}

func (s *ConversationService) GetConversation(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
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

	// Load items
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
	conversation.UpdatedAt = time.Now().Unix()
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

	// Get the item
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to find item: %w", err)
	}

	if item == nil {
		return nil, ErrItemNotFound
	}

	// Verify the item belongs to the conversation
	// We need to get the conversation ID from the item and compare
	// For now, let's check if the item was created as part of this conversation
	// by getting all items from the conversation and checking if our item is there
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

	return item, nil
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

func (s *ConversationService) generateConversationPublicID() (string, error) {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	key := base64.URLEncoding.EncodeToString(bytes)
	key = strings.TrimRight(key, "=")

	if len(key) > 16 {
		key = key[:16]
	} else if len(key) < 16 {
		extra := make([]byte, 16-len(key))
		_, err := rand.Read(extra)
		if err != nil {
			return "", err
		}
		key += base64.URLEncoding.EncodeToString(extra)[:16-len(key)]
	}

	return fmt.Sprintf("conv_%s", key), nil
}

func (s *ConversationService) generateItemPublicID() (string, error) {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	key := base64.URLEncoding.EncodeToString(bytes)
	key = strings.TrimRight(key, "=")

	if len(key) > 16 {
		key = key[:16]
	} else if len(key) < 16 {
		extra := make([]byte, 16-len(key))
		_, err := rand.Read(extra)
		if err != nil {
			return "", err
		}
		key += base64.URLEncoding.EncodeToString(extra)[:16-len(key)]
	}

	return fmt.Sprintf("msg_%s", key), nil
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
