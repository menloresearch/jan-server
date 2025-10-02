# Organization Models API Examples

This document provides examples of how to use the simplified `/v1/organization/models` API endpoints.

## Prerequisites

1. Application must be deployed in Kubernetes cluster
2. At least one GPU node available (for GPU models)
3. At least one storage class available

## API Endpoints

### 1. List Models

**GET** `/v1/organization/models`

List all models for the organization.

**Response:**

```json
{
  "models": [
    {
      "id": "llama-7b",
      "organization_id": 123,
      "display_name": "Llama 2 7B Chat",
      "description": "Llama 2 7B model for chat completion",
      "status": "running",
      "huggingface_id": "meta-llama/Llama-2-7b-chat-hf",
      "requirements": {
        "cpu": "1",
        "memory": "2Gi",
        "gpu": {
          "min_vram": "8Gi",
          "preferred_vram": "16Gi",
          "gpu_type": "nvidia",
          "min_gpus": 1,
          "max_gpus": 1
        }
      },
      "namespace": "default",
      "deployment_name": "llama-7b",
      "service_name": "llama-7b",
      "endpoint_url": "http://llama-7b.default.svc.cluster.local:8000",
      "tags": ["llama", "chat", "7b"],
      "managed": true,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 1
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
### 2. Create New Model (Simplified API)

**POST** `/v1/organization/models`

Create and deploy a new AI model with simplified configuration.

**Request:**

```json
{
  "name": "llama-7b",
  "display_name": "Llama 2 7B Chat", 
  "image": "vllm/vllm-openai:latest",
  "hugging_face_token": "hf_your_token_here",
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--served-model-name", "llama-7b",
    "--model", "meta-llama/Llama-2-7b-chat-hf"
  ],
  "gpu_count": 1,
  "storage_class": "fast-ssd"
}
```

**Minimal Request (Recommended):**

```json
{
  "name": "qwen-coder-7b",
  "display_name": "Qwen Coder 7B",
  "image": "vllm/vllm-openai:latest", 
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--served-model-name", "qwen-coder-7b",
    "--model", "Qwen/Qwen2.5-Coder-7B-Instruct"
  ],
  "gpu_count": 1
}
```

**Optional Fields Note:**
- `hugging_face_token`: Sets HF_TOKEN env var only when provided  
- `storage_class`: Uses cluster default when not provided
```

**Response:**

```json
{
  "model": {
    "id": "llama-7b",
    "organization_id": 123,
    "display_name": "Llama 2 7B Chat",
    "description": "Llama 2 7B model for chat completion",
    "status": "creating",
    "huggingface_id": "meta-llama/Llama-2-7b-chat-hf",
    "requirements": {
      "cpu": "1",
      "memory": "2Gi",
      "gpu": {
        "min_vram": "8Gi",
        "preferred_vram": "16Gi",
        "gpu_type": "nvidia",
        "min_gpus": 1,
        "max_gpus": 1
      }
    },
    "namespace": "default",
    "deployment_name": "llama-7b",
    "service_name": "llama-7b",
    "tags": ["llama", "chat", "7b"],
    "managed": true,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 3. Create Model from YAML

**POST** `/v1/organization/models/yaml`

Create a model from custom YAML manifest for advanced configurations.

**Request:**

```json
{
  "name": "custom-model",
  "display_name": "Custom Model Deployment",
  "description": "Custom model deployed via YAML",
  "yaml_content": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: custom-model\n  labels:\n    app: custom-model\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: custom-model\n  template:\n    metadata:\n      labels:\n        app: custom-model\n    spec:\n      containers:\n      - name: model\n        image: custom/model:latest\n        ports:\n        - containerPort: 8000",
  "tags": ["custom", "yaml"]
}
```

**Response:**

```json
{
  "model": {
    "id": "custom-model",
    "organization_id": 123,
    "display_name": "Custom Model Deployment",
    "description": "Custom model deployed via YAML",
    "status": "creating",
    "namespace": "default",
    "deployment_name": "custom-model",
    "service_name": "custom-model", 
    "tags": ["custom", "yaml"],
    "managed": false,
    "created_at": "2024-01-15T10:35:00Z",
    "updated_at": "2024-01-15T10:35:00Z"
  }
}
      }
```

### 4. Get Model by ID

**GET** `/v1/organization/models/{model_id}`

Get details of a specific model.

**Response:**

```json
{
  "model": {
    "id": "llama-7b",
    "organization_id": 123,
    "display_name": "Llama 2 7B Chat",
    "description": "Llama 2 7B model for chat completion",
    "status": "running",
    "huggingface_id": "meta-llama/Llama-2-7b-chat-hf",
    "requirements": {
      "cpu": "1",
      "memory": "2Gi",
      "gpu": {
        "min_vram": "8Gi",
        "preferred_vram": "16Gi",
        "gpu_type": "nvidia",
        "min_gpus": 1,
        "max_gpus": 1
      }
    },
    "namespace": "default",
    "deployment_name": "llama-7b",
    "service_name": "llama-7b",
    "endpoint_url": "http://llama-7b.default.svc.cluster.local:8000",
    "tags": ["llama", "chat", "7b"],
    "managed": true,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 5. Update Model

**PUT** `/v1/organization/models/{model_id}`

Update model metadata (display_name, description, tags).

**Request:**

```json
{
  "display_name": "Updated Model Name",
  "description": "Updated description",
  "tags": ["updated", "tags"]
}
```

### 6. Delete Model

**DELETE** `/v1/organization/models/{model_id}`

Delete a model and its Kubernetes resources.

**Response:**

```json
{
  "message": "Model llama-7b deleted successfully"
}
```

## Simplified API Benefits

### Before (Complex API)
- 50+ configuration fields
- Nested deployment config structures  
- Complex autoscaling configurations
- Multiple environment variable arrays
- Kubernetes-specific configurations

### After (Simplified API)
- **11 essential fields only**
- **Automatic defaults** for most settings
- **Template-based** deployment
- **YAML upload** for advanced users
- **Cleaner error handling**

## Default Values

The simplified API automatically sets these defaults:

```go
// Default values applied by req.SetDefaults()
Replicas: 1              // if not specified
GPUCount: 1              // if not specified  
InitialDelaySeconds: 60  // if not specified
StorageSize: 20          // if not specified (20Gi)

// Hardcoded defaults in deployment
ImagePullPolicy: "IfNotPresent"
Port: 8000
CPU: "1"
Memory: "2Gi" 
GPU_VRAM: "8Gi" (min), "16Gi" (preferred)
```

## Kubernetes Template Variables

The simplified API is based on these template variables:

```yaml
<model-id>              → req.Name
<docker-image>          → req.Image  
<huggingface-model-id>  → req.HuggingFaceID
<huggingface-token>     → req.HuggingFaceToken
<gpu-count>             → req.GPUCount
<replicas>              → req.Replicas
<initial-delay>         → req.InitialDelaySeconds
<storage-class>         → req.StorageClass
<storage-size>          → req.StorageSize + "Gi"
<command>               → req.Command array
```
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