# Grafana Monitoring Setup

This directory contains all Grafana and Prometheus monitoring components for K6 load tests.

## 📁 Directory Structure

```
grafana/
├── README.md                           # This file
├── docker-compose.yml                  # Docker Compose for Grafana + Prometheus
├── grafana-dashboard.json              # Pre-built K6 dashboard
├── prometheus.yml                      # Prometheus configuration
└── grafana-provisioning/               # Grafana auto-provisioning
    ├── dashboards/
    │   └── dashboard.yml              # Dashboard provisioning config
    └── datasources/
        └── prometheus.yml              # Prometheus datasource config

../ (parent directory)
├── setup-monitoring.sh                 # Linux/Mac setup script
├── setup-monitoring.bat                # Windows setup script
├── run-test-with-monitoring.sh         # Linux/Mac test runner with metrics
└── run-test-with-monitoring.bat        # Windows test runner with metrics
```

## 🚀 Quick Start

### 1. Start Monitoring Stack

**Linux/Mac:**
```bash
../setup-monitoring.sh
```

**Windows:**
```cmd
..\setup-monitoring.bat
```

### 2. Run Tests with Metrics

**Linux/Mac:**
```bash
../run-test-with-monitoring.sh test-completion-standard
```

**Windows:**
```cmd
..\run-test-with-monitoring.bat test-completion-standard
```

### 3. View Results

- **Grafana Dashboard**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

## 📊 What's Included

### Grafana Dashboard Features
- **HTTP Performance Metrics**: Request duration, throughput, error rates
- **Test Segmentation**: Filter by Test ID and Test Case
- **Real-time Monitoring**: 5-second refresh intervals
- **Comprehensive Coverage**: All K6 built-in metrics

### Prometheus Configuration
- **Remote Write Receiver**: Enabled for K6 metrics
- **Data Retention**: 15 days
- **Scrape Intervals**: Optimized for load testing

### Auto-Provisioning
- **Dashboard**: Automatically loads K6 dashboard on startup
- **Datasource**: Prometheus connection configured automatically
- **No Manual Setup**: Everything works out of the box

## 🔧 Manual Setup (Alternative)

If you prefer manual setup:

```bash
# Start services
docker-compose up -d

# Wait for services to start
sleep 10

# Run test with metrics
export K6_PROMETHEUS_RW_SERVER_URL="http://localhost:9090/api/v1/write"
k6 run --out experimental-prometheus-rw ../src/test-completion-standard.js
```

## 📈 Available Metrics

The dashboard displays these K6 metrics:

- `k6_http_reqs_total` - Total HTTP requests
- `k6_http_req_duration_p95` - 95th percentile response time
- `k6_http_req_failed_rate` - HTTP error rate
- `k6_vus` - Virtual users
- `k6_iterations_total` - Total test iterations
- `k6_checks_rate` - Check success rate

## 🛠️ Troubleshooting

### Services Not Starting
```bash
# Check logs
docker-compose logs

# Restart services
docker-compose down
docker-compose up -d
```

### No Metrics in Grafana
1. Verify Prometheus is running: http://localhost:9090
2. Check K6 environment variable: `K6_PROMETHEUS_RW_SERVER_URL`
3. Ensure test is using `--out experimental-prometheus-rw`

### Dashboard Not Loading
1. Check Grafana logs: `docker-compose logs grafana`
2. Verify dashboard file: `grafana-dashboard.json`
3. Check provisioning config: `grafana-provisioning/dashboards/dashboard.yml`

## 📚 Documentation

- **Complete Setup Guide**: See `HOW_TO_READ_RESULT.md` in this directory
- **Main Test Documentation**: See `../README.md`
- **Adding New Tests**: See `../HOW_TO_ADD_TESTS.md`

## 🔄 Updates

When updating the dashboard or configuration:

1. **Dashboard**: Update `grafana-dashboard.json`
2. **Config**: Update `prometheus.yml` or provisioning files
3. **Restart**: Run `docker-compose restart` to apply changes

## 🌐 External Access

To access from other machines:

1. Update `docker-compose.yml` ports (e.g., `3000:3000` → `0.0.0.0:3000:3000`)
2. Update Prometheus URL in scripts: `http://YOUR_IP:9090/api/v1/write`
3. Restart services: `docker-compose restart`
