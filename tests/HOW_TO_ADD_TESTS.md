# How to Add New Test Cases

This document shows you how to add new test cases to the load testing framework.

## Overview

The system automatically detects test cases by scanning the `src/` directory for `.js` files.
**No manual registration required** - just add your `.js` file and it will be available immediately.

## Current Test Structure

The framework includes three main test categories:

1. **`test-completion-standard.js`**: Basic completion flows with guest authentication
2. **`test-completion-conversation.js`**: Conversation management with both non-streaming and streaming messages  
3. **`test-responses.js`**: Response API testing with tools and streaming support (extended timeouts for tool calls)

## Example: Adding a Health Check Test

### Step 1: Create the k6 script in src/

Copy the example file and customize it:

```bash
cd tests/
cp src/health-check.js.example src/health-check.js
# Edit src/health-check.js as needed
```

### Step 2: Test immediately

That's it! The test is now available:

```bash
# Test the specific case
k6 run src/health-check.js

# Test all cases (including your new one)
./run-loadtest.sh

# List all available cases (auto-detected)
./run-loadtest.sh --list
```

## File Structure

```text
tests/
├── src/                              # Test scripts directory
│   ├── test-completion-standard.js   # Standard completion tests
│   ├── test-completion-conversation.js # Conversation flow tests
│   ├── test-responses.js             # Response API tests
│   ├── health-check.js.example       # Example template
│   └── your-new-test.js              # Your new test (auto-detected)
├── run-loadtest.sh                   # Test runner
├── Dockerfile                        # Docker container for testing
├── docker-entrypoint.sh              # Docker entrypoint script
├── run-docker-test.bat               # Windows Docker test runner
├── .env                              # Environment config
└── results/                          # Test results
```

## Auto-Detection

The system automatically:

- Scans `src/*.js` files
- Extracts test case names from filenames
- Makes them available in CLI and reports
- Validates file existence before running

## Environment Variables

All test cases share the same environment variables:

### Core Configuration
- `BASE` - API base URL (default: `https://api-stag.jan.ai`)
- `MODEL` - Model name (default: `jan-v1-4b`)
- `DEBUG` - Enable debug logging (default: `false`)
- `SINGLE_RUN` - Single iteration mode (default: `false`)

### Load Testing Configuration
- `NONSTREAM_RPS` - Non-streaming requests per second (default: `2`)
- `STREAM_RPS` - Streaming requests per second (default: `1`)
- `DURATION_MIN` - Test duration in minutes (default: `5`)

### Custom Variables
- Add test-specific variables as needed

**No authentication variables needed!** All tests use guest authentication automatically.

Your new test can access these via `__ENV.VARIABLE_NAME` in k6.

## Best Practices

1. **Follow naming convention**: `{test-name}.js`
2. **Use shared env variables**: Don't hardcode URLs or auth
3. **Add custom metrics**: Use meaningful metric names with prefixes
4. **Set appropriate thresholds**: Define what "success" means
5. **Test locally first**: Always test before committing
6. **Document your test**: Add description in the list function
7. **Use guest authentication**: No API keys needed - tests handle auth automatically
8. **Include token refresh**: Refresh tokens before requests to prevent timeouts
9. **Handle streaming properly**: Wait for `data: [DONE]` signals in streaming tests
10. **Add debug logging**: Use `DEBUG=true` for detailed request/response information
11. **Consider tool call timeouts**: Tool calls may take longer - use extended thresholds if needed

## Threshold Guidelines

### Standard Response Times
- **Guest login**: `p(95)<2000ms` (2 seconds)
- **Token refresh**: `p(95)<2000ms` (2 seconds)
- **Regular responses**: `p(95)<60000ms` (1 minute)
- **Streaming responses**: `p(95)<60000ms` (1 minute)

### Tool Call Response Times
- **Tool call responses**: `p(95)<300000ms` (5 minutes)
- **Tool call streaming**: `p(95)<300000ms` (5 minutes)

Tool calls require extended timeouts because they may involve external API calls and complex processing.

## Example Environment Variables

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

## Guest Authentication Pattern

All tests should follow this authentication pattern:

```javascript
// Global state
let accessToken = '';
let refreshToken = '';

// Guest login function
function guestLogin() {
  const res = http.post(`${BASE}/v1/auth/guest-login`, JSON.stringify({}));
  const body = JSON.parse(res.body);
  accessToken = body.access_token;
  
  // Extract refresh token from Set-Cookie header
  const setCookieHeader = res.headers['Set-Cookie'];
  if (setCookieHeader) {
    const refreshTokenMatch = setCookieHeader.match(/jan_refresh_token=([^;]+)/);
    if (refreshTokenMatch) {
      refreshToken = refreshTokenMatch[1];
    }
  }
}

// Token refresh function
function refreshAccessToken() {
  const headers = {
    'Content-Type': 'application/json',
    'Cookie': `jan_refresh_token=${refreshToken}`,
    'Authorization': `Bearer ${accessToken}`
  };
  
  const res = http.get(`${BASE}/v1/auth/refresh-token`, { headers });
  const body = JSON.parse(res.body);
  accessToken = body.access_token;
  
  // Update refresh token from new Set-Cookie header
  const setCookieHeader = res.headers['Set-Cookie'];
  if (setCookieHeader) {
    const refreshTokenMatch = setCookieHeader.match(/jan_refresh_token=([^;]+)/);
    if (refreshTokenMatch) {
      refreshToken = refreshTokenMatch[1];
    }
  }
}
```

## Docker Testing

### Building the Docker Image

```bash
# From the tests directory
docker build -t janai/k6-tests:local .
```

### Running Tests with Docker

```bash
# Run all tests
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local

# Run specific test
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local test-responses

# Run with volume mount (for local development)
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true -v "${PWD}":/tests janai/k6-tests:local
```

### Windows PowerShell Commands

```powershell
# Build
docker build -t janai/k6-tests:local .

# Run all tests
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local

# Run specific test
docker run --rm -it -e BASE=https://api-stag.jan.ai -e MODEL=jan-v1-4b -e DEBUG=true janai/k6-tests:local test-responses
```

### Docker Features

- **Alpine Linux base**: Lightweight and secure
- **Bash support**: Full bash scripting capabilities
- **jq included**: JSON parsing for metrics
- **Auto-detection**: Automatically finds and runs test scripts
- **Volume mounting**: Use local files with `-v` flag
- **Environment variables**: Full support for all test configuration
