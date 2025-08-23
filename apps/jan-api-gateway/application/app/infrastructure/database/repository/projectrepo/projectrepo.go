package projectrepo

import (
	"context"

	"gorm.io/gorm"
	domain "menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/utils/functional"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type ProjectGormRepository struct {
	query *gormgen.Query
	db    *gorm.DB
}

// AddMember implements project.ProjectRepository.
func (repo *ProjectGormRepository) AddMember(ctx context.Context, m *domain.ProjectMember) error {
	panic("unimplemented")
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
func (repo *ProjectGormRepository) applyFilter(query gormgen.IProjectDo, filter domain.ProjectFilter) gormgen.IProjectDo {
	if filter.PublicID != nil {
		query = query.Where(repo.query.Project.PublicID.Eq(*filter.PublicID))
	}
	if filter.Status != nil {
		query = query.Where(repo.query.Project.Status.Eq(*filter.Status))
	}
	if filter.OrganizationID != nil {
		query = query.Where(repo.query.Project.OrganizationID.Eq(*filter.OrganizationID))
	}
	if filter.Archived == ptr.ToBool(true) {
		query = query.Where(repo.query.Project.ArchivedAt.IsNotNull())
	}
	return query
}

// Create persists a new project to the database.
func (repo *ProjectGormRepository) Create(ctx context.Context, p *domain.Project) error {
	model := dbschema.NewSchemaProject(p)
	err := repo.query.Project.WithContext(ctx).Create(model)
	if err != nil {
		return err
	}
	p.ID = model.ID
	return nil
}

// Update modifies an existing project.
func (repo *ProjectGormRepository) Update(ctx context.Context, p *domain.Project) error {
	project := dbschema.NewSchemaProject(p)
	return repo.query.Project.WithContext(ctx).Save(project)
}

// DeleteByID removes a project by its ID.
func (repo *ProjectGormRepository) DeleteByID(ctx context.Context, id uint) error {
	return repo.db.WithContext(ctx).Delete(&dbschema.Project{}, id).Error
}

// FindByID retrieves a project by its primary key.
func (repo *ProjectGormRepository) FindByID(ctx context.Context, id uint) (*domain.Project, error) {
	model, err := repo.query.Project.WithContext(ctx).Where(repo.query.Project.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByPublicID retrieves a project by its public ID.
func (repo *ProjectGormRepository) FindByPublicID(ctx context.Context, publicID string) (*domain.Project, error) {
	model, err := repo.query.Project.WithContext(ctx).Where(repo.query.Project.PublicID.Eq(publicID)).First()
	if err != nil {
		return nil, err
	}
	return model.EtoD(), nil
}

// FindByFilter retrieves a list of projects matching filter + pagination.
func (repo *ProjectGormRepository) FindByFilter(ctx context.Context, filter domain.ProjectFilter, p *query.Pagination) ([]*domain.Project, error) {
	q := repo.query.Project.WithContext(ctx)
	q = repo.applyFilter(q, filter)
	if p != nil {
		q = q.Limit(p.PageSize).Offset((p.PageNumber - 1) * p.PageSize)
	}
	rows, err := q.Find()
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
	q := repo.query.Project.WithContext(ctx)
	q = repo.applyFilter(q, filter)
	return q.Count()
}

// NewProjectGormRepository creates a new Project repo instance.
func NewProjectGormRepository(db *gorm.DB) domain.ProjectRepository {
	return &ProjectGormRepository{
		query: gormgen.Use(db),
		db:    db,
	}
}
