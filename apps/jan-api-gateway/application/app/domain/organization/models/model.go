package models

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"menlo.ai/jan-api-gateway/app/infrastructure/kubernetes"
)

type ModelCreateRequest struct {
	Name                string   `json:"name" binding:"required"`
	DisplayName         string   `json:"display_name" binding:"required"`
	Description         string   `json:"description"`
	Image               string   `json:"image" binding:"required"`
	HuggingFaceToken    string   `json:"hugging_face_token,omitempty"`
	Command             []string `json:"command,omitempty"`
	Replicas            int      `json:"replicas"`
	GPUCount            int      `json:"gpu_count"`
	InitialDelaySeconds int      `json:"initial_delay_seconds"`
	StorageClass        string   `json:"storage_class,omitempty"`
	StorageSize         int      `json:"storage_size"`
	Tags                []string `json:"tags"`
}

// SetDefaults sets default values for the create request
func (req *ModelCreateRequest) SetDefaults() {
	if req.Replicas == 0 {
		req.Replicas = 1
	}
	if req.GPUCount == 0 {
		req.GPUCount = 1
	}
	if req.InitialDelaySeconds == 0 {
		req.InitialDelaySeconds = 60
	}
	if req.StorageSize == 0 {
		req.StorageSize = 20
	}
}

// ValidateServedModelName validates that --served-model-name in command matches the model name
// This is required for proper model identification and autoscaling
func (req *ModelCreateRequest) ValidateServedModelName() error {
	if len(req.Command) == 0 {
		return fmt.Errorf("command is required and must contain --served-model-name parameter")
	}

	// Convert entire command to a single string for searching
	fullCommand := strings.Join(req.Command, " ")

	// Look for --served-model-name parameter with the expected model name
	expectedParam := fmt.Sprintf("--served-model-name %s", req.Name)

	if strings.Contains(fullCommand, expectedParam) {
		return nil // Found and validated successfully
	}

	// Also check for --served-model-name=modelname format
	expectedParamEquals := fmt.Sprintf("--served-model-name=%s", req.Name)
	if strings.Contains(fullCommand, expectedParamEquals) {
		return nil // Found and validated successfully
	}

	// If we reach here, the expected --served-model-name was not found
	return fmt.Errorf("--served-model-name parameter must match model name '%s'. Please ensure your command contains '--served-model-name %s'", req.Name, req.Name)
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
	ModelStatusPending           ModelStatus = "pending"
	ModelStatusCreating          ModelStatus = "creating"
	ModelStatusRunning           ModelStatus = "running"
	ModelStatusFailed            ModelStatus = "failed"
	ModelStatusStopped           ModelStatus = "stopped"
	ModelStatusCrashLoopBackOff  ModelStatus = "crash_loop_back_off"
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
	ID             string      `json:"id"` // Model name from Kubernetes
	OrganizationID uint        `json:"organization_id"`
	DisplayName    string      `json:"display_name"`
	Description    string      `json:"description"`
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
	Managed         bool      `json:"managed"` // true if managed by jan-server, false if unmanaged (e.g., Aibrix)
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedByUserID string    `json:"created_by_user_id"` // User public ID (e.g. user_abc123)

	// Runtime status info (populated from Kubernetes)
	RestartCount int32  `json:"restart_count,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	LastEvent    string `json:"last_event,omitempty"`
}

// ModelFilter represents filtering options for model queries
type ModelFilter struct {
	OrganizationID  *uint        `json:"organization_id"`
	Status          *ModelStatus `json:"status"`
	Managed         *bool        `json:"managed"` // filter by managed/unmanaged models
	Tags            []string     `json:"tags"`
	CreatedByUserID *string      `json:"created_by_user_id"`
}

// KubernetesStatus represents the availability of Kubernetes APIs
type KubernetesStatus struct {
	Available bool   `json:"available"`
	InCluster bool   `json:"in_cluster"`
	Message   string `json:"message,omitempty"`
}

// ClusterStatus represents comprehensive cluster validation information
type ClusterStatus struct {
	Valid        bool                         `json:"valid"`
	Dependencies ClusterDependenciesStatus    `json:"dependencies"`
	GPUStatus    *kubernetes.ClusterGPUStatus `json:"gpu_status,omitempty"`
	Warnings     []string                     `json:"warnings,omitempty"`
	Errors       []string                     `json:"errors,omitempty"`
}

// ClusterDependenciesStatus represents the status of required dependencies
type ClusterDependenciesStatus struct {
	AibrixOperator  DependencyStatus `json:"aibrix_operator"`
	GPUOperator     DependencyStatus `json:"gpu_operator"`
	KuberayOperator DependencyStatus `json:"kuberay_operator"`
	EnvoyGateway    DependencyStatus `json:"envoy_gateway"`
	StorageClasses  DependencyStatus `json:"storage_classes"`
	Namespace       DependencyStatus `json:"namespace"`
}

// DependencyStatus represents the status of a single dependency
type DependencyStatus struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Message   string `json:"message,omitempty"`
}

// GPUResources represents comprehensive GPU resources information
type GPUResources struct {
	TotalNodes   int                       `json:"total_nodes"`
	GPUNodes     []*kubernetes.NodeGPUInfo `json:"gpu_nodes"`
	Summary      GPUResourcesSummary       `json:"summary"`
	Availability GPUAvailability           `json:"availability"`
}

// GPUResourcesSummary provides aggregate GPU information
type GPUResourcesSummary struct {
	TotalGPUs     int      `json:"total_gpus"`
	AvailableGPUs int      `json:"available_gpus"`
	GPUTypes      []string `json:"gpu_types"`
	TotalVRAM     string   `json:"total_vram"`
	AvailableVRAM string   `json:"available_vram"`
}

// GPUAvailability provides availability details per GPU type
type GPUAvailability struct {
	ByType map[string]GPUTypeAvailability `json:"by_type"`
}

// GPUTypeAvailability represents availability for a specific GPU type
type GPUTypeAvailability struct {
	Total     int    `json:"total"`
	Available int    `json:"available"`
	VRAM      string `json:"vram_per_gpu"`
}
