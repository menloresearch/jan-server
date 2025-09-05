package inviterepo

import (
	"context"

	domain "menlo.ai/jan-api-gateway/app/domain/invite"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type InviteGormRepository struct {
	db *transaction.Database
}

var _ domain.InviteRepository = (*InviteGormRepository)(nil)

func (repo *InviteGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IInviteDo, filter domain.InvitesFilter) gormgen.IInviteDo {
	if filter.PublicID != nil {
		sql = sql.Where(query.Invite.PublicID.Eq(*filter.PublicID))
	}
	if filter.OrganizationID != nil {
		sql = sql.Where(query.Invite.OrganizationID.Eq(*filter.OrganizationID))
	}
	return sql
}

func (repo *InviteGormRepository) Create(ctx context.Context, i *domain.Invite) error {
	model := dbschema.NewSchemaInvite(i)
	query := repo.db.GetQuery(ctx)
	err := query.Invite.WithContext(ctx).Create(model)
	if err != nil {
		return err
	}
	i.ID = model.ID
	return nil
}

func (repo *InviteGormRepository) Update(ctx context.Context, i *domain.Invite) error {
	invite := dbschema.NewSchemaInvite(i)
	query := repo.db.GetQuery(ctx)
	return query.Invite.WithContext(ctx).Save(invite)
}

func (repo *InviteGormRepository) DeleteByID(ctx context.Context, id uint) error {
	return repo.db.GetTx(ctx).Delete(&dbschema.Invite{}, id).Error
}

func (repo *InviteGormRepository) FindByFilter(ctx context.Context, filter domain.InvitesFilter, p *query.Pagination) ([]*domain.Invite, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.Invite.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.Invite.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.Invite.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.Invite.ID.Desc())
		} else {
			sql = sql.Order(query.Invite.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(item *dbschema.Invite) *domain.Invite {
		return item.EtoD()
	})
	return result, nil
}

func (repo *InviteGormRepository) Count(ctx context.Context, filter domain.InvitesFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	q := query.Invite.WithContext(ctx)
	q = repo.applyFilter(query, q, filter)
	return q.Count()
}

func NewInviteGormRepository(db *transaction.Database) domain.InviteRepository {
	return &InviteGormRepository{
		db: db,
	}
}
