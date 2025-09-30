package modelproviderrepo

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	domain "menlo.ai/jan-api-gateway/app/domain/modelprovider"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
)

type ModelProviderGormRepository struct {
	db *transaction.Database
}

var _ domain.ModelProviderRepository = (*ModelProviderGormRepository)(nil)

func NewModelProviderGormRepository(db *transaction.Database) domain.ModelProviderRepository {
	return &ModelProviderGormRepository{db: db}
}

func (r *ModelProviderGormRepository) Create(ctx context.Context, provider *domain.ModelProvider) error {
	model := dbschema.NewSchemaModelProvider(provider)
	tx := r.db.GetTx(ctx)
	if err := tx.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	provider.ID = model.ID
	provider.CreatedAt = model.CreatedAt
	provider.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *ModelProviderGormRepository) Update(ctx context.Context, provider *domain.ModelProvider) error {
	model := dbschema.NewSchemaModelProvider(provider)
	tx := r.db.GetTx(ctx)
	return tx.WithContext(ctx).Model(&dbschema.ModelProvider{}).
		Where("id = ?", provider.ID).
		Updates(model).Error
}

func (r *ModelProviderGormRepository) DeleteByID(ctx context.Context, id uint) error {
	tx := r.db.GetTx(ctx)
	return tx.WithContext(ctx).Delete(&dbschema.ModelProvider{}, id).Error
}

func (r *ModelProviderGormRepository) FindByID(ctx context.Context, id uint) (*domain.ModelProvider, error) {
	tx := r.db.GetTx(ctx)
	var model dbschema.ModelProvider
	if err := tx.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (r *ModelProviderGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.ModelProvider, error) {
	tx := r.db.GetTx(ctx)
	var model dbschema.ModelProvider
	if err := tx.WithContext(ctx).
		Where("public_id = ?", publicID).
		First(&model).Error; err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (r *ModelProviderGormRepository) Find(ctx context.Context, filter domain.ProviderFilter, pagination *query.Pagination) ([]*domain.ModelProvider, error) {
	tx := r.db.GetTx(ctx)
	query := tx.WithContext(ctx).Model(&dbschema.ModelProvider{})
	query = applyFilter(query, filter)
	query = applyPagination(query, pagination)

	var models []dbschema.ModelProvider
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	providers := make([]*domain.ModelProvider, len(models))
	for i := range models {
		providers[i] = models[i].EtoD()
	}
	return providers, nil
}

func (r *ModelProviderGormRepository) Count(ctx context.Context, filter domain.ProviderFilter) (int64, error) {
	tx := r.db.GetTx(ctx)
	query := tx.WithContext(ctx).Model(&dbschema.ModelProvider{})
	query = applyFilter(query, filter)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func applyFilter(query *gorm.DB, filter domain.ProviderFilter) *gorm.DB {
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
	}
	if filter.PublicID != nil {
		query = query.Where("public_id = ?", *filter.PublicID)
	}
	if filter.OrganizationID != nil {
		query = query.Where("organization_id = ?", *filter.OrganizationID)
	}
	if filter.ProjectIDs != nil && len(*filter.ProjectIDs) > 0 {
		query = query.Where("project_id IN ?", *filter.ProjectIDs)
	} else if filter.ProjectID != nil {
		query = query.Where("project_id = ?", *filter.ProjectID)
	}
	if filter.Type != nil {
		query = query.Where("type = ?", filter.Type.String())
	}
	if filter.Vendor != nil {
		query = query.Where("vendor = ?", filter.Vendor.String())
	}
	if filter.Active != nil {
		query = query.Where("active = ?", *filter.Active)
	}
	if filter.Search != nil {
		like := fmt.Sprintf("%%%s%%", strings.ToLower(*filter.Search))
		query = query.Where("LOWER(name) LIKE ?", like)
	}
	return query
}

func applyPagination(query *gorm.DB, pagination *query.Pagination) *gorm.DB {
	if pagination == nil {
		return query.Order("id ASC")
	}
	if pagination.Limit != nil {
		query = query.Limit(*pagination.Limit)
	}
	if pagination.Offset != nil {
		query = query.Offset(*pagination.Offset)
	}
	order := "ASC"
	if strings.ToLower(pagination.Order) == "desc" {
		order = "DESC"
	}
	if pagination.After != nil {
		if order == "DESC" {
			query = query.Where("id < ?", *pagination.After)
		} else {
			query = query.Where("id > ?", *pagination.After)
		}
	}
	return query.Order("id " + order)
}
