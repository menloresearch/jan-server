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
	PVCSize      string `json:"pvc_size"` // Size for PVC (e.g., "30Gi")

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
	// Ensure namespace exists
	if err := mdm.ensureNamespace(ctx, spec.Namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create PVC if enabled
	if spec.EnablePVC {
		if err := mdm.ensurePVC(ctx, spec.Namespace, spec.PVCName, spec.PVCSize, spec.StorageClass); err != nil {
			return fmt.Errorf("failed to ensure PVC: %w", err)
		}
	}

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
	}

	// Only set limits if they are specified (> 0)
	if !spec.CPULimit.IsZero() || !spec.MemoryLimit.IsZero() {
		resources.Limits = corev1.ResourceList{}
		if !spec.CPULimit.IsZero() {
			resources.Limits[corev1.ResourceCPU] = spec.CPULimit
		}
		if !spec.MemoryLimit.IsZero() {
			resources.Limits[corev1.ResourceMemory] = spec.MemoryLimit
		}
	}

	// Add GPU resources if specified
	if spec.GPUCount > 0 {
		gpuQuantity := resource.MustParse(fmt.Sprintf("%d", spec.GPUCount))
		resources.Requests["nvidia.com/gpu"] = gpuQuantity
		if resources.Limits == nil {
			resources.Limits = corev1.ResourceList{}
		}
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
			PeriodSeconds:       30,
			FailureThreshold:    15,
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
			FailureThreshold:    15,
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
	labelSelector := fmt.Sprintf("%s=jan-server", managedLabel)

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

	// If namespace is empty, search all namespaces
	var deployments *appsv1.DeploymentList
	var err error

	if namespace == "" {
		// Search all namespaces
		deployments, err = mdm.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	} else {
		// Search specific namespace
		deployments, err = mdm.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var models []*ModelInfo
	for _, deployment := range deployments.Items {
		isManaged := false
		if deployment.Labels[managedLabel] == "jan-server" {
			isManaged = true
		}

		// Get detailed status including pod info and events
		modelInfo := &ModelInfo{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			IsManaged: isManaged,
			Replicas:  deployment.Status.ReadyReplicas,
			Labels:    deployment.Labels,
			CreatedAt: deployment.CreationTimestamp.Time,
		}

		// Get detailed status with pod information
		mdm.enrichModelInfoWithPodDetails(ctx, modelInfo, &deployment)

		models = append(models, modelInfo)
	}

	return models, nil
}

// ModelInfo contains information about a deployed model
type ModelInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	IsManaged    bool              `json:"is_managed"`
	Status       string            `json:"status"`
	Replicas     int32             `json:"replicas"`
	Labels       map[string]string `json:"labels"`
	CreatedAt    time.Time         `json:"created_at"`
	RestartCount int32             `json:"restart_count,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	LastEvent    string            `json:"last_event,omitempty"`
}

// enrichModelInfoWithPodDetails gets detailed pod status and events
func (mdm *ModelDeploymentManager) enrichModelInfoWithPodDetails(ctx context.Context, modelInfo *ModelInfo, deployment *appsv1.Deployment) {
	// Default status based on deployment
	status := "Unknown"
	if deployment.Status.ReadyReplicas > 0 {
		status = "Running"
	} else if deployment.Status.Replicas > 0 {
		status = "Creating"
	}

	// Get pods for this deployment
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
	pods, err := mdm.clientset.CoreV1().Pods(deployment.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		modelInfo.Status = status
		modelInfo.ErrorMessage = fmt.Sprintf("Failed to get pod info: %v", err)
		return
	}

	// Analyze pod status
	if len(pods.Items) == 0 {
		modelInfo.Status = "Pending"
		return
	}

	// Check pod conditions and restart counts
	var maxRestartCount int32
	var lastErrorMessage string
	var criticalEvents []string

	for _, pod := range pods.Items {
		// Count restarts
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.RestartCount > maxRestartCount {
				maxRestartCount = containerStatus.RestartCount
			}

			// Check for crash loop back off
			if containerStatus.State.Waiting != nil {
				reason := containerStatus.State.Waiting.Reason
				message := containerStatus.State.Waiting.Message

				if reason == "CrashLoopBackOff" || reason == "ImagePullBackOff" || reason == "ErrImagePull" {
					status = reason
					lastErrorMessage = message
				}
			}

			// Check for terminated containers
			if containerStatus.State.Terminated != nil {
				if containerStatus.State.Terminated.ExitCode != 0 {
					status = "Failed"
					lastErrorMessage = containerStatus.State.Terminated.Message
				}
			}
		}

		// Check pod phase
		if pod.Status.Phase == corev1.PodFailed {
			status = "Failed"
			if lastErrorMessage == "" {
				lastErrorMessage = pod.Status.Message
			}
		}
	}

	// Get recent events for this deployment
	events, err := mdm.clientset.CoreV1().Events(deployment.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.kind=Pod,involvedObject.namespace=%s", deployment.Namespace),
	})

	if err == nil {
		// Filter events for this deployment's pods
		for _, event := range events.Items {
			// Check if this event is for one of our pods
			for _, pod := range pods.Items {
				if event.InvolvedObject.Name == pod.Name {
					// Focus on warning/error events
					if event.Type == "Warning" || event.Type == "Error" {
						eventMsg := fmt.Sprintf("%s: %s", event.Reason, event.Message)
						criticalEvents = append(criticalEvents, eventMsg)
					}
				}
			}
		}
	}

	// Set restart-based status
	if maxRestartCount >= 3 {
		if status == "Creating" || status == "Unknown" {
			status = "CrashLoopBackOff"
		}
	}

	// Combine critical events into last event
	if len(criticalEvents) > 0 {
		// Get the most recent event (events are usually ordered by time)
		modelInfo.LastEvent = criticalEvents[len(criticalEvents)-1]
		if lastErrorMessage == "" {
			lastErrorMessage = modelInfo.LastEvent
		}
	}

	modelInfo.Status = status
	modelInfo.RestartCount = maxRestartCount
	modelInfo.ErrorMessage = lastErrorMessage
}

// ensureNamespace creates a namespace if it doesn't exist
func (mdm *ModelDeploymentManager) ensureNamespace(ctx context.Context, namespaceName string) error {
	// Check if namespace exists
	_, err := mdm.clientset.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists
		return nil
	}

	// Create namespace if it doesn't exist
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "jan-server",
				"jan-server.menlo.ai/purpose":  "model-deployment",
			},
		},
	}

	_, err = mdm.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespaceName, err)
	}

	return nil
}

// ensurePVC creates a PVC if it doesn't exist, or resize if needed
func (mdm *ModelDeploymentManager) ensurePVC(ctx context.Context, namespace, pvcName, size, storageClass string) error {
	// Parse requested size
	requestedSize := resource.MustParse(size)

	// Check if PVC exists
	existingPVC, err := mdm.clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err == nil {
		// PVC already exists - check if resize is needed
		existingSize := existingPVC.Spec.Resources.Requests[corev1.ResourceStorage]

		// Compare sizes
		if requestedSize.Cmp(existingSize) > 0 {
			// Requested size is larger - resize PVC
			fmt.Printf("Resizing PVC %s from %s to %s\n", pvcName, existingSize.String(), requestedSize.String())

			existingPVC.Spec.Resources.Requests[corev1.ResourceStorage] = requestedSize
			_, err = mdm.clientset.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, existingPVC, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to resize PVC %s: %w", pvcName, err)
			}
		} else if requestedSize.Cmp(existingSize) < 0 {
			// Requested size is smaller - log warning but continue
			fmt.Printf("Warning: PVC %s already exists with size %s, cannot resize to smaller size %s. Using existing size.\n",
				pvcName, existingSize.String(), requestedSize.String())
		}
		// If sizes are equal, no action needed
		return nil
	}

	// PVC doesn't exist - create new one
	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteMany, // Try RWX first for shared storage
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "jan-server",
				"jan-server.menlo.ai/purpose":  "model-storage",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: requestedSize,
				},
			},
		},
	}

	// Set storage class if provided
	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	_, err = mdm.clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create PVC %s: %w", pvcName, err)
	}

	fmt.Printf("Created new PVC %s with size %s\n", pvcName, requestedSize.String())
	return nil
}

// Helper function
func int32Ptr(i int32) *int32 {
	return &i
}
