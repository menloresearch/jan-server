package conversationrepo

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

type ConversationGormRepository struct {
	db *transaction.Database
}

var _ domain.ConversationRepository = (*ConversationGormRepository)(nil)

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
	conversation.CreatedAt = model.CreatedAt
	conversation.UpdatedAt = model.UpdatedAt
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

	// select to update workspace nil as removing
	_, err := query.Conversation.WithContext(ctx).
		Where(query.Conversation.ID.Eq(conversation.ID)).
		Select(
			query.Conversation.PublicID,
			query.Conversation.Title,
			query.Conversation.UserID,
			query.Conversation.WorkspacePublicID,
			query.Conversation.Status,
			query.Conversation.Metadata,
			query.Conversation.IsPrivate,
		).
		Updates(model)
	return err
}

func (r *ConversationGormRepository) Delete(ctx context.Context, id uint) error {
	query := r.db.GetQuery(ctx)
	_, err := query.Conversation.WithContext(ctx).Where(query.Conversation.ID.Eq(id)).Delete()
	return err
}

func (r *ConversationGormRepository) DeleteByWorkspacePublicID(ctx context.Context, workspacePublicID string) error {
	query := r.db.GetQuery(ctx)
	_, err := query.Conversation.WithContext(ctx).Where(query.Conversation.WorkspacePublicID.Eq(workspacePublicID)).Delete()
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

// BulkAddItems adds multiple items to a conversation in a single transaction
func (r *ConversationGormRepository) BulkAddItems(ctx context.Context, conversationID uint, items []*domain.Item) error {
	if len(items) == 0 {
		return nil
	}

	models := make([]*dbschema.Item, len(items))
	for i, item := range items {
		model := dbschema.NewSchemaItem(item)
		model.ConversationID = conversationID
		models[i] = model
	}

	// Use batch insert for better performance
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

func (repo *ConversationGormRepository) FindByFilter(ctx context.Context, filter domain.ConversationFilter, p *query.Pagination) ([]*domain.Conversation, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.Conversation.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.Conversation.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.Conversation.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.Conversation.ID.Desc())
		} else {
			sql = sql.Order(query.Conversation.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(item *dbschema.Conversation) *domain.Conversation {
		return item.EtoD()
	})
	return result, nil
}

func (repo *ConversationGormRepository) applyFilter(
	query *gormgen.Query,
	sql gormgen.IConversationDo,
	filter domain.ConversationFilter,
) gormgen.IConversationDo {
	if filter.PublicID != nil {
		sql = sql.Where(query.Conversation.PublicID.Eq(*filter.PublicID))
	}
	if filter.UserID != nil {
		sql = sql.Where(query.Conversation.UserID.Eq(*filter.UserID))
	}
	if filter.WorkspacePublicID != nil {
		if strings.EqualFold(*filter.WorkspacePublicID, "none") {
			sql = sql.Where(query.Conversation.WorkspacePublicID.IsNull())
		} else {
			sql = sql.Where(query.Conversation.WorkspacePublicID.Eq(*filter.WorkspacePublicID))
		}
	}
	return sql
}

func (repo *ConversationGormRepository) Count(ctx context.Context, filter domain.ConversationFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	q := query.Conversation.WithContext(ctx)
	q = repo.applyFilter(query, q, filter)
	return q.Count()
}
