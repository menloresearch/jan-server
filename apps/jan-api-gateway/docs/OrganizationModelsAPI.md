# Organization Models API Reference

This document provides complete reference for the simplified Organization Models API with 4 core endpoints.

## Prerequisites

1. Application must be deployed in Kubernetes cluster
2. Valid JWT token for authentication
3. At least one GPU node available (for GPU models)
4. Required CRDs: Aibrix, GPU Operator

## API Endpoints Overview

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/organization/models` | List models with filtering |
| POST | `/v1/organization/models` | Create new model |
| GET | `/v1/organization/models/{model_id}` | Get specific model |
| DELETE | `/v1/organization/models/{model_id}` | Delete model |

## Endpoint Details

### 1. List Models

**GET** `/v1/organization/models`

List all models for the organization with optional filtering.

**Query Parameters:**
- `status` (optional): Filter by model status (pending, running, failed, etc.)
- `managed` (optional): Filter by managed status (true/false)

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
      "namespace": "jan-models",
      "deployment_name": "llama-7b",
      "service_name": "llama-7b",
      "tags": ["llama", "chat", "7b"],
      "managed": true,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 1
}
```

### 2. Create Model

**POST** `/v1/organization/models`

Create and deploy a new AI model with simplified configuration.

**Request Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Model identifier (used for Kubernetes resources) |
| display_name | string | No | Human-readable model name |
| description | string | No | Model description |
| image | string | Yes | Container image (e.g., "vllm/vllm-openai:latest") |
| command | array | Yes | Container command |
| gpu_count | integer | No | Number of GPUs (default: 1) |
| initial_delay_seconds | integer | No | Health check delay (default: 60) |
| storage_size | integer | No | PVC size in GB (default: 20) |
| storage_class | string | No | Storage class (uses cluster default if empty) |
| hugging_face_token | string | No | HF token for private models |
| tags | array | No | Model tags |

**Example Request:**

```json
{
  "name": "llama-7b",
  "display_name": "Llama 2 7B Chat",
  "description": "Llama 2 7B model for chat completion",
  "image": "vllm/vllm-openai:latest",
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--served-model-name", "llama-7b",
    "--model", "meta-llama/Llama-2-7b-chat-hf"
  ],
  "gpu_count": 1,
  "storage_size": 30,
  "tags": ["llama", "chat", "7b"]
}
```

**Response:**

```json
{
  "model": {
    "id": "llama-7b",
    "organization_id": 123,
    "display_name": "Llama 2 7B Chat",
    "description": "Llama 2 7B model for chat completion",
    "status": "pending",
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
    "namespace": "jan-models",
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

## Model Status Values

The following status values are returned for models:

- `pending`: Model deployment is being prepared
- `creating`: Kubernetes resources are being created
- `running`: Model is successfully deployed and ready
- `failed`: Model deployment failed
- `stopped`: Model has been stopped
- `crash_loop_back_off`: Model container is crashing repeatedly

## Automatic Defaults

The API automatically applies these default values:

```json
{
  "gpu_count": 1,
  "initial_delay_seconds": 60,
  "storage_size": 20,
  "cpu_request": "1",
  "memory_request": "2Gi",
  "image_pull_policy": "IfNotPresent",
  "port": 8000,
  "namespace": "jan-models"
}
```

## Kubernetes Resources Created

When you create a model, the following Kubernetes resources are automatically created:

1. **Deployment**: Runs the model container with specified GPU/CPU resources
2. **Service**: Exposes the model internally on port 8000
3. **PVC**: Optional persistent volume for model storage (per-model PVCs)
4. **ServiceMonitor**: Prometheus monitoring (if operator available)

## Resource Labels

All managed resources are labeled with:

- `app.kubernetes.io/managed-by: jan-server`
- `model.aibrix.ai/name: {model-name}`
- `jan-server.menlo.ai/organization: {organization-id}`

## Validation Requirements

### Served Model Name Validation

For proper integration with Aibrix autoscaling, the `--served-model-name` parameter in your command **must match** the model name:

```json
{
  "name": "llama-7b",
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--served-model-name", "llama-7b",
    "--model", "meta-llama/Llama-2-7b-chat-hf"
  ]
}
```

**Validation Error Example:**

```json
{
  "error": "--served-model-name 'wrong-name' must match model name 'llama-7b' (required by Aibrix)"
}
```

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

### 400 Bad Request - Invalid Served Model Name

```json
{
  "error": "--served-model-name parameter must match model name 'llama-7b'. Please ensure your command contains '--served-model-name llama-7b'"
}
```

### 404 Not Found - Model Not Found

```json
{
  "error": "model not found in organization"
}
```

### 400 Bad Request - Missing Required Fields

```json
{
  "error": "validation failed: command is required and must contain --served-model-name parameter"
}
```
