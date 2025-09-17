package userrepo

import (
	"context"

	"menlo.ai/jan-api-gateway/app/domain/query"
	domain "menlo.ai/jan-api-gateway/app/domain/user"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type UserGormRepository struct {
	db *transaction.Database
}

var _ domain.UserRepository = (*UserGormRepository)(nil)

func NewUserGormRepository(db *transaction.Database) domain.UserRepository {
	return &UserGormRepository{
		db: db,
	}
}

func (r *UserGormRepository) Create(ctx context.Context, u *domain.User) error {
	model := dbschema.NewSchemaUser(u)
	if err := r.db.GetQuery(ctx).User.WithContext(ctx).Create(model); err != nil {
		return err
	}
	u.ID = model.ID
	return nil
}

func (r *UserGormRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	query := r.db.GetQuery(ctx)
	model, err := query.User.WithContext(ctx).Where(query.User.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}

	return model.EtoD(), nil
}

func (repo *UserGormRepository) FindFirst(ctx context.Context, filter domain.UserFilter) (*domain.User, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.User.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	item, err := sql.First()
	if err != nil {
		return nil, err
	}
	return item.EtoD(), nil
}

func (repo *UserGormRepository) FindByFilter(ctx context.Context, filter domain.UserFilter, p *query.Pagination) ([]*domain.User, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.User.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.Project.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.Project.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.Project.ID.Desc())
		} else {
			sql = sql.Order(query.Project.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(item *dbschema.User) *domain.User {
		return item.EtoD()
	})
	return result, nil
}

// applyFilter applies conditions dynamically to the query.
func (repo *UserGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IUserDo, filter domain.UserFilter) gormgen.IUserDo {
	if filter.PublicID != nil {
		sql = sql.Where(query.User.PublicID.Eq(*filter.PublicID))
	}
	if filter.Email != nil {
		sql = sql.Where(query.User.Email.Eq(*filter.Email))
	}
	if filter.Enabled != nil {
		sql = sql.Where(query.User.Enabled.Is(*filter.Enabled))
	}
	if filter.OrganizationId != nil {
		sql = sql.
			Join(query.OrganizationMember, query.OrganizationMember.UserID.EqCol(query.User.ID)).
			Where(query.OrganizationMember.OrganizationID.Eq(*filter.OrganizationId))
	}
	return sql
}

func (r *UserGormRepository) Update(ctx context.Context, u *domain.User) error {
	user := dbschema.NewSchemaUser(u)
	query := r.db.GetQuery(ctx)
	return query.User.WithContext(ctx).Save(user)
}
