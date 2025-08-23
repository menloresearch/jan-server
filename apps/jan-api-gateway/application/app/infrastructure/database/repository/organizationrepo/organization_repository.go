package organizationrepo

import (
	"context"

	"gorm.io/gorm"
	domain "menlo.ai/jan-api-gateway/app/domain/organization"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type OrganizationGormRepository struct {
	query *gormgen.Query
	db    *gorm.DB
}

// applyFilter is a helper function to conditionally apply filter clauses to the GORM query.
func (repo *OrganizationGormRepository) applyFilter(query gormgen.IOrganizationDo, filter domain.OrganizationFilter) gormgen.IOrganizationDo {
	if filter.PublicID != nil {
		query = query.Where(repo.query.Organization.PublicID.Eq(*filter.PublicID))
	}
	// If the Enabled filter is not nil, add a WHERE clause.
	if filter.Enabled != nil {
		query = query.Where(repo.query.Organization.Enabled.Is(*filter.Enabled))
	}
	return query
}

// Create persists a new organization to the database.
func (repo *OrganizationGormRepository) Create(ctx context.Context, o *domain.Organization) error {
	model := dbschema.NewSchemaOrganization(o)
	err := repo.query.Organization.WithContext(ctx).Create(model)
	if err != nil {
		return err
	}
	o.ID = model.ID
	return nil
}

// Update modifies an existing organization.
func (repo *OrganizationGormRepository) Update(ctx context.Context, o *domain.Organization) error {
	organization := dbschema.NewSchemaOrganization(o)
	return repo.query.Organization.WithContext(ctx).Save(organization)
}

// DeleteByID removes an organization by its ID.
func (repo *OrganizationGormRepository) DeleteByID(ctx context.Context, id uint) error {
	return repo.db.WithContext(ctx).Delete(&dbschema.Organization{}, id).Error
}

// FindByID retrieves an organization by its primary key ID.
func (repo *OrganizationGormRepository) FindByID(ctx context.Context, id uint) (*domain.Organization, error) {
	model, err := repo.query.Organization.WithContext(ctx).Where(repo.query.Organization.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (repo *OrganizationGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Organization, error) {
	model, err := repo.query.Organization.WithContext(ctx).Where(repo.query.Organization.PublicID.Eq(publicID)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByFilter retrieves a list of organizations based on a filter and pagination.
func (repo *OrganizationGormRepository) FindByFilter(ctx context.Context, filter domain.OrganizationFilter, p *query.Pagination) ([]*domain.Organization, error) {
	query := repo.query.Organization.WithContext(ctx)
	query = repo.applyFilter(query, filter)
	if p != nil {
		query = query.Limit(p.PageSize).Offset(p.PageNumber - 1)
	}
	rows, err := query.Find()
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
	query := repo.query.Organization.WithContext(ctx)
	query = repo.applyFilter(query, filter)
	return query.Count()
}

// NewOrganizationGormRepository creates a new repository instance.
func NewOrganizationGormRepository(db *gorm.DB) domain.OrganizationRepository {
	return &OrganizationGormRepository{
		query: gormgen.Use(db),
		db:    db,
	}
}
