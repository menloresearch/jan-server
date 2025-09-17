package responserepo

import (
	"context"

	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/response"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

type ResponseGormRepository struct {
	db *transaction.Database
}

var _ response.ResponseRepository = (*ResponseGormRepository)(nil)

func NewResponseGormRepository(db *transaction.Database) response.ResponseRepository {
	return &ResponseGormRepository{
		db: db,
	}
}

// Create creates a new response in the database
func (r *ResponseGormRepository) Create(ctx context.Context, resp *response.Response) error {
	// Generate public ID if not provided
	if resp.PublicID == "" {
		id, err := idgen.GenerateSecureID("resp", 42)
		if err != nil {
			return err
		}
		resp.PublicID = id
	}

	model := dbschema.NewSchemaResponse(resp)
	if err := r.db.GetQuery(ctx).Response.WithContext(ctx).Create(model); err != nil {
		return err
	}
	resp.ID = model.ID
	return nil
}

// Update updates an existing response in the database
func (r *ResponseGormRepository) Update(ctx context.Context, resp *response.Response) error {
	model := dbschema.NewSchemaResponse(resp)
	model.ID = resp.ID

	query := r.db.GetQuery(ctx)
	_, err := query.Response.WithContext(ctx).Where(query.Response.ID.Eq(resp.ID)).Updates(model)
	return err
}

// DeleteByID deletes a response by ID
func (r *ResponseGormRepository) DeleteByID(ctx context.Context, id uint) error {
	query := r.db.GetQuery(ctx)
	_, err := query.Response.WithContext(ctx).Where(query.Response.ID.Eq(id)).Delete()
	return err
}

// FindByID finds a response by ID
func (r *ResponseGormRepository) FindByID(ctx context.Context, id uint) (*response.Response, error) {
	query := r.db.GetQuery(ctx)
	model, err := query.Response.WithContext(ctx).Where(query.Response.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

// FindByPublicID finds a response by public ID
func (r *ResponseGormRepository) FindByPublicID(ctx context.Context, publicID string) (*response.Response, error) {
	query := r.db.GetQuery(ctx)
	model, err := query.Response.WithContext(ctx).Where(query.Response.PublicID.Eq(publicID)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

// FindByFilter finds responses by filter criteria
func (r *ResponseGormRepository) FindByFilter(ctx context.Context, filter response.ResponseFilter, p *query.Pagination) ([]*response.Response, error) {
	query := r.db.GetQuery(ctx)
	sql := query.Response.WithContext(ctx)
	sql = r.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.Response.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.Response.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.Response.ID.Desc())
		} else {
			sql = sql.Order(query.Response.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(item *dbschema.Response) *response.Response {
		return item.EtoD()
	})
	return result, nil
}

// Count counts responses by filter criteria
func (r *ResponseGormRepository) Count(ctx context.Context, filter response.ResponseFilter) (int64, error) {
	query := r.db.GetQuery(ctx)
	q := query.Response.WithContext(ctx)
	q = r.applyFilter(query, q, filter)
	return q.Count()
}

// FindByUserID finds responses by user ID
func (r *ResponseGormRepository) FindByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*response.Response, error) {
	filter := response.ResponseFilter{UserID: &userID}
	return r.FindByFilter(ctx, filter, pagination)
}

// FindByConversationID finds responses by conversation ID
func (r *ResponseGormRepository) FindByConversationID(ctx context.Context, conversationID uint, pagination *query.Pagination) ([]*response.Response, error) {
	filter := response.ResponseFilter{ConversationID: &conversationID}
	return r.FindByFilter(ctx, filter, pagination)
}

// applyFilter applies conditions dynamically to the query
func (r *ResponseGormRepository) applyFilter(
	query *gormgen.Query,
	sql gormgen.IResponseDo,
	filter response.ResponseFilter,
) gormgen.IResponseDo {
	if filter.PublicID != nil {
		sql = sql.Where(query.Response.PublicID.Eq(*filter.PublicID))
	}
	if filter.UserID != nil {
		sql = sql.Where(query.Response.UserID.Eq(*filter.UserID))
	}
	if filter.ConversationID != nil {
		sql = sql.Where(query.Response.ConversationID.Eq(*filter.ConversationID))
	}
	if filter.Model != nil {
		sql = sql.Where(query.Response.Model.Eq(*filter.Model))
	}
	if filter.Status != nil {
		sql = sql.Where(query.Response.Status.Eq(string(*filter.Status)))
	}
	if filter.CreatedAfter != nil {
		sql = sql.Where(query.Response.CreatedAt.Gte(*filter.CreatedAfter))
	}
	if filter.CreatedBefore != nil {
		sql = sql.Where(query.Response.CreatedAt.Lte(*filter.CreatedBefore))
	}
	return sql
}
