package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"menlo.ai/jan-api-gateway/app/infrastructure/cache"
	"menlo.ai/jan-api-gateway/app/infrastructure/kubernetes"
	"menlo.ai/jan-api-gateway/app/utils/idgen"
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

// createPublicID generates a unique public ID for a model
func (s *ModelService) createPublicID() (string, error) {
	return idgen.GenerateSecureID("mdl", 16)
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
func (s *ModelService) CreateModel(ctx context.Context, orgID uint, userID uint, req *ModelCreateRequest) (*Model, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	// Validate resource requirements against cluster capabilities
	if err := s.validateResourceRequirements(ctx, &req.Requirements); err != nil {
		return nil, fmt.Errorf("resource requirements validation failed: %w", err)
	}

	// Validate deployment configuration
	if err := s.validateDeploymentConfig(ctx, &req.DeploymentConfig); err != nil {
		return nil, fmt.Errorf("deployment configuration validation failed: %w", err)
	}

	// Validate served-model-name consistency (required by Aibrix)
	if err := s.validateServedModelName(req.Name, req.DeploymentConfig.Args); err != nil {
		return nil, fmt.Errorf("served-model-name validation failed: %w", err)
	}

	publicID, err := s.createPublicID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate public ID: %w", err)
	}

	// Set defaults for deployment config
	s.setDeploymentDefaults(&req.DeploymentConfig)

	model := &Model{
		PublicID:        publicID,
		OrganizationID:  orgID,
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		ModelType:       req.ModelType,
		Status:          ModelStatusPending,
		HuggingFaceID:   req.HuggingFaceID,
		Requirements:    req.Requirements,
		Namespace:       DefaultNamespace,
		DeploymentName:  req.Name,
		ServiceName:     req.Name,
		Tags:            req.Tags,
		IsPublic:        req.IsPublic,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		CreatedByUserID: userID,
	}

	// Deploy to Kubernetes
	if err := s.deployToKubernetes(ctx, model, &req.DeploymentConfig); err != nil {
		return nil, fmt.Errorf("failed to deploy model to Kubernetes: %w", err)
	}

	// Update model status
	model.Status = ModelStatusCreating

	// Invalidate cache
	_ = s.invalidateModelCache(ctx, orgID, req.Name)

	return model, nil
}

// UpdateModel updates an existing model
func (s *ModelService) UpdateModel(ctx context.Context, orgID uint, modelID string, req *ModelUpdateRequest) (*Model, error) {
	if err := s.ValidateModelAPIAccess(ctx); err != nil {
		return nil, err
	}

	model, err := s.GetModel(ctx, orgID, modelID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.DisplayName != nil {
		model.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		model.Description = *req.Description
	}
	if req.Requirements != nil {
		if err := s.validateResourceRequirements(ctx, req.Requirements); err != nil {
			return nil, fmt.Errorf("resource requirements validation failed: %w", err)
		}
		model.Requirements = *req.Requirements
	}
	if req.Tags != nil {
		model.Tags = req.Tags
	}
	if req.IsPublic != nil {
		model.IsPublic = *req.IsPublic
	}

	model.UpdatedAt = time.Now()

	// Update Kubernetes labels if needed
	if err := s.updateKubernetesLabels(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to update Kubernetes labels: %w", err)
	}

	// Invalidate cache
	_ = s.invalidateModelCache(ctx, orgID, model.Name)

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
	if err := s.deploymentManager.DeleteModelDeployment(ctx, model.Name, DefaultNamespace); err != nil {
		return fmt.Errorf("failed to delete model from Kubernetes: %w", err)
	}

	// Invalidate cache
	_ = s.invalidateModelCache(ctx, orgID, model.Name)

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
	// First get all models and find the one with matching public ID
	models, err := s.getModelsFromKubernetes(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if model.PublicID == modelID {
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
	modelTypeStr := modelInfo.Labels[ModelTypeLabelKey]
	publicID := modelInfo.Labels["public-id"]

	// Default values if not found in labels
	if publicID == "" {
		publicID = fmt.Sprintf("mdl_%s", modelInfo.Name)
	}

	modelType := ModelTypeChat
	if modelTypeStr != "" {
		modelType = ModelType(modelTypeStr)
	}

	// Determine status from Kubernetes status
	status := ModelStatusPending
	switch modelInfo.Status {
	case "Running":
		status = ModelStatusRunning
	case "Starting":
		status = ModelStatusCreating
	default:
		status = ModelStatusPending
	}

	return &Model{
		PublicID:       publicID,
		OrganizationID: orgID,
		Name:           modelInfo.Name,
		DisplayName:    displayName,
		Description:    description,
		ModelType:      modelType,
		Status:         status,
		Namespace:      modelInfo.Namespace,
		DeploymentName: modelInfo.Name,
		ServiceName:    modelInfo.Name,
		CreatedAt:      modelInfo.CreatedAt,
		UpdatedAt:      modelInfo.CreatedAt, // Use creation time as fallback
		// TODO: Extract more fields from labels/annotations if needed
	}
}

// deployToKubernetes deploys a model to Kubernetes
func (s *ModelService) deployToKubernetes(ctx context.Context, model *Model, config *ModelDeploymentConfig) error {
	// Parse CPU and Memory from string to resource.Quantity
	cpuRequest, err := resource.ParseQuantity(model.Requirements.CPU.String())
	if err != nil {
		return fmt.Errorf("invalid CPU request: %w", err)
	}

	memoryRequest, err := resource.ParseQuantity(model.Requirements.Memory.String())
	if err != nil {
		return fmt.Errorf("invalid memory request: %w", err)
	}

	// Create deployment spec
	spec := &kubernetes.ModelDeploymentSpec{
		Name:                model.Name,
		Namespace:           model.Namespace,
		Image:               config.Image,
		ImagePullPolicy:     config.ImagePullPolicy,
		Command:             config.Command,
		Args:                config.Args,
		Port:                8000,
		CPURequest:          cpuRequest,
		MemoryRequest:       memoryRequest,
		GPUCount:            config.GPUCount,
		InitialDelaySeconds: int32(config.InitialDelaySeconds),
		EnablePVC:           config.EnablePVC,
		StorageClass:        config.StorageClass,
		EnableAutoscaling:   config.EnableAutoscaling,
		ExtraEnv:            convertEnvVars(config.ExtraEnv),
		ManagedLabels: map[string]string{
			ManagedByLabelKey:    ManagedByLabelValue,
			ModelNameLabelKey:    model.Name,
			OrganizationLabelKey: fmt.Sprintf("%d", model.OrganizationID),
			ModelTypeLabelKey:    string(model.ModelType),
		},
	}

	if config.AutoscalingConfig != nil {
		spec.AutoscalingConfig = &kubernetes.ModelAutoscalingConfig{
			MinReplicas:    int32(config.AutoscalingConfig.MinReplicas),
			MaxReplicas:    int32(config.AutoscalingConfig.MaxReplicas),
			TargetMetric:   config.AutoscalingConfig.TargetMetric,
			TargetValue:    config.AutoscalingConfig.TargetValue,
			ScaleDownDelay: config.AutoscalingConfig.ScaleDownDelay,
		}
	}

	return s.deploymentManager.CreateModelDeployment(ctx, spec)
}

// updateKubernetesLabels updates labels on Kubernetes resources
func (s *ModelService) updateKubernetesLabels(ctx context.Context, model *Model) error {
	// TODO: Implement label updates on K8s resources
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
	if filter.ModelType != nil && model.ModelType != *filter.ModelType {
		return false
	}
	if filter.Status != nil && model.Status != *filter.Status {
		return false
	}
	if filter.IsPublic != nil && model.IsPublic != *filter.IsPublic {
		return false
	}
	if filter.CreatedByUserID != nil && model.CreatedByUserID != *filter.CreatedByUserID {
		return false
	}
	// TODO: Implement tag filtering if needed
	return true
}

// validateResourceRequirements validates resource requirements against cluster capabilities
func (s *ModelService) validateResourceRequirements(ctx context.Context, requirements *ResourceRequirement) error {
	// Basic validation
	if requirements.CPU.String() == "" {
		return fmt.Errorf("CPU requirement is required")
	}
	if requirements.Memory.String() == "" {
		return fmt.Errorf("memory requirement is required")
	}

	// Enhanced cluster capacity validation using cached data
	return s.validateResourcesAgainstCluster(ctx, requirements)
}

// validateResourcesAgainstCluster validates resources against actual cluster capacity
func (s *ModelService) validateResourcesAgainstCluster(ctx context.Context, requirements *ResourceRequirement) error {
	// Get cached GPU resources to validate GPU requirements
	if requirements.GPU != nil && requirements.GPU.MinGPUs > 0 {
		gpuResources, err := s.GetGPUResources(ctx)
		if err != nil {
			// If we can't get GPU resources, warn but don't fail
			// (deployment might still work if GPU resources become available)
			return nil
		}

		// Check if cluster has enough GPUs
		if gpuResources.Summary.AvailableGPUs < requirements.GPU.MinGPUs {
			return fmt.Errorf("insufficient GPU resources: requested minimum %d GPUs but only %d available in cluster",
				requirements.GPU.MinGPUs, gpuResources.Summary.AvailableGPUs)
		}

		// Check if cluster has any GPUs at all
		if gpuResources.Summary.TotalGPUs == 0 {
			return fmt.Errorf("no GPU resources found in cluster but %d minimum GPUs requested", requirements.GPU.MinGPUs)
		}

		// Validate GPU type if specified
		if requirements.GPU.GPUType != "" {
			typeAvailable := false
			for _, availableType := range gpuResources.Summary.GPUTypes {
				if availableType == requirements.GPU.GPUType {
					typeAvailable = true
					break
				}
			}
			if !typeAvailable {
				return fmt.Errorf("requested GPU type '%s' not available in cluster. Available types: %v",
					requirements.GPU.GPUType, gpuResources.Summary.GPUTypes)
			}

			// Check available GPUs of the specific type
			if typeAvail, exists := gpuResources.Availability.ByType[requirements.GPU.GPUType]; exists {
				if typeAvail.Available < requirements.GPU.MinGPUs {
					return fmt.Errorf("insufficient '%s' GPUs: requested minimum %d but only %d available",
						requirements.GPU.GPUType, requirements.GPU.MinGPUs, typeAvail.Available)
				}
			}
		}

		// TODO: Add VRAM validation when GPU nodes provide detailed VRAM info
		// Currently NodeGPUInfo has TotalVRAM and AvailableVRAM but it's at node level
	}

	// Get cached cluster status to validate general capacity
	clusterStatus, err := s.GetClusterStatus(ctx)
	if err != nil {
		// If we can't get cluster status, allow deployment (it will fail at K8s level if invalid)
		return nil
	}

	// Check if cluster dependencies are valid for model deployment
	if !clusterStatus.Valid {
		warnings := []string{}
		for _, warning := range clusterStatus.Warnings {
			warnings = append(warnings, warning)
		}
		for _, error := range clusterStatus.Errors {
			warnings = append(warnings, error)
		}

		if len(warnings) > 0 {
			// Log warnings but don't fail - allow deployment attempt
			// In production, you might want to fail here depending on criticality
		}
	}

	return nil
}

// validateDeploymentConfig validates deployment configuration
func (s *ModelService) validateDeploymentConfig(ctx context.Context, config *ModelDeploymentConfig) error {
	if config.Image == "" {
		return fmt.Errorf("container image is required")
	}

	if len(config.Command) == 0 {
		return fmt.Errorf("container command is required")
	}

	if config.GPUCount < 0 {
		return fmt.Errorf("GPU count cannot be negative")
	}

	if config.InitialDelaySeconds < 0 {
		return fmt.Errorf("initial delay seconds cannot be negative")
	}

	// Validate storage class if PVC is enabled
	if config.EnablePVC {
		if config.StorageClass == "" {
			// Try to get default storage class
			defaultSC, err := s.k8sService.GetDefaultStorageClass(ctx)
			if err != nil {
				return fmt.Errorf("PVC enabled but no storage class specified and no default found: %w", err)
			}
			config.StorageClass = defaultSC
		}
	}

	return nil
}

// validateServedModelName validates that --served-model-name in args matches the model name
// This is required by Aibrix for proper model identification and autoscaling
func (s *ModelService) validateServedModelName(modelName string, args []string) error {
	// Look for --served-model-name parameter in args
	for i, arg := range args {
		if arg == "--served-model-name" {
			// Check if there's a next argument
			if i+1 >= len(args) {
				return fmt.Errorf("--served-model-name flag found but no value provided")
			}

			servedModelName := args[i+1]
			if servedModelName != modelName {
				return fmt.Errorf("--served-model-name '%s' must match model name '%s' (required by Aibrix for autoscaling)", servedModelName, modelName)
			}

			return nil // Found and validated
		}

		// Also check for combined format like --served-model-name=model-name
		if strings.HasPrefix(arg, "--served-model-name=") {
			servedModelName := strings.TrimPrefix(arg, "--served-model-name=")
			if servedModelName != modelName {
				return fmt.Errorf("--served-model-name '%s' must match model name '%s' (required by Aibrix for autoscaling)", servedModelName, modelName)
			}

			return nil // Found and validated
		}
	}

	// If we're using vLLM image, --served-model-name is recommended for Aibrix integration
	return fmt.Errorf("--served-model-name parameter not found in args. For Aibrix autoscaling compatibility, please add '--served-model-name %s' to your vLLM command", modelName)
}

// setDeploymentDefaults sets default values for deployment configuration
func (s *ModelService) setDeploymentDefaults(config *ModelDeploymentConfig) {
	if config.ImagePullPolicy == "" {
		config.ImagePullPolicy = "IfNotPresent"
	}

	if config.InitialDelaySeconds == 0 {
		config.InitialDelaySeconds = 240
	}

	if config.EnableAutoscaling && config.AutoscalingConfig == nil {
		config.AutoscalingConfig = &ModelAutoscalingConfig{
			MinReplicas:    1,
			MaxReplicas:    10,
			TargetMetric:   "num_requests_running",
			TargetValue:    "40",
			ScaleDownDelay: "3m",
		}
	}
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

	// Get all deployments from Kubernetes (managed and unmanaged)
	allModels, err := s.deploymentManager.GetAllModels(ctx, DefaultNamespace, ManagedByLabelKey)
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

// ListModels returns models for an organization with filtering (backward compatibility for API)
func (s *ModelService) ListModels(ctx context.Context, orgID uint, filter *ModelFilter) ([]*Model, error) {
	// Use the main GetModels method with filtering
	return s.GetModels(ctx, orgID, filter)
}

// ModelInfo represents basic model information for listing
type ModelInfo struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	IsManaged   bool      `json:"is_managed"`
	Status      string    `json:"status"`
	Replicas    int32     `json:"replicas"`
	CreatedAt   time.Time `json:"created_at"`
	ModelType   string    `json:"model_type,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Description string    `json:"description,omitempty"`
}

// Helper functions

// convertEnvVars converts domain EnvVar to Kubernetes EnvVar
func convertEnvVars(envVars []EnvVar) []corev1.EnvVar {
	var k8sEnvVars []corev1.EnvVar
	for _, env := range envVars {
		k8sEnvVars = append(k8sEnvVars, corev1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}
	return k8sEnvVars
}

// convertFromKubernetesModelInfo converts Kubernetes ModelInfo to domain ModelInfo
func convertFromKubernetesModelInfo(k8sModels []*kubernetes.ModelInfo) []ModelInfo {
	var models []ModelInfo
	for _, k8sModel := range k8sModels {
		models = append(models, ModelInfo{
			Name:        k8sModel.Name,
			Namespace:   k8sModel.Namespace,
			IsManaged:   k8sModel.IsManaged,
			Status:      k8sModel.Status,
			Replicas:    k8sModel.Replicas,
			CreatedAt:   k8sModel.CreatedAt,
			ModelType:   k8sModel.Labels[ModelTypeLabelKey],
			DisplayName: k8sModel.Labels["display-name"],
			Description: k8sModel.Labels["description"],
		})
	}
	return models
}

// convertFromKubernetesModelInfoToPointers converts Kubernetes ModelInfo to domain ModelInfo pointers
func convertFromKubernetesModelInfoToPointers(k8sModels []*kubernetes.ModelInfo) []*ModelInfo {
	var models []*ModelInfo
	for _, k8sModel := range k8sModels {
		model := &ModelInfo{
			Name:        k8sModel.Name,
			Namespace:   k8sModel.Namespace,
			IsManaged:   k8sModel.IsManaged,
			Status:      k8sModel.Status,
			Replicas:    k8sModel.Replicas,
			CreatedAt:   k8sModel.CreatedAt,
			ModelType:   k8sModel.Labels[ModelTypeLabelKey],
			DisplayName: k8sModel.Labels["display-name"],
			Description: k8sModel.Labels["description"],
		}
		models = append(models, model)
	}
	return models
}
