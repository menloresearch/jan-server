# How to Create New Test Scenarios

This guide explains how to create new K6 test scenarios for the Jan Server API.

## 🎯 Overview

Our test framework supports three main test types:
1. **Standard Completion Tests** - Basic API functionality
2. **Conversation Management Tests** - Conversation lifecycle
3. **Response API Tests** - Response endpoint testing

## 📁 File Structure

```
tests/
├── src/
│   ├── test-completion-standard.js      # Basic completion flows
│   ├── test-completion-conversation.js  # Conversation management
│   ├── test-responses.js               # Response API testing
│   └── your-new-test.js                 # Your new test scenario
├── grafana/                             # Monitoring setup
├── results/                             # Test results
└── run-loadtest.sh                      # Test runner
```

## 🚀 Creating a New Test

### Step 1: Copy Template

Start by copying an existing test file:

```bash
cp src/test-completion-standard.js src/test-your-scenario.js
```

### Step 2: Update Test Configuration

```javascript
// ====== Test Configuration ======
const TEST_ID = `test-your-scenario-${Date.now()}`;
const TEST_CASE = 'your-scenario';

// ====== Custom metrics ======
const yourMetric = new Trend('your_metric_ms', true);
const errors = new Counter('your_errors');
const successes = new Counter('your_successes');

// ====== Options ======
export const options = {
  iterations: 1,
  vus: 1,
  thresholds: {
    'http_req_failed': ['rate<0.05'],
    'your_metric_ms': ['p(95)<10000'],
  },
  discardResponseBodies: false,
  tags: {
    testid: TEST_ID,
    test_case: TEST_CASE,
  },
};
```

### Step 3: Implement Test Functions

```javascript
// ====== Test Functions ======
function yourTestFunction() {
  console.log('[YOUR TEST] Starting your test...');
  
  const startTime = Date.now();
  
  // Your test logic here
  const res = http.post(`${BASE}/v1/your-endpoint`, {
    // Your payload
  }, {
    headers: buildHeaders()
  });
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  yourMetric.add(duration);
  
  const ok = check(res, {
    'your test status 200': (r) => r.status === 200,
    'your test has data': (r) => r.body && r.body.length > 0
  });
  
  if (ok) {
    successes.add(1);
    console.log('[YOUR TEST] ✅ Success!');
  } else {
    errors.add(1);
    console.log('[YOUR TEST] ❌ Failed!');
  }
}
```

### Step 4: Create Main Test Function

```javascript
export default function() {
  console.log('========================================');
  console.log('   YOUR SCENARIO TESTS');
  console.log('========================================');
  console.log(`Base URL: ${BASE}`);
  console.log(`Model: ${MODEL}`);
  console.log(`Debug Mode: ${DEBUG ? 'ENABLED' : 'DISABLED'}`);
  console.log();
  
  // Authentication
  guestLogin();
  refreshToken();
  
  // Your test steps
  yourTestFunction();
  
  console.log();
  console.log('========================================');
  console.log('            TEST SUMMARY');
  console.log('========================================');
  console.log('✅ Your test completed!');
  console.log('========================================');
}
```

## 🔧 Helper Functions

### Authentication Helpers

**Complete Guest Authentication Pattern:**

```javascript
// Global state for authentication
let accessToken = '';
let refreshToken = '';

function guestLogin() {
  console.log('[GUEST LOGIN] Starting guest login...');
  
  const headers = buildHeaders();
  const res = http.post(`${BASE}/v1/auth/guest-login`, {}, { headers });
  
  debugResponse(res);
  
  const ok = check(res, {
    'guest login status 200': (r) => r.status === 200,
    'guest login has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.access_token.length > 0;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    const body = JSON.parse(res.body);
    accessToken = body.access_token;
    __ENV.ACCESS_TOKEN = accessToken;
    
    // Extract refresh token from Set-Cookie header
    const setCookieHeader = res.headers['Set-Cookie'];
    if (setCookieHeader) {
      const refreshTokenMatch = setCookieHeader.match(/jan_refresh_token=([^;]+)/);
      if (refreshTokenMatch) {
        refreshToken = refreshTokenMatch[1];
      }
    }
    
    console.log('[GUEST LOGIN] ✅ Success!');
  } else {
    console.log('[GUEST LOGIN] ❌ Failed!');
  }
}

function refreshToken() {
  console.log('[REFRESH TOKEN] Refreshing access token...');
  
  const headers = {
    'Content-Type': 'application/json',
    'Cookie': `jan_refresh_token=${refreshToken}`,
    'Authorization': `Bearer ${accessToken}`
  };
  
  const res = http.get(`${BASE}/v1/auth/refresh-token`, { headers });
  
  debugResponse(res);
  
  const ok = check(res, {
    'refresh token status 200': (r) => r.status === 200,
    'refresh token has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.access_token.length > 0;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    const body = JSON.parse(res.body);
    accessToken = body.access_token;
    __ENV.ACCESS_TOKEN = accessToken;
    
    // Update refresh token from new Set-Cookie header
    const setCookieHeader = res.headers['Set-Cookie'];
    if (setCookieHeader) {
      const refreshTokenMatch = setCookieHeader.match(/jan_refresh_token=([^;]+)/);
      if (refreshTokenMatch) {
        refreshToken = refreshTokenMatch[1];
      }
    }
    
    console.log('[REFRESH TOKEN] ✅ Success!');
  } else {
    console.log('[REFRESH TOKEN] ❌ Failed!');
  }
}
```

**Key Points:**
- **No API keys needed**: All tests use guest authentication automatically
- **Token refresh**: Always refresh tokens before requests to prevent timeouts
- **Cookie handling**: Extract refresh tokens from Set-Cookie headers
- **Global state**: Store tokens in global variables for reuse

### Utility Functions

```javascript
function buildHeaders() {
  const headers = {
    'Content-Type': 'application/json'
  };
  
  if (__ENV.ACCESS_TOKEN) {
    headers['Authorization'] = `Bearer ${__ENV.ACCESS_TOKEN}`;
  }
  
  return headers;
}

function debugResponse(response) {
  if (DEBUG) {
    console.log('[DEBUG] ====== RESPONSE ======');
    console.log(`[DEBUG] Status: ${response.status}`);
    console.log(`[DEBUG] Headers:`, response.headers);
    console.log(`[DEBUG] Body:`, response.body);
    console.log('[DEBUG] =====================');
  }
}
```

## 📊 Metrics and Monitoring

### Custom Metrics

```javascript
// ====== Custom metrics ======
const yourMetric = new Trend('your_metric_ms', true);
const yourCounter = new Counter('your_counter');
const yourGauge = new Gauge('your_gauge');

// Usage
yourMetric.add(duration);
yourCounter.add(1);
yourGauge.add(value);
```

### Thresholds

```javascript
thresholds: {
  'http_req_failed': ['rate<0.05'],           // Error rate < 5%
  'your_metric_ms': ['p(95)<10000'],          // 95th percentile < 10s
  'http_req_duration': ['p(99)<15000'],       // 99th percentile < 15s
}
```

### Tags for Filtering

```javascript
tags: {
  testid: TEST_ID,           // Unique test identifier
  test_case: TEST_CASE,      // Test category
  scenario: 'your-scenario', // Specific scenario
  method: 'POST',            // HTTP method
  status: '200'              // Response status
}
```

## 🧪 Test Patterns

### API Endpoint Testing

```javascript
function testApiEndpoint() {
  const startTime = Date.now();
  
  const res = http.post(`${BASE}/v1/your-endpoint`, {
    // Your payload
  }, {
    headers: buildHeaders()
  });
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  
  const ok = check(res, {
    'endpoint status 200': (r) => r.status === 200,
    'endpoint has expected data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.expected_field !== undefined;
      } catch (e) {
        return false;
      }
    }
  });
  
  return { ok, duration, response: res };
}
```

### Streaming Response Testing

```javascript
function testStreamingEndpoint() {
  const startTime = Date.now();
  
  const res = http.post(`${BASE}/v1/streaming-endpoint`, {
    stream: true
  }, {
    headers: buildHeaders()
  });
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  
  // Check for streaming response
  const isStreaming = res.headers['Content-Type'] && 
                     res.headers['Content-Type'].includes('text/event-stream');
  
  // Check for completion signal
  const hasCompletionSignal = res.body.includes('data: [DONE]');
  
  const ok = check(res, {
    'streaming status 200': (r) => r.status === 200,
    'streaming content type': (r) => isStreaming,
    'streaming completion signal': (r) => hasCompletionSignal
  });
  
  return { ok, duration, response: res };
}
```

### Error Handling

```javascript
function testWithErrorHandling() {
  try {
    const res = http.post(`${BASE}/v1/endpoint`, payload, { headers });
    
    if (res.status >= 400) {
      console.log(`[ERROR] HTTP ${res.status}: ${res.body}`);
      return false;
    }
    
    return check(res, {
      'successful response': (r) => r.status === 200
    });
    
  } catch (error) {
    console.log(`[ERROR] Request failed: ${error.message}`);
    return false;
  }
}
```

## 🔄 Integration with Test Runner

### Update run-loadtest.sh

Add your test to the test runner:

```bash
# In run-loadtest.sh
case "$TEST_CASE" in
  "test-completion-standard")
    TEST_FILE="src/test-completion-standard.js"
    ;;
  "test-completion-conversation")
    TEST_FILE="src/test-completion-conversation.js"
    ;;
  "test-responses")
    TEST_FILE="src/test-responses.js"
    ;;
  "test-your-scenario")                    # Add your test
    TEST_FILE="src/test-your-scenario.js"
    ;;
  *)
    echo "Unknown test case: $TEST_CASE"
    exit 1
    ;;
esac
```

### Run Your Test

```bash
# Using test runner
./run-loadtest.sh test-your-scenario

# Direct execution
k6 run src/test-your-scenario.js
```

## 📈 Best Practices

### 1. Test Structure
- **Single Responsibility**: Each test should focus on one scenario
- **Clear Naming**: Use descriptive names for functions and variables
- **Consistent Format**: Follow existing test patterns
- **Auto-Detection**: Tests are automatically detected by scanning `src/*.js` files

### 2. Error Handling
- **Graceful Failures**: Handle errors without crashing the test
- **Meaningful Messages**: Provide clear error descriptions
- **Debug Information**: Include relevant debugging data

### 3. Performance
- **Reasonable Thresholds**: Set achievable performance targets
- **Resource Management**: Don't overwhelm the API
- **Monitoring**: Include relevant metrics

### 4. Documentation
- **Comments**: Explain complex logic
- **README**: Document your test's purpose
- **Examples**: Provide usage examples

## 🎯 Threshold Guidelines

### Standard Response Times
- **Guest login**: `p(95)<2000ms` (2 seconds)
- **Token refresh**: `p(95)<2000ms` (2 seconds)
- **Regular responses**: `p(95)<10000ms` (10 seconds)
- **Streaming responses**: `p(95)<15000ms` (15 seconds)

### Tool Call Response Times
- **Tool call responses**: `p(95)<300000ms` (5 minutes)
- **Tool call streaming**: `p(95)<300000ms` (5 minutes)

Tool calls require extended timeouts because they may involve external API calls and complex processing.

### Example Threshold Configuration
```javascript
thresholds: {
  'http_req_failed': ['rate<0.05'],           // Error rate < 5%
  'guest_login_time_ms': ['p(95)<2000'],      // Guest login < 2s
  'refresh_token_time_ms': ['p(95)<2000'],    // Token refresh < 2s
  'completion_time_ms': ['p(95)<10000'],      // Regular completion < 10s
  'streaming_time_ms': ['p(95)<15000'],       // Streaming < 15s
  'response_time_with_tools_ms': ['p(95)<300000'], // Tool calls < 5min
}
```

## 🔄 Auto-Detection System

The framework automatically:
- Scans `src/*.js` files for test scripts
- Extracts test case names from filenames
- Makes them available in CLI and reports
- Validates file existence before running
- **No manual registration required** - just add your `.js` file and it will be available immediately

### Example: Adding a Health Check Test
```bash
# Copy the example template
cp src/health-check.js.example src/health-check.js

# Edit the file as needed
# The test is now automatically available:
k6 run src/health-check.js
./run-loadtest.sh health-check
```

## 🧪 Testing Your New Test

### 1. Basic Validation
```bash
# Test syntax
k6 run --dry-run src/test-your-scenario.js

# Test execution
k6 run src/test-your-scenario.js
```

### 2. Debug Mode
```bash
# Enable debug logging
DEBUG=true k6 run src/test-your-scenario.js
```

### 3. Performance Testing
```bash
# Load test
DURATION_MIN=5 NONSTREAM_RPS=5 k6 run src/test-your-scenario.js
```

### 4. Integration Testing
```bash
# Use test runner
./run-loadtest.sh test-your-scenario
```

## 📝 Example: Complete Test File

```javascript
import http from 'k6/http';
import { check } from 'k6';
import { Trend, Counter } from 'k6/metrics';

// ====== Test Configuration ======
const TEST_ID = `test-example-${Date.now()}`;
const TEST_CASE = 'example';

// ====== Environment Variables ======
const BASE = __ENV.BASE || 'https://api-dev.jan.ai';
const MODEL = __ENV.MODEL || 'jan-v1-4b';
const DEBUG = __ENV.DEBUG === 'true';

// ====== Custom metrics ======
const exampleTime = new Trend('example_time_ms', true);
const errors = new Counter('example_errors');
const successes = new Counter('example_successes');

// ====== Options ======
export const options = {
  iterations: 1,
  vus: 1,
  thresholds: {
    'http_req_failed': ['rate<0.05'],
    'example_time_ms': ['p(95)<5000'],
  },
  discardResponseBodies: false,
  tags: {
    testid: TEST_ID,
    test_case: TEST_CASE,
  },
};

// ====== Helper Functions ======
function buildHeaders() {
  const headers = { 'Content-Type': 'application/json' };
  if (__ENV.ACCESS_TOKEN) {
    headers['Authorization'] = `Bearer ${__ENV.ACCESS_TOKEN}`;
  }
  return headers;
}

function debugResponse(response) {
  if (DEBUG) {
    console.log('[DEBUG] ====== RESPONSE ======');
    console.log(`[DEBUG] Status: ${response.status}`);
    console.log(`[DEBUG] Body:`, response.body);
    console.log('[DEBUG] =====================');
  }
}

// ====== Test Functions ======
function guestLogin() {
  console.log('[GUEST LOGIN] Starting guest login...');
  
  const headers = buildHeaders();
  const res = http.post(`${BASE}/v1/auth/guest-login`, {}, { headers });
  
  debugResponse(res);
  
  const ok = check(res, {
    'guest login status 200': (r) => r.status === 200,
    'guest login has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.access_token.length > 0;
      } catch (e) {
        return false;
      }
    }
  });
  
  if (ok) {
    const body = JSON.parse(res.body);
    __ENV.ACCESS_TOKEN = body.access_token;
    console.log('[GUEST LOGIN] ✅ Success!');
  } else {
    console.log('[GUEST LOGIN] ❌ Failed!');
  }
}

function testExampleEndpoint() {
  console.log('[EXAMPLE] Testing example endpoint...');
  
  const startTime = Date.now();
  
  const res = http.post(`${BASE}/v1/example`, {
    message: 'Hello, world!'
  }, {
    headers: buildHeaders()
  });
  
  const endTime = Date.now();
  const duration = endTime - startTime;
  exampleTime.add(duration);
  
  debugResponse(res);
  
  const ok = check(res, {
    'example status 200': (r) => r.status === 200,
    'example has response': (r) => r.body && r.body.length > 0
  });
  
  if (ok) {
    successes.add(1);
    console.log('[EXAMPLE] ✅ Success!');
  } else {
    errors.add(1);
    console.log('[EXAMPLE] ❌ Failed!');
  }
}

// ====== Main Test Function ======
export default function() {
  console.log('========================================');
  console.log('   EXAMPLE TESTS');
  console.log('========================================');
  console.log(`Base URL: ${BASE}`);
  console.log(`Model: ${MODEL}`);
  console.log(`Debug Mode: ${DEBUG ? 'ENABLED' : 'DISABLED'}`);
  console.log();
  
  // Test steps
  guestLogin();
  testExampleEndpoint();
  
  console.log();
  console.log('========================================');
  console.log('            TEST SUMMARY');
  console.log('========================================');
  console.log('✅ Example tests completed!');
  console.log('========================================');
}
```

## 🚀 Next Steps

1. **Create your test file** using the template above
2. **Implement your test logic** following the patterns
3. **Add to test runner** for easy execution
4. **Test thoroughly** with different configurations
5. **Document your test** in the README
6. **Share with team** for review and integration

## 📚 Additional Resources

- **K6 Documentation**: [k6.io/docs](https://k6.io/docs)
- **Existing Tests**: Study `src/test-*.js` files
- **Test Runner**: See `run-loadtest.sh`
- **Monitoring**: See `grafana/README.md`
