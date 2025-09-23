# Jan Server Load Tests

Comprehensive K6 load testing framework for the Jan Server API, including authentication, completions, conversations, and response endpoints.

## 🚀 Quick Start

### Run Tests Locally
```bash
# Basic test run
k6 run src/test-completion-standard.js

# Using test runner
./run-loadtest.sh test-completion-standard
```

### With Monitoring
```bash
# Start Grafana monitoring with Prometheus
./setup-monitoring.sh

# Run test with metrics automatically sent to Grafana
./run-test-with-monitoring.sh test-completion-standard
```

## 📚 Documentation

- **[HOW_TO_RUN_TESTS_LOCALLY.md](HOW_TO_RUN_TESTS_LOCALLY.md)** - Complete guide for running tests locally
- **[HOW_TO_CREATE_NEW_TEST_SCENARIOS.md](HOW_TO_CREATE_NEW_TEST_SCENARIOS.md)** - Guide for creating new test scenarios
- **[grafana/README.md](grafana/README.md)** - Grafana monitoring setup and usage

## 🧪 Test Scenarios

### 1. Standard Completion Tests (`test-completion-standard.js`)
- Guest authentication with token refresh
- Model listing and validation
- Non-streaming chat completions
- Streaming chat completions

### 2. Conversation Management Tests (`test-completion-conversation.js`)
- Conversation creation and management
- Message addition (non-streaming and streaming)
- Conversation listing and retrieval
- Message persistence validation

### 3. Response API Tests (`test-responses.js`)
- Non-streaming responses (with/without tools)
- Streaming responses (with/without tools)
- Tool call handling and validation

## ⚙️ Configuration

### Environment Variables
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
```

### Test Parameters
| Variable | Description | Default |
|----------|-------------|---------|
| `BASE` | API base URL | `https://api-dev.jan.ai` |
| `MODEL` | LLM model to test | `jan-v1-4b` |
| `LOADTEST_TOKEN` | Cloudflare load test token (required) | - |
| `DEBUG` | Enable debug logging | `false` |
| `DURATION_MIN` | Test duration (minutes) | `1` |
| `NONSTREAM_RPS` | Non-streaming RPS | `2` |
| `STREAM_RPS` | Streaming RPS | `1` |
| `SINGLE_RUN` | Run once vs load test | `false` |

## 📊 Monitoring

### Grafana Dashboard
- **Status**: ✅ **Working with Prometheus integration**
- **Location**: `grafana/grafana-dashboard.json`
- **Setup**: `./setup-monitoring.sh`
- **Access**: http://localhost:3000 (admin/admin)
- **Metrics**: Automatically sent to Prometheus and displayed in Grafana

### Available Metrics
- HTTP performance metrics (response time, throughput, error rates)
- Custom completion timing metrics
- Test segmentation by Test ID and Test Case
- Real-time monitoring with 5s refresh

## 🔧 Installation

### K6 Installation

**macOS:**
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

### Docker (Alternative)
   ```bash
docker run --rm -i grafana/k6 run - <src/test-completion-standard.js
   ```

## 🏃‍♂️ Running Tests

### Basic Execution
   ```bash
# Single test
k6 run src/test-completion-standard.js

# All tests
./run-loadtest.sh test-completion-standard
./run-loadtest.sh test-completion-conversation
./run-loadtest.sh test-responses
```

### With Custom Configuration
```bash
BASE=https://api-stag.jan.ai MODEL=gpt-oss-20b k6 run src/test-completion-standard.js
```

### Load Testing
```bash
DURATION_MIN=5 NONSTREAM_RPS=10 STREAM_RPS=5 ./run-loadtest.sh test-completion-standard
```

## 📈 Performance Thresholds

Tests include built-in performance thresholds:
- HTTP error rate < 5%
- Response times < 10 seconds
- Authentication time < 2 seconds
- Custom completion timing thresholds

## 🌐 Environment Support

### Development
```bash
BASE=https://api-dev.jan.ai ./run-loadtest.sh test-completion-standard
```

### Staging
```bash
BASE=https://api-stag.jan.ai ./run-loadtest.sh test-completion-standard
```

### Production
```bash
BASE=https://api.jan.ai ./run-loadtest.sh test-completion-standard
```

## 📁 Project Structure

```
tests/
├── src/                                    # Test scripts
│   ├── test-completion-standard.js         # Basic completion flows
│   ├── test-completion-conversation.js     # Conversation management
│   └── test-responses.js                   # Response API testing
├── grafana/                                # Monitoring setup
│   ├── README.md                           # Grafana documentation
│   ├── docker-compose.yml                  # Monitoring stack
│   ├── grafana-dashboard.json              # Pre-built dashboard
│   └── prometheus.yml                      # Prometheus config
├── results/                                # Test results
├── HOW_TO_RUN_TESTS_LOCALLY.md             # Local testing guide
├── HOW_TO_CREATE_NEW_TEST_SCENARIOS.md     # New test creation guide
├── setup-monitoring.sh                     # Monitoring setup script
├── setup-monitoring.bat                    # Windows monitoring setup
├── run-test-with-monitoring.sh             # Test runner with metrics
├── run-test-with-monitoring.bat            # Windows test runner with metrics
├── run-loadtest.sh                         # Test runner script
└── README.md                               # This file
```

## 🔍 Troubleshooting

### Common Issues
1. **Connection errors**: Check internet connection and API URL
2. **Authentication failures**: Tests use guest auth (no API key needed)
3. **Model not found**: Verify model availability with `curl $BASE/v1/models`
4. **Timeouts**: Reduce load or increase timeout thresholds

### Debug Mode
```bash
DEBUG=true ./run-loadtest.sh test-completion-standard
```

### Verbose Output
```bash
k6 run --verbose src/test-completion-standard.js
```

## 📊 Results Analysis

### Understanding Output
- ✅ Green checkmark = Test passed
- ❌ Red X = Test failed
- Metrics show response times, error rates, and custom timing

### Saving Results
   ```bash
# JSON output
k6 run --out json=results/my-test.json src/test-completion-standard.js

# CSV output
k6 run --out csv=results/my-test.csv src/test-completion-standard.js
```

## 🤝 Contributing

### Adding New Tests
1. Follow the guide in `HOW_TO_CREATE_NEW_TEST_SCENARIOS.md`
2. Use existing tests as templates
3. Include proper error handling and metrics
4. Update test runner scripts
5. Document your test scenario

### Best Practices
- Single responsibility per test file
- Clear naming and documentation
- Reasonable performance thresholds
- Comprehensive error handling
- Consistent test structure

## 📚 Additional Resources

- **K6 Documentation**: [k6.io/docs](https://k6.io/docs)
- **Local Testing Guide**: [HOW_TO_RUN_TESTS_LOCALLY.md](HOW_TO_RUN_TESTS_LOCALLY.md)
- **New Test Creation**: [HOW_TO_CREATE_NEW_TEST_SCENARIOS.md](HOW_TO_CREATE_NEW_TEST_SCENARIOS.md)
- **Monitoring Setup**: [grafana/README.md](grafana/README.md)

## 🆘 Support

1. **Check documentation**: Review the specific guides above
2. **Enable debug mode**: Use `DEBUG=true` for detailed output
3. **Verify setup**: Run `k6 version` and check prerequisites
4. **Test connectivity**: Try `curl $BASE/v1/models`
5. **Review logs**: Check test output for specific error messages