package models

import (
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

type ModelCreateRequest struct {
	Name          string              `json:"name" binding:"required"`
	DisplayName   string              `json:"display_name" binding:"required"`
	Description   string              `json:"description"`
	ModelType     ModelType           `json:"model_type" binding:"required"`
	HuggingFaceID string              `json:"huggingface_id"`
	RepositoryURL string              `json:"repository_url"`
	Version       string              `json:"version"`
	Requirements  ResourceRequirement `json:"requirements" binding:"required"`
	Tags          []string            `json:"tags"`
	IsPublic      bool                `json:"is_public"`

	// Deployment configuration
	DeploymentConfig ModelDeploymentConfig `json:"deployment_config"`
}

// ModelDeploymentConfig contains Kubernetes deployment configuration
type ModelDeploymentConfig struct {
	// Container image
	Image           string `json:"image" binding:"required"`
	ImagePullPolicy string `json:"image_pull_policy"`

	// Command and arguments
	Command []string `json:"command"`
	Args    []string `json:"args"`

	// Resource configuration
	GPUCount int `json:"gpu_count"`

	// Probe configuration
	InitialDelaySeconds int `json:"initial_delay_seconds"`

	// Storage configuration
	EnablePVC    bool   `json:"enable_pvc"`
	StorageClass string `json:"storage_class,omitempty"`

	// Autoscaling configuration
	EnableAutoscaling bool                    `json:"enable_autoscaling"`
	AutoscalingConfig *ModelAutoscalingConfig `json:"autoscaling_config,omitempty"`

	// Environment variables
	ExtraEnv []EnvVar `json:"extra_env"`

	// Optional Hugging Face token for private models
	HuggingFaceToken string `json:"hugging_face_token,omitempty"`
}

// ModelAutoscalingConfig contains autoscaling configuration
type ModelAutoscalingConfig struct {
	MinReplicas    int    `json:"min_replicas"`
	MaxReplicas    int    `json:"max_replicas"`
	TargetMetric   string `json:"target_metric"`
	TargetValue    string `json:"target_value"`
	ScaleDownDelay string `json:"scale_down_delay"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ModelType represents the type of AI model
type ModelType string

const (
	ModelTypeChat       ModelType = "chat"
	ModelTypeCompletion ModelType = "completion"
	ModelTypeEmbedding  ModelType = "embedding"
	ModelTypeVision     ModelType = "vision"
)

// ModelStatus represents the current status of a model deployment
type ModelStatus string

const (
	ModelStatusPending  ModelStatus = "pending"
	ModelStatusCreating ModelStatus = "creating"
	ModelStatusRunning  ModelStatus = "running"
	ModelStatusFailed   ModelStatus = "failed"
	ModelStatusStopped  ModelStatus = "stopped"
)

// GPURequirement represents GPU requirements for a model
type GPURequirement struct {
	MinVRAM       resource.Quantity `json:"min_vram"`
	PreferredVRAM resource.Quantity `json:"preferred_vram"`
	GPUType       string            `json:"gpu_type,omitempty"` // nvidia, amd, etc.
	MinGPUs       int               `json:"min_gpus"`
	MaxGPUs       int               `json:"max_gpus"`
}

// ResourceRequirement represents compute resource requirements
type ResourceRequirement struct {
	CPU    resource.Quantity `json:"cpu"`
	Memory resource.Quantity `json:"memory"`
	GPU    *GPURequirement   `json:"gpu,omitempty"`
}

// Model represents an AI model in the organization
type Model struct {
	ID             uint        `json:"id"`
	PublicID       string      `json:"public_id"`
	OrganizationID uint        `json:"organization_id"`
	Name           string      `json:"name"`
	DisplayName    string      `json:"display_name"`
	Description    string      `json:"description"`
	ModelType      ModelType   `json:"model_type"`
	Status         ModelStatus `json:"status"`

	// Model source information
	HuggingFaceID string `json:"huggingface_id,omitempty"`
	RepositoryURL string `json:"repository_url,omitempty"`
	Version       string `json:"version"`

	// Resource requirements
	Requirements ResourceRequirement `json:"requirements"`

	// Kubernetes deployment info
	Namespace      string `json:"namespace,omitempty"`
	DeploymentName string `json:"deployment_name,omitempty"`
	ServiceName    string `json:"service_name,omitempty"`

	// API endpoint information
	EndpointURL string `json:"endpoint_url,omitempty"`
	InternalURL string `json:"internal_url,omitempty"`

	// Metadata
	Tags            []string  `json:"tags"`
	IsPublic        bool      `json:"is_public"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedByUserID uint      `json:"created_by_user_id"`
}

// ModelUpdateRequest represents a request to update an existing model
type ModelUpdateRequest struct {
	DisplayName  *string              `json:"display_name"`
	Description  *string              `json:"description"`
	Requirements *ResourceRequirement `json:"requirements"`
	Tags         []string             `json:"tags"`
	IsPublic     *bool                `json:"is_public"`
}

// ModelFilter represents filtering options for model queries
type ModelFilter struct {
	OrganizationID  *uint        `json:"organization_id"`
	ModelType       *ModelType   `json:"model_type"`
	Status          *ModelStatus `json:"status"`
	IsPublic        *bool        `json:"is_public"`
	Tags            []string     `json:"tags"`
	CreatedByUserID *uint        `json:"created_by_user_id"`
}
