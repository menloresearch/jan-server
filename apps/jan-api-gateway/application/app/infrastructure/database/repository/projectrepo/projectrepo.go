package projectrepo

import (
	"context"

	"gorm.io/gorm/clause"
	domain "menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/repository/transaction"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ProjectGormRepository struct {
	db *transaction.Database
}

var _ domain.ProjectRepository = (*ProjectGormRepository)(nil)

// AddMember implements project.ProjectRepository.
func (repo *ProjectGormRepository) AddMember(ctx context.Context, m *domain.ProjectMember) error {
	model := dbschema.NewSchemaProjectMember(m)
	query := repo.db.GetQuery(ctx)
	err := query.ProjectMember.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: query.ProjectMember.UserID.ColumnName().String()},
			{Name: query.ProjectMember.ProjectID.ColumnName().String()},
		},
		DoNothing: false,
		UpdateAll: true,
	}).Create(model)
	if err != nil {
		return err
	}
	m.ID = model.ID
	return nil
}

// ListMembers implements project.ProjectRepository.
func (repo *ProjectGormRepository) ListMembers(ctx context.Context, projectID uint) ([]*domain.ProjectMember, error) {
	panic("unimplemented")
}

// RemoveMember implements project.ProjectRepository.
func (repo *ProjectGormRepository) RemoveMember(ctx context.Context, projectID uint, userID uint) error {
	panic("unimplemented")
}

// UpdateMemberRole implements project.ProjectRepository.
func (repo *ProjectGormRepository) UpdateMemberRole(ctx context.Context, projectID uint, userID uint, role string) error {
	panic("unimplemented")
}

// applyFilter applies conditions dynamically to the query.
func (repo *ProjectGormRepository) applyFilter(query *gormgen.Query, sql gormgen.IProjectDo, filter domain.ProjectFilter) gormgen.IProjectDo {
	if filter.PublicID != nil {
		sql = sql.Where(query.Project.PublicID.Eq(*filter.PublicID))
	}
	if filter.Status != nil {
		sql = sql.Where(query.Project.Status.Eq(*filter.Status))
	}
	if filter.OrganizationID != nil {
		sql = sql.Where(query.Project.OrganizationID.Eq(*filter.OrganizationID))
	}
	if filter.Archived == ptr.ToBool(true) {
		sql = sql.Where(query.Project.ArchivedAt.IsNotNull())
	}
	if filter.PublicIDs != nil {
		sql = sql.Where(query.Project.PublicID.In(*filter.PublicIDs...))
	}
	return sql
}

// Create persists a new project to the database.
func (repo *ProjectGormRepository) Create(ctx context.Context, p *domain.Project) error {
	model := dbschema.NewSchemaProject(p)
	query := repo.db.GetQuery(ctx)
	err := query.Project.WithContext(ctx).Create(model)
	if err != nil {
		return err
	}
	p.ID = model.ID
	return nil
}

// Update modifies an existing project.
func (repo *ProjectGormRepository) Update(ctx context.Context, p *domain.Project) error {
	project := dbschema.NewSchemaProject(p)
	query := repo.db.GetQuery(ctx)
	return query.Project.WithContext(ctx).Save(project)
}

// DeleteByID removes a project by its ID.
func (repo *ProjectGormRepository) DeleteByID(ctx context.Context, id uint) error {
	return repo.db.GetTx(ctx).Delete(&dbschema.Project{}, id).Error
}

// FindByID retrieves a project by its primary key.
func (repo *ProjectGormRepository) FindByID(ctx context.Context, id uint) (*domain.Project, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.Project.WithContext(ctx).Where(query.Project.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByPublicID retrieves a project by its public ID.
func (repo *ProjectGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Project, error) {
	query := repo.db.GetQuery(ctx)
	model, err := query.Project.WithContext(ctx).Where(query.Project.PublicID.Eq(publicID)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByFilter retrieves a list of projects matching filter + pagination.
func (repo *ProjectGormRepository) FindByFilter(ctx context.Context, filter domain.ProjectFilter, p *query.Pagination) ([]*domain.Project, error) {
	query := repo.db.GetQuery(ctx)
	sql := query.Project.WithContext(ctx)
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
	result := functional.Map(rows, func(item *dbschema.Project) *domain.Project {
		return item.EtoD()
	})
	return result, nil
}

// Count returns number of projects that match filter.
func (repo *ProjectGormRepository) Count(ctx context.Context, filter domain.ProjectFilter) (int64, error) {
	query := repo.db.GetQuery(ctx)
	q := query.Project.WithContext(ctx)
	q = repo.applyFilter(query, q, filter)
	return q.Count()
}

// NewProjectGormRepository creates a new Project repo instance.
func NewProjectGormRepository(db *transaction.Database) domain.ProjectRepository {
	return &ProjectGormRepository{
		db: db,
	}
}
