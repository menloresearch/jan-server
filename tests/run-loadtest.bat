@echo off
setlocal enabledelayedexpansion

REM Load Test Runner Script for Jan Server (Windows Batch Version)
REM Usage: run-loadtest.bat [test_case_name]
REM Examples:
REM   run-loadtest.bat                    # Run all test cases
REM   run-loadtest.bat health-check      # Run only health-check test
REM   run-loadtest.bat --list            # Show available test cases

echo Jan Server Load Test Runner
echo ============================

REM Load environment variables from .env file if it exists
if exist ".env" (
    echo Loading environment from .env file...
    for /f "usebackq tokens=1,2 delims==" %%a in (".env") do (
        if not "%%a"=="" if not "%%a:~0,1%"=="#" (
            set "%%a=%%b"
        )
    )
)

REM Default configuration
set "DEFAULT_BASE_URL=https://api-dev.jan.ai"
set "DEFAULT_MODEL=jan-v1-4b"
set "DEFAULT_DURATION_MIN=5"
set "DEFAULT_NONSTREAM_RPS=2"
set "DEFAULT_STREAM_RPS=1"

REM Environment variables (can be overridden)
if not defined BASE set "BASE=%DEFAULT_BASE_URL%"
if not defined MODEL set "MODEL=%DEFAULT_MODEL%"
if not defined DURATION_MIN set "DURATION_MIN=%DEFAULT_DURATION_MIN%"
if not defined NONSTREAM_RPS set "NONSTREAM_RPS=%DEFAULT_NONSTREAM_RPS%"
if not defined STREAM_RPS set "STREAM_RPS=%DEFAULT_STREAM_RPS%"
if not defined LOADTEST_TOKEN set "LOADTEST_TOKEN="
if not defined API_KEY set "API_KEY="

REM Set k6 executable path
set "K6_EXE=k6"

REM Validate environment
if "%BASE%"=="" (
    echo [ERROR] BASE URL is required
    exit /b 1
)

if "%API_KEY%"=="" if "%LOADTEST_TOKEN%"=="" (
    echo [WARNING] Neither API_KEY nor LOADTEST_TOKEN is set. Test might fail.
)

REM Handle different arguments
set "TEST_CASE=%~1"

if "%TEST_CASE%"=="--list" goto :list_test_cases
if "%TEST_CASE%"=="-l" goto :list_test_cases
if "%TEST_CASE%"=="--help" goto :list_test_cases
if "%TEST_CASE%"=="-h" goto :list_test_cases

REM Get available test cases
set "TEST_CASES="
if not exist "src" (
    echo [ERROR] Source directory not found: src
    exit /b 1
)

for %%f in (src\*.js) do (
    set "filename=%%~nf"
    set "TEST_CASES=!TEST_CASES! !filename!"
)

if "%TEST_CASE%"=="" (
    REM No argument provided - run all test cases
    echo [INFO] No specific test case provided, running all test cases...
    goto :run_all_tests
)

REM Specific test case provided
echo [INFO] Running specific test case: %TEST_CASE%
goto :run_single_test

:list_test_cases
echo [INFO] Available test cases:
for %%f in (src\*.js) do (
    echo   - %%~nf (src\%%~nf.js)
)
echo.
echo Usage:
echo   %~nx0                    # Run all test cases
echo   %~nx0 [test_case_name]   # Run specific test case
echo.
echo Examples:
echo   %~nx0                    # Run all tests
echo   %~nx0 health-check       # Run only health-check test
echo   %~nx0 completion-flow    # Run full completion API flow test
echo   %~nx0 chat-completion    # Run chat completion test
echo   %~nx0 --list             # Show this help
goto :end

:run_all_tests
echo [INFO] Running all test cases
echo ====================================================

set "FAILED_TESTS="
set "TOTAL_TESTS=0"

for %%t in (%TEST_CASES%) do (
    set /a TOTAL_TESTS+=1
    echo.
    echo [INFO] 📋 Running test case: %%t
    echo [INFO] ----------------------------------------------------
    
    call :run_test "%%t"
    if errorlevel 1 (
        echo [ERROR] ❌ Test case '%%t' failed
        set "FAILED_TESTS=!FAILED_TESTS! %%t"
    ) else (
        echo [SUCCESS] ✅ Test case '%%t' completed successfully
    )
    
    REM Add a delay between tests
    if !TOTAL_TESTS! gtr 1 (
        echo [INFO] Waiting 10 seconds before next test...
        timeout /t 10 /nobreak >nul
    )
)

REM Summary
echo.
echo ====================================================
echo [INFO] 📊 TEST EXECUTION SUMMARY
echo ====================================================
echo [INFO] Total tests: %TOTAL_TESTS%

set "FAILED_COUNT=0"
for %%t in (%FAILED_TESTS%) do set /a FAILED_COUNT+=1
set /a PASSED_COUNT=%TOTAL_TESTS%-%FAILED_COUNT%

echo [INFO] Passed: %PASSED_COUNT%
echo [INFO] Failed: %FAILED_COUNT%

if %FAILED_COUNT%==0 (
    echo [SUCCESS] 🎉 All tests passed!
    exit /b 0
) else (
    echo [ERROR] 💥 Failed tests:%FAILED_TESTS%
    exit /b 1
)
goto :end

:run_single_test
call :run_test "%TEST_CASE%"
exit /b %errorlevel%

:run_test
set "TEST_CASE=%~1"
set "TIMESTAMP=%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%"
set "TIMESTAMP=%TIMESTAMP: =0%"
set "RESULTS_DIR=results"
set "OUTPUT_FILE=%RESULTS_DIR%\%TEST_CASE%_%TIMESTAMP%.json"
set "TEST_FILE=src\%TEST_CASE%.js"

REM Check if test file exists
if not exist "%TEST_FILE%" (
    echo [ERROR] Test file not found: %TEST_FILE%
    echo [INFO] Available test cases:
    for %%f in (src\*.js) do (
        echo   - %%~nf
    )
    exit /b 1
)

REM Create results directory if it doesn't exist
if not exist "%RESULTS_DIR%" mkdir "%RESULTS_DIR%"

echo [INFO] Running test case: %TEST_CASE%
echo [INFO] Test file: %TEST_FILE%
echo [INFO] Configuration:
echo [INFO]   Base URL: %BASE%
echo [INFO]   Model: %MODEL%
echo [INFO]   Duration: %DURATION_MIN% minutes
echo [INFO]   Non-stream RPS: %NONSTREAM_RPS%
echo [INFO]   Stream RPS: %STREAM_RPS%
echo [INFO]   Output: %OUTPUT_FILE%

REM Generate unique test ID for metrics segmentation
set "TEST_ID=%TEST_CASE%_%TIMESTAMP%_%RANDOM%"
echo [INFO] Test ID: %TEST_ID%

REM Execute k6
echo [INFO] Running k6 test...
"%K6_EXE%" run ^
    --summary-export="%OUTPUT_FILE%" ^
    --out json="%OUTPUT_FILE%" ^
    --tag testid="%TEST_ID%" ^
    --tag test_case="%TEST_CASE%" ^
    --tag environment="%BASE%" ^
    "%TEST_FILE%"

REM Check if test completed successfully
if errorlevel 1 (
    echo [ERROR] Test case '%TEST_CASE%' failed
    exit /b 1
) else (
    echo [SUCCESS] Test case '%TEST_CASE%' completed successfully
    
    REM Show results file location
    if exist "%OUTPUT_FILE%" (
        echo [INFO] Test Results Summary:
        echo ==================== METRICS SUMMARY ====================
        echo [INFO] Results saved to: %OUTPUT_FILE%
        echo ==========================================================
    )
    exit /b 0
)

:end