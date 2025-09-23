#!/bin/bash

# Test Script with Grafana Monitoring
# This script runs a K6 test and sends metrics to Grafana

set -euo pipefail

echo "========================================"
echo "  K6 Test with Grafana Monitoring"
echo "========================================"
echo

# Check if monitoring stack is running
if ! curl -s http://localhost:9090/api/v1/query?query=up >/dev/null 2>&1; then
    echo "❌ Prometheus is not running. Please start the monitoring stack first:"
    echo "   ./setup-monitoring.sh"
    echo "   or"
    echo "   docker-compose -f grafana/docker-compose.yml up -d"
    exit 1
fi

echo "✅ Prometheus is running"
echo

# Set environment variables for metrics
export K6_PROMETHEUS_RW_SERVER_URL="http://localhost:9090/api/v1/write"
export K6_PROMETHEUS_RW_TREND_STATS="p(95),p(99),min,max"
export K6_PROMETHEUS_RW_PUSH_INTERVAL="5s"

echo "📊 Metrics will be sent to: $K6_PROMETHEUS_RW_SERVER_URL"
echo

# Get test case from command line or use default
TEST_CASE="${1:-test-completion-standard}"

echo "🧪 Running test: $TEST_CASE"
echo

# Run the test
if [ -f "./run-loadtest.sh" ]; then
    ./run-loadtest.sh "$TEST_CASE"
else
    echo "❌ run-loadtest.sh not found. Running k6 directly..."
    k6 run --out experimental-prometheus-rw "src/$TEST_CASE.js"
fi

echo
echo "✅ Test completed!"
echo
echo "📈 View results in Grafana:"
echo "   http://localhost:3000"
echo "   Username: admin"
echo "   Password: admin"
echo
echo "🔍 Check Prometheus metrics:"
echo "   http://localhost:9090"
echo
