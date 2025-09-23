package organizationrepo

import (
	"context"

	"gorm.io/gorm/clause"
	domain "menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type OrganizationGormRepository struct {
	db *transaction.Database
}

var _ domain.OrganizationRepository = (*OrganizationGormRepository)(nil)

// applyFilter is a helper function to conditionally apply filter clauses to the GORM query.
func (repo *OrganizationGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IOrganizationDo, filter domain.OrganizationFilter) gormgen.IOrganizationDo {
	if filter.PublicID != nil {
		sql = sql.Where(query.Organization.PublicID.Eq(*filter.PublicID))
	}
	// If the Enabled filter is not nil, add a WHERE clause.
	if filter.Enabled != nil {
		sql = sql.Where(query.Organization.Enabled.Is(*filter.Enabled))
	}
	return sql
}

// Create persists a new organization to the database.
func (repo *OrganizationGormRepository) Create(ctx context.Context, o *domain.Organization) error {
	model := dbschema.NewSchemaOrganization(o)
	query := repo.db.GetQuery(ctx)
	err := query.Organization.WithContext(ctx).Create(model)
	if err != nil {
		return err
	}
	o.ID = model.ID
	return nil
}

// Update modifies an existing organization.
func (repo *OrganizationGormRepository) Update(ctx context.Context, o *domain.Organization) error {
	organization := dbschema.NewSchemaOrganization(o)
	query := repo.db.GetQuery(ctx)
	return query.Organization.WithContext(ctx).Save(organization)
}

// DeleteByID removes an organization by its ID.
func (repo *OrganizationGormRepository) DeleteByID(ctx context.Context, id uint) error {
	return repo.db.GetTx(ctx).Delete(&dbschema.Organization{}, id).Error
}

// FindByID retrieves an organization by its primary key ID.
func (repo *OrganizationGormRepository) FindByID(ctx context.Context, id uint) (*domain.Organization, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.Organization.WithContext(ctx).Where(query.Organization.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (repo *OrganizationGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Organization, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.Organization.WithContext(ctx).Where(query.Organization.PublicID.Eq(publicID)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByFilter retrieves a list of organizations based on a filter and pagination.
func (repo *OrganizationGormRepository) FindByFilter(ctx context.Context, filter domain.OrganizationFilter, p *query.Pagination) ([]*domain.Organization, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.WithContext(ctx).Organization
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.Organization.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.Organization.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.Organization.ID.Desc())
		} else {
			// Default to ascending order
			sql = sql.Order(query.Organization.ID.Asc())
		}
	}

	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(org *dbschema.Organization) *domain.Organization {
		return org.EtoD()
	})
	return result, nil
}

// Count returns the total number of organizations matching a given filter.
func (repo *OrganizationGormRepository) Count(ctx context.Context, filter domain.OrganizationFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.WithContext(ctx).Organization
	sql = repo.applyFilter(query, sql, filter)
	return sql.Count()
}

// AddMember implements organization.OrganizationRepository.
func (repo *OrganizationGormRepository) AddMember(ctx context.Context, m *domain.OrganizationMember) error {
	model := dbschema.NewSchemaOrganizationMember(m)
	query := repo.db.GetQuery(ctx)
	return query.OrganizationMember.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: query.OrganizationMember.UserID.ColumnName().String()},
			{Name: query.OrganizationMember.OrganizationID.ColumnName().String()},
		},
		DoNothing: false,
		UpdateAll: true,
	}).Create(model)
}

// FindMemberByFilter implements organization.OrganizationRepository.
func (repo *OrganizationGormRepository) FindMemberByFilter(ctx context.Context, filter domain.OrganizationMemberFilter, p *query.Pagination) ([]*domain.OrganizationMember, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.WithContext(ctx).OrganizationMember
	sql = repo.applyMemberFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.OrganizationMember.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.OrganizationMember.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.OrganizationMember.ID.Desc())
		} else {
			// Default to ascending order
			sql = sql.Order(query.OrganizationMember.ID.Asc())
		}
	}

	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(org *dbschema.OrganizationMember) *domain.OrganizationMember {
		return org.EtoD()
	})
	return result, nil
}

// applyFilter is a helper function to conditionally apply filter clauses to the GORM query.
func (repo *OrganizationGormRepository) applyMemberFilter(query *gormgen.Query, sql gormgen.IOrganizationMemberDo, filter domain.OrganizationMemberFilter) gormgen.IOrganizationMemberDo {
	if filter.UserID != nil {
		sql = sql.Where(query.OrganizationMember.UserID.Eq(*filter.UserID))
	}
	// If the Enabled filter is not nil, add a WHERE clause.
	if filter.OrganizationID != nil {
		sql = sql.Where(query.OrganizationMember.OrganizationID.Eq(*filter.OrganizationID))
	}
	if filter.Role != nil {
		sql = sql.Where(query.OrganizationMember.Role.Eq(*filter.Role))
	}
	return sql
}

// NewOrganizationGormRepository creates a new repository instance.
func NewOrganizationGormRepository(db *transaction.Database) domain.OrganizationRepository {
	return &OrganizationGormRepository{
		db: db,
	}
}
