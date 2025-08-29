package conversationrepo

import (
	"context"
	"strings"

	domain "menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
)

type ConversationGormRepository struct {
	db *transaction.Database
}

func NewConversationGormRepository(db *transaction.Database) domain.ConversationRepository {
	return &ConversationGormRepository{
		db: db,
	}
}

func (r *ConversationGormRepository) Create(ctx context.Context, conversation *domain.Conversation) error {
	model := dbschema.NewSchemaConversation(conversation)
	if err := r.db.GetQuery(ctx).Conversation.WithContext(ctx).Create(model); err != nil {
		return err
	}
	conversation.ID = model.ID
	return nil
}

func (r *ConversationGormRepository) FindByID(ctx context.Context, id uint) (*domain.Conversation, error) {
	query := r.db.GetQuery(ctx)
	model, err := query.Conversation.WithContext(ctx).Where(query.Conversation.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

func (r *ConversationGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Conversation, error) {
	query := r.db.GetQuery(ctx)
	model, err := query.Conversation.WithContext(ctx).Where(query.Conversation.PublicID.Eq(publicID)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

func (r *ConversationGormRepository) Update(ctx context.Context, conversation *domain.Conversation) error {
	model := dbschema.NewSchemaConversation(conversation)
	model.ID = conversation.ID

	query := r.db.GetQuery(ctx)
	_, err := query.Conversation.WithContext(ctx).Where(query.Conversation.ID.Eq(conversation.ID)).Updates(model)
	return err
}

func (r *ConversationGormRepository) Delete(ctx context.Context, id uint) error {
	query := r.db.GetQuery(ctx)
	_, err := query.Conversation.WithContext(ctx).Where(query.Conversation.ID.Eq(id)).Delete()
	return err
}

func (r *ConversationGormRepository) AddItem(ctx context.Context, conversationID uint, item *domain.Item) error {
	model := dbschema.NewSchemaItem(item)
	model.ConversationID = conversationID

	if err := r.db.GetQuery(ctx).Item.WithContext(ctx).Create(model); err != nil {
		return err
	}
	item.ID = model.ID
	return nil
}

func (r *ConversationGormRepository) SearchItems(ctx context.Context, conversationID uint, query string) ([]*domain.Item, error) {
	searchTerm := "%" + strings.ToLower(query) + "%"

	gormQuery := r.db.GetQuery(ctx)
	models, err := gormQuery.Item.WithContext(ctx).
		Where(gormQuery.Item.ConversationID.Eq(conversationID)).
		Where(gormQuery.Item.Content.Like(searchTerm)).
		Order(gormQuery.Item.CreatedAt.Asc()).
		Find()

	if err != nil {
		return nil, err
	}

	items := make([]*domain.Item, len(models))
	for i, model := range models {
		items[i] = model.EtoD()
	}

	return items, nil
}
