package itemrepo

import (
	"context"
	"strings"

	domain "menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
)

type ItemGormRepository struct {
	db *transaction.Database
}

func NewItemGormRepository(db *transaction.Database) domain.ItemRepository {
	return &ItemGormRepository{
		db: db,
	}
}

func (r *ItemGormRepository) Create(ctx context.Context, item *domain.Item) error {
	model := dbschema.NewSchemaItem(item)
	if err := r.db.GetQuery(ctx).Item.WithContext(ctx).Create(model); err != nil {
		return err
	}
	item.ID = model.ID
	return nil
}

func (r *ItemGormRepository) FindByID(ctx context.Context, id uint) (*domain.Item, error) {
	query := r.db.GetQuery(ctx)
	model, err := query.Item.WithContext(ctx).Where(query.Item.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

func (r *ItemGormRepository) FindByConversationID(ctx context.Context, conversationID uint) ([]*domain.Item, error) {
	query := r.db.GetQuery(ctx)
	models, err := query.Item.WithContext(ctx).
		Where(query.Item.ConversationID.Eq(conversationID)).
		Order(query.Item.CreatedAt.Asc()).
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

func (r *ItemGormRepository) Search(ctx context.Context, conversationID uint, searchQuery string) ([]*domain.Item, error) {
	searchTerm := "%" + strings.ToLower(searchQuery) + "%"

	query := r.db.GetQuery(ctx)
	models, err := query.Item.WithContext(ctx).
		Where(query.Item.ConversationID.Eq(conversationID)).
		Where(query.Item.Content.Like(searchTerm)).
		Order(query.Item.CreatedAt.Asc()).
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

func (r *ItemGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Item, error) {
	// Temporary implementation using raw GORM until generated code is updated
	var model dbschema.Item
	err := r.db.GetTx(ctx).WithContext(ctx).Where("public_id = ?", publicID).First(&model).Error
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

func (r *ItemGormRepository) Delete(ctx context.Context, id uint) error {
	query := r.db.GetQuery(ctx)
	_, err := query.Item.WithContext(ctx).Where(query.Item.ID.Eq(id)).Delete()
	return err
}

func (r *ItemGormRepository) DeleteByPublicID(ctx context.Context, publicID string) error {
	// Temporary implementation using raw GORM until generated code is updated
	err := r.db.GetTx(ctx).WithContext(ctx).Where("public_id = ?", publicID).Delete(&dbschema.Item{}).Error
	return err
}
