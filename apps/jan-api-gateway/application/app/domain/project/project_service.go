package project

import (
	"context"
	"fmt"

	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

// ProjectService provides business logic for managing projects.
type ProjectService struct {
	// The service has a dependency on the repository interface.
	repo ProjectRepository
}

// NewService is the constructor for ProjectService.
// It injects the repository dependency.
func NewService(repo ProjectRepository) *ProjectService {
	return &ProjectService{
		repo: repo,
	}
}

func (s *ProjectService) createPublicID() (string, error) {
	return idgen.GenerateSecureID("proj", 16)
}

// CreateProjectWithPublicID creates a new project and automatically
// assigns a unique public ID before saving it to the repository.
func (s *ProjectService) CreateProjectWithPublicID(ctx context.Context, p *Project) (*Project, error) {
	publicID, err := s.createPublicID()
	if err != nil {
		return nil, err
	}
	p.PublicID = publicID

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to create project in repository: %w", err)
	}
	return p, nil
}

// UpdateProject updates an existing project.
func (s *ProjectService) UpdateProject(ctx context.Context, p *Project) (*Project, error) {
	// Basic validation could be added here before calling the repository.
	if p.ID == 0 {
		return nil, fmt.Errorf("cannot update project with an ID of 0")
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}
	return p, nil
}

// DeleteProjectByID deletes a project by its ID.
func (s *ProjectService) DeleteProjectByID(ctx context.Context, id uint) error {
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete project by ID: %w", err)
	}
	return nil
}

// FindProjectByID finds a project by its unique ID.
func (s *ProjectService) FindProjectByID(ctx context.Context, id uint) (*Project, error) {
	return s.repo.FindByID(ctx, id)
}

// FindProjectByPublicID finds a project by its unique public ID.
func (s *ProjectService) FindProjectByPublicID(ctx context.Context, publicID string) (*Project, error) {
	return s.repo.FindByPublicID(ctx, publicID)
}

// FindProjects retrieves a list of projects based on a filter and pagination.
func (s *ProjectService) Find(ctx context.Context, filter ProjectFilter, pagination *query.Pagination) ([]*Project, error) {
	return s.repo.FindByFilter(ctx, filter, pagination)
}

func (s *ProjectService) FindOne(ctx context.Context, filter ProjectFilter) (*Project, error) {
	projectEntities, err := s.repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(projectEntities) != 1 {
		return nil, err
	}
	return projectEntities[0], nil
}

// CountProjects counts the number of projects matching a given filter.
func (s *ProjectService) CountProjects(ctx context.Context, filter ProjectFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}

func (s *ProjectService) AddMember(ctx context.Context, projectID, userID uint, role string) error {
	return s.repo.AddMember(ctx, &ProjectMember{
		UserID:    userID,
		ProjectID: projectID,
		Role:      role,
	})
}
