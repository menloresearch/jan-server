package itemrepo

import (
	"context"
	"strconv"
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

// FindByConversationIDPaginated returns paginated items for a conversation
func (r *ItemGormRepository) FindByConversationIDPaginated(ctx context.Context, conversationID uint, opts domain.PaginationOptions) (*domain.PaginatedResult[*domain.Item], error) {
	query := r.db.GetQuery(ctx).Item.WithContext(ctx).Where(r.db.GetQuery(ctx).Item.ConversationID.Eq(conversationID))

	// Apply ordering
	if opts.Order == "desc" {
		query = query.Order(r.db.GetQuery(ctx).Item.CreatedAt.Desc())
	} else {
		query = query.Order(r.db.GetQuery(ctx).Item.CreatedAt.Asc())
	}

	// Apply cursor-based pagination
	if opts.Cursor != "" {
		// Parse cursor as ID (simplified - in production, use encrypted cursor)
		if cursorID, err := strconv.ParseUint(opts.Cursor, 10, 64); err == nil {
			if opts.Order == "desc" {
				query = query.Where(r.db.GetQuery(ctx).Item.ID.Lt(uint(cursorID)))
			} else {
				query = query.Where(r.db.GetQuery(ctx).Item.ID.Gt(uint(cursorID)))
			}
		}
	}

	// Fetch one extra item to determine if there are more
	limit := opts.Limit
	if limit <= 0 {
		limit = 20 // Default limit
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	models, err := query.Limit(limit + 1).Find()
	if err != nil {
		return nil, err
	}

	hasMore := len(models) > limit
	if hasMore {
		models = models[:limit] // Remove the extra item
	}

	items := make([]*domain.Item, len(models))
	for i, model := range models {
		items[i] = model.EtoD()
	}

	result := &domain.PaginatedResult[*domain.Item]{
		Data:    items,
		HasMore: hasMore,
	}

	// Set next cursor
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		result.NextCursor = strconv.FormatUint(uint64(lastItem.ID), 10)
	}

	return result, nil
}

// SearchPaginated returns paginated search results for items in a conversation
func (r *ItemGormRepository) SearchPaginated(ctx context.Context, conversationID uint, searchQuery string, opts domain.PaginationOptions) (*domain.PaginatedResult[*domain.Item], error) {
	searchTerm := "%" + strings.ToLower(searchQuery) + "%"

	query := r.db.GetQuery(ctx).Item.WithContext(ctx).
		Where(r.db.GetQuery(ctx).Item.ConversationID.Eq(conversationID)).
		Where(r.db.GetQuery(ctx).Item.Content.Like(searchTerm))

	// Apply ordering
	if opts.Order == "desc" {
		query = query.Order(r.db.GetQuery(ctx).Item.CreatedAt.Desc())
	} else {
		query = query.Order(r.db.GetQuery(ctx).Item.CreatedAt.Asc())
	}

	// Apply cursor-based pagination (similar to above)
	if opts.Cursor != "" {
		if cursorID, err := strconv.ParseUint(opts.Cursor, 10, 64); err == nil {
			if opts.Order == "desc" {
				query = query.Where(r.db.GetQuery(ctx).Item.ID.Lt(uint(cursorID)))
			} else {
				query = query.Where(r.db.GetQuery(ctx).Item.ID.Gt(uint(cursorID)))
			}
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	models, err := query.Limit(limit + 1).Find()
	if err != nil {
		return nil, err
	}

	hasMore := len(models) > limit
	if hasMore {
		models = models[:limit]
	}

	items := make([]*domain.Item, len(models))
	for i, model := range models {
		items[i] = model.EtoD()
	}

	result := &domain.PaginatedResult[*domain.Item]{
		Data:    items,
		HasMore: hasMore,
	}

	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		result.NextCursor = strconv.FormatUint(uint64(lastItem.ID), 10)
	}

	return result, nil
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
