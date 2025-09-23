#!/bin/bash

# Grafana Monitoring Setup Script for K6 Tests
# This script sets up Grafana and Prometheus for monitoring K6 test results

set -euo pipefail

echo "========================================"
echo "  K6 Test Monitoring Setup"
echo "========================================"
echo

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose >/dev/null 2>&1; then
    echo "❌ docker-compose is not installed. Please install docker-compose and try again."
    exit 1
fi

echo "✅ Docker and docker-compose are available"
echo

# Start the monitoring stack
echo "🚀 Starting Grafana and Prometheus..."
docker-compose -f grafana/docker-compose.yml up -d

echo
echo "⏳ Waiting for services to start..."
sleep 10

# Check if services are running
if docker-compose -f grafana/docker-compose.yml ps | grep -q "Up"; then
    echo "✅ Services started successfully!"
    echo
    echo "📊 Access your monitoring dashboard:"
    echo "   Grafana:    http://localhost:3000"
    echo "   Prometheus: http://localhost:9090"
    echo
    echo "🔐 Grafana credentials:"
    echo "   Username: admin"
    echo "   Password: admin"
    echo
    echo "🧪 To run tests with metrics:"
    echo "   export K6_PROMETHEUS_RW_SERVER_URL=\"http://localhost:9090/api/v1/write\""
    echo "   ./run-loadtest.sh test-completion-standard"
    echo "   or"
    echo "   ./run-test-with-monitoring.sh test-completion-standard"
    echo
    echo "📈 The K6 dashboard will be automatically loaded in Grafana"
    echo
else
    echo "❌ Failed to start services. Check the logs:"
    echo "   docker-compose -f grafana/docker-compose.yml logs"
    exit 1
fi
