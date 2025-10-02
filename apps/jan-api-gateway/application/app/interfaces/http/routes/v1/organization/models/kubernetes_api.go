package models

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/jan-api-gateway/app/domain/auth"
	orgModels "menlo.ai/jan-api-gateway/app/domain/organization/models"
	"menlo.ai/jan-api-gateway/app/infrastructure/kubernetes"
)

// KubernetesAPI handles HTTP requests for Kubernetes cluster information
type KubernetesAPI struct {
	modelService *orgModels.ModelService
	authService  *auth.AuthService
}

// NewKubernetesAPI creates a new KubernetesAPI instance
func NewKubernetesAPI(modelService *orgModels.ModelService, authService *auth.AuthService) *KubernetesAPI {
	return &KubernetesAPI{
		modelService: modelService,
		authService:  authService,
	}
}

// RegisterRouter registers the Kubernetes routes
func (api *KubernetesAPI) RegisterRouter(router gin.IRouter) {
	k8sRouter := router.Group("/kubernetes")

	k8sRouter.GET("", api.GetKubernetesStatus)
	k8sRouter.GET("/cluster-status", api.GetClusterValidation)
	k8sRouter.GET("/gpu-resources", api.GetGPUResources)
}

// GetKubernetesStatus checks if the application is running in Kubernetes
// @Summary Check Kubernetes availability
// @Description Check if the application is running in a Kubernetes cluster and has access to K8s APIs
// @Tags Kubernetes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} KubernetesStatusResponse "Kubernetes status"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/kubernetes [get]
func (api *KubernetesAPI) GetKubernetesStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Use shared cached method from ModelService
	status, err := api.modelService.GetKubernetesStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, KubernetesStatusResponse{
		Available: status.Available,
		InCluster: status.InCluster,
		Message:   status.Message,
	})
}

// GetClusterValidation validates if the cluster has all required dependencies for managed models
// @Summary Validate cluster dependencies
// @Description Check if the Kubernetes cluster has all required dependencies and configurations for managed model deployment
// @Tags Kubernetes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} ClusterValidationResponse "Cluster validation results"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 403 {object} ErrorResponse "Forbidden - Kubernetes not available"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/kubernetes/cluster-status [get]
func (api *KubernetesAPI) GetClusterValidation(c *gin.Context) {
	ctx := c.Request.Context()

	// Validate access first
	if err := api.modelService.ValidateModelAPIAccess(ctx); err != nil {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	// Get cluster status
	// Use shared cached method from ModelService
	clusterStatus, err := api.modelService.GetClusterStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get cluster status: " + err.Error(),
		})
		return
	}

	// Build validation response using the enhanced ClusterStatus
	response := &ClusterValidationResponse{
		Valid:        clusterStatus.Valid,
		Dependencies: clusterStatus.Dependencies,
		Warnings:     clusterStatus.Warnings,
		Errors:       clusterStatus.Errors,
	}

	// Include GPU status if available
	if clusterStatus.GPUStatus != nil {
		response.GPUStatus = *clusterStatus.GPUStatus
	}

	c.JSON(http.StatusOK, response)
}

// GetGPUResources returns detailed GPU resource information
// @Summary Get GPU resources information
// @Description Get detailed information about GPU resources in the Kubernetes cluster including availability and usage
// @Tags Kubernetes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} GPUResourcesResponse "GPU resources information"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /v1/organization/kubernetes/gpu-resources [get]
func (api *KubernetesAPI) GetGPUResources(c *gin.Context) {
	ctx := c.Request.Context()

	// Use shared cached method from ModelService
	gpuResources, err := api.modelService.GetGPUResources(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get GPU resources: " + err.Error(),
		})
		return
	}

	// Convert to response format
	response := &GPUResourcesResponse{
		TotalNodes:   gpuResources.TotalNodes,
		GPUNodes:     make([]kubernetes.NodeGPUInfo, len(gpuResources.GPUNodes)),
		Summary:      gpuResources.Summary,
		Availability: gpuResources.Availability,
	}

	// Convert pointer slice to value slice
	for i, node := range gpuResources.GPUNodes {
		response.GPUNodes[i] = *node
	}

	c.JSON(http.StatusOK, response)
}
