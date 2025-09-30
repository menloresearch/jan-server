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
