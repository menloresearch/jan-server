# How to Add New Test Cases

This document shows you how to add new test cases to the load testing framework.

## Overview

The system automatically detects test cases by scanning the `src/` directory for `.js` files.
**No manual registration required** - just add your `.js` file and it will be available immediately.

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
./run-loadtest.sh health-check

# Test all cases (including your new one)
./run-loadtest.sh

# List all available cases (auto-detected)
./run-loadtest.sh --list
```

## File Structure

```text
tests/
├── src/                          # Test scripts directory
│   ├── chat-completion.js        # Existing test
│   ├── health-check.js.example   # Example template
│   └── your-new-test.js          # Your new test (auto-detected)
├── run-loadtest.sh               # Test runner
├── .env                          # Environment config
└── results/                      # Test results
```

## Auto-Detection

The system automatically:

- Scans `src/*.js` files
- Extracts test case names from filenames
- Makes them available in CLI and reports
- Validates file existence before running

## Environment Variables

All test cases share the same environment variables from `.env` file:

- `BASE` - API base URL
- `API_KEY` / `LOADTEST_TOKEN` - Authentication
- `DURATION_MIN` - Test duration
- Custom variables (add as needed)

Your new test can access these via `__ENV.VARIABLE_NAME` in k6.

## Best Practices

1. **Follow naming convention**: `{test-name}.js`
2. **Use shared env variables**: Don't hardcode URLs or auth
3. **Add custom metrics**: Use meaningful metric names with prefixes
4. **Set appropriate thresholds**: Define what "success" means
5. **Test locally first**: Always test before committing
6. **Document your test**: Add description in the list function

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
