@echo off
REM Grafana Monitoring Setup Script for K6 Tests
REM This script sets up Grafana and Prometheus for monitoring K6 test results

echo ========================================
echo   K6 Test Monitoring Setup
echo ========================================
echo.

REM Check if Docker is running
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker is not running. Please start Docker and try again.
    pause
    exit /b 1
)

REM Check if docker-compose is available
docker-compose --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ docker-compose is not installed. Please install docker-compose and try again.
    pause
    exit /b 1
)

echo ✅ Docker and docker-compose are available
echo.

REM Start the monitoring stack
echo 🚀 Starting Grafana and Prometheus...
docker-compose -f grafana\docker-compose.yml up -d

echo.
echo ⏳ Waiting for services to start...
timeout /t 10 /nobreak >nul

REM Check if services are running
docker-compose -f grafana\docker-compose.yml ps | findstr "Up" >nul
if %errorlevel% equ 0 (
    echo ✅ Services started successfully!
    echo.
    echo 📊 Access your monitoring dashboard:
    echo    Grafana:    http://localhost:3000
    echo    Prometheus: http://localhost:9090
    echo.
    echo 🔐 Grafana credentials:
    echo    Username: admin
    echo    Password: admin
    echo.
    echo 🧪 To run tests with metrics:
    echo    set K6_PROMETHEUS_RW_SERVER_URL=http://localhost:9090/api/v1/write
    echo    .\run-loadtest.bat test-completion-standard
    echo    or
    echo    .\run-test-with-monitoring.bat test-completion-standard
    echo.
    echo 📈 The K6 dashboard will be automatically loaded in Grafana
    echo.
    echo 🌐 Opening Grafana in your browser...
    start http://localhost:3000
) else (
    echo ❌ Failed to start services. Check the logs:
    echo    docker-compose -f grafana\docker-compose.yml logs
    pause
    exit /b 1
)

pause
