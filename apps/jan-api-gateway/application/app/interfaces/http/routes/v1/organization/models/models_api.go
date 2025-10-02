package models

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	"menlo.ai/jan-api-gateway/app/domain/organization"
	orgModels "menlo.ai/jan-api-gateway/app/domain/organization/models"
)

// ModelsAPI handles HTTP requests for organization models
type ModelsAPI struct {
	modelService *orgModels.ModelService
	authService  *auth.AuthService
}

// NewModelsAPI creates a new ModelsAPI instance
func NewModelsAPI(modelService *orgModels.ModelService, authService *auth.AuthService) *ModelsAPI {
	return &ModelsAPI{
		modelService: modelService,
		authService:  authService,
	}
}

// RegisterRouter registers the models routes
func (api *ModelsAPI) RegisterRouter(router gin.IRouter) {
	modelsRouter := router.Group("/models")

	modelsRouter.GET("", api.ListModels)
	modelsRouter.POST("", api.CreateModel)
	modelsRouter.GET("/:model_id", api.GetModel)
	modelsRouter.DELETE("/:model_id", api.DeleteModel)
}

// getOrganizationFromContext extracts organization from gin context
func (api *ModelsAPI) getOrganizationFromContext(c *gin.Context) (*organization.Organization, error) {
	org, exists := c.Get(string(auth.OrganizationContextKeyEntity))
	if !exists {
		return nil, fmt.Errorf("organization context not found")
	}

	organization, ok := org.(*organization.Organization)
	if !ok {
		return nil, fmt.Errorf("invalid organization context")
	}

	return organization, nil
}

// ListModels returns all models for the organization
// @Summary List organization models
// @Description Get all models belonging to the organization (both managed and unmanaged)
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param status query string false "Filter by model status"
// @Param managed query boolean false "Filter by managed (true) or unmanaged (false) models"
// @Success 200 {object} ModelsListResponse "List of models"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models [get]
func (api *ModelsAPI) ListModels(c *gin.Context) {
	ctx := c.Request.Context()

	organization, err := api.getOrganizationFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	// Parse query parameters for filtering
	filter := &orgModels.ModelFilter{}

	if status := c.Query("status"); status != "" {
		ms := orgModels.ModelStatus(status)
		filter.Status = &ms
	}

	if managedStr := c.Query("managed"); managedStr != "" {
		if managed, err := strconv.ParseBool(managedStr); err == nil {
			filter.Managed = &managed
		}
	}

	models, err := api.modelService.ListModels(ctx, organization.ID, filter)
	if err != nil {
		if err.Error() == "models API only available when running in Kubernetes cluster" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, ModelsListResponse{
		Models: models,
		Total:  len(models),
	})
}

// CreateModel creates a new model
// @Summary Create a new model
// @Description Create a new AI model for the organization
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model body orgModels.ModelCreateRequest true "Model creation request"
// @Success 201 {object} ModelResponse "Created model"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models [post]
func (api *ModelsAPI) CreateModel(c *gin.Context) {
	ctx := c.Request.Context()

	organization, err := api.getOrganizationFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	userID, exists := c.Get(string(auth.UserContextKeyID))
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "user context not found",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid user ID format",
		})
		return
	}

	var req orgModels.ModelCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	model, err := api.modelService.CreateModel(ctx, organization.ID, userIDStr, &req)
	if err != nil {
		if err.Error() == "models API only available when running in Kubernetes cluster" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: err.Error(),
			})
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, ModelResponse{
		Model: *model,
	})
}

// GetModel returns a specific model
// @Summary Get a model by name
// @Description Get details of a specific model by its name
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model_id path string true "Model name"
// @Success 200 {object} ModelResponse "Model details"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 404 {object} ErrorResponse "Model not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models/{model_id} [get]
func (api *ModelsAPI) GetModel(c *gin.Context) {
	ctx := c.Request.Context()

	organization, err := api.getOrganizationFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	modelID := c.Param("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "model_id is required",
		})
		return
	}

	model, err := api.modelService.GetModel(ctx, organization.ID, modelID)
	if err != nil {
		if err.Error() == "models API only available when running in Kubernetes cluster" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: err.Error(),
			})
		} else if err.Error() == "model not found in organization" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, ModelResponse{
		Model: *model,
	})
}

// DeleteModel deletes a model
// @Summary Delete a model
// @Description Delete a model and its associated Kubernetes resources
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model_id path string true "Model name"
// @Success 204 "Model deleted successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 404 {object} ErrorResponse "Model not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models/{model_id} [delete]
func (api *ModelsAPI) DeleteModel(c *gin.Context) {
	ctx := c.Request.Context()

	organization, err := api.getOrganizationFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	modelID := c.Param("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "model_id is required",
		})
		return
	}

	err = api.modelService.DeleteModel(ctx, organization.ID, modelID)
	if err != nil {
		if err.Error() == "models API only available when running in Kubernetes cluster" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error: err.Error(),
			})
		} else if err.Error() == "model not found in organization" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: err.Error(),
			})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
