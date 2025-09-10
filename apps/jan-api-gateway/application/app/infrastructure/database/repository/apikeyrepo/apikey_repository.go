package apikeyrepo

import (
	"context"

	domain "menlo.ai/jan-api-gateway/app/domain/apikey"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type ApiKeyGormRepository struct {
	db *transaction.Database
}

// Count implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) Count(ctx context.Context, filter domain.ApiKeyFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.WithContext(ctx).ApiKey
	sql = repo.applyFilter(query, sql, filter)
	return sql.Count()
}

// Create implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) Create(ctx context.Context, a *domain.ApiKey) error {
	model := dbschema.NewSchemaApiKey(a)
	query := repo.db.GetQuery(ctx)
	err := query.ApiKey.WithContext(ctx).Create(model)
	if err != nil {
		return err
	}
	a.ID = model.ID
	return nil
}

// DeleteByID implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) DeleteByID(ctx context.Context, id uint) error {
	return repo.db.GetTx(ctx).Delete(&dbschema.ApiKey{}, id).Error
}

// FindByID implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) FindByID(ctx context.Context, id uint) (*domain.ApiKey, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.ApiKey.WithContext(ctx).Where(query.ApiKey.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByKeyHash implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) FindByKeyHash(ctx context.Context, keyHash string) (*domain.ApiKey, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.ApiKey.WithContext(ctx).Where(query.ApiKey.KeyHash.Eq(keyHash)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// Update implements apikey.ApiKeyRepository.
func (repo *ApiKeyGormRepository) Update(ctx context.Context, u *domain.ApiKey) error {
	query := repo.db.GetQuery(ctx)
	apiKey := dbschema.NewSchemaApiKey(u)
	return query.ApiKey.WithContext(ctx).Save(apiKey)
}

func (repo *ApiKeyGormRepository) FindByFilter(ctx context.Context, filter domain.ApiKeyFilter, p *query.Pagination) ([]*domain.ApiKey, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.WithContext(ctx).ApiKey
	sql = repo.applyFilter(query, sql, filter)
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			sql = sql.Limit(*p.Limit)
		}
		if p.After != nil {
			if p.Order == "desc" {
				sql = sql.Where(query.ApiKey.ID.Lt(*p.After))
			} else {
				sql = sql.Where(query.ApiKey.ID.Gt(*p.After))
			}
		}
		if p.Order == "desc" {
			sql = sql.Order(query.ApiKey.ID.Desc())
		} else {
			// Default to ascending order
			sql = sql.Order(query.ApiKey.ID.Asc())
		}
	}
	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}
	result := functional.Map(rows, func(item *dbschema.ApiKey) *domain.ApiKey {
		return item.EtoD()
	})
	return result, nil
}

func (repo *ApiKeyGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IApiKeyDo, filter domain.ApiKeyFilter) gormgen.IApiKeyDo {
	if filter.ApikeyType != nil {
		sql = sql.Where(query.ApiKey.ApikeyType.Eq(*filter.ApikeyType))
	}
	if filter.OwnerPublicID != nil {
		sql = sql.Where(query.ApiKey.OwnerPublicID.Eq(*filter.OwnerPublicID))
	}
	if filter.OrganizationID != nil {
		sql = sql.Where(query.ApiKey.OrganizationID.Eq(*filter.OrganizationID))
	}
	if filter.PublicID != nil {
		sql = sql.Where(query.ApiKey.PublicID.Eq(*filter.PublicID))
	}
	if filter.ProjectID != nil {
		sql = sql.Where(query.ApiKey.ProjectID.Eq(*filter.ProjectID))
	}
	return sql
}

func NewApiKeyGormRepository(db *transaction.Database) domain.ApiKeyRepository {
	return &ApiKeyGormRepository{
		db: db,
	}
}
