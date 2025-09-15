package itemrepo

import (
	"context"
	"strings"

	domain "menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
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

// BulkCreate creates multiple items in a single batch operation
func (r *ItemGormRepository) BulkCreate(ctx context.Context, items []*domain.Item) error {
	if len(items) == 0 {
		return nil
	}

	models := make([]*dbschema.Item, len(items))
	for i, item := range items {
		models[i] = dbschema.NewSchemaItem(item)
	}

	query := r.db.GetQuery(ctx)
	if err := query.Item.WithContext(ctx).CreateInBatches(models, 100); err != nil {
		return err
	}

	// Update the items with their assigned IDs
	for i, model := range models {
		items[i].ID = model.ID
	}

	return nil
}

// CountByConversation counts items in a conversation
func (r *ItemGormRepository) CountByConversation(ctx context.Context, conversationID uint) (int64, error) {
	query := r.db.GetQuery(ctx)
	return query.Item.WithContext(ctx).Where(query.Item.ConversationID.Eq(conversationID)).Count()
}

// ExistsByIDAndConversation efficiently checks if an item exists in a conversation
func (r *ItemGormRepository) ExistsByIDAndConversation(ctx context.Context, itemID uint, conversationID uint) (bool, error) {
	query := r.db.GetQuery(ctx)
	count, err := query.Item.WithContext(ctx).
		Where(query.Item.ID.Eq(itemID)).
		Where(query.Item.ConversationID.Eq(conversationID)).
		Count()

	return count > 0, err
}

// Count implements conversation.ItemRepository.
func (repo *ItemGormRepository) Count(ctx context.Context, filter domain.ItemFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	q := query.Item.WithContext(ctx)
	q = repo.applyFilter(query, q, filter)
	return q.Count()
}

// FindByFilter implements conversation.ItemRepository.
func (repo *ItemGormRepository) FindByFilter(ctx context.Context, filter domain.ItemFilter, p *query.Pagination) ([]*domain.Item, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.Item.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.Item.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.Item.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.Item.ID.Desc())
		} else {
			sql = sql.Order(query.Item.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(item *dbschema.Item) *domain.Item {
		return item.EtoD()
	})
	return result, nil
}

func (repo *ItemGormRepository) applyFilter(
	query *gormgen.Query,
	sql gormgen.IItemDo,
	filter domain.ItemFilter,
) gormgen.IItemDo {
	if filter.PublicID != nil {
		sql = sql.Where(query.Item.PublicID.Eq(*filter.PublicID))
	}
	if filter.ConversationID != nil {
		sql = sql.Where(query.Item.ConversationID.Eq(*filter.ConversationID))
	}
	if filter.Role != nil {
		sql = sql.Where(query.Item.Role.Eq(string(*filter.Role)))
	}
	if filter.ResponseID != nil {
		sql = sql.Where(query.Item.ResponseID.Eq(*filter.ResponseID))
	}
	return sql
}
