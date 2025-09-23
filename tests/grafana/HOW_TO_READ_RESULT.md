# How to Read Test Results with Grafana

This guide shows you how to set up Grafana with Docker to visualize your K6 test results locally.

## Overview

This setup provides:
- **Local Grafana instance** running in Docker
- **Prometheus** for metrics storage
- **Pre-built dashboard** for K6 test visualization
- **Real-time monitoring** of test performance
- **Historical analysis** of test trends

## Prerequisites

- Docker and Docker Compose installed
- Basic understanding of Docker containers
- Ports 3000 (Grafana) and 9090 (Prometheus) available

## Quick Start

### 1. Create Docker Compose Setup

Create a `docker-compose.yml` file in your `tests/` directory:

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: janai-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: janai-grafana
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana-dashboard.json:/var/lib/grafana/dashboards/k6-dashboard.json
      - ./grafana-provisioning:/etc/grafana/provisioning
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    networks:
      - monitoring

volumes:
  prometheus_data:
  grafana_data:

networks:
  monitoring:
    driver: bridge
```

### 2. Create Prometheus Configuration

Create `prometheus.yml` in your `tests/` directory:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'k6'
    static_configs:
      - targets: ['host.docker.internal:9090']
    metrics_path: /api/v1/write
    scheme: http
```

### 3. Create Grafana Provisioning

Create the directory structure and files:

```bash
mkdir -p grafana-provisioning/datasources
mkdir -p grafana-provisioning/dashboards
```

Create `grafana-provisioning/datasources/prometheus.yml`:

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

Create `grafana-provisioning/dashboards/dashboard.yml`:

```yaml
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
```

### 4. Start the Monitoring Stack

```bash
# Start Grafana and Prometheus
docker-compose up -d

# Check if containers are running
docker-compose ps
```

### 5. Access Grafana

Open your browser and go to:
- **Grafana**: http://localhost:3000
- **Username**: `admin`
- **Password**: `admin`

## Running Tests with Metrics

### Method 1: Direct K6 with Prometheus Remote Write

```bash
# Set environment variables
export K6_PROMETHEUS_RW_SERVER_URL="http://localhost:9090/api/v1/write"
export K6_PROMETHEUS_RW_TREND_STATS="p(95),p(99),min,max"
export K6_PROMETHEUS_RW_PUSH_INTERVAL="5s"

# Run test with metrics export
k6 run --out experimental-prometheus-rw src/test-completion-standard.js
```

### Method 2: Using Docker with Host Network

```bash
# Run K6 test with metrics to local Prometheus
docker run --rm -it \
   --network host \
   -e BASE=https://api-stag.jan.ai \
   -e MODEL=jan-v1-4b \
   -e DEBUG=true \
   -e K6_PROMETHEUS_RW_SERVER_URL="http://localhost:9090/api/v1/write" \
   -e K6_PROMETHEUS_RW_TREND_STATS="p(95),p(99),min,max" \
   -e K6_PROMETHEUS_RW_PUSH_INTERVAL="5s" \
   janai/k6-tests:local run test-completion-standard
```

### Method 3: Using Our Test Runner

```bash
# Set environment variables
export K6_PROMETHEUS_RW_SERVER_URL="http://localhost:9090/api/v1/write"
export K6_PROMETHEUS_RW_TREND_STATS="p(95),p(99),min,max"
export K6_PROMETHEUS_RW_PUSH_INTERVAL="5s"

# Run using our test runner
./run-loadtest.sh test-completion-standard
```

## Dashboard Features

### Main Panels

1. **LLM Performance Overview**
   - Response time percentiles (p95, p99)
   - Tokens per second
   - Time to first byte (TTFB)
   - Queue time metrics

2. **HTTP Performance**
   - Request duration trends
   - Request rate (RPS)
   - Error rate percentage
   - Response size distribution

3. **Custom K6 Metrics**
   - Guest login time
   - Token refresh time
   - Completion time (non-streaming)
   - Streaming completion time
   - Tool call response time (extended)

4. **Test Segmentation**
   - By test case (test-completion-standard, test-responses, etc.)
   - By test ID (individual run tracking)
   - By environment (staging vs production)

### Key Metrics to Monitor

```prometheus
# Response times
k6_http_req_duration{testid="test-completion-standard_20250923_042450_1"}

# Custom completion metrics
k6_completion_time_ms{testid="test-completion-standard_20250923_042450_1"}
k6_guest_login_time_ms{testid="test-completion-standard_20250923_042450_1"}

# Tool call metrics (extended timeouts)
k6_response_time_with_tools_ms{testid="test-responses_20250923_042450_1"}

# Error rates
k6_http_req_failed{testid="test-completion-standard_20250923_042450_1"}

# Throughput
k6_http_reqs{testid="test-completion-standard_20250923_042450_1"}
```

## Dashboard Navigation

### Time Range Selection
- **Last 5 minutes**: For real-time monitoring
- **Last hour**: For recent test analysis
- **Last 24 hours**: For daily trends
- **Last 7 days**: For weekly patterns

### Panel Interactions
- **Click and drag**: Zoom into specific time ranges
- **Panel menu**: Access panel options and edit
- **Refresh**: Manual refresh or auto-refresh (5s, 10s, 30s, 1m, 5m, 15m, 30m, 1h)

### Alerting
- **Threshold alerts**: Set up alerts for response time limits
- **Error rate alerts**: Monitor failure rates
- **Performance regression**: Detect performance degradation

## Troubleshooting

### Common Issues

1. **Metrics not appearing in Grafana**
   ```bash
   # Check Prometheus targets
   curl http://localhost:9090/api/v1/targets
   
   # Check if K6 is sending metrics
   curl http://localhost:9090/api/v1/query?query=k6_http_reqs
   ```

2. **Connection refused errors**
   ```bash
   # Check if containers are running
   docker-compose ps
   
   # Check container logs
   docker-compose logs prometheus
   docker-compose logs grafana
   ```

3. **Dashboard not loading**
   ```bash
   # Check Grafana logs
   docker-compose logs grafana
   
   # Restart Grafana
   docker-compose restart grafana
   ```

### Port Conflicts

If ports 3000 or 9090 are in use:

```yaml
# Modify docker-compose.yml
services:
  grafana:
    ports:
      - "3001:3000"  # Change to 3001
  prometheus:
    ports:
      - "9091:9090"  # Change to 9091
```

Then update your K6 command:
```bash
export K6_PROMETHEUS_RW_SERVER_URL="http://localhost:9091/api/v1/write"
```

## Advanced Configuration

### Custom Dashboard Panels

You can add custom panels to monitor specific metrics:

1. **Add Panel**: Click "+" → "Add panel"
2. **Query**: Use Prometheus queries like:
   ```prometheus
   # Average response time
   avg(k6_http_req_duration)
   
   # Error rate percentage
   rate(k6_http_req_failed[5m]) * 100
   
   # Requests per second
   rate(k6_http_reqs[5m])
   ```

### Alerting Rules

Create alerting rules in Prometheus:

```yaml
# prometheus-alerts.yml
groups:
  - name: k6-alerts
    rules:
      - alert: HighResponseTime
        expr: k6_http_req_duration > 5
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "High response time detected"
          
      - alert: HighErrorRate
        expr: rate(k6_http_req_failed[5m]) > 0.05
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
```

## Cleanup

To stop and remove all containers:

```bash
# Stop containers
docker-compose down

# Remove volumes (WARNING: This deletes all data)
docker-compose down -v

# Remove images
docker-compose down --rmi all
```

## Example Workflow

1. **Start monitoring stack**:
   ```bash
   docker-compose up -d
   ```

2. **Run a test with metrics**:
   ```bash
   export K6_PROMETHEUS_RW_SERVER_URL="http://localhost:9090/api/v1/write"
   ./run-loadtest.sh test-completion-standard
   ```

3. **View results in Grafana**:
   - Open http://localhost:3000
   - Login with admin/admin
   - Navigate to "K6 Load Test Dashboard"
   - Select time range "Last 5 minutes"

4. **Analyze performance**:
   - Check response time trends
   - Monitor error rates
   - Compare different test runs
   - Set up alerts for thresholds

## Next Steps

- **Set up alerts** for performance thresholds
- **Create custom dashboards** for specific use cases
- **Integrate with CI/CD** for automated monitoring
- **Export dashboards** for team sharing
- **Configure retention policies** for long-term analysis

This setup gives you comprehensive visibility into your K6 test performance with real-time monitoring and historical analysis capabilities! 📊✨
