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
	// For now, just validate that the resources are properly formatted
	// The actual cluster validation would need to be implemented based on cluster status
	if requirements.CPU.String() == "" {
		return fmt.Errorf("CPU requirement is required")
	}
	if requirements.Memory.String() == "" {
		return fmt.Errorf("memory requirement is required")
	}

	// TODO: Add actual cluster capacity validation when needed
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

// GetClusterStatus returns the current cluster status for models
func (s *ModelService) GetClusterStatus(ctx context.Context) (*kubernetes.ClusterGPUStatus, error) {
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
