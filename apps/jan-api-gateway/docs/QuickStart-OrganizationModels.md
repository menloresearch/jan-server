# Quick Start Guide: Organization Models API (Simplified)# Quick Start Guide: Organization Models API (Simplified)# Quick Start Guide: Organization Models API (Simplified)



## Prerequisites



- Application deployed in Kubernetes cluster## Prerequisites## Prerequisites

- At least one GPU node available (for GPU models)

- Application deployed in Kubernetes cluster- Application deployed in Kubernetes cluster

## Basic Usage

- At least one GPU node available (for GPU models)- At least one GPU node available (for GPU models)

### 1. List existing models



```bash

curl -X GET http://localhost:8080/api/v1/organization/models \## Basic Usage## Basic Usage

  -H "Authorization: Bearer your_jwt_token"

```



Expected response:### 1. List existing models### 1. List existing models



```json```bash

{

  "models": [```bashcurl -X GET http://localhost:8080/api/v1/organization/models \

    {

      "id": "llama-7b",curl -X GET http://localhost:8080/api/v1/organization/models \  -H "Authorization: Bearer your_jwt_token"

      "organization_id": 123,

      "display_name": "Llama 2 7B Chat",  -H "Authorization: Bearer your_jwt_token"```

      "status": "running",

      "managed": true```

    }

  ],Expected response:

  "total": 1

}Expected response:```json

```

```json{

### 2. Deploy your first model (Simplified API)

{  "models": [

```bash

curl -X POST http://localhost:8080/api/v1/organization/models \  "models": [    {

  -H "Authorization: Bearer your_jwt_token" \

  -H "Content-Type: application/json" \    {      "id": "llama-7b",

  -d '{

    "name": "llama-7b",      "id": "llama-7b",      "organization_id": 123,

    "display_name": "Llama 2 7B Chat",

    "description": "Llama 2 7B model for chat completion",      "organization_id": 123,      "display_name": "Llama 2 7B Chat",

    "image": "vllm/vllm-openai:latest",

    "huggingface_id": "meta-llama/Llama-2-7b-chat-hf",      "display_name": "Llama 2 7B Chat",      "status": "running",

    "hugging_face_token": "hf_your_token_here",

    "command": [      "status": "running",      "managed": true

      "python", "-m", "vllm.entrypoints.openai.api_server",

      "--host", "0.0.0.0",      "managed": true    }

      "--port", "8000",

      "--model", "meta-llama/Llama-2-7b-chat-hf",    }  ],

      "--served-model-name", "llama-7b"

    ],  ],  "total": 1

    "replicas": 1,

    "gpu_count": 1,  "total": 1}

    "initial_delay_seconds": 60,

    "storage_size": 20,}```

    "tags": ["llama", "chat", "7b"]

  }'```

```

### 2. Deploy your first model (Simplified API)

**⚠️ Important**: The `--served-model-name` parameter in the command **must match** the model `name` field. This validation is enforced and will return a 400 error if they don't match.

### 2. Deploy your first model (Simplified API)```bash

### 3. Deploy from YAML (Advanced)

curl -X POST http://localhost:8080/api/v1/organization/models \

```bash

curl -X POST http://localhost:8080/api/v1/organization/models/yaml \```bash  -H "Authorization: Bearer your_jwt_token" \

  -H "Authorization: Bearer your_jwt_token" \

  -H "Content-Type: application/json" \curl -X POST http://localhost:8080/api/v1/organization/models \  -H "Content-Type: application/json" \

  -d '{

    "name": "custom-model",  -H "Authorization: Bearer your_jwt_token" \  -d '{

    "display_name": "Custom Model",

    "description": "Custom deployment",  -H "Content-Type: application/json" \    "name": "llama-7b",

    "yaml_content": "apiVersion: apps/v1\nkind: Deployment\n...",

    "tags": ["custom"]  -d '{    "display_name": "Llama 2 7B Chat",

  }'

```    "name": "llama-7b",    "description": "Llama 2 7B model for chat completion",



### 4. Check deployment status    "display_name": "Llama 2 7B Chat",    "image": "vllm/vllm-openai:latest",



```bash    "description": "Llama 2 7B model for chat completion",    "huggingface_id": "meta-llama/Llama-2-7b-chat-hf",

curl -X GET http://localhost:8080/api/v1/organization/models/llama-7b \

  -H "Authorization: Bearer your_jwt_token"    "image": "vllm/vllm-openai:latest",    "hugging_face_token": "hf_your_token_here",

```

    "huggingface_id": "meta-llama/Llama-2-7b-chat-hf",    "command": ["python", "-m", "vllm.entrypoints.openai.api_server"],

### 5. Use the deployed model

    "hugging_face_token": "hf_your_token_here",    "replicas": 1,

Once status is "running", you can make requests to the model:

    "command": ["python", "-m", "vllm.entrypoints.openai.api_server"],    "gpu_count": 1,

```bash

curl -X POST http://llama-7b.default.svc.cluster.local:8000/v1/chat/completions \    "replicas": 1,    "initial_delay_seconds": 60,

  -H "Content-Type: application/json" \

  -d '{    "gpu_count": 1,    "storage_size": 20,

    "model": "llama-7b",

    "messages": [    "initial_delay_seconds": 60,    "tags": ["llama", "chat", "7b"]

      {"role": "user", "content": "Hello, how are you?"}

    ]    "storage_size": 20,  }'

  }'

```    "tags": ["llama", "chat", "7b"]```



### 6. Delete model when done  }'



```bash```### 3. Deploy from YAML (Advanced)

curl -X DELETE http://localhost:8080/api/v1/organization/models/llama-7b \

  -H "Authorization: Bearer your_jwt_token"```bash

```

### 3. Deploy from YAML (Advanced)curl -X POST http://localhost:8080/api/v1/organization/models/yaml \

## Common Examples

  -H "Authorization: Bearer your_jwt_token" \

### Simple Text Generation Model

```bash  -H "Content-Type: application/json" \

```json

{curl -X POST http://localhost:8080/api/v1/organization/models/yaml \  -d '{

  "name": "qwen-7b",

  "display_name": "Qwen 7B",  -H "Authorization: Bearer your_jwt_token" \    "name": "custom-model",

  "image": "vllm/vllm-openai:latest",

  "huggingface_id": "Qwen/Qwen2.5-7B-Instruct",  -H "Content-Type: application/json" \    "display_name": "Custom Model",

  "command": [

    "python", "-m", "vllm.entrypoints.openai.api_server",  -d '{    "description": "Custom deployment",

    "--served-model-name", "qwen-7b"

  ],    "name": "custom-model",    "yaml_content": "apiVersion: apps/v1\nkind: Deployment\n...",

  "gpu_count": 1,

  "tags": ["qwen", "generation"]    "display_name": "Custom Model",    "tags": ["custom"]

}

```    "description": "Custom deployment",  }'



### Code Generation Model    "yaml_content": "apiVersion: apps/v1\nkind: Deployment\n...",```



```json    "tags": ["custom"]

{

  "name": "codellama-13b",  }'### 4. Check deployment status

  "display_name": "CodeLlama 13B",

  "image": "vllm/vllm-openai:latest", ``````bash

  "huggingface_id": "codellama/CodeLlama-13b-Instruct-hf",

  "command": [curl -X GET http://localhost:8080/v1/organization/models \

    "python", "-m", "vllm.entrypoints.openai.api_server",

    "--served-model-name", "codellama-13b"### 4. Check deployment status  -H "Authorization: Bearer your_jwt_token"

  ],

  "gpu_count": 2,```

  "storage_size": 30,

  "tags": ["code", "llama"]```bash

}

```curl -X GET http://localhost:8080/api/v1/organization/models/llama-7b \### 4. Use the deployed model



## Validation Rules  -H "Authorization: Bearer your_jwt_token"Once status shows "running", you can send requests to:



### Served Model Name Validation``````



The API enforces that `--served-model-name` in the command must match the model `name`:http://llama2-7b-chat.jan-models.svc.cluster.local:8000/v1/chat/completions



#### ✅ Valid Examples### 5. Use the deployed model```



```json

{

  "name": "my-model",Once status is "running", you can make requests to the model:## Common Patterns

  "command": ["python", "-m", "vllm.entrypoints.openai.api_server", "--served-model-name", "my-model"]

}

```

```bash### Chat Model with Autoscaling

#### ❌ Invalid Examples

curl -X POST http://llama-7b.default.svc.cluster.local:8000/v1/chat/completions \

**Missing --served-model-name:**

  -H "Content-Type: application/json" \```json

```json

{  -d '{{

  "name": "my-model",

  "command": ["python", "-m", "vllm.entrypoints.openai.api_server"]    "model": "llama-7b",  "name": "qwen3-coder-30b",

}

```    "messages": [  "model_type": "completion", 



Error: `--served-model-name parameter is required in command`      {"role": "user", "content": "Hello, how are you?"}  "deployment_config": {



**Mismatched name:**    ]    "image": "registry.menlo.ai/dockerhub/vllm/vllm-openai:v0.10.2",



```json  }'    "args": [

{

  "name": "my-model",```      "python3", "-m", "vllm.entrypoints.openai.api_server",

  "command": ["python", "-m", "vllm.entrypoints.openai.api_server", "--served-model-name", "different-name"]

}      "--model", "Qwen/Qwen3-Coder-30B-A3B-Instruct-FP8",

```

### 6. Delete model when done      "--served-model-name", "qwen3-coder-30b"

Error: `--served-model-name 'different-name' must match model name 'my-model'`

    ],

## Simplified API Benefits

```bash    "gpu_count": 2,

### What Changed

curl -X DELETE http://localhost:8080/api/v1/organization/models/llama-7b \    "enable_autoscaling": true,

- **From 50+ fields** → **12 essential fields**

- **Automatic defaults** for common settings  -H "Authorization: Bearer your_jwt_token"    "autoscaling_config": {

- **Template-based** deployment

- **YAML upload** for advanced cases```      "min_replicas": 1,

- **Strict validation** for --served-model-name

      "max_replicas": 5,

### Default Values

## Common Examples      "target_value": "20"

```yaml

replicas: 1    }

gpu_count: 1

initial_delay_seconds: 60### Simple Text Generation Model  }

storage_size: 20

image_pull_policy: "IfNotPresent"}

port: 8000

cpu: "1"```json```

memory: "2Gi"

```{



## Troubleshooting  "name": "qwen-7b",**Important**: Note that `--served-model-name qwen3-coder-30b` matches the model name exactly. This is required by Aibrix for autoscaling.



### Model stuck in "creating" status  "display_name": "Qwen 7B",



Check pod events:  "image": "vllm/vllm-openai:latest",### Private Model with HuggingFace Token



```bash  "huggingface_id": "Qwen/Qwen2.5-7B-Instruct",```json

kubectl describe pod -l app=llama-7b -n default

```  "gpu_count": 1,{



### Validation errors  "tags": ["qwen", "generation"]  "name": "private-model",



Check that your command includes the correct `--served-model-name` parameter that matches your model name.}  "deployment_config": {



### GPU not available```    "hugging_face_token": "hf_your_token_here",



The API will automatically detect GPU requirements and fail gracefully if insufficient resources.    "enable_pvc": true



### YAML deployment failed### Code Generation Model  }



Check the logs:}



```bash```json```

kubectl logs -l app=custom-model -n default

```{

  "name": "codellama-13b",## Troubleshooting

  "display_name": "CodeLlama 13B",

  "image": "vllm/vllm-openai:latest", ### Model stuck in "creating" status

  "huggingface_id": "codellama/CodeLlama-13b-Instruct-hf",Check cluster events:

  "gpu_count": 2,```bash

  "storage_size": 30,kubectl get events -n jan-models --sort-by='.lastTimestamp'

  "tags": ["code", "llama"]```

}

```### Insufficient GPU resources

The API will return specific error:

## Simplified API Benefits```json

{

### What Changed  "error": "no GPU node can satisfy the requirements: min 2 GPUs with 48Gi VRAM"

}

- **From 50+ fields** → **12 essential fields**```

- **Automatic defaults** for common settings

- **Template-based** deployment### Missing CRDs

- **YAML upload** for advanced casesInstall required operators first:

```bash

### Default Values# Install Aibrix operator

kubectl apply -f https://github.com/aibrix-ai/operator/releases/latest/download/install.yaml

```yaml

replicas: 1# Install GPU Operator  

gpu_count: 1kubectl apply -f https://raw.githubusercontent.com/NVIDIA/gpu-operator/main/deployments/gpu-operator.yaml

initial_delay_seconds: 60```
storage_size: 20
image_pull_policy: "IfNotPresent"
port: 8000
cpu: "1"
memory: "2Gi"
```

## Troubleshooting

### Model stuck in "creating" status

Check pod events:

```bash
kubectl describe pod -l app=llama-7b -n default
```

### GPU not available

The API will automatically detect GPU requirements and fail gracefully if insufficient resources.

### YAML deployment failed

Check the logs:

```bash
kubectl logs -l app=custom-model -n default
```