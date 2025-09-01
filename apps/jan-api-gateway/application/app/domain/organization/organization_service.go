package organization

import (
	"context"
	"fmt"

	"menlo.ai/jan-api-gateway/app/domain/project"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/domain/shared/id"
)

// OrganizationService provides business logic for managing organizations.
type OrganizationService struct {
	// The service has a dependency on the repository interface.
	repo           OrganizationRepository
	projectService *project.ProjectService
	idService      *id.IDService
}

// NewService is the constructor for OrganizationService.
// It injects the repository dependency.
func NewService(repo OrganizationRepository, projectService *project.ProjectService, idService *id.IDService) *OrganizationService {
	return &OrganizationService{
		repo:           repo,
		projectService: projectService,
		idService:      idService,
	}
}

func (s *OrganizationService) createPublicID() (string, error) {
	return s.idService.GenerateOrganizationID()
}

// CreateOrganizationWithPublicID creates a new organization and automatically
// assigns a unique public ID before saving it to the repository.
func (s *OrganizationService) CreateOrganizationWithPublicID(ctx context.Context, o *Organization) (*Organization, error) {
	publicID, err := s.createPublicID()
	if err != nil {
		return nil, err
	}
	o.PublicID = publicID
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, err
	}

	projectEntity, err := s.projectService.CreateProjectWithPublicID(ctx, &project.Project{
		Name:           "Default Project",
		Status:         string(project.ProjectStatusActive),
		OrganizationID: o.ID,
	})

	if err != nil {
		return nil, err
	}

	err = s.projectService.AddMember(ctx, projectEntity.ID, o.OwnerID, string(project.ProjectMemberRoleOwner))
	if err != nil {
		return nil, err
	}
	return o, nil
}

// UpdateOrganization updates an existing organization.
func (s *OrganizationService) UpdateOrganization(ctx context.Context, o *Organization) (*Organization, error) {
	// Basic validation could be added here before calling the repository.
	if o.ID == 0 {
		return nil, fmt.Errorf("cannot update organization with an ID of 0")
	}
	if err := s.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	return o, nil
}

// DeleteOrganizationByID deletes an organization by its ID.
func (s *OrganizationService) DeleteOrganizationByID(ctx context.Context, id uint) error {
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete organization by ID: %w", err)
	}
	return nil
}

// FindOrganizationByID finds an organization by its unique ID.
func (s *OrganizationService) FindOrganizationByID(ctx context.Context, id uint) (*Organization, error) {
	return s.repo.FindByID(ctx, id)
}

// FindOrganizationByPublicID finds an organization by its unique public ID.
func (s *OrganizationService) FindOrganizationByPublicID(ctx context.Context, publicID string) (*Organization, error) {
	return s.repo.FindByPublicID(ctx, publicID)
}

// FindOrganizations retrieves a list of organizations based on a filter and pagination.
func (s *OrganizationService) FindOrganizations(ctx context.Context, filter OrganizationFilter, pagination *query.Pagination) ([]*Organization, error) {
	return s.repo.FindByFilter(ctx, filter, pagination)
}

// CountOrganizations counts the number of organizations matching a given filter.
func (s *OrganizationService) CountOrganizations(ctx context.Context, filter OrganizationFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}
