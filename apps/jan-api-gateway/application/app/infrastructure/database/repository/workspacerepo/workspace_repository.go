package workspacerepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"menlo.ai/jan-api-gateway/app/domain/query"
	domain "menlo.ai/jan-api-gateway/app/domain/workspace"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
)

type WorkspaceGormRepository struct {
	db *transaction.Database
}

var _ domain.WorkspaceRepository = (*WorkspaceGormRepository)(nil)

func NewWorkspaceGormRepository(db *transaction.Database) domain.WorkspaceRepository {
	return &WorkspaceGormRepository{db: db}
}

func (r *WorkspaceGormRepository) Create(ctx context.Context, workspace *domain.Workspace) error {
	model := dbschema.NewSchemaWorkspace(workspace)
	tx := r.db.GetTx(ctx).WithContext(ctx)
	if err := tx.Create(model).Error; err != nil {
		return err
	}

	created := model.EtoD()
	workspace.ID = created.ID
	workspace.CreatedAt = created.CreatedAt
	workspace.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *WorkspaceGormRepository) Update(ctx context.Context, workspace *domain.Workspace) error {
	tx := r.db.GetTx(ctx).WithContext(ctx)

	updates := map[string]interface{}{
		"name":       workspace.Name,
		"updated_at": time.Now(),
	}

	if workspace.Instruction != nil {
		updates["instruction"] = *workspace.Instruction
	} else {
		updates["instruction"] = nil
	}

	if err := tx.Model(&dbschema.Workspace{}).Where("id = ?", workspace.ID).Updates(updates).Error; err != nil {
		return err
	}

	var updatedModel dbschema.Workspace
	if err := tx.Where("id = ?", workspace.ID).First(&updatedModel).Error; err != nil {
		return err
	}
	updated := updatedModel.EtoD()
	workspace.Name = updated.Name
	workspace.Instruction = updated.Instruction
	workspace.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *WorkspaceGormRepository) Delete(ctx context.Context, id uint) error {
	tx := r.db.GetTx(ctx).WithContext(ctx)
	return tx.Delete(&dbschema.Workspace{}, id).Error
}

func (r *WorkspaceGormRepository) FindByID(ctx context.Context, id uint) (*domain.Workspace, error) {
	tx := r.db.GetTx(ctx).WithContext(ctx)
	var model dbschema.Workspace
	if err := tx.Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (r *WorkspaceGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Workspace, error) {
	tx := r.db.GetTx(ctx).WithContext(ctx)
	var model dbschema.Workspace
	if err := tx.Where("public_id = ?", publicID).First(&model).Error; err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

func (r *WorkspaceGormRepository) FindByFilter(ctx context.Context, filter domain.WorkspaceFilter, pagination *query.Pagination) ([]*domain.Workspace, error) {
	tx := r.db.GetTx(ctx).WithContext(ctx).Model(&dbschema.Workspace{})
	tx = applyFilter(tx, filter)
	tx = applyPagination(tx, pagination)

	var models []dbschema.Workspace
	if err := tx.Find(&models).Error; err != nil {
		return nil, err
	}

	results := make([]*domain.Workspace, len(models))
	for i := range models {
		results[i] = models[i].EtoD()
	}
	return results, nil
}

func (r *WorkspaceGormRepository) Count(ctx context.Context, filter domain.WorkspaceFilter) (int64, error) {
	tx := r.db.GetTx(ctx).WithContext(ctx).Model(&dbschema.Workspace{})
	tx = applyFilter(tx, filter)

	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func applyFilter(tx *gorm.DB, filter domain.WorkspaceFilter) *gorm.DB {
	if filter.UserID != nil {
		tx = tx.Where("user_id = ?", *filter.UserID)
	}
	if filter.PublicID != nil {
		tx = tx.Where("public_id = ?", *filter.PublicID)
	}
	if filter.PublicIDs != nil && len(*filter.PublicIDs) > 0 {
		tx = tx.Where("public_id IN ?", *filter.PublicIDs)
	}
	if filter.IDs != nil && len(*filter.IDs) > 0 {
		tx = tx.Where("id IN ?", *filter.IDs)
	}
	return tx
}

func applyPagination(tx *gorm.DB, pagination *query.Pagination) *gorm.DB {
	if pagination == nil {
		return tx.Order("id ASC")
	}

	order := "ASC"
	if pagination.Order == "desc" {
		order = "DESC"
	}

	if pagination.After != nil {
		if order == "DESC" {
			tx = tx.Where("id < ?", *pagination.After)
		} else {
			tx = tx.Where("id > ?", *pagination.After)
		}
	}

	if pagination.Offset != nil {
		tx = tx.Offset(*pagination.Offset)
	}
	if pagination.Limit != nil {
		tx = tx.Limit(*pagination.Limit)
	}

	return tx.Order("id " + order)
}
