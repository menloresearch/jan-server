package invite

import (
	"context"
	"fmt"

	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

// InviteService provides business logic for managing invitations.
type InviteService struct {
	repo InviteRepository
}

// NewInviteService is the constructor for InviteService.
func NewInviteService(repo InviteRepository) *InviteService {
	return &InviteService{
		repo: repo,
	}
}

func (s *InviteService) createPublicID() (string, error) {
	return idgen.GenerateSecureID("invite", 16)
}

// CreateInviteWithPublicID creates a new invitation and assigns it a unique
// public ID before saving it to the repository.
func (s *InviteService) CreateInviteWithPublicID(ctx context.Context, invite *Invite) (*Invite, error) {
	publicID, err := s.createPublicID()
	if err != nil {
		return nil, err
	}
	invite.PublicID = publicID
	if err := s.repo.Create(ctx, invite); err != nil {
		return nil, err
	}
	return invite, nil
}

// UpdateInvite updates an existing invitation.
func (s *InviteService) UpdateInvite(ctx context.Context, invite *Invite) (*Invite, error) {
	if invite.ID == 0 {
		return nil, fmt.Errorf("cannot update invite with an ID of 0")
	}
	if err := s.repo.Update(ctx, invite); err != nil {
		return nil, fmt.Errorf("failed to update invite: %w", err)
	}
	return invite, nil
}

// DeleteInviteByID deletes an invitation by its ID.
func (s *InviteService) DeleteInviteByID(ctx context.Context, id uint) error {
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete invite by ID: %w", err)
	}
	return nil
}

// FindInvites retrieves a list of invitations based on a filter and pagination.
func (s *InviteService) FindInvites(ctx context.Context, filter InvitesFilter, pagination *query.Pagination) ([]*Invite, error) {
	return s.repo.FindByFilter(ctx, filter, pagination)
}

// CountInvites counts the number of invitations matching a given filter.
func (s *InviteService) CountInvites(ctx context.Context, filter InvitesFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}
