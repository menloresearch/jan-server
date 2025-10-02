# Served Model Name Validation Examples

## Valid Request Examples

### 1. ✅ Correct --served-model-name
```json
{
  "name": "llama-7b",
  "display_name": "Llama 2 7B Chat",
  "image": "vllm/vllm-openai:latest",
  "hugging_face_token": "hf_your_token_here",
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--host", "0.0.0.0",
    "--port", "8000", 
    "--model", "meta-llama/Llama-2-7b-chat-hf",
    "--served-model-name", "llama-7b"
  ],
  "gpu_count": 1,
  "storage_class": "fast-ssd"
}
```

### 2. ✅ Without optional fields (recommended)

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

- `hugging_face_token`: If not provided, no `HF_TOKEN` environment variable will be set
- `storage_class`: If not provided, Kubernetes will use the default storage class

## Invalid Request Examples

### 1. ❌ Missing --served-model-name

```json
{
  "name": "llama-7b",
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--model", "meta-llama/Llama-2-7b-chat-hf"
  ]
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "validation failed: --served-model-name parameter is required in command. Please add --served-model-name llama-7b to your command array"
}
```

### 2. ❌ Mismatched --served-model-name
```json
{
  "name": "llama-7b",
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--served-model-name", "wrong-name",
    "--model", "meta-llama/Llama-2-7b-chat-hf"
  ]
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "validation failed: --served-model-name 'wrong-name' must match model name 'llama-7b'. Please update your command to use --served-model-name llama-7b"
}
```

### 3. ❌ --served-model-name without value
```json
{
  "name": "llama-7b", 
  "command": [
    "python", "-m", "vllm.entrypoints.openai.api_server",
    "--served-model-name"
  ]
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "validation failed: --served-model-name parameter found but no value provided. Usage: --served-model-name llama-7b"
}
```

### 4. ❌ Empty command array
```json
{
  "name": "llama-7b",
  "command": []
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "validation failed: command is required and must contain --served-model-name parameter"
}
```

## Why This Validation Is Important

1. **Model Identification**: The `--served-model-name` parameter is used by clients to identify the model when making requests
2. **Autoscaling**: Some autoscaling systems rely on this parameter to match models correctly
3. **Consistency**: Ensures the model name in the API matches what's actually served by the model server
4. **API Compatibility**: Maintains compatibility with OpenAI API format where model names must match

## Testing the Validation

```bash
# Test invalid request
curl -X POST http://localhost:8080/api/v1/organization/models \
  -H "Authorization: Bearer your_jwt_token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-model",
    "display_name": "Test Model",
    "image": "vllm/vllm-openai:latest",
    "huggingface_id": "microsoft/DialoGPT-medium",
    "command": [
      "python", "-m", "vllm.entrypoints.openai.api_server",
      "--served-model-name", "wrong-name"
    ]
  }'

# Expected response: 400 Bad Request with validation error
```

```bash
# Test valid request  
curl -X POST http://localhost:8080/api/v1/organization/models \
  -H "Authorization: Bearer your_jwt_token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-model",
    "display_name": "Test Model", 
    "image": "vllm/vllm-openai:latest",
    "huggingface_id": "microsoft/DialoGPT-medium",
    "command": [
      "python", "-m", "vllm.entrypoints.openai.api_server",
      "--served-model-name", "test-model"
    ]
  }'

# Expected response: 201 Created with model details
```