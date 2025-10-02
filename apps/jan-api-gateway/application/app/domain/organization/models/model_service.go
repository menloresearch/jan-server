package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/infrastructure/kubernetes"
)

// Constants for managed model identification
const (
	// Standard Kubernetes label for managed resources
	ManagedByLabelKey   = "app.kubernetes.io/managed-by"
	ManagedByLabelValue = "jan-server"

	// Jan-specific labels
	ModelNameLabelKey    = "model.aibrix.ai/name"
	OrganizationLabelKey = "jan-server.menlo.ai/organization"
	ModelTypeLabelKey    = "jan-server.menlo.ai/model-type"

	DefaultNamespace = "jan-models"
	DefaultPVCName   = "hf-hub-cache"

	// Redis cache keys and TTL
	ModelCacheKeyPrefix = "jan:models:"
	ClusterCacheKey     = "jan:cluster:status"
	CacheTTL            = 300 // 5 minutes
)

// ModelService provides business logic for managing organization models
type ModelService struct {
	k8sService        *kubernetes.KubernetesService
	deploymentManager *kubernetes.ModelDeploymentManager
	cache             *cache.RedisCacheService
}

// NewModelService creates a new ModelService instance
func NewModelService(k8sService *kubernetes.KubernetesService, cacheService *cache.RedisCacheService) (*ModelService, error) {
	deploymentManager, err := k8sService.NewModelDeploymentManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment manager: %w", err)
	}

	return &ModelService{
		k8sService:        k8sService,
		deploymentManager: deploymentManager,
		cache:             cacheService,
	}, nil
}

// getCacheKey generates a cache key for model data
func (s *ModelService) getCacheKey(prefix, identifier string) string {
	return fmt.Sprintf("%s%s:%s", ModelCacheKeyPrefix, prefix, identifier)
}

// cacheModelData stores model data in Redis with TTL
func (s *ModelService) cacheModelData(ctx context.Context, key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal model data: %w", err)
	}

	return s.cache.Set(ctx, key, string(jsonData), time.Duration(CacheTTL)*time.Second)
}

// getCachedModelData retrieves and unmarshals model data from Redis
func (s *ModelService) getCachedModelData(ctx context.Context, key string, dest interface{}) error {
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// invalidateModelCache removes cached model data
func (s *ModelService) invalidateModelCache(ctx context.Context, orgID uint, modelName string) error {
	// Remove specific model cache
	modelKey := s.getCacheKey("model", fmt.Sprintf("%d:%s", orgID, modelName))
	_ = s.cache.Delete(ctx, modelKey)

	// Remove organization models list cache
	orgKey := s.getCacheKey("org", fmt.Sprintf("%d", orgID))
	_ = s.cache.Delete(ctx, orgKey)

	// Remove all models cache
	allKey := s.getCacheKey("all", "models")
	_ = s.cache.Delete(ctx, allKey)

	return nil
}

// ValidateModelAPIAccess checks if the models API is available
func (s *ModelService) ValidateModelAPIAccess(ctx context.Context) error {
	if s.k8sService == nil {
		return fmt.Errorf("models API only available when running in Kubernetes cluster")
	}

	return s.k8sService.ValidateModelDeploymentRequirements(ctx)
}

// GetKubernetesStatus returns cached Kubernetes availability status
func (s *ModelService) GetKubernetesStatus(ctx context.Context) (*KubernetesStatus, error) {
	// Try cache first
	cacheKey := s.getCacheKey("k8s", "status")
	var cachedStatus KubernetesStatus
	if err := s.getCachedModelData(ctx, cacheKey, &cachedStatus); err == nil {
		return &cachedStatus, nil
	}

	// Check if we can access K8s
	err := s.ValidateModelAPIAccess(ctx)

	status := &KubernetesStatus{
		Available: err == nil,
		InCluster: s.k8sService != nil,
	}

	if err != nil {
		status.Message = err.Error()
	} else {
		status.Message = "Kubernetes API accessible"
	}

	// Cache for 2 minutes (shorter than other data as this can change)
	_ = s.cacheModelData(ctx, cacheKey, status)

	return status, nil
}

// GetClusterStatus returns cached cluster validation status with enhanced information
func (s *ModelService) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	// Try cache first
	cacheKey := s.getCacheKey("k8s", "cluster_status")
	var cachedStatus ClusterStatus
	if err := s.getCachedModelData(ctx, cacheKey, &cachedStatus); err == nil {
		return &cachedStatus, nil
	}

	// Get basic GPU status from existing method
	gpuStatus, err := s.k8sService.GetClusterGPUStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster GPU status: %w", err)
	}

	// TODO: Add dependency validation when available
	// For now, create basic validation based on GPU status
	status := &ClusterStatus{
		Valid:     true, // Assume valid if we can get GPU status
		GPUStatus: gpuStatus,
		Dependencies: ClusterDependenciesStatus{
			// TODO: Implement actual dependency checks
			AibrixOperator:  DependencyStatus{Available: true, Message: "Not validated"},
			GPUOperator:     DependencyStatus{Available: true, Message: "Not validated"},
			KuberayOperator: DependencyStatus{Available: true, Message: "Not validated"},
			EnvoyGateway:    DependencyStatus{Available: true, Message: "Not validated"},
			StorageClasses:  DependencyStatus{Available: true, Message: "Not validated"},
			Namespace:       DependencyStatus{Available: true, Message: "Not validated"},
		},
		Warnings: []string{},
		Errors:   []string{},
	}

	// Cache for 5 minutes
	_ = s.cacheModelData(ctx, cacheKey, status)

	return status, nil
}

// GetGPUResources returns cached GPU resources information based on cluster status
func (s *ModelService) GetGPUResources(ctx context.Context) (*GPUResources, error) {
	// Try cache first
	cacheKey := s.getCacheKey("k8s", "gpu_resources")
	var cachedResources GPUResources
	if err := s.getCachedModelData(ctx, cacheKey, &cachedResources); err == nil {
		return &cachedResources, nil
	}

	// Get GPU status from cluster
	clusterGPUStatus, err := s.k8sService.GetClusterGPUStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster GPU status: %w", err)
	}

	// Convert ClusterGPUStatus to GPUResources format
	resources := &GPUResources{
		TotalNodes: clusterGPUStatus.TotalNodes,
		GPUNodes:   make([]*kubernetes.NodeGPUInfo, len(clusterGPUStatus.GPUNodes)),
		Summary: GPUResourcesSummary{
			TotalGPUs:     clusterGPUStatus.TotalGPUs,
			AvailableGPUs: clusterGPUStatus.TotalGPUs, // Assume all available for now
			GPUTypes:      []string{},
			TotalVRAM:     "Unknown",
			AvailableVRAM: "Unknown",
		},
		Availability: GPUAvailability{
			ByType: make(map[string]GPUTypeAvailability),
		},
	}

	// Copy GPU nodes and extract type information
	gpuTypeMap := make(map[string]int)
	gpuTypeVRAMMap := make(map[string]resource.Quantity)
	var totalVRAM resource.Quantity

	for i, node := range clusterGPUStatus.GPUNodes {
		resources.GPUNodes[i] = &node

		// Extract GPU type information
		if node.GPUType != "" {
			gpuTypeMap[node.GPUType] += node.GPUCount

			// Track VRAM per GPU type (assume all GPUs of same type have same VRAM)
			if !node.TotalVRAM.IsZero() {
				gpuTypeVRAMMap[node.GPUType] = node.TotalVRAM
				// Add to total VRAM (multiply by GPU count on this node)
				nodeVRAM := node.TotalVRAM.DeepCopy()
				nodeVRAM.Set(nodeVRAM.Value() * int64(node.GPUCount))
				totalVRAM.Add(nodeVRAM)
			}
		}
	}

	// Update summary with calculated VRAM
	if !totalVRAM.IsZero() {
		resources.Summary.TotalVRAM = totalVRAM.String()
		resources.Summary.AvailableVRAM = totalVRAM.String() // Assume all available for now
	}

	// Extract unique GPU types
	for gpuType := range gpuTypeMap {
		resources.Summary.GPUTypes = append(resources.Summary.GPUTypes, gpuType)
	}

	// Build availability map with VRAM information
	for gpuType, total := range gpuTypeMap {
		vramPerGPU := "Unknown"
		if vram, exists := gpuTypeVRAMMap[gpuType]; exists && !vram.IsZero() {
			vramPerGPU = vram.String()
		}

		resources.Availability.ByType[gpuType] = GPUTypeAvailability{
			Total:     total,
			Available: total, // Assume all available for now
			VRAM:      vramPerGPU,
		}
	}

	// Cache for 3 minutes (GPU status can change fairly quickly)
	_ = s.cacheModelData(ctx, cacheKey, resources)

	return resources, nil
}

// GetModels returns all models for an organization with optional filtering
func (s *ModelService) GetModels(ctx context.Context, orgID uint, filter *ModelFilter) ([]*Model, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	// Try cache first
	cacheKey := s.getCacheKey("org", fmt.Sprintf("%d", orgID))
	var cachedModels []*Model
	if err := s.getCachedModelData(ctx, cacheKey, &cachedModels); err == nil {
		return s.applyFilter(cachedModels, filter), nil
	}

	// Get models from Kubernetes
	models, err := s.getModelsFromKubernetes(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get models from Kubernetes: %w", err)
	}

	// Cache the results
	_ = s.cacheModelData(ctx, cacheKey, models)

	return s.applyFilter(models, filter), nil
}

// GetModel returns a specific model by public ID
func (s *ModelService) GetModel(ctx context.Context, orgID uint, modelID string) (*Model, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	// Try cache first
	cacheKey := s.getCacheKey("model", fmt.Sprintf("%d:%s", orgID, modelID))
	var cachedModel Model
	if err := s.getCachedModelData(ctx, cacheKey, &cachedModel); err == nil {
		return &cachedModel, nil
	}

	// Get from Kubernetes
	model, err := s.getModelFromKubernetes(ctx, orgID, modelID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	_ = s.cacheModelData(ctx, cacheKey, model)

	return model, nil
}

// CreateModel creates a new model for an organization
func (s *ModelService) CreateModel(ctx context.Context, orgID uint, userID string, req *ModelCreateRequest) (*Model, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	// Set defaults
	req.SetDefaults()

	// Validate served-model-name consistency (required for proper model identification)
	if err := req.ValidateServedModelName(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Build resource requirements from simplified request
	resourceReq := ResourceRequirement{
		CPU:    resource.MustParse("1"),
		Memory: resource.MustParse("2Gi"),
	}
	if req.GPUCount > 0 {
		resourceReq.GPU = &GPURequirement{
			MinVRAM:       resource.MustParse("8Gi"),
			PreferredVRAM: resource.MustParse("16Gi"),
			GPUType:       "nvidia",
			MinGPUs:       req.GPUCount,
			MaxGPUs:       req.GPUCount,
		}
	}

	model := &Model{
		ID:              req.Name, // Use model name as ID
		OrganizationID:  orgID,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		Status:          ModelStatusPending,
		Requirements:    resourceReq,
		Namespace:       DefaultNamespace,
		DeploymentName:  req.Name,
		ServiceName:     req.Name,
		Tags:            req.Tags,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		CreatedByUserID: userID,
		Managed:         true,
	}

	// Deploy to Kubernetes with simplified config
	if err := s.deployModelToKubernetes(ctx, model, req); err != nil {
		return nil, fmt.Errorf("failed to deploy model to Kubernetes: %w", err)
	}

	// Update model status
	model.Status = ModelStatusCreating

	// Invalidate cache
	_ = s.invalidateModelCache(ctx, orgID, req.Name)

	return model, nil
}

// DeleteModel removes a model and its Kubernetes resources
func (s *ModelService) DeleteModel(ctx context.Context, orgID uint, modelID string) error {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return err
	}

	model, err := s.GetModel(ctx, orgID, modelID)
	if err != nil {
		return err
	}

	// Delete from Kubernetes
	if err := s.deploymentManager.DeleteModelDeployment(ctx, model.ID, DefaultNamespace); err != nil {
		return fmt.Errorf("failed to delete model from Kubernetes: %w", err)
	}

	// Invalidate cache
	_ = s.invalidateModelCache(ctx, orgID, model.ID)

	return nil
}

// getModelsFromKubernetes retrieves models from Kubernetes API
func (s *ModelService) getModelsFromKubernetes(ctx context.Context, orgID uint) ([]*Model, error) {
	// Get all deployments with our managed label
	models, err := s.deploymentManager.GetManagedModels(ctx, DefaultNamespace, ManagedByLabelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed deployments: %w", err)
	}

	var orgModels []*Model
	for _, modelInfo := range models {
		// Filter by organization (check label)
		if orgLabel, exists := modelInfo.Labels[OrganizationLabelKey]; exists && orgLabel == fmt.Sprintf("%d", orgID) {
			model := s.convertModelInfoToModel(modelInfo, orgID)
			if model != nil {
				orgModels = append(orgModels, model)
			}
		}
	}

	return orgModels, nil
}

// getModelFromKubernetes retrieves a specific model from Kubernetes
func (s *ModelService) getModelFromKubernetes(ctx context.Context, orgID uint, modelID string) (*Model, error) {
	// First get all models and find the one with matching ID
	models, err := s.getModelsFromKubernetes(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if model.ID == modelID {
			return model, nil
		}
	}

	return nil, fmt.Errorf("model not found")
}

// convertModelInfoToModel converts a Kubernetes ModelInfo to a Model
func (s *ModelService) convertModelInfoToModel(modelInfo *kubernetes.ModelInfo, orgID uint) *Model {
	// Extract information from labels
	displayName := modelInfo.Labels["display-name"]
	if displayName == "" {
		displayName = modelInfo.Name
	}

	description := modelInfo.Labels["description"]

	// For unmanaged models, don't use database fields
	var createdByUserID string

	if modelInfo.IsManaged {
		// For managed models, createdByUserID would be set from database
	} else {
		// Leave createdByUserID as empty for unmanaged models
	}

	// Determine status from Kubernetes status with enhanced error handling
	status := ModelStatusPending
	switch modelInfo.Status {
	case "Running":
		status = ModelStatusRunning
	case "Starting":
		status = ModelStatusCreating
	case "CrashLoopBackOff":
		if modelInfo.RestartCount >= 3 {
			status = ModelStatusCrashLoopBackOff
		} else {
			status = ModelStatusPending
		}
	default:
		status = ModelStatusPending
	}

	// Extract actual resource requirements from deployment
	requirements := s.extractResourceRequirements(modelInfo)

	return &Model{
		ID:              modelInfo.Name, // Use model name from Kubernetes as ID
		OrganizationID:  orgID,
		DisplayName:     displayName,
		Description:     description,
		Status:          status,
		Managed:         modelInfo.IsManaged,
		Namespace:       modelInfo.Namespace,
		DeploymentName:  modelInfo.Name,
		ServiceName:     modelInfo.Name,
		Requirements:    requirements,
		RestartCount:    modelInfo.RestartCount,
		ErrorMessage:    modelInfo.ErrorMessage,
		LastEvent:       modelInfo.LastEvent,
		CreatedAt:       modelInfo.CreatedAt,
		UpdatedAt:       modelInfo.CreatedAt,
		CreatedByUserID: createdByUserID, // 0 for unmanaged models
	}
}

// extractResourceRequirements extracts actual resource requirements from Kubernetes ModelInfo
func (s *ModelService) extractResourceRequirements(modelInfo *kubernetes.ModelInfo) ResourceRequirement {
	// TODO: This should be implemented to extract actual resources from the deployment
	// For now, return empty/zero values - this is where we would:
	// 1. Query the deployment's resource requests/limits
	// 2. Detect GPU usage and type
	// 3. Get CPU/Memory requirements
	// 4. Determine which node it's running on and GPU details

	return ResourceRequirement{
		CPU:    resource.MustParse("0"),
		Memory: resource.MustParse("0"),
		GPU:    nil, // Will be populated with actual GPU info
	}
}

// deployModelToKubernetes deploys a model to Kubernetes with simplified configuration
func (s *ModelService) deployModelToKubernetes(ctx context.Context, model *Model, req *ModelCreateRequest) error {
	// Parse CPU and Memory from requirements
	cpuRequest, err := resource.ParseQuantity(model.Requirements.CPU.String())
	if err != nil {
		return fmt.Errorf("invalid CPU request: %w", err)
	}

	memoryRequest, err := resource.ParseQuantity(model.Requirements.Memory.String())
	if err != nil {
		return fmt.Errorf("invalid memory request: %w", err)
	}

	// Build environment variables for Hugging Face
	var envVars []corev1.EnvVar
	if req.HuggingFaceToken != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "HF_TOKEN",
			Value: req.HuggingFaceToken,
		})
	}

	// Create deployment spec
	spec := &kubernetes.ModelDeploymentSpec{
		Name:            model.ID,
		Namespace:       model.Namespace,
		Image:           req.Image,
		ImagePullPolicy: "IfNotPresent",
		Command:         req.Command,
		Port:            8000,
		CPURequest:      cpuRequest,
		MemoryRequest:   memoryRequest,
		// No CPU/Memory limits - allow full node utilization for GPU workloads
		GPUCount:            req.GPUCount,
		InitialDelaySeconds: int32(req.InitialDelaySeconds),
		EnablePVC:           req.StorageSize > 0,                  // Enable PVC if storage size specified
		PVCName:             fmt.Sprintf("%s-storage", model.ID),  // Each model has its own PVC
		PVCSize:             fmt.Sprintf("%dGi", req.StorageSize), // Convert to Gi format
		ExtraEnv:            envVars,
		ManagedLabels: map[string]string{
			ManagedByLabelKey:    ManagedByLabelValue,
			ModelNameLabelKey:    model.ID,
			OrganizationLabelKey: fmt.Sprintf("%d", model.OrganizationID),
		},
	}

	// Set storage class only if provided (let Kubernetes use default if empty)
	if req.StorageClass != "" {
		spec.StorageClass = req.StorageClass
	}

	// Deploy using deployment manager
	if err := s.deploymentManager.CreateModelDeployment(ctx, spec); err != nil {
		return fmt.Errorf("failed to deploy model: %w", err)
	}

	return nil
}

// applyFilter applies filtering to model list
func (s *ModelService) applyFilter(models []*Model, filter *ModelFilter) []*Model {
	if filter == nil {
		return models
	}

	var filtered []*Model
	for _, model := range models {
		if s.matchesFilter(model, filter) {
			filtered = append(filtered, model)
		}
	}

	return filtered
}

// matchesFilter checks if a model matches the given filter
func (s *ModelService) matchesFilter(model *Model, filter *ModelFilter) bool {
	if filter.Status != nil && model.Status != *filter.Status {
		return false
	}
	if filter.CreatedByUserID != nil && model.CreatedByUserID != *filter.CreatedByUserID {
		return false
	}
	// TODO: Implement tag filtering if needed
	return true
}

// ListAllModels returns all models in the cluster (managed and unmanaged)
func (s *ModelService) ListAllModels(ctx context.Context, orgID uint) ([]*ModelInfo, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	// Try cache first
	cacheKey := s.getCacheKey("all", "models")
	var cachedModels []*ModelInfo
	if err := s.getCachedModelData(ctx, cacheKey, &cachedModels); err == nil {
		return cachedModels, nil
	}

	// Get all deployments from Kubernetes (managed and unmanaged) across all namespaces
	allModels, err := s.deploymentManager.GetAllModels(ctx, "", ManagedByLabelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get all models from Kubernetes: %w", err)
	}

	// Cache the results
	models := convertFromKubernetesModelInfoToPointers(allModels)
	_ = s.cacheModelData(ctx, cacheKey, models)

	return models, nil
}

// GetClusterGPUStatus returns the current cluster GPU status (legacy method for compatibility)
func (s *ModelService) GetClusterGPUStatus(ctx context.Context) (*kubernetes.ClusterGPUStatus, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	return s.k8sService.GetClusterGPUStatus(ctx)
}

// ListModels returns models for an organization with filtering and caching
func (s *ModelService) ListModels(ctx context.Context, orgID uint, filter *ModelFilter) ([]*Model, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	// Generate cache key based on filter
	cacheKey := s.getFilteredCacheKey(orgID, filter)

	// Try cache first
	var cachedModels []*Model
	if err := s.getCachedModelData(ctx, cacheKey, &cachedModels); err == nil {
		return cachedModels, nil
	}

	// Get all models from Kubernetes (managed and unmanaged) across all namespaces
	allModelInfos, err := s.deploymentManager.GetAllModels(ctx, "", ManagedByLabelKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get models from Kubernetes: %w", err)
	}

	// Convert ModelInfo to Model and apply filters
	var models []*Model
	for _, modelInfo := range allModelInfos {
		// Convert kubernetes.ModelInfo to Model (using existing method)
		model := s.convertModelInfoToModel(modelInfo, orgID)

		// Apply filters
		if s.shouldIncludeModel(model, filter) {
			models = append(models, model)
		}
	}

	// Cache the filtered results
	_ = s.cacheModelData(ctx, cacheKey, models)

	return models, nil
}

// ModelInfo represents basic model information for listing
type ModelInfo struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	IsManaged    bool      `json:"is_managed"`
	Status       string    `json:"status"`
	Replicas     int32     `json:"replicas"`
	CreatedAt    time.Time `json:"created_at"`
	DisplayName  string    `json:"display_name,omitempty"`
	Description  string    `json:"description,omitempty"`
	RestartCount int32     `json:"restart_count,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	LastEvent    string    `json:"last_event,omitempty"`
}

// Helper functions

// convertFromKubernetesModelInfoToPointers converts Kubernetes ModelInfo to domain ModelInfo pointers
func convertFromKubernetesModelInfoToPointers(k8sModels []*kubernetes.ModelInfo) []*ModelInfo {
	var models []*ModelInfo
	for _, k8sModel := range k8sModels {
		model := &ModelInfo{
			Name:         k8sModel.Name,
			Namespace:    k8sModel.Namespace,
			IsManaged:    k8sModel.IsManaged,
			Status:       k8sModel.Status,
			Replicas:     k8sModel.Replicas,
			CreatedAt:    k8sModel.CreatedAt,
			DisplayName:  k8sModel.Labels["display-name"],
			Description:  k8sModel.Labels["description"],
			RestartCount: k8sModel.RestartCount,
			ErrorMessage: k8sModel.ErrorMessage,
			LastEvent:    k8sModel.LastEvent,
		}
		models = append(models, model)
	}
	return models
}

// getFilteredCacheKey generates a cache key based on organization and filter parameters
func (s *ModelService) getFilteredCacheKey(orgID uint, filter *ModelFilter) string {
	key := fmt.Sprintf("%sorg:%d", ModelCacheKeyPrefix, orgID)

	if filter == nil {
		return key + ":all"
	}

	if filter.Status != nil {
		key += fmt.Sprintf(":status:%s", *filter.Status)
	}
	if filter.Managed != nil {
		key += fmt.Sprintf(":managed:%t", *filter.Managed)
	}

	return key
}

// shouldIncludeModel checks if a model should be included based on filter criteria
func (s *ModelService) shouldIncludeModel(model *Model, filter *ModelFilter) bool {
	if filter == nil {
		return true
	}

	if filter.Status != nil && model.Status != *filter.Status {
		return false
	}

	if filter.Managed != nil && model.Managed != *filter.Managed {
		return false
	}

	return true
}
