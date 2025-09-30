package kubernetes

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ModelDeploymentSpec contains specification for deploying a model
type ModelDeploymentSpec struct {
	// Basic model info
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	// Container configuration
	Image           string   `json:"image"`
	ImagePullPolicy string   `json:"image_pull_policy"`
	Command         []string `json:"command"`
	Args            []string `json:"args"`
	Port            int32    `json:"port"`

	// Resource requirements
	GPUCount      int               `json:"gpu_count"`
	CPURequest    resource.Quantity `json:"cpu_request"`
	CPULimit      resource.Quantity `json:"cpu_limit"`
	MemoryRequest resource.Quantity `json:"memory_request"`
	MemoryLimit   resource.Quantity `json:"memory_limit"`

	// Probe configuration
	InitialDelaySeconds int32 `json:"initial_delay_seconds"`

	// Storage configuration
	EnablePVC    bool   `json:"enable_pvc"`
	PVCName      string `json:"pvc_name"`
	StorageClass string `json:"storage_class"`

	// Environment variables
	ExtraEnv []corev1.EnvVar `json:"extra_env"`

	// Autoscaling configuration
	EnableAutoscaling bool                    `json:"enable_autoscaling"`
	AutoscalingConfig *ModelAutoscalingConfig `json:"autoscaling_config,omitempty"`

	// Labels for managed identification
	ManagedLabels map[string]string `json:"managed_labels"`
}

// ModelAutoscalingConfig contains PodAutoscaler configuration
type ModelAutoscalingConfig struct {
	MinReplicas    int32  `json:"min_replicas"`
	MaxReplicas    int32  `json:"max_replicas"`
	TargetMetric   string `json:"target_metric"`
	TargetValue    string `json:"target_value"`
	ScaleDownDelay string `json:"scale_down_delay"`
}

// ModelDeploymentManager handles Kubernetes model deployments
type ModelDeploymentManager struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewModelDeploymentManager creates a new deployment manager
func (ks *KubernetesService) NewModelDeploymentManager() (*ModelDeploymentManager, error) {
	dynamicClient, err := dynamic.NewForConfig(ks.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &ModelDeploymentManager{
		clientset:     ks.clientset,
		dynamicClient: dynamicClient,
	}, nil
}

// NewModelDeploymentManager creates a new deployment manager for Wire DI
func NewModelDeploymentManager(ks *KubernetesService) (*ModelDeploymentManager, error) {
	return ks.NewModelDeploymentManager()
}

// CreateModelDeployment creates all resources for a model deployment
func (mdm *ModelDeploymentManager) CreateModelDeployment(ctx context.Context, spec *ModelDeploymentSpec) error {
	// Create Deployment
	deployment := mdm.createDeployment(spec)
	_, err := mdm.clientset.AppsV1().Deployments(spec.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service
	service := mdm.createService(spec)
	_, err = mdm.clientset.CoreV1().Services(spec.Namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Create ServiceMonitor (optional, may fail if Prometheus operator not installed)
	if err := mdm.createServiceMonitor(ctx, spec); err != nil {
		// Log warning but don't fail the deployment
		fmt.Printf("Warning: failed to create ServiceMonitor: %v\n", err)
	}

	// Create PodAutoscaler if enabled
	if spec.EnableAutoscaling && spec.AutoscalingConfig != nil {
		if err := mdm.createPodAutoscaler(ctx, spec); err != nil {
			return fmt.Errorf("failed to create PodAutoscaler: %w", err)
		}
	}

	return nil
}

// createDeployment creates a Kubernetes Deployment
func (mdm *ModelDeploymentManager) createDeployment(spec *ModelDeploymentSpec) *appsv1.Deployment {
	labels := map[string]string{
		"model.aibrix.ai/name": spec.Name,
		"model.aibrix.ai/port": fmt.Sprintf("%d", spec.Port),
	}

	// Add managed labels
	for k, v := range spec.ManagedLabels {
		labels[k] = v
	}

	// Container resources
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    spec.CPURequest,
			corev1.ResourceMemory: spec.MemoryRequest,
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    spec.CPULimit,
			corev1.ResourceMemory: spec.MemoryLimit,
		},
	}

	// Add GPU resources if specified
	if spec.GPUCount > 0 {
		gpuQuantity := resource.MustParse(fmt.Sprintf("%d", spec.GPUCount))
		resources.Requests["nvidia.com/gpu"] = gpuQuantity
		resources.Limits["nvidia.com/gpu"] = gpuQuantity
	}

	// Volume mounts for shared storage
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume

	if spec.EnablePVC {
		volumeMounts = []corev1.VolumeMount{
			{
				Name:      "hf-cache",
				MountPath: "/root/.cache/huggingface/hub",
				SubPath:   "hf-hub",
			},
			{
				Name:      "hf-cache",
				MountPath: "/root/.cache/vllm/torch_compile_cache",
				SubPath:   "vllm-compile",
			},
		}

		volumes = []corev1.Volume{
			{
				Name: "hf-cache",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: spec.PVCName,
					},
				},
			},
		}
	}

	// Container spec
	container := corev1.Container{
		Name:            "vllm-openai",
		Image:           spec.Image,
		ImagePullPolicy: corev1.PullPolicy(spec.ImagePullPolicy),
		Command:         spec.Command,
		Args:            spec.Args,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: spec.Port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources:    resources,
		VolumeMounts: volumeMounts,
		Env:          spec.ExtraEnv,
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt(int(spec.Port)),
				},
			},
			InitialDelaySeconds: spec.InitialDelaySeconds,
			PeriodSeconds:       5,
			FailureThreshold:    3,
			TimeoutSeconds:      1,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt(int(spec.Port)),
				},
			},
			InitialDelaySeconds: spec.InitialDelaySeconds,
			PeriodSeconds:       30,
			FailureThreshold:    5,
			TimeoutSeconds:      1,
		},
	}

	// Pod annotations for Prometheus
	podAnnotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   fmt.Sprintf("%d", spec.Port),
		"prometheus.io/path":   "/metrics",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"model.aibrix.ai/name": spec.Name,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
					Volumes:    volumes,
				},
			},
		},
	}

	return deployment
}

// createService creates a Kubernetes Service
func (mdm *ModelDeploymentManager) createService(spec *ModelDeploymentSpec) *corev1.Service {
	labels := map[string]string{
		"model.aibrix.ai/name": spec.Name,
		"prometheus-discovery": "true",
	}

	// Add managed labels
	for k, v := range spec.ManagedLabels {
		labels[k] = v
	}

	annotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   fmt.Sprintf("%d", spec.Port),
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Namespace:   spec.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "serve",
					Port:       spec.Port,
					TargetPort: intstr.FromInt(int(spec.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"model.aibrix.ai/name": spec.Name,
			},
		},
	}

	return service
}

// createServiceMonitor creates a ServiceMonitor for Prometheus monitoring
func (mdm *ModelDeploymentManager) createServiceMonitor(ctx context.Context, spec *ModelDeploymentSpec) error {
	// ServiceMonitor is a CRD from Prometheus operator
	gvr := schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "servicemonitors",
	}

	labels := map[string]string{
		"model.aibrix.ai/name": spec.Name,
	}

	// Add managed labels
	for k, v := range spec.ManagedLabels {
		labels[k] = v
	}

	serviceMonitor := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "monitoring.coreos.com/v1",
			"kind":       "ServiceMonitor",
			"metadata": map[string]interface{}{
				"name":      spec.Name,
				"namespace": spec.Namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"endpoints": []interface{}{
					map[string]interface{}{
						"port":     "serve",
						"path":     "/metrics",
						"interval": "30s",
					},
				},
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"model.aibrix.ai/name": spec.Name,
					},
				},
			},
		},
	}

	_, err := mdm.dynamicClient.Resource(gvr).Namespace(spec.Namespace).Create(ctx, serviceMonitor, metav1.CreateOptions{})
	return err
}

// createPodAutoscaler creates a PodAutoscaler for Aibrix autoscaling
func (mdm *ModelDeploymentManager) createPodAutoscaler(ctx context.Context, spec *ModelDeploymentSpec) error {
	gvr := schema.GroupVersionResource{
		Group:    "autoscaling.aibrix.ai",
		Version:  "v1alpha1",
		Resource: "podautoscalers",
	}

	labels := map[string]string{
		"app.kubernetes.io/name": "aibrix",
	}

	// Add managed labels
	for k, v := range spec.ManagedLabels {
		labels[k] = v
	}

	annotations := map[string]string{
		"kpa.autoscaling.aibrix.ai/scale-down-delay": spec.AutoscalingConfig.ScaleDownDelay,
	}

	podAutoscaler := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "autoscaling.aibrix.ai/v1alpha1",
			"kind":       "PodAutoscaler",
			"metadata": map[string]interface{}{
				"name":        spec.Name,
				"namespace":   spec.Namespace,
				"labels":      labels,
				"annotations": annotations,
			},
			"spec": map[string]interface{}{
				"scalingStrategy": "KPA",
				"minReplicas":     spec.AutoscalingConfig.MinReplicas,
				"maxReplicas":     spec.AutoscalingConfig.MaxReplicas,
				"metricsSources": []interface{}{
					map[string]interface{}{
						"metricSourceType": "pod",
						"protocolType":     "http",
						"port":             fmt.Sprintf("%d", spec.Port),
						"path":             "metrics",
						"targetMetric":     spec.AutoscalingConfig.TargetMetric,
						"targetValue":      spec.AutoscalingConfig.TargetValue,
					},
				},
				"scaleTargetRef": map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"name":       spec.Name,
				},
			},
		},
	}

	_, err := mdm.dynamicClient.Resource(gvr).Namespace(spec.Namespace).Create(ctx, podAutoscaler, metav1.CreateOptions{})
	return err
}

// DeleteModelDeployment deletes all resources for a model deployment
func (mdm *ModelDeploymentManager) DeleteModelDeployment(ctx context.Context, name, namespace string) error {
	// Delete PodAutoscaler
	gvr := schema.GroupVersionResource{
		Group:    "autoscaling.aibrix.ai",
		Version:  "v1alpha1",
		Resource: "podautoscalers",
	}
	_ = mdm.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	// Delete ServiceMonitor
	gvr = schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "servicemonitors",
	}
	_ = mdm.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	// Delete Service
	err := mdm.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Delete Deployment
	err = mdm.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	return nil
}

// GetManagedModels returns all models managed by jan-server
func (mdm *ModelDeploymentManager) GetManagedModels(ctx context.Context, namespace, managedLabel string) ([]*ModelInfo, error) {
	labelSelector := fmt.Sprintf("%s=true", managedLabel)

	deployments, err := mdm.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var models []*ModelInfo
	for _, deployment := range deployments.Items {
		status := "Unknown"
		if deployment.Status.ReadyReplicas > 0 {
			status = "Running"
		} else if deployment.Status.Replicas > 0 {
			status = "Starting"
		}

		model := &ModelInfo{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			IsManaged: true,
			Status:    status,
			Replicas:  deployment.Status.ReadyReplicas,
			Labels:    deployment.Labels,
			CreatedAt: deployment.CreationTimestamp.Time,
		}
		models = append(models, model)
	}

	return models, nil
}

// GetAllModels returns all models (managed and unmanaged) in the cluster
func (mdm *ModelDeploymentManager) GetAllModels(ctx context.Context, namespace, managedLabel string) ([]*ModelInfo, error) {
	// Get all deployments with aibrix model labels
	labelSelector := "model.aibrix.ai/name"

	deployments, err := mdm.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var models []*ModelInfo
	for _, deployment := range deployments.Items {
		isManaged := false
		if deployment.Labels[managedLabel] == "true" {
			isManaged = true
		}

		status := "Unknown"
		if deployment.Status.ReadyReplicas > 0 {
			status = "Running"
		} else if deployment.Status.Replicas > 0 {
			status = "Starting"
		}

		model := &ModelInfo{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			IsManaged: isManaged,
			Status:    status,
			Replicas:  deployment.Status.ReadyReplicas,
			Labels:    deployment.Labels,
			CreatedAt: deployment.CreationTimestamp.Time,
		}
		models = append(models, model)
	}

	return models, nil
}

// ModelInfo contains information about a deployed model
type ModelInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	IsManaged bool              `json:"is_managed"`
	Status    string            `json:"status"`
	Replicas  int32             `json:"replicas"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"created_at"`
}

// Helper function
func int32Ptr(i int32) *int32 {
	return &i
}
