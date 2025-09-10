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

Mount your working directory so the script is visible inside the container:

```bash
docker run --rm -it \
  -e BASE=https://api.jan.ai \
  -e MODEL=jan-v1-4b \
  -e NONSTREAM_RPS=3 \
  -e STREAM_RPS=1 \
  -e DURATION_MIN=1 \
  -e API_KEY=your_api_key \
  -e LOADTEST_TOKEN=your_secret_bypass_token \
  -v "$PWD":/work -w /work \
  grafana/k6 run /work/chat-completion.js
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
