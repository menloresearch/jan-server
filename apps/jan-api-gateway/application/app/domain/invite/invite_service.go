package invite

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/utils/emailservice"
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
	invite.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
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

func (s *InviteService) FindOne(ctx context.Context, filter InvitesFilter) (*Invite, error) {
	entities, err := s.repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return nil, nil
	}
	if len(entities) != 1 {
		return nil, fmt.Errorf("more than one record")
	}
	return entities[0], err
}

// CountInvites counts the number of invitations matching a given filter.
func (s *InviteService) CountInvites(ctx context.Context, filter InvitesFilter) (int64, error) {
	return s.repo.Count(ctx, filter)
}

type EmailMetadata struct {
	InviterEmail string
	OrgName      string
	OrgPublicID  string
	InviteLink   string
}

func (s *InviteService) SendInviteEmail(ctx context.Context, e EmailMetadata, to string) error {
	templateString := `<html><body><div style="font-family: Arial, sans-serif; max-width: 600px; margin: auto; border: 1px solid #ddd; border-radius: 8px; overflow: hidden;">
    <div style="background-color: #f7f7f7; padding: 20px; text-align: center; border-bottom: 1px solid #ddd;">
        <h2 style="margin: 0; color: #333;">You were invited to the organization {{.OrgName}} on JanAI</h2>
    </div>
    <div style="padding: 20px; background-color: #ffffff;">
        <p style="font-size: 16px; color: #555; line-height: 1.6;">
            <strong>{{.InviterEmail}}</strong> invited you to be a member of the organization {{.OrgName}} ({{.OrgPublicID}}) on the JanAI API.
        </p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.InviteLink}}" style="background-color: #007bff; color: #ffffff; padding: 12px 25px; text-decoration: none; border-radius: 5px; font-size: 16px; font-weight: bold;">
                Accept Invite
            </a>
        </div>
        <p style="font-size: 14px; color: #888; text-align: center; margin-top: 20px;">
            This invite will expire in 7 days.
        </p>
    </div>
    <div style="background-color: #f7f7f7; padding: 15px; text-align: center; font-size: 12px; color: #999; border-top: 1px solid #ddd;">
        If you don't recognize this, you may safely ignore it. If you have any additional questions or concerns, please visit our help center.
    </div>
</div></body></html>`
	tmpl, err := template.New("email").Parse(templateString)
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, e); err != nil {
		return err
	}
	emailBody := buffer.String()
	return emailservice.SendEmail(
		to,
		fmt.Sprintf("You were invited to the organization %s on JanAI", e.OrgName),
		emailBody,
	)
}
