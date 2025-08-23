package apikeyrepo

import (
	"context"

	domain "menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/utils/functional"

	"gorm.io/gorm"
)

type ApiKeyGormRepository struct {
	query *gormgen.Query
	db    *gorm.DB
}

// Count implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) Count(ctx context.Context, filter domain.ApiKeyFilter) (int64, error) {
	query := repo.query.ApiKey.WithContext(ctx)
	query = repo.applyFilter(query, filter)
	return query.Count()
}

// Create implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) Create(ctx context.Context, a *domain.ApiKey) error {
	model := dbschema.NewSchemaApiKey(a)
	err := repo.query.ApiKey.WithContext(ctx).Create(model)
	if err != nil {
		return err
	}
	a.ID = model.ID
	return nil
}

// DeleteByID implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) DeleteByID(ctx context.Context, id uint) error {
	return repo.db.WithContext(ctx).Delete(&dbschema.ApiKey{}, id).Error
}

// FindByID implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) FindByID(ctx context.Context, id uint) (*domain.ApiKey, error) {
	model, err := repo.query.ApiKey.WithContext(ctx).Where(repo.query.ApiKey.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByKeyHash implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) FindByKeyHash(ctx context.Context, keyHash string) (*domain.ApiKey, error) {
	model, err := repo.query.ApiKey.WithContext(ctx).Where(repo.query.ApiKey.KeyHash.Eq(keyHash)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// Update implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) Update(ctx context.Context, u *domain.ApiKey) error {
	apiKey := dbschema.NewSchemaApiKey(u)
	return repo.query.ApiKey.WithContext(ctx).Save(apiKey)
}

func (repo *ApiKeyGormRepository) FindByFilter(ctx context.Context, filter domain.ApiKeyFilter, p *query.Pagination) ([]*domain.ApiKey, error) {
	query := repo.query.ApiKey.WithContext(ctx)
	query = repo.applyFilter(query, filter)
	if p != nil {
		query = query.Limit(p.PageSize).Offset(p.PageNumber - 1)
	}
	rows, err := query.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(item *dbschema.ApiKey) *domain.ApiKey {
		return item.EtoD()
	})
	return result, nil
}

func (repo *ApiKeyGormRepository) applyFilter(query gormgen.IApiKeyDo, filter domain.ApiKeyFilter) gormgen.IApiKeyDo {
	if filter.OwnerType != nil {
		query = query.Where(repo.query.ApiKey.OwnerType.Eq(*filter.OwnerType))
	}
	if filter.OwnerID != nil {
		query = query.Where(repo.query.ApiKey.OwnerID.Eq(*filter.OwnerID))
	}
	return query
}

func NewApiKeyGormRepository(db *gorm.DB) domain.ApiKeyRepository {
	return &ApiKeyGormRepository{
		query: gormgen.Use(db),
		db:    db,
	}
}
