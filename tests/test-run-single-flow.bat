@echo off
REM Jan Server Test Runner - Single Flow Tests
REM Usage: test-run-single-flow.bat [BASE_URL] [MODEL] [DEBUG] [TEST_TYPE]

set BASE_URL=%1
if "%BASE_URL%"=="" set BASE_URL=https://api-stag.jan.ai

set MODEL=%2
if "%MODEL%"=="" set MODEL=jan-v1-4b

set DEBUG_MODE=%3
if "%DEBUG_MODE%"=="" set DEBUG_MODE=true

set TEST_TYPE=%4
if "%TEST_TYPE%"=="" set TEST_TYPE=conversation

echo ========================================
echo   JAN SERVER TEST - SINGLE RUN
echo ========================================
echo Base URL: %BASE_URL%
echo Model: %MODEL%
echo Debug Mode: %DEBUG_MODE%
echo Test Type: %TEST_TYPE%
echo ========================================
echo.

REM Set environment variables for single run
set BASE=%BASE_URL%
set MODEL=%MODEL%
set SINGLE_RUN=true
set DEBUG=%DEBUG_MODE%

REM Run the selected test
if "%TEST_TYPE%"=="standard" (
    echo Running Standard Completion Test...
    k6 run src\test-completion-standard.js
) else if "%TEST_TYPE%"=="conversation" (
    echo Running Conversation Flow Test...
    k6 run src\test-completion-conversation.js
) else if "%TEST_TYPE%"=="responses" (
    echo Running Response API Test...
    k6 run src\test-responses.js
) else (
    echo Invalid test type. Available options: standard, conversation, responses
    echo Running default conversation test...
    k6 run src\test-completion-conversation.js
)

echo.
echo ========================================
echo   TEST COMPLETED
echo ========================================
pause
