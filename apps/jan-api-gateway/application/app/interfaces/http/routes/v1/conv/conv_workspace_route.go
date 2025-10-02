package conv

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/conversation"
	"menlo.ai/jan-api-gateway/app/domain/workspace"
	"menlo.ai/jan-api-gateway/app/interfaces/http/responses"
	"menlo.ai/jan-api-gateway/app/utils/ptr"
)

type WorkspaceRoute struct {
	authService         *auth.AuthService
	workspaceService    *workspace.WorkspaceService
	conversationService *conversation.ConversationService
}

type CreateWorkspaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Instruction *string `json:"instruction"`
}

type UpdateWorkspaceNameRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateWorkspaceInstructionRequest struct {
	Instruction *string `json:"instruction"`
}

type WorkspaceResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Instruction *string `json:"instruction,omitempty"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

type WorkspaceDeletedResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

func NewWorkspaceRoute(authService *auth.AuthService, workspaceService *workspace.WorkspaceService, conversationService *conversation.ConversationService) *WorkspaceRoute {
	return &WorkspaceRoute{
		authService:         authService,
		workspaceService:    workspaceService,
		conversationService: conversationService,
	}
}

func (route *WorkspaceRoute) RegisterRouter(router gin.IRouter) {
	convRouter := router.Group("/conv",
		route.authService.AppUserAuthMiddleware(),
		route.authService.RegisteredUserMiddleware(),
	)

	workspacesRouter := convRouter.Group("/workspaces")
	workspacesRouter.POST("", route.CreateWorkspace)
	workspacesRouter.GET("", route.ListWorkspaces)

	workspaceMiddleware := route.workspaceService.GetWorkspaceMiddleware()
	workspacesRouter.PATCH(
		fmt.Sprintf("/:%s/name", workspace.WorkspaceContextKeyPublicID),
		workspaceMiddleware,
		route.UpdateWorkspaceName,
	)
	workspacesRouter.PATCH(
		fmt.Sprintf("/:%s/instruction", workspace.WorkspaceContextKeyPublicID),
		workspaceMiddleware,
		route.UpdateWorkspaceInstruction,
	)
	workspacesRouter.DELETE(
		fmt.Sprintf("/:%s", workspace.WorkspaceContextKeyPublicID),
		workspaceMiddleware,
		route.DeleteWorkspace,
	)
}

// CreateWorkspace godoc
// @Summary Create Workspace
// @Description Creates a new workspace for the authenticated user.
// @Tags conv Workspaces API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateWorkspaceRequest true "Workspace creation payload"
// @Success 201 {object} WorkspaceCreateResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/conv/workspaces [post]
func (route *WorkspaceRoute) CreateWorkspace(reqCtx *gin.Context) {
	var request CreateWorkspaceRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "e663a5c4-bfc8-452f-bbe9-6d3f65d3d7dc",
			Error: "invalid request payload",
		})
		return
	}

	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "f1b9907a-1560-4c7b-8eb4-874fa6ab54b2",
			Error: "user not found",
		})
		return
	}

	ctx := reqCtx.Request.Context()
	workspaceEntity, err := route.workspaceService.CreateWorkspace(ctx, user.ID, request.Name, request.Instruction)
	if err != nil {
		status := http.StatusInternalServerError
		if err.GetCode() == "3a5dcb2f-9f1c-4f4b-8893-4a62f72f7a00" || err.GetCode() == "94a6a12b-d4f0-4594-8125-95de7f9ce3d6" {
			status = http.StatusBadRequest
		}
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusCreated, responses.GeneralResponse[WorkspaceResponse]{
		Status: responses.ResponseCodeOk,
		Result: toWorkspaceResponse(workspaceEntity),
	})
}

// ListWorkspaces godoc
// @Summary List Workspaces
// @Description Lists all workspaces for the authenticated user.
// @Tags conv Workspaces API
// @Security BearerAuth
// @Produce json
// @Success 200 {object} WorkspaceListResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/conv/workspaces [get]
func (route *WorkspaceRoute) ListWorkspaces(reqCtx *gin.Context) {
	user, ok := auth.GetUserFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Code:  "b7a4fb56-451f-4b44-8d85-d5f222528b0d",
			Error: "user not found",
		})
		return
	}

	ctx := reqCtx.Request.Context()
	workspaces, err := route.workspaceService.FindWorkspacesByFilter(ctx, workspace.WorkspaceFilter{
		UserID: &user.ID,
	}, nil)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	responsesList := make([]WorkspaceResponse, len(workspaces))
	for i, entity := range workspaces {
		responsesList[i] = toWorkspaceResponse(entity)
	}

	var firstID *string
	var lastID *string
	if len(workspaces) > 0 {
		firstID = ptr.ToString(workspaces[0].PublicID)
		lastID = ptr.ToString(workspaces[len(workspaces)-1].PublicID)
	}

	reqCtx.JSON(http.StatusOK, responses.ListResponse[WorkspaceResponse]{
		Status:  responses.ResponseCodeOk,
		Total:   int64(len(workspaces)),
		Results: responsesList,
		FirstID: firstID,
		LastID:  lastID,
		HasMore: false,
	})
}

// UpdateWorkspaceName godoc
// @Summary Update Workspace Name
// @Description Updates the name of a workspace.
// @Tags conv Workspaces API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param request body UpdateWorkspaceNameRequest true "Workspace name update payload"
// @Success 200 {object} WorkspaceCreateResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/conv/workspaces/{workspace_id}/name [patch]
func (route *WorkspaceRoute) UpdateWorkspaceName(reqCtx *gin.Context) {
	var request UpdateWorkspaceNameRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "6dc4fb8f-8af9-4b66-a67a-c2efdec59f5e",
			Error: "invalid request payload",
		})
		return
	}

	workspaceEntity, ok := workspace.GetWorkspaceFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "c8bc424c-5b20-4cf9-8ca1-7d9ad1b098c8",
			Error: "workspace not found",
		})
		return
	}

	ctx := reqCtx.Request.Context()
	updated, err := route.workspaceService.UpdateWorkspaceName(ctx, workspaceEntity, request.Name)
	if err != nil {
		status := http.StatusInternalServerError
		if err.GetCode() == "71cf6385-8ca9-4f25-9ad5-2f3ec0e0f765" || err.GetCode() == "d36f9e9f-db49-4d06-81db-75adf127cd7c" {
			status = http.StatusBadRequest
		}
		reqCtx.AbortWithStatusJSON(status, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[WorkspaceResponse]{
		Status: responses.ResponseCodeOk,
		Result: toWorkspaceResponse(updated),
	})
}

// UpdateWorkspaceInstruction godoc
// @Summary Update Workspace Instruction
// @Description Updates the shared instruction for a workspace.
// @Tags conv Workspaces API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Param request body UpdateWorkspaceInstructionRequest true "Workspace instruction update payload"
// @Success 200 {object} WorkspaceCreateResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/conv/workspaces/{workspace_id}/instruction [patch]
func (route *WorkspaceRoute) UpdateWorkspaceInstruction(reqCtx *gin.Context) {
	var request UpdateWorkspaceInstructionRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusBadRequest, responses.ErrorResponse{
			Code:  "c26be0cb-0c74-486f-bc72-7bb67ad839b1",
			Error: "invalid request payload",
		})
		return
	}

	workspaceEntity, ok := workspace.GetWorkspaceFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "c8bc424c-5b20-4cf9-8ca1-7d9ad1b098c8",
			Error: "workspace not found",
		})
		return
	}

	ctx := reqCtx.Request.Context()
	updated, err := route.workspaceService.UpdateWorkspaceInstruction(ctx, workspaceEntity, request.Instruction)
	if err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[WorkspaceResponse]{
		Status: responses.ResponseCodeOk,
		Result: toWorkspaceResponse(updated),
	})
}

// DeleteWorkspace godoc
// @Summary Delete Workspace
// @Description Deletes a workspace and cascades to its conversations.
// @Tags conv Workspaces API
// @Security BearerAuth
// @Param workspace_id path string true "Workspace ID"
// @Success 200 {object} WorkspaceDeleteResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/conv/workspaces/{workspace_id} [delete]
func (route *WorkspaceRoute) DeleteWorkspace(reqCtx *gin.Context) {
	workspaceEntity, ok := workspace.GetWorkspaceFromContext(reqCtx)
	if !ok {
		reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
			Code:  "c8bc424c-5b20-4cf9-8ca1-7d9ad1b098c8",
			Error: "workspace not found",
		})
		return
	}

	ctx := reqCtx.Request.Context()
	if err := route.workspaceService.DeleteWorkspaceWithConversations(ctx, workspaceEntity); err != nil {
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, responses.ErrorResponse{
			Code:  err.GetCode(),
			Error: err.Error(),
		})
		return
	}

	reqCtx.JSON(http.StatusOK, responses.GeneralResponse[WorkspaceDeletedResponse]{
		Status: responses.ResponseCodeOk,
		Result: WorkspaceDeletedResponse{
			ID:      workspaceEntity.PublicID,
			Deleted: true,
		},
	})
}

func toWorkspaceResponse(entity *workspace.Workspace) WorkspaceResponse {
	var instruction *string
	if entity.Instruction != nil {
		instruction = ptr.ToString(*entity.Instruction)
	}

	return WorkspaceResponse{
		ID:          entity.PublicID,
		Name:        entity.Name,
		Instruction: instruction,
		CreatedAt:   entity.CreatedAt.Unix(),
		UpdatedAt:   entity.UpdatedAt.Unix(),
	}
}

type WorkspaceCreateResponse struct {
	Status string            `json:"status"`
	Result WorkspaceResponse `json:"result"`
}

type WorkspaceListResponse struct {
	Status  string              `json:"status"`
	Total   int64               `json:"total"`
	Results []WorkspaceResponse `json:"results"`
	FirstID *string             `json:"first_id"`
	LastID  *string             `json:"last_id"`
	HasMore bool                `json:"has_more"`
}

type WorkspaceDeleteResponse struct {
	Status string                   `json:"status"`
	Result WorkspaceDeletedResponse `json:"result"`
}
