package workspace

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/common"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/query"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
)

type WorkspaceContextKey string

const (
	WorkspaceContextKeyPublicID WorkspaceContextKey = "workspace_public_id"
	WorkspaceContextEntity      WorkspaceContextKey = "WorkspaceContextEntity"
)

type WorkspaceService struct {
	repo             WorkspaceRepository
	conversationRepo conversation.ConversationRepository
}

func NewWorkspaceService(repo WorkspaceRepository, conversationRepo conversation.ConversationRepository) *WorkspaceService {
	return &WorkspaceService{
		repo:             repo,
		conversationRepo: conversationRepo,
	}
}

func (s *WorkspaceService) FindWorkspacesByFilter(ctx context.Context, filter WorkspaceFilter, pagination *query.Pagination) ([]*Workspace, *common.Error) {
	workspaces, err := s.repo.FindByFilter(ctx, filter, pagination)
	if err != nil {
		return nil, common.NewError(err, "13df5d74-32c4-4b87-9066-6f9c546f4ad2")
	}
	return workspaces, nil
}

func (s *WorkspaceService) CountWorkspacesByFilter(ctx context.Context, filter WorkspaceFilter) (int64, *common.Error) {
	count, err := s.repo.Count(ctx, filter)
	if err != nil {
		return 0, common.NewError(err, "2d7b2075-f64f-4fc9-9d74-29f738ff3f0a")
	}
	return count, nil
}

func (s *WorkspaceService) CreateWorkspace(ctx context.Context, userID uint, name string, instruction *string) (*Workspace, *common.Error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, common.NewErrorWithMessage("workspace name is required", "3a5dcb2f-9f1c-4f4b-8893-4a62f72f7a00")
	}
	if len([]rune(trimmedName)) > 120 {
		return nil, common.NewErrorWithMessage("workspace name is too long", "94a6a12b-d4f0-4594-8125-95de7f9ce3d6")
	}

	sanitizedInstruction := sanitizeInstruction(instruction)

	publicID, err := idgen.GenerateSecureID("ws", 24)
	if err != nil {
		return nil, common.NewError(err, "6d4af582-0c23-4f91-b45e-253956218b64")
	}

	workspace := &Workspace{
		PublicID:    publicID,
		UserID:      userID,
		Name:        trimmedName,
		Instruction: sanitizedInstruction,
	}

	if err := s.repo.Create(ctx, workspace); err != nil {
		return nil, common.NewError(err, "7ef72c57-90f8-4d59-8d08-2b2edf61d8da")
	}

	return workspace, nil
}

func (s *WorkspaceService) GetWorkspaceByPublicIDAndUserID(ctx context.Context, publicID string, userID uint) (*Workspace, *common.Error) {
	if publicID == "" {
		return nil, common.NewErrorWithMessage("workspace id is required", "70d9041a-a3a5-4654-af30-2b530eb3e734")
	}

	workspaces, err := s.repo.FindByFilter(ctx, WorkspaceFilter{
		PublicID: &publicID,
		UserID:   &userID,
	}, nil)
	if err != nil {
		return nil, common.NewError(err, "ad9be074-4c1e-4d43-828d-fc9e7efc0c52")
	}
	if len(workspaces) == 0 {
		return nil, common.NewErrorWithMessage("workspace not found", "c8bc424c-5b20-4cf9-8ca1-7d9ad1b098c8")
	}
	if len(workspaces) > 1 {
		return nil, common.NewErrorWithMessage("multiple workspaces found", "0d0ff761-aa21-4d0b-91c3-acc0f3fa652f")
	}
	return workspaces[0], nil
}

func (s *WorkspaceService) GetWorkspaceByID(ctx context.Context, id uint) (*Workspace, *common.Error) {
	if id == 0 {
		return nil, common.NewErrorWithMessage("workspace id is required", "d7e0a86f-4f69-421a-9a3f-9048b79ecf5e")
	}

	workspace, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, common.NewError(err, "2225cf26-18c6-44fb-b8f8-8f2f09bc5bf6")
	}
	if workspace == nil {
		return nil, common.NewErrorWithMessage("workspace not found", "4002c7b1-b2a0-4a0b-8684-4c37ff7d9f1f")
	}
	return workspace, nil
}

func (s *WorkspaceService) GetWorkspaceByIDAndUserID(ctx context.Context, id uint, userID uint) (*Workspace, *common.Error) {
	workspace, err := s.GetWorkspaceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if workspace.UserID != userID {
		return nil, common.NewErrorWithMessage("workspace does not belong to user", "c18b514b-5a46-4ed5-8c1c-1928367d5872")
	}
	return workspace, nil
}

func (s *WorkspaceService) UpdateWorkspaceName(ctx context.Context, workspace *Workspace, name string) (*Workspace, *common.Error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, common.NewErrorWithMessage("workspace name is required", "71cf6385-8ca9-4f25-9ad5-2f3ec0e0f765")
	}
	if len([]rune(trimmedName)) > 120 {
		return nil, common.NewErrorWithMessage("workspace name is too long", "d36f9e9f-db49-4d06-81db-75adf127cd7c")
	}

	workspace.Name = trimmedName
	if err := s.repo.Update(ctx, workspace); err != nil {
		return nil, common.NewError(err, "4e4c3a63-9e3c-420a-84f7-4415a7c21e61")
	}
	return workspace, nil
}

func (s *WorkspaceService) UpdateWorkspaceInstruction(ctx context.Context, workspace *Workspace, instruction *string) (*Workspace, *common.Error) {
	workspace.Instruction = sanitizeInstruction(instruction)
	if err := s.repo.Update(ctx, workspace); err != nil {
		return nil, common.NewError(err, "1c59f37a-56fa-4f64-9d8c-8a6c99b2e3ee")
	}
	return workspace, nil
}

func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspace *Workspace) *common.Error {
	if err := s.repo.Delete(ctx, workspace.ID); err != nil {
		return common.NewError(err, "4cfb58ef-8016-4f24-8fcb-48d414d351d2")
	}
	return nil
}

func (s *WorkspaceService) DeleteWorkspaceWithConversations(ctx context.Context, workspace *Workspace) *common.Error {
	if workspace == nil {
		return common.NewErrorWithMessage("workspace is required", "5d35c9b3-61f6-4c40-b6f8-31e0de1d7688")
	}
	if workspace.ID == 0 {
		return common.NewErrorWithMessage("workspace id is required", "7e2f82a6-1c4f-4f67-9ef6-8790896eb99c")
	}
	if s.conversationRepo != nil {
		if err := s.conversationRepo.DeleteByWorkspacePublicID(ctx, workspace.PublicID); err != nil {
			return common.NewError(err, "2adf58f7-df2c-4f7f-bc11-2e9a2928c1f9")
		}
	}
	if err := s.repo.Delete(ctx, workspace.ID); err != nil {
		return common.NewError(err, "4cfb58ef-8016-4f24-8fcb-48d414d351d2")
	}
	return nil
}

func (s *WorkspaceService) GetWorkspaceMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		workspaceID := reqCtx.Param(string(WorkspaceContextKeyPublicID))
		if workspaceID == "" {
			reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
				Code:  "8dbbdf92-0ff6-4b70-99ee-0a6fe48eab8a",
				Error: "missing workspace id",
			})
			return
		}

		user, ok := auth.GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code:  "19d3e0aa-38db-42f4-9ed0-d4f02b8c7c2d",
				Error: "user not found",
			})
			return
		}

		workspace, err := s.GetWorkspaceByPublicIDAndUserID(ctx, workspaceID, user.ID)
		if err != nil {
			status := http.StatusInternalServerError
			if err.GetCode() == "c8bc424c-5b20-4cf9-8ca1-7d9ad1b098c8" {
				status = http.StatusNotFound
			}
			reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
				Code:  err.GetCode(),
				Error: err.Error(),
			})
			return
		}

		SetWorkspaceOnContext(reqCtx, workspace)
		reqCtx.Next()
	}
}

func sanitizeInstruction(instruction *string) *string {
	if instruction == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*instruction)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func SetWorkspaceOnContext(reqCtx *gin.Context, workspace *Workspace) {
	reqCtx.Set(string(WorkspaceContextEntity), workspace)
}

func GetWorkspaceFromContext(reqCtx *gin.Context) (*Workspace, bool) {
	value, ok := reqCtx.Get(string(WorkspaceContextEntity))
	if !ok {
		return nil, false
	}
	workspace, ok := value.(*Workspace)
	if !ok {
		return nil, false
	}
	return workspace, true
}
