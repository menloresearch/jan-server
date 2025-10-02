package workspace

import (
	"context"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/query"
)

type Workspace struct {
	ID          uint
	PublicID    string
	UserID      uint
	Name        string
	Instruction *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkspaceFilter struct {
	UserID    *uint
	PublicID  *string
	PublicIDs *[]string
	IDs       *[]uint
}

type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *Workspace) error
	Update(ctx context.Context, workspace *Workspace) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Workspace, error)
	FindByPublicID(ctx context.Context, publicID string) (*Workspace, error)
	FindByFilter(ctx context.Context, filter WorkspaceFilter, pagination *query.Pagination) ([]*Workspace, error)
	Count(ctx context.Context, filter WorkspaceFilter) (int64, error)
}
