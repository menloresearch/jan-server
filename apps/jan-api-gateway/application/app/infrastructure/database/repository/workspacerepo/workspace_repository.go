package workspacerepo

import (
	"context"

	"menlo.ai/jan-api-gateway/app/domain/query"
	domain "menlo.ai/jan-api-gateway/app/domain/workspace"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
)

type WorkspaceGormRepository struct {
	db *transaction.Database
}

var _ domain.WorkspaceRepository = (*WorkspaceGormRepository)(nil)

func NewWorkspaceGormRepository(db *transaction.Database) domain.WorkspaceRepository {
	return &WorkspaceGormRepository{db: db}
}

func (repo *WorkspaceGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IWorkspaceDo, filter domain.WorkspaceFilter) gormgen.IWorkspaceDo {
	if filter.UserID != nil {
		sql = sql.Where(query.Workspace.UserID.Eq(*filter.UserID))
	}
	if filter.PublicID != nil {
		sql = sql.Where(query.Workspace.PublicID.Eq(*filter.PublicID))
	}
	if filter.PublicIDs != nil && len(*filter.PublicIDs) > 0 {
		sql = sql.Where(query.Workspace.PublicID.In((*filter.PublicIDs)...))
	}
	if filter.IDs != nil && len(*filter.IDs) > 0 {
		sql = sql.Where(query.Workspace.ID.In((*filter.IDs)...))
	}
	return sql
}

func (repo *WorkspaceGormRepository) Create(ctx context.Context, workspace *domain.Workspace) error {
	model := dbschema.NewSchemaWorkspace(workspace)
	query := repo.db.GetQuery(ctx)
	if err := query.Workspace.WithContext(ctx).Create(model); err != nil {
		return err
	}
	workspace.ID = model.ID
	workspace.CreatedAt = model.CreatedAt
	workspace.UpdatedAt = model.UpdatedAt
	return nil
}

func (repo *WorkspaceGormRepository) Update(ctx context.Context, workspace *domain.Workspace) error {
	model := dbschema.NewSchemaWorkspace(workspace)
	query := repo.db.GetQuery(ctx)
	if err := query.Workspace.WithContext(ctx).Save(model); err != nil {
		return err
	}
	workspace.UpdatedAt = model.UpdatedAt
	return nil
}

func (repo *WorkspaceGormRepository) Delete(ctx context.Context, id uint) error {
	query := repo.db.GetQuery(ctx)
	_, err := query.Workspace.WithContext(ctx).Where(query.Workspace.ID.Eq(id)).Delete()
	return err
}

func (repo *WorkspaceGormRepository) FindByID(ctx context.Context, id uint) (*domain.Workspace, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.Workspace.WithContext(ctx).Where(query.Workspace.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (repo *WorkspaceGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Workspace, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.Workspace.WithContext(ctx).Where(query.Workspace.PublicID.Eq(publicID)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (repo *WorkspaceGormRepository) FindByFilter(ctx context.Context, filter domain.WorkspaceFilter, pagination *query.Pagination) ([]*domain.Workspace, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.Workspace.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)

	if pagination != nil {
		if pagination.Limit != nil && *pagination.Limit > 0 {
			sql = sql.Limit(*pagination.Limit)
		}
		if pagination.Offset != nil {
			sql = sql.Offset(*pagination.Offset)
		}
		if pagination.After != nil {
			if pagination.Order == "desc" {
				sql = sql.Where(query.Workspace.ID.Lt(*pagination.After))
			} else {
				sql = sql.Where(query.Workspace.ID.Gt(*pagination.After))
			}
		}
		if pagination.Order == "desc" {
			sql = sql.Order(query.Workspace.ID.Desc())
		} else {
			sql = sql.Order(query.Workspace.ID.Asc())
		}
	} else {
		sql = sql.Order(query.Workspace.ID.Asc())
	}

	rows, err := sql.Find()
	if err != nil {
		return nil, err
	}

	return functional.Map(rows, func(item *dbschema.Workspace) *domain.Workspace {
		return item.EtoD()
	}), nil
}

func (repo *WorkspaceGormRepository) Count(ctx context.Context, filter domain.WorkspaceFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.Workspace.WithContext(ctx)
	sql = repo.applyFilter(query, sql, filter)
	return sql.Count()
}
