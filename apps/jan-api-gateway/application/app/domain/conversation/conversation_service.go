package conversation

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"

	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/user"
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
	convs, err := s.conversationRepo.FindByFilter(ctx, ConversationFilter{
		UserID:   &userID,
		PublicID: &publicID,
	}, nil)
	if err != nil {
		return nil, err
	}
	if len(convs) != 1 {
		return nil, fmt.Errorf("conversation not found")
	}
	return convs[0], nil
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
		user, ok := user.GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "f5742805-2c6e-45a8-b6a8-95091b9d46f0",
			})
			return
		}
		entities, err := s.FindConversationsByFilter(ctx, ConversationFilter{
			PublicID: &publicID,
			UserID:   &user.ID,
		}, nil)

		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:          "1fe94ab8-ba2c-4356-a446-f091c256e260",
				ErrorInstance: err,
			})
			return
		}

		if len(entities) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "e91636c2-fced-4a89-bf08-55309005365f",
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
				Code: "0f5c3304-bf46-45ce-8719-7c03a3485b37",
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
				Code:          "bff3c8bf-c259-46a1-8ff0-7c2b2dbfe1b2",
				ErrorInstance: err,
			})
			return
		}

		if len(entities) == 0 {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "25647b40-4967-497e-9cbd-a85243ccef58",
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
