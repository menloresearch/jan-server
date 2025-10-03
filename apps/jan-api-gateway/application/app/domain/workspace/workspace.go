package workspace

import (
	"context"
	"strings"
	"time"

	"menlo.ai/jan-api-gateway/app/domain/common"
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

func (w *Workspace) Normalize() error {
	trimmedName := strings.TrimSpace(w.Name)
	if trimmedName == "" {
		return common.NewErrorWithMessage("workspace name is required", "3a5dcb2f-9f1c-4f4b-8893-4a62f72f7a00")
	}
	if len([]rune(trimmedName)) > 50 {
		return common.NewErrorWithMessage("workspace name is too long", "94a6a12b-d4f0-4594-8125-95de7f9ce3d6")
	}
	return nil
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
