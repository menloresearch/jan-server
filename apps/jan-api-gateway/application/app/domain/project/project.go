package project

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type Project struct {
	ID             uint
	Name           string
	PublicID       string
	Status         string
	OrganizationID uint
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ArchivedAt     *time.Time
	IsDefault      bool
}

type ProjectMember struct {
	ID        uint
	UserID    uint
	ProjectID uint
	Role      string
}

type ProjectFilter struct {
	PublicID       *string
	Status         *string
	OrganizationID *uint
	Archived       *bool
	PublicIDs      *[]string
}

type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusArchived ProjectStatus = "archived"
)

type ProjectMemberRole string

const (
	ProjectMemberRoleOwner  ProjectMemberRole = "owner"
	ProjectMemberRoleMember ProjectMemberRole = "member"
)

type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	Update(ctx context.Context, p *Project) error
	DeleteByID(ctx context.Context, id uint) error

	FindByID(ctx context.Context, id uint) (*Project, error)
	FindByPublicID(ctx context.Context, publicID string) (*Project, error)
	FindByFilter(ctx context.Context, filter ProjectFilter, p *query.Pagination) ([]*Project, error)
	Count(ctx context.Context, filter ProjectFilter) (int64, error)

	AddMember(ctx context.Context, m *ProjectMember) error
	RemoveMember(ctx context.Context, projectID, userID uint) error
	ListMembers(ctx context.Context, projectID uint) ([]*ProjectMember, error)
	UpdateMemberRole(ctx context.Context, projectID, userID uint, role string) error
}
