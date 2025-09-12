# Jan Inference Model

A high-performance inference service for Jan models using vLLM, providing OpenAI-compatible API endpoints for seamless integration.

## Quick Start with Docker

The fastest way to get started is using Docker, which provides a pre-configured environment with all dependencies.

### Building Docker Images

#### Prerequisites for Building

- Docker installed and running
- NVIDIA Docker runtime (for production image with GPU support)
- At least 8GB free disk space for the production image

#### Production Image (vLLM with Jan Model)

Build the production image with vLLM and Jan v1-4B model:

```bash
# Build the production image
docker build -t jan-inference-model:latest .

# Build with specific tag/version
docker build -t jan-inference-model:v1.0.0 .

```

**Note**: The production build will download the Jan v1-4B model (~8GB) during the build process, which may take some time depending on your internet connection.

#### Mock Image (Development/Testing)

Build the lightweight mock server for development:

```bash
# Build the mock image
docker build -f Dockerfile.mock -t jan-inference-model:mock .

```

### Running Docker Containers

#### Production Setup

Run the Jan v1-4B model with vLLM:

```bash
docker run -d \
  --name jan-inference \
  --gpus all \
  -p 8101:8101 \
  jan-inference-model:latest
```

#### Development/Testing Setup

For development or testing without GPU requirements, use the mock server:

```bash
docker run -d \
  --name jan-inference-mock \
  -p 8101:8101 \
  jan-inference-model:mock
```

### Checking if vLLM is Ready

After starting the container, check the logs to see when vLLM is ready:

```bash
# Check container logs
docker logs jan-inference

# Follow logs in real-time
docker logs -f jan-inference
```

## Manual Setup (Alternative)

If you prefer to set up vLLM manually, follow these steps:

### Prerequisites

- Python 3.8+
- CUDA-compatible GPU (recommended)
- At least 8GB GPU memory for Jan v1-4B model

### Installation

1. **Install vLLM** (choose one method):

   **Using pip:**
   ```bash
   pip install vllm
   ```

   **Using uv (recommended):**
   ```bash
   uv pip install vllm --torch-backend=auto
   ```

   **Using conda:**
   ```bash
   conda create -n jan-inference python=3.12 -y
   conda activate jan-inference
   pip install --upgrade uv
   uv pip install vllm --torch-backend=auto
   ```

2. **Download the Jan model:**
   ```bash
   huggingface-cli download janhq/Jan-v1-4B --local-dir ./models/Jan-v1-4B
   ```

### Running the Server

Start the vLLM server with the Jan model:

```bash
vllm serve ./models/Jan-v1-4B \
  --served-model-name jan-v1-4b \
  --host 0.0.0.0 \
  --port 8101 \
  --max-num-batched-tokens 1024 \
  --enable-auto-tool-choice \
  --tool-call-parser hermes \
  --reasoning-parser qwen3
```

## API Usage

The service provides OpenAI-compatible endpoints:

### Chat Completions

```bash
curl http://localhost:8101/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "temperature": 0.7,
    "max_tokens": 100
  }'
```

### Python Client Example

```python
from openai import OpenAI

client = OpenAI(
    api_key="EMPTY",
    base_url="http://localhost:8101/v1"
)

response = client.chat.completions.create(
    model="jan-v1-4b",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Tell me about artificial intelligence."}
    ],
    temperature=0.7,
    max_tokens=150
)

print(response.choices[0].message.content)
```

## Configuration

### Environment Variables

- `VLLM_API_KEY`: Set API key for authentication (optional)
- `VLLM_USE_MODELSCOPE`: Set to `True` to use ModelScope instead of Hugging Face
- `VLLM_ATTENTION_BACKEND`: Manually set attention backend (`FLASH_ATTN`, `FLASHINFER`, or `XFORMERS`)

### Server Arguments

Key vLLM server arguments for Jan models:

- `--model`: Path to the model directory
- `--served-model-name`: Name for the model in API responses
- `--max-num-batched-tokens`: Maximum tokens per batch (adjust based on GPU memory)
- `--enable-auto-tool-choice`: Enable automatic tool selection
- `--tool-call-parser`: Parser for tool calls (`hermes` recommended)
- `--reasoning-parser`: Parser for reasoning outputs (`qwen3` recommended)

## Performance Optimization

### Memory Management

- Adjust `--max-num-batched-tokens` based on your GPU memory
- Use `--gpu-memory-utilization` to control GPU memory usage
- Consider quantization for lower memory usage

### Attention Backends

vLLM automatically selects the best attention backend, but you can manually configure:

```bash
export VLLM_ATTENTION_BACKEND=FLASH_ATTN  # or FLASHINFER, XFORMERS
```

## Troubleshooting

### Common Issues

1. **CUDA Out of Memory**: Reduce `--max-num-batched-tokens` or use a smaller model
2. **Model Loading Errors**: Ensure the model path is correct and accessible
3. **Port Already in Use**: Change the port with `--port` argument

### Logs

Check container logs:
```bash
docker logs jan-inference
```

## Development

### Mock Server

For development without GPU requirements, the mock server provides a lightweight alternative:

```bash
# Run mock server locally
cd application
pip install -r requirements.txt
uvicorn mockserver:app --host 0.0.0.0 --port 8101
```

### Advanced Docker Build Options

#### Build Optimization

For faster builds and smaller images:

```bash
# Build with no cache (clean build)
docker build --no-cache -t jan-inference-model:latest .

# Build with specific Dockerfile
docker build -f Dockerfile -t jan-inference-model:latest .

# Build and tag multiple versions
docker build -t jan-inference-model:latest -t jan-inference-model:stable .
```

#### Verifying Builds

After building, verify your images:

```bash
# List built images
docker images | grep jan-inference-model

# Check image size and details
docker inspect jan-inference-model:latest

# Test the built image
docker run --rm jan-inference-model:latest --help
```

## References

- [vLLM Documentation](https://docs.vllm.ai/en/latest/getting_started/quickstart.html)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [Jan Model Hub](https://huggingface.co/janhq)