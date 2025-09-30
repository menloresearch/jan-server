package models

import (
	"menlo.ai/jan-api-gateway/app/domain/organization/models"
	"menlo.ai/jan-api-gateway/app/infrastructure/kubernetes"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// ClusterStatusResponse represents the cluster status response
type ClusterStatusResponse struct {
	OrganizationID uint                        `json:"organization_id"`
	ClusterStatus  kubernetes.ClusterGPUStatus `json:"cluster_status"`
}

// KubernetesStatusResponse represents Kubernetes availability status
type KubernetesStatusResponse struct {
	Available bool   `json:"available"`
	InCluster bool   `json:"in_cluster"`
	Message   string `json:"message,omitempty"`
}

// ClusterValidationResponse represents cluster dependency validation
type ClusterValidationResponse struct {
	Valid        bool                             `json:"valid"`
	Dependencies models.ClusterDependenciesStatus `json:"dependencies"`
	GPUStatus    kubernetes.ClusterGPUStatus      `json:"gpu_status"`
	Warnings     []string                         `json:"warnings,omitempty"`
	Errors       []string                         `json:"errors,omitempty"`
}

// GPUResourcesResponse represents GPU resources information
type GPUResourcesResponse struct {
	TotalNodes   int                        `json:"total_nodes"`
	GPUNodes     []kubernetes.NodeGPUInfo   `json:"gpu_nodes"`
	Summary      models.GPUResourcesSummary `json:"summary"`
	Availability models.GPUAvailability     `json:"availability"`
}

// ModelResponse represents a single model response
type ModelResponse struct {
	Model models.Model `json:"model"`
}

// ModelsListResponse represents a list of models response
type ModelsListResponse struct {
	Models []*models.Model `json:"models"`
	Total  int             `json:"total"`
}

// AllModelsListResponse represents a list of all models (managed + unmanaged) response
type AllModelsListResponse struct {
	Models []*models.ModelInfo `json:"models"`
	Total  int                 `json:"total"`
}
