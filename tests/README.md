# LLM Load Tests with k6

This directory contains k6 scripts to load test the Jan Server's LLM-style chat completions API, including both streaming and non-streaming scenarios. The tests use **guest authentication** (no API keys required) and cover comprehensive conversation flows with conversation management, message persistence, and response validation.

## What this test covers

- **Authentication**: Guest login with automatic token refresh
- **Standard Completions**: Non-streaming and streaming chat completions
- **Conversation Management**: Create, retrieve, and list conversations
- **Message Persistence**: Store and retrieve conversation items
- **Response API**: Non-streaming and streaming responses with/without tools
- **Comprehensive Flow Testing**: End-to-end conversation lifecycle validation

### Test Categories

1. **`test-completion-standard.js`**: Basic completion flows (guest login, models, completions)
2. **`test-completion-conversation.js`**: Conversation management with both non-streaming and streaming messages
3. **`test-responses.js`**: Response API testing with tools and streaming support

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

- **`test-completion-standard.js`** — Basic completion flows:
  - Guest authentication with token refresh
  - Model listing and validation
  - Non-streaming and streaming completions
  - Comprehensive error handling and metrics

- **`test-completion-conversation.js`** — Conversation management flows:
  - Conversation creation and retrieval
  - Non-streaming and streaming message exchange
  - Conversation listing and item persistence
  - End-to-end conversation lifecycle testing

- **`test-responses.js`** — Response API testing:
  - Non-streaming and streaming responses
  - Tool integration testing with extended timeouts (5 minutes)
  - Response validation and completion detection
  - Separate metrics for tool call responses

## Quick start (local)

### Method 1: Using the batch script (Windows)

1. **Run individual tests**:

   ```batch
   # Test standard completions
   k6 run src\test-completion-standard.js
   
   # Test conversation flows
   k6 run src\test-completion-conversation.js
   
   # Test response API
   k6 run src\test-responses.js
   ```

2. **Run with custom configuration**:

   ```batch
   # Set environment variables
   set BASE=https://api-dev.jan.ai
   set MODEL=jan-v1-4b
   set DEBUG=true
   
   # Run test
   k6 run src\test-completion-conversation.js
   ```

### Method 2: Direct k6 execution

From the `tests/` directory:

```bash
# Basic completion test
BASE=https://api-dev.jan.ai \
MODEL=jan-v1-4b \
DEBUG=true \
k6 run src/test-completion-standard.js

# Conversation test with both streaming and non-streaming
BASE=https://api-dev.jan.ai \
MODEL=jan-v1-4b \
DEBUG=true \
k6 run src/test-completion-conversation.js

# Response API test
BASE=https://api-dev.jan.ai \
MODEL=jan-v1-4b \
DEBUG=true \
k6 run src/test-responses.js
```

### Method 3: Using the test runner script (Recommended for CI/CD)

```bash
# Run all tests
./run-loadtest.sh

# Run specific test
./run-loadtest.sh test-completion-conversation

# List available tests
./run-loadtest.sh --list
```

## Run via Docker

You can run these tests without installing k6 locally. The Docker setup has been tested and works on both Linux and Windows.

### Option A: Use the provided Dockerfile (recommended)

Build the image once:

```bash
docker build -t janai/k6-tests:local ./tests
```

Run with your local tests mounted (so changes are picked up and results persist):

**Using run-loadtest.sh script (Recommended):**
```bash
docker run --rm -it \
   -e BASE=https://api-stag.jan.ai \
   -e MODEL=jan-v1-4b \
   -e NONSTREAM_RPS=3 \
   -e STREAM_RPS=1 \
   -e DURATION_MIN=2 \
   -e K6_PROMETHEUS_RW_SERVER_URL="$K6_PROMETHEUS_RW_SERVER_URL" \
   -e K6_PROMETHEUS_RW_USERNAME="$K6_PROMETHEUS_RW_USERNAME" \
   -e K6_PROMETHEUS_RW_PASSWORD="$K6_PROMETHEUS_RW_PASSWORD" \
   -e K6_PROMETHEUS_RW_TREND_STATS="p(95),p(99),min,max" \
   -e K6_PROMETHEUS_RW_PUSH_INTERVAL="5s" \
   -v "$PWD/tests":/tests \
   janai/k6-tests:local run-all
```

Run a specific test case:

```bash
docker run --rm -it \
   -e BASE=https://api-stag.jan.ai \
   -e MODEL=jan-v1-4b \
   -e DEBUG=true \
   -v "$PWD/tests":/tests \
   janai/k6-tests:local run test-completion-conversation
```

### Docker Features

- **Alpine Linux base**: Lightweight and secure
- **Bash support**: Full bash scripting capabilities  
- **jq included**: JSON parsing for metrics
- **Line ending fixes**: Handles Windows CRLF → Unix LF conversion
- **Auto-detection**: Automatically finds and runs test scripts
- **Volume mounting**: Use local files with `-v` flag
- **Environment variables**: Full support for all test configuration

### Windows PowerShell Commands

```powershell
# Build the image
docker build -t janai/k6-tests:local .

# Run all tests
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local

# Run specific test
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local test-responses

# Run with volume mount
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true -v "${PWD}":/tests janai/k6-tests:local
```

Notes:

- If you don't mount `/tests`, the container runs with a baked-in copy at `/app` (copied at build time).
- If a `.env` file exists inside the mounted `/tests`, it's automatically loaded.
- The Docker image has been tested and works on both Linux and Windows environments.

### Option B: Use upstream `grafana/k6` image

```bash
docker run --rm -it \
   -e BASE=https://api-stag.jan.ai \
   -e MODEL=jan-v1-4b \
   -e DEBUG=true \
   -v "$PWD/tests":/work -w /work \
   grafana/k6 run src/test-completion-conversation.js
```

## Environment variables (knobs)

### Core Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE` | `https://api-stag.jan.ai` | Base URL for your Jan Server API |
| `MODEL` | `jan-v1-4b` | Model name passed to the API |
| `DEBUG` | `false` | Enable detailed request/response logging |
| `SINGLE_RUN` | `false` | Run tests in single iteration mode (for validation) |

### Load Testing Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `NONSTREAM_RPS` | `2` | Target arrival rate (req/s) for non-streaming scenarios |
| `STREAM_RPS` | `1` | Target arrival rate (req/s) for streaming scenarios (set 0 to disable) |
| `DURATION_MIN` | `5` | Test duration (minutes) for each scenario stage (ramped) |

### Authentication

**No API keys required!** All tests use **guest authentication**:

- Tests automatically perform guest login (`POST /v1/auth/guest-login`)
- Access tokens are refreshed before each request to prevent timeouts
- Refresh tokens are extracted from `Set-Cookie` headers
- No manual token management needed

## Testing Modes

### Functional Testing (`SINGLE_RUN=true`)
- **Purpose**: Validate API functionality and flow correctness
- **Behavior**: Runs each test step once
- **Use case**: Development, debugging, CI/CD validation
- **Performance**: Fast execution, detailed logging

### Load Testing (`SINGLE_RUN=false` or omitted)
- **Purpose**: Test API performance under load
- **Behavior**: Runs multiple iterations with specified RPS
- **Use case**: Performance testing, capacity planning, SLA validation
- **Performance**: Sustained load over specified duration

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

## Test Flow Details

### Standard Completion Test (`test-completion-standard.js`)
1. **Guest Login**: Authenticate and get access token
2. **Token Refresh**: Refresh access token to prevent timeouts
3. **List Models**: Validate available models
4. **Non-Streaming Completion**: Test standard chat completion
5. **Streaming Completion**: Test streaming with `data: [DONE]` detection

### Conversation Test (`test-completion-conversation.js`)
1. **Guest Login**: Authenticate and get access token
2. **Token Refresh**: Refresh access token
3. **Create Conversation**: Create new conversation
4. **Non-Streaming Message**: Add first message (non-streaming)
5. **Streaming Message**: Add second message (streaming)
6. **Get Conversation**: Retrieve conversation details
7. **List Conversations**: Get all conversations
8. **Get Conversation Items**: Retrieve stored messages

### Response API Test (`test-responses.js`)
1. **Guest Login**: Authenticate and get access token
2. **Token Refresh**: Refresh access token
3. **Non-Streaming Response**: Test without tools (30s timeout)
4. **Non-Streaming Response with Tools**: Test with tool integration (5min timeout)
5. **Streaming Response**: Test streaming without tools (30s timeout)
6. **Streaming Response with Tools**: Test streaming with tools (5min timeout)

## What you'll see in the output

### Test Results Summary

Each test provides comprehensive output including:

- **Step-by-step progress**: Clear indication of each test step
- **Success/failure indicators**: ✅ for success, ❌ for failures
- **Debug information**: Detailed request/response data when `DEBUG=true`
- **Performance metrics**: Response times, token counts, completion rates
- **Conversation details**: IDs, titles, message counts, and content previews

### Key Metrics

- **Authentication time**: Guest login and token refresh duration
- **Completion time**: Non-streaming and streaming response times
- **Conversation metrics**: Creation, retrieval, and listing performance
- **Error rates**: Failed requests and validation failures
- **Streaming metrics**: Chunk counts and completion detection

### Debug Mode

When `DEBUG=true`, you'll see:
- Full HTTP request details (method, URL, headers, body)
- Complete HTTP response details (status, headers, body)
- Parsed JSON data and extracted values
- Token refresh operations and cookie management

## Interpreting results

### Test Success Indicators

- **All steps completed**: Each test step shows ✅ success
- **Low error rates**: `http_req_failed` should be 0% or very low
- **Fast response times**: Completion times under thresholds
- **Proper streaming**: `data: [DONE]` signals received
- **Data persistence**: Conversation items stored and retrieved correctly

### Performance Expectations

- **Guest login**: Should complete in < 2 seconds
- **Token refresh**: Should complete in < 2 seconds  
- **Non-streaming completions**: 1-30 seconds depending on model and prompt length
- **Streaming completions**: First token in < 1 second, full completion varies
- **Conversation operations**: < 3 seconds for CRUD operations
- **Tool call responses**: Up to 5 minutes for complex tool operations

### Debugging Failed Tests

When tests fail, check:
1. **Network connectivity**: Can reach the API endpoint
2. **API availability**: Server is running and responding
3. **Model availability**: Specified model exists and is accessible
4. **Authentication**: Guest login is working (no API key issues)
5. **Rate limits**: Not hitting API rate limits

## Common pitfalls & tips

- **File not found**: Run k6 from the `tests/` directory, or pass correct paths (`k6 run src/test-completion-conversation.js`)
- **Docker**: Mount your directory (`-v "$PWD/tests":/work -w /work`) so scripts are visible inside the container
- **Docker line endings**: If you get `$'\r': command not found` errors, the script has Windows line endings - use the provided Dockerfile which handles this
- **Docker shell issues**: The provided Dockerfile uses bash instead of sh for full compatibility
- **Authentication issues**: Guest login should work automatically - no API keys needed
- **Token timeouts**: Tests automatically refresh tokens before each request
- **Streaming issues**: Ensure `data: [DONE]` signals are properly detected
- **Tool call timeouts**: Tool calls may take up to 5 minutes - this is normal and expected
- **Debug mode**: Use `DEBUG=true` to see detailed request/response information
- **Model availability**: Verify the specified model exists and is accessible
- **Rate limits**: Tests use single iterations by default to avoid rate limiting

## Extending

- **Add new test scenarios**: Create additional `.js` files in `src/` directory
- **Custom conversation flows**: Test specific conversation patterns or edge cases
- **Additional API endpoints**: Test other Jan Server endpoints beyond completions
- **Performance thresholds**: Add custom performance requirements for your use case
- **Integration testing**: Test with external services or tool integrations
- **Load testing**: Modify tests to run multiple iterations for performance testing

## Example commands

### Functional Testing (Single Run)

```bash
# Basic completion test
BASE=https://api-stag.jan.ai \
MODEL=jan-v1-4b \
DEBUG=true \
SINGLE_RUN=true \
k6 run src/test-completion-standard.js

# Conversation flow test
BASE=https://api-stag.jan.ai \
MODEL=jan-v1-4b \
DEBUG=true \
SINGLE_RUN=true \
k6 run src/test-completion-conversation.js
```

### Load Testing (Performance)

```bash
# Load test with moderate traffic
BASE=https://api-stag.jan.ai \
MODEL=jan-v1-4b \
NONSTREAM_RPS=3 \
STREAM_RPS=1 \
DURATION_MIN=2 \
k6 run src/test-completion-conversation.js

# High load test
BASE=https://api-stag.jan.ai \
MODEL=jan-v1-4b \
NONSTREAM_RPS=10 \
STREAM_RPS=5 \
DURATION_MIN=5 \
k6 run src/test-completion-conversation.js

# Streaming only test
BASE=https://api-stag.jan.ai \
MODEL=jan-v1-4b \
NONSTREAM_RPS=0 \
STREAM_RPS=3 \
DURATION_MIN=3 \
k6 run src/test-completion-conversation.js
```

### Test against local development server

```bash
# Functional test
BASE=http://localhost:8080 \
MODEL=jan-v1-4b \
DEBUG=true \
SINGLE_RUN=true \
k6 run src/test-completion-conversation.js

# Load test
BASE=http://localhost:8080 \
MODEL=jan-v1-4b \
NONSTREAM_RPS=2 \
STREAM_RPS=1 \
DURATION_MIN=1 \
k6 run src/test-completion-conversation.js
```

## CI/CD Integration

The repository includes a GitHub Actions workflow for automated load testing that uses the `run-loadtest.sh` script for all deployment and setup commands.

### Setup GitHub Secrets

**No authentication secrets required!** Tests use guest authentication automatically.

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
   - Base URL (default: `https://api-stag.jan.ai`)
   - Model name (default: `jan-v1-4b`)
   - Debug mode (default: `false`)

**Default behavior:** If no test case is selected, all tests will be executed automatically.

### Workflow Features

- **Manual trigger**: Workflow dispatch with configurable parameters
- **All tests by default**: Runs all available tests unless specific test is selected
- **Auto-detection**: Automatically discovers test cases from `src/` directory
- **Test results**: Automatically uploaded as artifacts
- **Guest authentication**: No API keys or tokens required
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
   k6 run src/new-test.js           # Test specific case
   ./run-loadtest.sh                # Test all cases including new one
   ```

3. **Update the workflow file** to include the new option in the dropdown

**Environment variables** are shared across all test cases, so new tests can use the same configuration.

**File structure:**

```text
tests/
├── src/                              # Test scripts directory
│   ├── test-completion-standard.js   # Standard completion tests
│   ├── test-completion-conversation.js # Conversation flow tests
│   ├── test-responses.js             # Response API tests
│   └── your-test.js                  # Your new test (auto-detected)
├── run-loadtest.sh                   # Test runner
└── .env                              # Shared environment
```

## Next steps

Happy testing! If needed, we can add:

- A minimal Makefile for common test configurations
- Docker Compose setup for integrated testing
- Additional test scenarios for specific use cases
- Performance regression tracking scripts

For questions or issues, check the main [Jan Server documentation](../../README.md) or create an issue in the repository.
