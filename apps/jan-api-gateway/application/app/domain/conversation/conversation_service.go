package conversation

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"
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
	publicID, err := s.generatePublicID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate public ID: %w", err)
	}

	conversation := &Conversation{
		PublicID:  publicID,
		Title:     title,
		UserID:    userID,
		Status:    ConversationStatusActive,
		IsPrivate: isPrivate,
		Metadata:  metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.conversationRepo.Create(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conversation, nil
}

func (s *ConversationService) GetConversation(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return nil, fmt.Errorf("conversation not found")
	}

	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, fmt.Errorf("access denied: conversation is private")
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

func (s *ConversationService) ListConversations(ctx context.Context, userID uint, filter ConversationFilter, limit *int, offset *int) ([]*Conversation, error) {
	// Always filter by user for privacy
	filter.UserID = &userID

	conversations, err := s.conversationRepo.Find(ctx, filter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	return conversations, nil
}

func (s *ConversationService) UpdateConversation(ctx context.Context, publicID string, userID uint, title *string, metadata map[string]string) (*Conversation, error) {
	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return nil, fmt.Errorf("conversation not found")
	}

	// Check access permissions
	if conversation.UserID != userID {
		return nil, fmt.Errorf("access denied: not the owner of this conversation")
	}

	// Update fields
	if title != nil {
		conversation.Title = title
	}
	if metadata != nil {
		conversation.Metadata = metadata
	}
	conversation.UpdatedAt = time.Now()

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
		return fmt.Errorf("conversation not found")
	}

	// Check access permissions
	if conversation.UserID != userID {
		return fmt.Errorf("access denied: not the owner of this conversation")
	}

	if err := s.conversationRepo.Delete(ctx, conversation.ID); err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

func (s *ConversationService) AddItem(ctx context.Context, publicID string, userID uint, itemType ItemType, role *ItemRole, content []Content) (*Item, error) {
	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return nil, fmt.Errorf("conversation not found")
	}

	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, fmt.Errorf("access denied: conversation is private")
	}

	item := &Item{
		Type:      itemType,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.conversationRepo.AddItem(ctx, conversation.ID, item); err != nil {
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	// Update conversation timestamp
	conversation.UpdatedAt = time.Now()
	if err := s.conversationRepo.Update(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to update conversation timestamp: %w", err)
	}

	return item, nil
}

func (s *ConversationService) SearchItems(ctx context.Context, publicID string, userID uint, query string) ([]*Item, error) {
	conversation, err := s.conversationRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if conversation == nil {
		return nil, fmt.Errorf("conversation not found")
	}

	// Check access permissions
	if conversation.IsPrivate && conversation.UserID != userID {
		return nil, fmt.Errorf("access denied: conversation is private")
	}

	items, err := s.itemRepo.Search(ctx, conversation.ID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search items: %w", err)
	}

	return items, nil
}

func (s *ConversationService) CountConversations(ctx context.Context, userID uint, filter ConversationFilter) (int64, error) {
	// Always filter by user for privacy
	filter.UserID = &userID

	count, err := s.conversationRepo.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count conversations: %w", err)
	}

	return count, nil
}

func (s *ConversationService) generatePublicID() (string, error) {
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
