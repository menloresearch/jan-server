# How to Run Load Tests Locally

This guide explains how to run K6 load tests locally on your machine.

## 🚀 Quick Start

### Prerequisites

- **K6 installed** (see installation section below)
- **Docker** (optional, for Grafana monitoring)
- **Internet connection** (tests run against live APIs)

### Basic Test Run

```bash
# Run a single test
k6 run src/test-completion-standard.js

# Run with specific environment variables
BASE=https://api-dev.jan.ai MODEL=jan-v1-4b k6 run src/test-completion-standard.js
```

### Using Test Runner Scripts

```bash
# Linux/Mac
./run-loadtest.sh test-completion-standard

# Windows
.\run-loadtest.bat test-completion-standard
```

## 📋 Available Test Scenarios

### 1. Standard Completion Tests
```bash
k6 run src/test-completion-standard.js
```
**What it tests:**
- Guest authentication
- Token refresh
- Model listing
- Non-streaming completions
- Streaming completions

### 2. Conversation Management Tests
```bash
k6 run src/test-completion-conversation.js
```
**What it tests:**
- Guest authentication
- Conversation creation
- Message addition (non-streaming)
- Message addition (streaming)
- Conversation listing
- Conversation items retrieval

### 3. Response API Tests
```bash
k6 run src/test-responses.js
```
**What it tests:**
- Guest authentication
- Non-streaming responses (without tools)
- Non-streaming responses (with tools)
- Streaming responses (without tools)
- Streaming responses (with tools)

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the `tests` directory:

```bash
# API Configuration
BASE=https://api-dev.jan.ai
MODEL=jan-v1-4b

# Cloudflare Configuration (Required)
LOADTEST_TOKEN=your_cloudflare_token

# Test Configuration
DEBUG=true
DURATION_MIN=1
NONSTREAM_RPS=2
STREAM_RPS=1
SINGLE_RUN=true

# Optional: API Keys (not required for guest auth)
# API_KEY=your_api_key
```

### Test Parameters

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `BASE` | API base URL | `https://api-dev.jan.ai` | `https://api-stag.jan.ai` |
| `MODEL` | LLM model to test | `jan-v1-4b` | `gpt-oss-20b` |
| `LOADTEST_TOKEN` | Cloudflare load test token (required) | - | `cf_1234567890abcdef` |
| `DEBUG` | Enable debug logging | `false` | `true` |
| `DURATION_MIN` | Test duration in minutes | `1` | `5` |
| `NONSTREAM_RPS` | Non-streaming requests per second | `2` | `5` |
| `STREAM_RPS` | Streaming requests per second | `1` | `3` |
| `SINGLE_RUN` | Run once instead of load test | `false` | `true` |

### Custom Environment Variables

You can add test-specific variables to `.env`:

```bash
# Health check specific
HEALTH_RPS=10
HEALTH_TIMEOUT=30

# Your custom test specific  
YOUR_TEST_PARAM=value
```

Then access in your k6 script:

```javascript
const HEALTH_RPS = Number(__ENV.HEALTH_RPS || 5);
const YOUR_PARAM = __ENV.YOUR_TEST_PARAM || 'default';
```

**Important Notes:**
- **LOADTEST_TOKEN is required!** This token is necessary for Cloudflare API access
- Tests handle authentication automatically via guest login
- Refresh tokens are managed automatically to prevent timeouts

## 🔧 Installation

### Install K6

**macOS (Homebrew):**
```bash
brew install k6
```

**Ubuntu/Debian:**
```bash
sudo apt-get update && sudo apt-get install -y gnupg ca-certificates
curl -fsSL https://dl.k6.io/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/k6-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install -y k6
```

**Windows:**
Download from [k6.io/docs/get-started/installation](https://k6.io/docs/get-started/installation)

**Docker (Alternative):**
```bash
docker run --rm -i grafana/k6 run - <src/test-completion-standard.js
```

### Docker Testing with Custom Image

**Building the Docker Image:**
```bash
# From the tests directory
docker build -t janai/k6-tests:local .
```

**Running Tests with Docker:**
```bash
# Run all tests
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local

# Run specific test
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local test-responses

# Run with volume mount (for local development)
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true -v "${PWD}":/tests janai/k6-tests:local
```

**Windows PowerShell Commands:**
```powershell
# Build
docker build -t janai/k6-tests:local .

# Run all tests
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local

# Run specific test
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local test-responses
```

**Docker Features:**
- **Alpine Linux base**: Lightweight and secure
- **Bash support**: Full bash scripting capabilities
- **jq included**: JSON parsing for metrics
- **Auto-detection**: Automatically finds and runs test scripts
- **Volume mounting**: Use local files with `-v` flag
- **Environment variables**: Full support for all test configuration

### Verify Installation

```bash
k6 version
```

## 📊 Output Formats

### JSON Output
```bash
k6 run --out json=results/test-results.json src/test-completion-standard.js
```

### Console Output
```bash
k6 run src/test-completion-standard.js
```

### With Grafana Monitoring

```bash
# Start Grafana monitoring with Prometheus
./setup-monitoring.sh

# Run test with metrics automatically sent to Grafana
./run-test-with-monitoring.sh test-completion-standard
```

**Access Dashboards:**
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

### Custom Metrics
```bash
k6 run --out experimental-prometheus-rw src/test-completion-standard.js
```

## 🐛 Troubleshooting

### Common Issues

**1. "No connection could be made"**
- Check your internet connection
- Verify the `BASE` URL is correct
- Ensure the API server is running

**2. "Authentication failed"**
- Tests use guest authentication (no API key required)
- **Check LOADTEST_TOKEN**: Ensure it's set correctly for Cloudflare access
- Check if the API server supports guest login
- Verify the API endpoint is accessible

**3. "Model not found"**
- Check available models: `curl $BASE/v1/models`
- Update the `MODEL` environment variable
- Ensure the model is available on the server

**4. "Test timeout"**
- Increase timeout in test configuration
- Check API response times
- Reduce load (`NONSTREAM_RPS`, `STREAM_RPS`)

### Debug Mode

Enable debug logging to see detailed request/response information:

```bash
DEBUG=true k6 run src/test-completion-standard.js
```

### Verbose Output

```bash
k6 run --verbose src/test-completion-standard.js
```

## 📈 Performance Testing

### Load Test Configuration

```bash
# High load test
DURATION_MIN=5 NONSTREAM_RPS=10 STREAM_RPS=5 k6 run src/test-completion-standard.js

# Stress test
DURATION_MIN=10 NONSTREAM_RPS=20 STREAM_RPS=10 k6 run src/test-completion-standard.js
```

### Thresholds

Tests include performance thresholds:
- HTTP error rate < 5%
- Response times < 10 seconds
- Authentication time < 2 seconds

### Custom Thresholds

Modify thresholds in test files:
```javascript
thresholds: {
  'http_req_failed': ['rate<0.05'],
  'http_req_duration': ['p(95)<10000'],
  'completion_time_ms': ['p(95)<10000'],
}
```

## 🔄 Continuous Testing

### Run All Tests

```bash
# Run all test scenarios
./run-loadtest.sh test-completion-standard
./run-loadtest.sh test-completion-conversation  
./run-loadtest.sh test-responses
```

### Automated Testing

Create a test script:
```bash
#!/bin/bash
# run-all-tests.sh

echo "Running all load tests..."

./run-loadtest.sh test-completion-standard
./run-loadtest.sh test-completion-conversation
./run-loadtest.sh test-responses

echo "All tests completed!"
```

## 📝 Results Analysis

### Understanding Output

**Thresholds:**
- ✅ Green checkmark = Test passed
- ❌ Red X = Test failed

**Metrics:**
- `http_req_duration`: Response time statistics
- `http_req_failed`: Error rate
- `completion_time_ms`: Custom completion timing
- `checks`: Test validation results

### Saving Results

```bash
# Save to JSON file
k6 run --out json=results/my-test.json src/test-completion-standard.js

# Save to CSV
k6 run --out csv=results/my-test.csv src/test-completion-standard.js
```

## 🌐 Different Environments

### Development
```bash
BASE=https://api-dev.jan.ai k6 run src/test-completion-standard.js
```

### Staging
```bash
BASE=https://api-stag.jan.ai k6 run src/test-completion-standard.js
```

### Production
```bash
BASE=https://api.jan.ai k6 run src/test-completion-standard.js
```

## 📚 Additional Resources

- **K6 Documentation**: [k6.io/docs](https://k6.io/docs)
- **Test Scripts**: See `src/` directory
- **Adding New Tests**: See `HOW_TO_CREATE_NEW_TEST_SCENARIOS.md`
- **Monitoring Setup**: See `grafana/README.md`

## 🆘 Getting Help

1. **Check logs**: Enable `DEBUG=true` for detailed output
2. **Verify setup**: Run `k6 version` and check prerequisites
3. **Test connectivity**: Try `curl $BASE/v1/models`
4. **Review documentation**: Check test-specific README files
5. **Check issues**: Look at test thresholds and error messages
