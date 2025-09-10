# LLM Load Tests with k6

This directory contains k6 scripts to load test the Jan Server's LLM-style chat completions API, including both streaming and non-streaming scenarios. The tests compare different prompt types ("short" vs "chain-of-thought") and record key latency metrics including TTFB (Time to First Byte), total duration, optional queue time, and tokens per second.

## What this test covers

- **Smoke tests**: `GET /v1/version`, `GET /v1/models`
- **Chat – non-stream**: Returns full JSON response after complete generation  
- **Chat – stream**: Server-Sent Events / chunked response for real-time streaming
- **Prompt variants**:
  - `short`: Concise instruction (no elaborate reasoning)
  - `cot`: Chain-of-thought style instruction (asks for step-by-step reasoning)

Metrics are tagged by scenario (`chat_nonstream` / `chat_stream`) and prompt type (`short` / `cot`), with TTFB thresholds applied only to successful responses (status=200).

## Prerequisites

- **k6 installed locally** (Node.js not required - k6 runs the script directly)
- **Or use Docker** if you prefer containerized execution

## Install k6

### macOS (Homebrew)

```bash
brew install k6
```

### Ubuntu/Debian

```bash
sudo apt-get update && sudo apt-get install -y gnupg ca-certificates
curl -fsSL https://dl.k6.io/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/k6-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install -y k6
```

### Windows

Download from [k6.io/docs/get-started/installation](https://k6.io/docs/get-started/installation)

## Files

- **`chat-completion.js`** — The main k6 script that defines:
  - Scenarios for smoke tests, non-stream, and stream
  - Prompt variants (short & chain-of-thought)  
  - Custom metrics and performance thresholds

## Quick start (local)

### Method 1: Using the bash script (Recommended)

1. **Configure environment (optional)**:

   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

2. **Run all test cases**:

   ```bash
   ./run-loadtest.sh
   ```

3. **Run a specific test case**:

   ```bash
   ./run-loadtest.sh chat-completion
   ```

4. **View available test cases**:

   ```bash
   ./run-loadtest.sh --list
   ```

### Method 2: Direct k6 execution

From the `tests/loadtests` directory:

```bash
BASE=https://api.jan.ai \
MODEL=jan-v1-4b \
NONSTREAM_RPS=3 \
STREAM_RPS=1 \
DURATION_MIN=1 \
k6 run chat-completion.js
```

### With authentication and Cloudflare bypass

```bash
API_KEY="your_api_key" \
LOADTEST_TOKEN="your_secret_bypass_token" \
BASE=https://api.jan.ai \
MODEL=jan-v1-4b \
NONSTREAM_RPS=3 \
STREAM_RPS=1 \
DURATION_MIN=1 \
k6 run chat-completion.js
```

The script will attach `Authorization: Bearer <API_KEY>` and `X-LoadTest-Token: <LOADTEST_TOKEN>` headers if provided.

## Run via Docker

You can run these tests without installing k6 locally.

### Option A: Use the provided Dockerfile (recommended)

Build the image once:

```bash
docker build -t janai/k6-tests:local ./tests
```

Run with your local tests mounted (so changes are picked up and results persist):

```bash
docker run --rm -it \
   -e BASE=https://api.jan.ai \
   -e MODEL=jan-v1-4b \
   -e NONSTREAM_RPS=3 \
   -e STREAM_RPS=1 \
   -e DURATION_MIN=1 \
   -e API_KEY=your_api_key \
   -e LOADTEST_TOKEN=your_secret_bypass_token \
   -e K6_PROMETHEUS_RW_SERVER_URL="$K6_PROMETHEUS_RW_SERVER_URL" \
   -e K6_PROMETHEUS_RW_USERNAME="$K6_PROMETHEUS_RW_USERNAME" \
   -e K6_PROMETHEUS_RW_PASSWORD="$K6_PROMETHEUS_RW_PASSWORD" \
   -e K6_PROMETHEUS_RW_TREND_STATS="p(95),p(99),min,max" \
   -e K6_PROMETHEUS_RW_PUSH_INTERVAL="5s" \
   -v "$PWD/tests":/tests \
   janai/k6-tests:local run-all
```

Run a specific test case (e.g., `chat-completion`):

```bash
docker run --rm -it \
   -e BASE=https://api.jan.ai \
   -e MODEL=jan-v1-4b \
   -e NONSTREAM_RPS=3 \
   -e STREAM_RPS=1 \
   -e DURATION_MIN=1 \
   -e API_KEY=your_api_key \
   -e LOADTEST_TOKEN=your_secret_bypass_token \
   -v "$PWD/tests":/tests \
   janai/k6-tests:local run chat-completion
```

Notes:

- If you don’t mount `/tests`, the container runs with a baked-in copy at `/app` (copied at build time).
- If a `.env` file exists inside the mounted `/tests`, it’s automatically loaded.

### Option B: Use upstream `grafana/k6` image

```bash
docker run --rm -it \
   -e BASE=https://api.jan.ai \
   -e MODEL=jan-v1-4b \
   -e NONSTREAM_RPS=3 \
   -e STREAM_RPS=1 \
   -e DURATION_MIN=1 \
   -e API_KEY=your_api_key \
   -e LOADTEST_TOKEN=your_secret_bypass_token \
   -v "$PWD/tests":/work -w /work \
   grafana/k6 run src/chat-completion.js
```

## Environment variables (knobs)

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE` | `https://api-dev.jan.ai` | Base URL for your Jan Server API |
| `MODEL` | `jan-v1-4b` | Model name passed to the API |
| `NONSTREAM_RPS` | `2` | Target arrival rate (req/s) for non-stream scenarios |
| `STREAM_RPS` | `1` | Target arrival rate (req/s) for stream scenarios (set 0 to disable) |
| `DURATION_MIN` | `5` | Test duration (minutes) for each scenario stage (ramped) |
| `API_KEY` | (empty) | Bearer token; adds `Authorization: Bearer ...` |
| `LOADTEST_TOKEN` | (empty) | Adds `X-LoadTest-Token` header (e.g., to bypass WAF rules) |

### Prometheus Remote Write Configuration (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `K6_PROMETHEUS_RW_SERVER_URL` | (empty) | Prometheus remote write endpoint (e.g., `https://prometheus.example.com/api/v1/write`) |
| `K6_PROMETHEUS_RW_USERNAME` | (empty) | Basic auth username for Prometheus endpoint |
| `K6_PROMETHEUS_RW_PASSWORD` | (empty) | Basic auth password for Prometheus endpoint |
| `K6_PROMETHEUS_RW_TREND_STATS` | `p(95),p(99),min,max` | Trend metrics to export to Prometheus |
| `K6_PROMETHEUS_RW_PUSH_INTERVAL` | `5s` | How often to push metrics to Prometheus |

**Note:** When Prometheus is configured, k6 will automatically export metrics using the `experimental-prometheus-rw` output.

Docker tip: pass these environment variables with `-e` flags as shown above. Make sure the endpoint is reachable from the container network.

## Disable scenarios

- **Disable stream scenarios**: Set `STREAM_RPS=0`
- **Disable non-stream scenarios**: Set `NONSTREAM_RPS=0`
- **Test only specific prompt types**: Comment out unwanted scenario blocks in the script

## What you'll see in the output

### Thresholds

- `http_req_failed` — Global error rate (must be < 2% by default)
- `llm_ttfb_ms{scenario:chat_stream,status:200,prompt:short} p(95)<1000` — Stream TTFB SLA for short prompts
- `llm_ttfb_ms{scenario:chat_stream,status:200,prompt:cot} p(95)<3000` — Stream TTFB SLA for chain-of-thought prompts

**Note**: TTFB metrics are filtered to `status=200` to avoid "passing" on error responses.

### Custom metrics (highlights)

- **`llm_ttfb_ms`** — Time-to-first-byte (aka time to first token for streaming)
- **`llm_receiving_ms`** — Time spent receiving data after TTFB (longer for streams with longer outputs)
- **`llm_total_ms`** — Total end-to-end duration per request
- **`llm_queue_ms`** — Parsed from `X-Queue-Time` header (if your backend sets it). Absent header = metric not recorded
- **`llm_tokens_per_sec`** — Completion tokens per second (if the response includes `usage.completion_tokens`)

### Built-in metrics

- **`http_req_duration{scenario:chat_nonstream}`** — Total time for non-stream responses (returns full JSON)
- **`http_req_duration{scenario:chat_stream}`** — Total time for stream to finish (first token + full stream duration)
- **`http_req_failed`** — Failure rate
- **Counts**: `http_reqs`, `iterations`, `VUs`, etc.

## Interpreting results

### User-perceived responsiveness (stream)

Look at `llm_ttfb_ms{scenario:chat_stream,status:200,prompt:*}`.

- `p95 < 1s` (short) and `p95 < 3s` (COT) are good starting SLAs.

### Full answer latency (non-stream)

Look at `http_req_duration{scenario:chat_nonstream}` p95.

- Expect higher values, because server returns only after the answer is fully generated.

### Throughput

`llm_tokens_per_sec` reflects generation speed (if usage is provided). Compare short vs COT prompts to quantify computational cost.

### Queue time

If your backend sets `X-Queue-Time`, use `llm_queue_ms` to separate queuing delay from compute/stream time.

## Common pitfalls & tips

- **File not found**: Run k6 from the directory that contains `chat-completion.js`, or pass a correct path (`k6 run ./tests/loadtests/chat-completion.js`)
- **Docker**: Mount your directory (`-v "$PWD":/work -w /work`) so the script is visible inside the container
- **Cloudflare/WAF interference**: If you load test production paths, use a scoped bypass header (e.g., `X-LoadTest-Token`) with a Cloudflare rule to skip WAF/bot/rate limits only for test traffic
- **Token safety**: If you exposed `LOADTEST_TOKEN` or API keys in a terminal/screenshot, rotate them
- **Prompt impact**: COT prompts usually increase total duration and reduce tokens/sec. Keep a "short baseline" test for SLA tracking

## Extending

- Add more prompt fixtures (longer input, bigger `max_tokens`) and attach a `prompt: name` tag
- Add thresholds for non-stream latency (e.g., `http_req_duration{scenario:chat_nonstream,prompt:short} p(95)<4000` if that's your SLA)
- Split network timing: Record `res.timings.connecting`, `tls_handshaking`, etc., into Trends if you need to isolate network vs compute

## Example commands

### Stream + non-stream for 1 minute, low RPS

```bash
BASE=https://api.jan.ai \
MODEL=jan-v1-4b \
NONSTREAM_RPS=3 \
STREAM_RPS=1 \
DURATION_MIN=1 \
k6 run chat-completion.js
```

### Stream-only quick check

```bash
STREAM_RPS=2 NONSTREAM_RPS=0 DURATION_MIN=1 \
k6 run chat-completion.js
```

### Non-stream only with higher RPS

```bash
NONSTREAM_RPS=5 STREAM_RPS=0 DURATION_MIN=2 \
k6 run chat-completion.js
```

### With auth and bypass header

```bash
API_KEY="sk-xxx" LOADTEST_TOKEN="lt-xxx" \
BASE=https://api.jan.ai \
MODEL=jan-v1-4b \
k6 run chat-completion.js
```

## CI/CD Integration

The repository includes a GitHub Actions workflow for automated load testing that can be triggered manually.

### Setup GitHub Secrets

Before using the CI workflow, configure the following repository secrets:

**Required secrets:**

- `LOADTEST_API_KEY` or `LOADTEST_TOKEN` - Authentication for the API

**Optional secrets (for Prometheus metrics - following k6 official docs):**

- `K6_PROMETHEUS_RW_SERVER_URL` - Prometheus remote write endpoint (e.g., `https://prometheus.example.com/api/v1/write`)
- `K6_PROMETHEUS_RW_USERNAME` - Basic auth username (optional)
- `K6_PROMETHEUS_RW_PASSWORD` - Basic auth password (optional)

**Optional variables (for advanced Prometheus config):**

- `K6_PROMETHEUS_RW_TREND_STATS` - Trend metrics to export (default: `p(95),p(99),min,max`)
- `K6_PROMETHEUS_RW_PUSH_INTERVAL` - Push interval (default: `5s`)

**Note:** Metrics are sent directly from k6 to Prometheus via the remote write protocol using k6's built-in experimental-prometheus-rw output.

### Running Tests via GitHub Actions

1. Go to your repository's **Actions** tab
2. Select the **Load Test** workflow
3. Click **Run workflow**
4. Configure the test parameters:
   - Test case: Leave empty to run **all tests**, or select specific test
   - Base URL
   - Model name
   - Duration in minutes
   - RPS settings

**Default behavior:** If no test case is selected, all tests will be executed automatically.

### Workflow Features

- **Manual trigger**: Workflow dispatch with configurable parameters
- **All tests by default**: Runs all available tests unless specific test is selected
- **Auto-detection**: Automatically discovers test cases from `src/` directory
- **Test results**: Automatically uploaded as artifacts
- **Direct metrics export**: k6 sends metrics directly to Prometheus remote write endpoint
- **Test ID tagging**: Each test run gets unique `testid` tag for metrics segmentation
- **Grafana dashboard**: Pre-built dashboard for monitoring LLM and HTTP metrics

## 📊 Grafana Dashboard

A comprehensive Grafana dashboard is provided for monitoring load test metrics:

- **Location**: `grafana-dashboard.json`
- **Setup Guide**: See `GRAFANA_SETUP.md` for detailed instructions
- **Features**:
  - LLM-specific metrics (TTFB, tokens/sec, queue time)
  - HTTP performance monitoring
  - Test segmentation by Test ID and Test Case
  - Real-time monitoring with 5s refresh

### Quick Dashboard Import

1. Open Grafana → Dashboards → New → Import
2. Upload `grafana-dashboard.json`
3. Configure Prometheus data source
4. Start monitoring your load tests!

### Adding New Test Cases

The system automatically detects test cases by scanning the `src/` directory for `.js` files.

To add new test cases:

1. **Create a new k6 script file** in the `src/` directory (e.g., `src/new-test.js`)
2. **Test immediately** - no registration needed:

   ```bash
   ./run-loadtest.sh new-test      # Test specific case
   ./run-loadtest.sh               # Test all cases including new one
   ```

3. **Update the workflow file** to include the new option in the dropdown

**Environment variables** are shared across all test cases, so new tests can use the same `.env` configuration.

**File structure:**

```text
tests/
├── src/                    # Test scripts directory
│   ├── chat-completion.js  # Auto-detected test
│   └── your-test.js        # Your new test (auto-detected)
├── run-loadtest.sh         # Test runner
└── .env                    # Shared environment
```

### Test against local development server

```bash
BASE=http://localhost:8080 \
MODEL=jan-v1-4b \
NONSTREAM_RPS=1 \
STREAM_RPS=1 \
DURATION_MIN=1 \
k6 run chat-completion.js
```

## Next steps

Happy testing! If needed, we can add:

- A minimal Makefile for common test configurations
- Docker Compose setup for integrated testing
- Additional test scenarios for specific use cases
- Performance regression tracking scripts

For questions or issues, check the main [Jan Server documentation](../../README.md) or create an issue in the repository.
