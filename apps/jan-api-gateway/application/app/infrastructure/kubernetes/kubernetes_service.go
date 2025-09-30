package kubernetes

import (
	"context"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

// KubernetesService provides functionality to interact with Kubernetes cluster
type KubernetesService struct {
	clientset           *kubernetes.Clientset
	apiExtensionsClient *apiextensionsclientset.Clientset
	metricsClient       *metricsv1beta1.Clientset
	config              *rest.Config
	isInCluster         bool
}

// NodeGPUInfo contains GPU information for a node
type NodeGPUInfo struct {
	NodeName      string            `json:"node_name"`
	GPUCount      int               `json:"gpu_count"`
	GPUType       string            `json:"gpu_type"`
	TotalVRAM     resource.Quantity `json:"total_vram"`
	AvailableVRAM resource.Quantity `json:"available_vram"`
	Labels        map[string]string `json:"labels"`
}

// ClusterGPUStatus contains overall cluster GPU status
type ClusterGPUStatus struct {
	HasGPUs        bool          `json:"has_gpus"`
	TotalNodes     int           `json:"total_nodes"`
	GPUNodes       []NodeGPUInfo `json:"gpu_nodes"`
	TotalGPUs      int           `json:"total_gpus"`
	GPUOperatorOK  bool          `json:"gpu_operator_ok"`
	AibrixOK       bool          `json:"aibrix_ok"`
	KuberayOK      bool          `json:"kuberay_ok"`
	EnvoyGatewayOK bool          `json:"envoy_gateway_ok"`
}

// NewKubernetesService creates a new Kubernetes service instance
func NewKubernetesService() (*KubernetesService, error) {
	var config *rest.Config
	var err error
	var isInCluster bool

	// Try in-cluster config first
	if config, err = rest.InClusterConfig(); err == nil {
		isInCluster = true
	} else {
		// Fall back to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
		isInCluster = false
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	// Create API extensions client for CRD operations
	apiExtensionsClient, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create api extensions client: %w", err)
	}

	// Create metrics client (optional, may fail)
	metricsClient, _ := metricsv1beta1.NewForConfig(config)

	return &KubernetesService{
		clientset:           clientset,
		apiExtensionsClient: apiExtensionsClient,
		metricsClient:       metricsClient,
		config:              config,
		isInCluster:         isInCluster,
	}, nil
}

// IsInCluster returns true if running inside a Kubernetes cluster
func (ks *KubernetesService) IsInCluster() bool {
	return ks.isInCluster
}

// IsKubernetesAvailable checks if Kubernetes API is accessible
func (ks *KubernetesService) IsKubernetesAvailable(ctx context.Context) bool {
	if ks.clientset == nil {
		return false
	}

	// Try to get server version as a connectivity test
	_, err := ks.clientset.Discovery().ServerVersion()
	return err == nil
}

// CheckCRDExists checks if a specific CRD exists in the cluster
func (ks *KubernetesService) CheckCRDExists(ctx context.Context, crdName string) (bool, error) {
	if ks.apiExtensionsClient == nil {
		return false, fmt.Errorf("API extensions client not available")
	}

	_, err := ks.apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return false, nil // CRD doesn't exist
	}
	return true, nil
}

// CheckRequiredCRDs checks all required CRDs for the models API
func (ks *KubernetesService) CheckRequiredCRDs(ctx context.Context) (map[string]bool, error) {
	requiredCRDs := map[string]string{
		"aibrix":        "podautoscalers.autoscaling.aibrix.ai",
		"gpu_operator":  "clusterpolicies.nvidia.com",
		"kuberay":       "rayclusters.ray.io",
		"envoy_gateway": "gatewayclasses.gateway.networking.k8s.io",
	}

	result := make(map[string]bool)

	for name, crd := range requiredCRDs {
		exists, err := ks.CheckCRDExists(ctx, crd)
		if err != nil {
			return nil, fmt.Errorf("failed to check CRD %s: %w", crd, err)
		}
		result[name] = exists
	}

	return result, nil
}

// GetGPUNodes returns information about nodes with GPUs
func (ks *KubernetesService) GetGPUNodes(ctx context.Context) ([]NodeGPUInfo, error) {
	if ks.clientset == nil {
		return nil, fmt.Errorf("kubernetes client not available")
	}

	nodes, err := ks.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var gpuNodes []NodeGPUInfo

	for _, node := range nodes.Items {
		gpuInfo := ks.extractGPUInfo(node)
		if gpuInfo.GPUCount > 0 {
			gpuNodes = append(gpuNodes, gpuInfo)
		}
	}

	return gpuNodes, nil
}

// extractGPUInfo extracts GPU information from a node
func (ks *KubernetesService) extractGPUInfo(node v1.Node) NodeGPUInfo {
	info := NodeGPUInfo{
		NodeName: node.Name,
		Labels:   node.Labels,
	}

	// Check for NVIDIA GPUs
	if gpuCount, exists := node.Status.Capacity["nvidia.com/gpu"]; exists {
		info.GPUCount = int(gpuCount.Value())
		info.GPUType = "nvidia"
	}

	// Check for AMD GPUs
	if gpuCount, exists := node.Status.Capacity["amd.com/gpu"]; exists && info.GPUCount == 0 {
		info.GPUCount = int(gpuCount.Value())
		info.GPUType = "amd"
	}

	// Extract GPU model from labels
	if gpuModel, exists := node.Labels["nvidia.com/gpu.product"]; exists {
		info.GPUType = gpuModel
	} else if gpuModel, exists := node.Labels["amd.com/gpu.device"]; exists {
		info.GPUType = gpuModel
	}

	// Extract memory information
	if memory, exists := node.Status.Capacity["nvidia.com/gpu.memory"]; exists {
		info.TotalVRAM = memory
	}

	// Calculate available VRAM (simplified - would need actual pod resource usage)
	info.AvailableVRAM = info.TotalVRAM

	return info
}

// GetClusterGPUStatus returns comprehensive GPU status of the cluster
func (ks *KubernetesService) GetClusterGPUStatus(ctx context.Context) (*ClusterGPUStatus, error) {
	status := &ClusterGPUStatus{}

	// Check required CRDs
	crdStatus, err := ks.CheckRequiredCRDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check CRDs: %w", err)
	}

	status.GPUOperatorOK = crdStatus["gpu_operator"]
	status.AibrixOK = crdStatus["aibrix"]
	status.KuberayOK = crdStatus["kuberay"]
	status.EnvoyGatewayOK = crdStatus["envoy_gateway"]

	// Get GPU nodes
	gpuNodes, err := ks.GetGPUNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU nodes: %w", err)
	}

	status.GPUNodes = gpuNodes
	status.HasGPUs = len(gpuNodes) > 0

	// Calculate totals
	for _, node := range gpuNodes {
		status.TotalGPUs += node.GPUCount
	}

	// Get total node count
	nodes, err := ks.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		status.TotalNodes = len(nodes.Items)
	}

	return status, nil
}

// ValidateModelDeploymentRequirements checks if the cluster can deploy models
func (ks *KubernetesService) ValidateModelDeploymentRequirements(ctx context.Context) error {
	if !ks.IsInCluster() {
		return fmt.Errorf("models API only available when running in Kubernetes cluster")
	}

	if !ks.IsKubernetesAvailable(ctx) {
		return fmt.Errorf("kubernetes API not accessible")
	}

	status, err := ks.GetClusterGPUStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cluster status: %w", err)
	}

	if !status.AibrixOK {
		return fmt.Errorf("aibrix CRD not found - required for model deployment")
	}

	if !status.GPUOperatorOK {
		return fmt.Errorf("GPU operator CRD not found - required for GPU model deployment")
	}

	if !status.HasGPUs {
		return fmt.Errorf("no GPU nodes found in cluster - required for model deployment")
	}

	// Check for available storage classes
	if err := ks.validateStorageClasses(ctx); err != nil {
		return fmt.Errorf("storage validation failed: %w", err)
	}

	return nil
}

// validateStorageClasses checks if there are any storage classes available
func (ks *KubernetesService) validateStorageClasses(ctx context.Context) error {
	storageClasses, err := ks.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list storage classes: %w", err)
	}

	if len(storageClasses.Items) == 0 {
		return fmt.Errorf("no storage classes found in cluster")
	}

	return nil
}

// GetDefaultStorageClass returns the default storage class or the first available one
func (ks *KubernetesService) GetDefaultStorageClass(ctx context.Context) (string, error) {
	storageClasses, err := ks.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list storage classes: %w", err)
	}

	if len(storageClasses.Items) == 0 {
		return "", fmt.Errorf("no storage classes found")
	}

	// First try to find the default storage class
	for _, sc := range storageClasses.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc.Name, nil
		}
	}

	// If no default found, return the first one
	return storageClasses.Items[0].Name, nil
}
