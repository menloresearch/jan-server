package models

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	orgModels "menlo.ai/jan-api-gateway/app/domain/organization/models"
	"menlo.ai/jan-api-gateway/app/utils/contextkeys"
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

	modelsRouter.GET("/status", api.GetClusterStatus)
	modelsRouter.GET("", api.ListModels)
	modelsRouter.GET("/all", api.ListAllModels) // Both managed and unmanaged
	modelsRouter.POST("", api.CreateModel)
	modelsRouter.GET("/:model_id", api.GetModel)
	modelsRouter.PUT("/:model_id", api.UpdateModel)
	modelsRouter.DELETE("/:model_id", api.DeleteModel)
} // GetClusterStatus returns the cluster status for model deployment
// @Summary Get cluster status for models
// @Description Get information about the Kubernetes cluster's capability for model deployment
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} ClusterStatusResponse "Cluster status information"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models/status [get]
func (api *ModelsAPI) GetClusterStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Get organization from context (set by auth middleware)
	orgID, exists := c.Get(contextkeys.OrganizationID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "organization context not found",
		})
		return
	}

	// Validate API access first
	if err := api.modelService.ValidateModelAPIAccess(ctx); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	status, err := api.modelService.GetClusterStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ClusterStatusResponse{
		OrganizationID: orgID.(uint),
		ClusterStatus:  *status,
	})
}

// ListModels returns all models for the organization
// @Summary List organization models
// @Description Get all models belonging to the organization
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model_type query string false "Filter by model type"
// @Param status query string false "Filter by model status"
// @Param is_public query boolean false "Filter by public/private models"
// @Success 200 {object} ModelsListResponse "List of models"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models [get]
func (api *ModelsAPI) ListModels(c *gin.Context) {
	ctx := c.Request.Context()

	orgID, exists := c.Get(contextkeys.OrganizationID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "organization context not found",
		})
		return
	}

	// Parse query parameters for filtering
	filter := &orgModels.ModelFilter{}

	if modelType := c.Query("model_type"); modelType != "" {
		mt := orgModels.ModelType(modelType)
		filter.ModelType = &mt
	}

	if status := c.Query("status"); status != "" {
		ms := orgModels.ModelStatus(status)
		filter.Status = &ms
	}

	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		if isPublic, err := strconv.ParseBool(isPublicStr); err == nil {
			filter.IsPublic = &isPublic
		}
	}

	models, err := api.modelService.ListModels(ctx, orgID.(uint), filter)
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

	orgID, exists := c.Get(contextkeys.OrganizationID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "organization context not found",
		})
		return
	}

	userID, exists := c.Get(contextkeys.UserID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "user context not found",
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

	model, err := api.modelService.CreateModel(ctx, orgID.(uint), userID.(uint), &req)
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
// @Summary Get a model by ID
// @Description Get details of a specific model by its public ID
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model_id path string true "Model public ID"
// @Success 200 {object} ModelResponse "Model details"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 404 {object} ErrorResponse "Model not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models/{model_id} [get]
func (api *ModelsAPI) GetModel(c *gin.Context) {
	ctx := c.Request.Context()

	orgID, exists := c.Get(contextkeys.OrganizationID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "organization context not found",
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

	model, err := api.modelService.GetModel(ctx, orgID.(uint), modelID)
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

// UpdateModel updates an existing model
// @Summary Update a model
// @Description Update an existing model's properties
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model_id path string true "Model public ID"
// @Param model body orgModels.ModelUpdateRequest true "Model update request"
// @Success 200 {object} ModelResponse "Updated model"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 404 {object} ErrorResponse "Model not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models/{model_id} [put]
func (api *ModelsAPI) UpdateModel(c *gin.Context) {
	ctx := c.Request.Context()

	orgID, exists := c.Get(contextkeys.OrganizationID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "organization context not found",
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

	var req orgModels.ModelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	model, err := api.modelService.UpdateModel(ctx, orgID.(uint), modelID, &req)
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
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, ModelResponse{
		Model: *model,
	})
}

// ListAllModels returns all models (managed and unmanaged) in the cluster
// @Summary List all models in cluster
// @Description Get all models in the cluster (both managed by jan-server and unmanaged)
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} AllModelsListResponse "List of all models"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models/all [get]
func (api *ModelsAPI) ListAllModels(c *gin.Context) {
	ctx := c.Request.Context()

	orgID, exists := c.Get(contextkeys.OrganizationID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "organization context not found",
		})
		return
	}

	models, err := api.modelService.ListAllModels(ctx, orgID.(uint))
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

	c.JSON(http.StatusOK, AllModelsListResponse{
		Models: models,
		Total:  len(models),
	})
}

// DeleteModel deletes a model
// @Summary Delete a model
// @Description Delete a model and its associated Kubernetes resources
// @Tags Models
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model_id path string true "Model public ID"
// @Success 204 "Model deleted successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - models API not available"
// @Failure 404 {object} ErrorResponse "Model not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/models/{model_id} [delete]
func (api *ModelsAPI) DeleteModel(c *gin.Context) {
	ctx := c.Request.Context()

	orgID, exists := c.Get(contextkeys.OrganizationID)
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "organization context not found",
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

	err := api.modelService.DeleteModel(ctx, orgID.(uint), modelID)
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
