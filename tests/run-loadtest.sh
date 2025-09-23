#!/bin/bash

# Load Test Runner Script for Jan Server
# Usage: ./run-loadtest.sh [test_case_name]

set -e

# Load environment variables from .env file if it exists
SCRIPT_DIR=$(dirname "$0")
if [[ -f "$SCRIPT_DIR/.env" ]]; then
    echo "Loading environment from .env file..."
    set -a  # automatically export all variables
    source "$SCRIPT_DIR/.env"
    set +a
fi

# Default configuration
DEFAULT_BASE_URL="https://api-stag.jan.ai"
DEFAULT_MODEL="jan-v1-4b"
DEFAULT_DURATION_MIN=5
DEFAULT_NONSTREAM_RPS=2
DEFAULT_STREAM_RPS=1

# Environment variables (can be overridden)
export BASE="${BASE:-$DEFAULT_BASE_URL}"
export MODEL="${MODEL:-$DEFAULT_MODEL}"
export DURATION_MIN="${DURATION_MIN:-$DEFAULT_DURATION_MIN}"
export NONSTREAM_RPS="${NONSTREAM_RPS:-$DEFAULT_NONSTREAM_RPS}"
export STREAM_RPS="${STREAM_RPS:-$DEFAULT_STREAM_RPS}"
export DEBUG="${DEBUG:-false}"
export SINGLE_RUN="${SINGLE_RUN:-false}"

# Cloudflare load test token (required for API access)
export LOADTEST_TOKEN="${LOADTEST_TOKEN:-}"

# Guest authentication - no API keys needed
# Tests automatically use guest login

# Prometheus remote write configuration (following k6 docs)
export K6_PROMETHEUS_RW_SERVER_URL="${K6_PROMETHEUS_RW_SERVER_URL:-}"
export K6_PROMETHEUS_RW_USERNAME="${K6_PROMETHEUS_RW_USERNAME:-}"
export K6_PROMETHEUS_RW_PASSWORD="${K6_PROMETHEUS_RW_PASSWORD:-}"
export K6_PROMETHEUS_RW_TREND_STATS="${K6_PROMETHEUS_RW_TREND_STATS:-p(95),p(99),min,max}"
export K6_PROMETHEUS_RW_PUSH_INTERVAL="${K6_PROMETHEUS_RW_PUSH_INTERVAL:-5s}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if k6 is installed
check_k6() {
    if ! command -v k6 &> /dev/null; then
        log_error "k6 is not installed. Please install k6 first."
        echo "macOS: brew install k6"
        echo "Ubuntu/Debian: See README.md for installation instructions"
        exit 1
    fi
}

# Function to validate environment
validate_env() {
    if [[ -z "$BASE" ]]; then
        log_error "BASE URL is required"
        exit 1
    fi
    
    # Check for Cloudflare load test token
    if [[ -z "$LOADTEST_TOKEN" ]]; then
        log_warning "LOADTEST_TOKEN is not set - this may be required for Cloudflare API access"
        log_info "Set LOADTEST_TOKEN environment variable or add it to .env file"
    else
        log_info "Cloudflare load test token configured: [CONFIGURED]"
    fi
    
    # Guest authentication - no API keys needed
    log_info "Using guest authentication (no API keys required)"
    
    # Validate Prometheus endpoint format if provided
    if [[ -n "$K6_PROMETHEUS_RW_SERVER_URL" ]]; then
        if [[ ! "$K6_PROMETHEUS_RW_SERVER_URL" =~ ^https?:// ]]; then
            log_error "K6_PROMETHEUS_RW_SERVER_URL must start with http:// or https://"
            exit 1
        fi
        log_info "Prometheus remote write endpoint configured"
    fi
}

# Function to get all available test cases by scanning src folder
get_available_test_cases() {
    local script_dir=$(dirname "$0")
    local src_dir="$script_dir/src"
    
    if [[ ! -d "$src_dir" ]]; then
        log_error "Source directory not found: $src_dir"
        return 1
    fi
    
    # Find all .js files in src directory and extract base names
    find "$src_dir" -name "*.js" -type f | while read -r file; do
        basename "$file" .js
    done | sort
}

# Function to run all test cases
run_all_test_cases() {
    local available_tests=($(get_available_test_cases))
    local failed_tests=()
    local total_tests=${#available_tests[@]}
    
    log_info "Running all test cases (${total_tests} total)"
    log_info "===================================================="
    
    for test_case in "${available_tests[@]}"; do
        log_info ""
        log_info "📋 Running test case: $test_case"
        log_info "----------------------------------------------------"
        
        if run_single_test_case "$test_case"; then
            log_success "✅ Test case '$test_case' completed successfully"
        else
            log_error "❌ Test case '$test_case' failed"
            failed_tests+=("$test_case")
        fi
        
        # Add a delay between tests to avoid overwhelming the system
        if [[ ${#available_tests[@]} -gt 1 ]]; then
            log_info "Waiting 10 seconds before next test..."
            sleep 10
        fi
    done
    
    # Summary
    log_info ""
    log_info "===================================================="
    log_info "📊 TEST EXECUTION SUMMARY"
    log_info "===================================================="
    log_info "Total tests: $total_tests"
    log_info "Passed: $((total_tests - ${#failed_tests[@]}))"
    log_info "Failed: ${#failed_tests[@]}"
    
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        log_success "🎉 All tests passed!"
        return 0
    else
        log_error "💥 Failed tests: ${failed_tests[*]}"
        return 1
    fi
}

# Function to run a specific test case (renamed from run_test_case)
run_single_test_case() {
    local test_case="$1"
    local script_dir=$(dirname "$0")
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local results_dir="$script_dir/results"
    local output_file="$results_dir/${test_case}_${timestamp}.json"
    local test_file="$script_dir/src/${test_case}.js"
    
    # Check if test file exists
    if [[ ! -f "$test_file" ]]; then
        log_error "Test file not found: $test_file"
        log_info "Available test cases:"
        local available_tests=($(get_available_test_cases))
        for available_test in "${available_tests[@]}"; do
            log_info "  - $available_test"
        done
        return 1
    fi
    
    # Create results directory if it doesn't exist
    mkdir -p "$results_dir"
    
    log_info "Running test case: $test_case"
    log_info "Test file: $test_file"
    log_info "Running command: k6 run $test_file"
    log_info "Configuration:"
    log_info "  Base URL: $BASE"
    log_info "  Model: $MODEL"
    log_info "  Duration: ${DURATION_MIN} minutes"
    log_info "  Non-stream RPS: $NONSTREAM_RPS"
    log_info "  Stream RPS: $STREAM_RPS"
    log_info "  Debug Mode: $DEBUG"
    log_info "  Single Run: $SINGLE_RUN"
    if [[ -n "$LOADTEST_TOKEN" ]]; then
        log_info "  Load Test Token: [CONFIGURED]"
    else
        log_info "  Load Test Token: [NOT SET]"
    fi
    log_info "  Output: $output_file"
    
    # Generate unique test ID for metrics segmentation
    local test_id="${test_case}_$(date +%Y%m%d_%H%M%S)_$$"
    log_info "Test ID: $test_id"
    
    # Execute k6 with conditional Prometheus output
    if [[ -n "$K6_PROMETHEUS_RW_SERVER_URL" ]]; then
        log_info "Prometheus remote write endpoint configured: [CONFIGURED]"
        
        # Validate that it's not localhost in CI environment
        if [[ "$K6_PROMETHEUS_RW_SERVER_URL" == *"localhost"* ]] || [[ "$K6_PROMETHEUS_RW_SERVER_URL" == *"127.0.0.1"* ]]; then
            log_warning "Prometheus endpoint appears to be localhost - this will not work in CI environment!"
        fi
        
        # Set optional k6 environment variables for Prometheus remote write
        if [[ -n "$K6_PROMETHEUS_RW_USERNAME" ]]; then
            log_info "Using Prometheus with basic auth (username: [CONFIGURED])"
            export K6_PROMETHEUS_RW_USERNAME
        fi
        
        if [[ -n "$K6_PROMETHEUS_RW_PASSWORD" ]]; then
            export K6_PROMETHEUS_RW_PASSWORD
        fi
        
        export K6_PROMETHEUS_RW_TREND_STATS
        export K6_PROMETHEUS_RW_PUSH_INTERVAL
        
        # Run k6 with Prometheus remote write output
        log_info "Running with Prometheus remote write metrics export"
        log_info "Trend stats: $K6_PROMETHEUS_RW_TREND_STATS"
        log_info "Push interval: $K6_PROMETHEUS_RW_PUSH_INTERVAL"
        
        k6 run \
            --summary-export="$output_file" \
            --out json="$output_file" \
            --out experimental-prometheus-rw \
            --tag testid="$test_id" \
            --tag test_case="$test_case" \
            --tag environment="${BASE##*/}" \
            "$test_file"
    else
        log_info "No Prometheus endpoint configured, running without metrics export"
        k6 run \
            --summary-export="$output_file" \
            --out json="$output_file" \
            --tag testid="$test_id" \
            --tag test_case="$test_case" \
            --tag environment="${BASE##*/}" \
            "$test_file"
    fi
    
    # Check if test completed successfully
    if [[ $? -eq 0 ]]; then
        log_success "Test case '$test_case' completed successfully"
        
        # Extract and display key metrics
        if [[ -f "$output_file" ]]; then
            log_info "Test Results Summary:"
            
            # Parse JSON output for key metrics (requires jq)
            if command -v jq &> /dev/null; then
                echo "==================== METRICS SUMMARY ===================="
                jq -r '.metrics | to_entries[] | select(.key | contains("completion_") or contains("conversation_") or contains("response_") or contains("guest_") or contains("refresh_")) | "\(.key): \(.value.avg // .value.count)"' "$output_file" 2>/dev/null || true
                echo "=========================================================="
            fi
            
            if [[ -n "$K6_PROMETHEUS_RW_SERVER_URL" ]]; then
                log_success "Metrics sent to Prometheus directly via k6"
            fi
        fi
        return 0
    else
        log_error "Test case '$test_case' failed"
        return 1
    fi
}

# Function to list available test cases
list_test_cases() {
    local available_tests=($(get_available_test_cases))
    
    if [[ ${#available_tests[@]} -eq 0 ]]; then
        log_warning "No test cases found in src/ directory"
        log_info "Create .js files in src/ directory to add test cases"
        return 1
    fi
    
    log_info "Available test cases (${#available_tests[@]} total):"
    for test_case in "${available_tests[@]}"; do
        local test_file="src/${test_case}.js"
        if [[ -f "$test_file" ]]; then
            log_info "  - $test_case (src/${test_case}.js)"
        else
            log_warning "  - $test_case (file missing: $test_file)"
        fi
    done
    log_info ""
    log_info "Usage:"
    log_info "  $0                    # Run all test cases"
    log_info "  $0 [test_case_name]   # Run specific test case"
    log_info ""
    log_info "Examples:"
    log_info "  $0                                    # Run all tests"
    log_info "  $0 test-completion-standard          # Run only standard completion test"
    log_info "  $0 test-completion-conversation      # Run only conversation test"
    log_info "  $0 test-responses                     # Run only response API test"
    log_info "  $0 --list                             # Show this help"
}

# Main execution
main() {
    local test_case="$1"
    
    log_info "Jan Server Load Test Runner"
    log_info "============================"
    
    # Check prerequisites
    check_k6
    validate_env
    
    # Handle different arguments
    case "$test_case" in
        "--list"|"-l"|"--help"|"-h")
            list_test_cases
            exit 0
            ;;
        "")
            # No argument provided - run all test cases
            log_info "No specific test case provided, running all test cases..."
            run_all_test_cases
            exit $?
            ;;
        *)
            # Specific test case provided
            log_info "Running specific test case: $test_case"
            run_single_test_case "$test_case"
            exit $?
            ;;
    esac
}

# Execute main function with all arguments
main "$@"
