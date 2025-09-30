# Organization Models API Examples

This document provides examples of how to use the `/v1/organization/models` API endpoints.

## Prerequisites

1. Application must be deployed in Kubernetes cluster
2. Required CRDs must be installed:
   - Aibrix (`podautoscalers.autoscaling.aibrix.ai`)
   - GPU Operator (`clusterpolicies.nvidia.com`)
   - KubeRay (`rayclusters.ray.io`)
   - Envoy Gateway (`gatewayclasses.gateway.networking.k8s.io`)

3. At least one GPU node available
4. At least one storage class available

## API Endpoints

### 1. Check Cluster Status

**GET** `/v1/organization/models/status`

Check if the cluster is ready for model deployment.

**Response:**

```json
{
  "organization_id": 123,
  "cluster_status": {
    "has_gpus": true,
    "total_nodes": 3,
    "gpu_nodes": [
      {
        "node_name": "gpu-node-1",
        "gpu_count": 2,
        "gpu_type": "nvidia-a100",
        "total_vram": "40Gi",
        "available_vram": "38Gi"
      }
    ],
    "total_gpus": 2,
    "gpu_operator_ok": true,
    "aibrix_ok": true,
    "kuberay_ok": true,
    "envoy_gateway_ok": true
  }
}
```

### 2. List All Models (Managed + Unmanaged)

**GET** `/v1/organization/models/all`

Get all models in the cluster, both managed by jan-server and unmanaged.

**Response:**

```json
{
  "models": [
    {
      "name": "llama2-7b-chat",
      "namespace": "jan-models", 
      "is_managed": true,
      "status": "Running",
      "replicas": 2,
      "created_at": "2025-01-01T10:00:00Z",
      "model_type": "chat",
      "display_name": "Llama 2 7B Chat",
      "description": "Llama 2 model for chat completions"
    },
    {
      "name": "external-model",
      "namespace": "default",
      "is_managed": false,
      "status": "Running", 
      "replicas": 1,
      "created_at": "2025-01-01T09:00:00Z"
    }
  ],
  "total": 2
}
```

### 3. Create New Model

**POST** `/v1/organization/models`

Create and deploy a new AI model.

**Request:**

```json
{
  "name": "qwen3-coder-30b",
  "display_name": "Qwen3 Coder 30B",
  "description": "Qwen3 Coder model for code completion",
  "model_type": "completion",
  "huggingface_id": "Qwen/Qwen3-Coder-30B-A3B-Instruct-FP8",
  "requirements": {
    "cpu": "2000m",
    "memory": "16Gi",
    "gpu": {
      "min_vram": "24Gi",
      "min_gpus": 1,
      "max_gpus": 1
    }
  },
  "deployment_config": {
    "image": "registry.menlo.ai/dockerhub/vllm/vllm-openai:v0.10.2",
    "image_pull_policy": "IfNotPresent",
    "command": ["sh"],
    "args": [
      "-c",
      "python3 -m vllm.entrypoints.openai.api_server --host 0.0.0.0 --port 8000 --uvicorn-log-level warning --model Qwen/Qwen3-Coder-30B-A3B-Instruct-FP8 --served-model-name qwen3-coder-30b --tool-call-parser qwen3_coder --enable-auto-tool-choice --api-server-count 4"
    ],
    "gpu_count": 1,
    "initial_delay_seconds": 240,
    "enable_pvc": true,
    "storage_class": "",
    "enable_autoscaling": true,
    "autoscaling_config": {
      "min_replicas": 1,
      "max_replicas": 8,
      "target_metric": "num_requests_running",
      "target_value": "40",
      "scale_down_delay": "3m"
    },
    "extra_env": [
      {
        "name": "CUDA_VISIBLE_DEVICES",
        "value": "0"
      }
    ],
    "hugging_face_token": "hf_your_token_here"
  },
  "tags": ["coding", "completion"],
  "is_public": false
}
```

**Response:**

```json
{
  "model": {
    "id": 1,
    "public_id": "mdl_abc123def456",
    "organization_id": 123,
    "name": "qwen3-coder-30b",
    "display_name": "Qwen3 Coder 30B", 
    "model_type": "completion",
    "status": "creating",
    "huggingface_id": "Qwen/Qwen3-Coder-30B-A3B-Instruct-FP8",
    "namespace": "jan-models",
    "created_at": "2025-01-01T10:00:00Z"
  }
}
```

### 4. List Managed Models Only

**GET** `/v1/organization/models`

Get only models managed by jan-server for this organization.

**Response:**

```json
{
  "models": [
    {
      "id": 1,
      "public_id": "mdl_abc123def456",
      "name": "qwen3-coder-30b",
      "display_name": "Qwen3 Coder 30B",
      "status": "running",
      "model_type": "completion"
    }
  ],
  "total": 1
}
```

### 5. Get Model Details

**GET** `/v1/organization/models/{model_id}`

Get detailed information about a specific managed model.

**Response:**

```json
{
  "model": {
    "id": 1,
    "public_id": "mdl_abc123def456",
    "organization_id": 123,
    "name": "qwen3-coder-30b",
    "display_name": "Qwen3 Coder 30B",
    "description": "Qwen3 Coder model for code completion",
    "model_type": "completion",
    "status": "running",
    "huggingface_id": "Qwen/Qwen3-Coder-30B-A3B-Instruct-FP8",
    "namespace": "jan-models",
    "deployment_name": "qwen3-coder-30b",
    "service_name": "qwen3-coder-30b",
    "endpoint_url": "http://qwen3-coder-30b.jan-models.svc.cluster.local:8000",
    "requirements": {
      "cpu": "2000m",
      "memory": "16Gi",
      "gpu": {
        "min_vram": "24Gi",
        "min_gpus": 1
      }
    },
    "tags": ["coding", "completion"],
    "is_public": false,
    "created_at": "2025-01-01T10:00:00Z",
    "updated_at": "2025-01-01T10:05:00Z"
  }
}
```

### 6. Delete Model

**DELETE** `/v1/organization/models/{model_id}`

Delete a managed model and all its Kubernetes resources.

**Response:** `204 No Content`

## Configuration Parameters

### Model Deployment Parameters

| Parameter | Description | Required | Default |
|-----------|-------------|----------|---------|
| `image` | Container image for the model | Yes | - |
| `command` | Container command | Yes | - |
| `args` | Container arguments | Yes | - |
| `gpu_count` | Number of GPUs needed | No | 0 |
| `initial_delay_seconds` | Probe initial delay | No | 240 |
| `enable_pvc` | Use shared storage PVC | No | false |
| `storage_class` | Storage class for PVC | No | default |
| `enable_autoscaling` | Enable PodAutoscaler | No | false |
| `hugging_face_token` | HF token for private models | No | - |

### Autoscaling Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `min_replicas` | Minimum replicas | 1 |
| `max_replicas` | Maximum replicas | 10 |
| `target_metric` | Metric to scale on | "num_requests_running" |
| `target_value` | Target metric value | "40" |
| `scale_down_delay` | Delay before scaling down | "3m" |

## Validation Requirements

### Aibrix Compatibility

For proper autoscaling with Aibrix, the following requirements must be met:

1. **Served Model Name Consistency**: The `--served-model-name` parameter in your vLLM args must exactly match the model name field.

   **Example:**

   ```json
   {
     "name": "qwen3-coder-30b",
     "deployment_config": {
       "args": [
         "python3 -m vllm.entrypoints.openai.api_server --served-model-name qwen3-coder-30b ..."
       ]
     }
   }
   ```

2. **Required for autoscaling**: If `enable_autoscaling: true`, the `--served-model-name` parameter is mandatory.

3. **Validation errors**: The API will return specific error messages if validation fails:

   ```json
   {
     "error": "--served-model-name 'wrong-name' must match model name 'qwen3-coder-30b' (required by Aibrix for autoscaling)"
   }
   ```

## Kubernetes Resources Created

When you create a model, the following Kubernetes resources are automatically created:

1. **Deployment** - Runs the model container with specified resources
2. **Service** - Exposes the model on port 8000
3. **ServiceMonitor** - For Prometheus monitoring (if Prometheus operator is installed)
4. **PodAutoscaler** - For auto-scaling (if enabled)

All resources are labeled with:

- `model.aibrix.ai/name: {model-name}`
- `jan-server.menlo.ai/managed: true` (for managed identification)

## Error Responses

### 403 Forbidden - Models API Not Available

```json
{
  "error": "models API only available when running in Kubernetes cluster"
}
```

### 403 Forbidden - Missing CRDs

```json
{
  "error": "aibrix CRD not found - required for model deployment"
}
```

### 400 Bad Request - Insufficient Resources

```json
{
  "error": "no GPU node can satisfy the requirements: min 2 GPUs with 48Gi VRAM"
}
```

### 400 Bad Request - Invalid Served Model Name

```json
{
  "error": "--served-model-name 'wrong-name' must match model name 'qwen3-coder-30b' (required by Aibrix for autoscaling)"
}
```