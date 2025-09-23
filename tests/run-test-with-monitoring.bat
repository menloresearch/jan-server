@echo off
REM Test Script with Grafana Monitoring
REM This script runs a K6 test and sends metrics to Grafana

echo ========================================
echo   K6 Test with Grafana Monitoring
echo ========================================
echo.

REM Check if monitoring stack is running
curl -s http://localhost:9090/api/v1/query?query=up >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Prometheus is not running. Please start the monitoring stack first:
    echo    .\setup-monitoring.bat
    echo    or
    echo    docker-compose -f grafana\docker-compose.yml up -d
    pause
    exit /b 1
)

echo ✅ Prometheus is running
echo.

REM Set environment variables for metrics
set K6_PROMETHEUS_RW_SERVER_URL=http://localhost:9090/api/v1/write
set K6_PROMETHEUS_RW_TREND_STATS=p(95),p(99),min,max
set K6_PROMETHEUS_RW_PUSH_INTERVAL=5s

echo 📊 Metrics will be sent to: %K6_PROMETHEUS_RW_SERVER_URL%
echo.

REM Get test case from command line or use default
if "%1"=="" (
    set TEST_CASE=test-completion-standard
) else (
    set TEST_CASE=%1
)

echo 🧪 Running test: %TEST_CASE%
echo.

REM Run the test
if exist ".\run-loadtest.bat" (
    .\run-loadtest.bat %TEST_CASE%
) else (
    echo ❌ run-loadtest.bat not found. Running k6 directly...
    k6 run --out experimental-prometheus-rw "src\%TEST_CASE%.js"
)

echo.
echo ✅ Test completed!
echo.
echo 📈 View results in Grafana:
echo    http://localhost:3000
echo    Username: admin
echo    Password: admin
echo.
echo 🔍 Check Prometheus metrics:
echo    http://localhost:9090
echo.

pause
