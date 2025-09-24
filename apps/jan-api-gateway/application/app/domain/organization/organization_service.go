package organization

import (
	"context"
	"fmt"
	"sync"

	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

// OrganizationService provides business logic for managing organizations.
type OrganizationService struct {
	// The service has a dependency on the repository interface.
	repo OrganizationRepository
}

// NewService is the constructor for OrganizationService.
// It injects the repository dependency.
func NewService(repo OrganizationRepository) *OrganizationService {
	return &OrganizationService{
		repo: repo,
	}
}

var DEFAULT_ORGANIZATION_ONCE sync.Once
var DEFAULT_ORGANIZATION *Organization

func UpdateDefaultOrganization(o *Organization) {
	DEFAULT_ORGANIZATION_ONCE.Do(func() {
		DEFAULT_ORGANIZATION = o
	})
}

func (s *OrganizationService) createPublicID() (string, error) {
	return idgen.GenerateSecureID("org", 16)
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

func (s *OrganizationService) FindOneByFilter(ctx context.Context, filter OrganizationFilter) (*Organization, error) {
	orgEntities, err := s.repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(orgEntities) == 0 {
		return nil, nil
	}
	if len(orgEntities) != 1 {
		return nil, fmt.Errorf("no records found")
	}
	return orgEntities[0], nil
}

// CountOrganizations counts the number of organizations matching a given filter.
func (s *OrganizationService) CountOrganizations(ctx context.Context, filter OrganizationFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}

// CountOrganizations counts the number of organizations matching a given filter.
func (s *OrganizationService) AddMember(ctx context.Context, m *OrganizationMember) error {
	return s.repo.AddMember(ctx, m)
}

func (s *OrganizationService) FindMembersByFilter(ctx context.Context, f OrganizationMemberFilter, p *query.Pagination) ([]*OrganizationMember, error) {
	return s.repo.FindMemberByFilter(ctx, f, p)
}

func (s *OrganizationService) FindOneMemberByFilter(ctx context.Context, f OrganizationMemberFilter) (*OrganizationMember, error) {
	entities, err := s.repo.FindMemberByFilter(ctx, f, nil)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, nil
	}
	if len(entities) != 1 {
		return nil, fmt.Errorf("no records")
	}
	return entities[0], err
}

func (s *OrganizationService) FindOrCreateDefaultOrganization(ctx context.Context) (*Organization, error) {
	orgEntity, err := s.FindOneByFilter(ctx, OrganizationFilter{
		Enabled: ptr.ToBool(true),
	})
	if err != nil {
		return nil, err
	}
	if orgEntity != nil {
		return orgEntity, nil
	}

	return s.CreateOrganizationWithPublicID(ctx, &Organization{
		Name:    "Default Organization",
		Enabled: true,
	})
}
