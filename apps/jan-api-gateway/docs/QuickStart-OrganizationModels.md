# Quick Start Guide: Organization Models API

## Prerequisites
- Application deployed in Kubernetes cluster
- Required CRDs installed (Aibrix, GPU Operator)
- At least one GPU node available

## Basic Usage

### 1. Check if cluster is ready
```bash
curl -X GET http://localhost:8080/v1/organization/models/status \
  -H "Authorization: Bearer your_jwt_token"
```

Expected response:
```json
{
  "organization_id": 123,
  "cluster_status": {
    "has_gpus": true,
    "total_gpus": 4,
    "aibrix_ok": true,
    "gpu_operator_ok": true
  }
}
```

### 2. Deploy your first model
```bash
curl -X POST http://localhost:8080/v1/organization/models \
  -H "Authorization: Bearer your_jwt_token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "llama2-7b-chat",
    "display_name": "Llama 2 7B Chat",
    "model_type": "chat",
    "huggingface_id": "meta-llama/Llama-2-7b-chat-hf",
    "deployment_config": {
      "image": "registry.menlo.ai/dockerhub/vllm/vllm-openai:v0.10.2",
      "gpu_count": 1,
      "enable_autoscaling": true
    }
  }'
```

### 3. Check deployment status
```bash
curl -X GET http://localhost:8080/v1/organization/models \
  -H "Authorization: Bearer your_jwt_token"
```

### 4. Use the deployed model
Once status shows "running", you can send requests to:
```
http://llama2-7b-chat.jan-models.svc.cluster.local:8000/v1/chat/completions
```

## Common Patterns

### Chat Model with Autoscaling

```json
{
  "name": "qwen3-coder-30b",
  "model_type": "completion", 
  "deployment_config": {
    "image": "registry.menlo.ai/dockerhub/vllm/vllm-openai:v0.10.2",
    "args": [
      "python3", "-m", "vllm.entrypoints.openai.api_server",
      "--model", "Qwen/Qwen3-Coder-30B-A3B-Instruct-FP8",
      "--served-model-name", "qwen3-coder-30b"
    ],
    "gpu_count": 2,
    "enable_autoscaling": true,
    "autoscaling_config": {
      "min_replicas": 1,
      "max_replicas": 5,
      "target_value": "20"
    }
  }
}
```

**Important**: Note that `--served-model-name qwen3-coder-30b` matches the model name exactly. This is required by Aibrix for autoscaling.

### Private Model with HuggingFace Token
```json
{
  "name": "private-model",
  "deployment_config": {
    "hugging_face_token": "hf_your_token_here",
    "enable_pvc": true
  }
}
```

## Troubleshooting

### Model stuck in "creating" status
Check cluster events:
```bash
kubectl get events -n jan-models --sort-by='.lastTimestamp'
```

### Insufficient GPU resources
The API will return specific error:
```json
{
  "error": "no GPU node can satisfy the requirements: min 2 GPUs with 48Gi VRAM"
}
```

### Missing CRDs
Install required operators first:
```bash
# Install Aibrix operator
kubectl apply -f https://github.com/aibrix-ai/operator/releases/latest/download/install.yaml

# Install GPU Operator  
kubectl apply -f https://raw.githubusercontent.com/NVIDIA/gpu-operator/main/deployments/gpu-operator.yaml
```