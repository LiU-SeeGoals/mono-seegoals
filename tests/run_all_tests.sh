#!/bin/bash

# Test runner for all integration tests
# Runs C unit tests and integration tests

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

PROJECT_ROOT="$(pwd)/.."
TESTS_DIR="$(pwd)/unit"
FIRMWARE_SHARED="$PROJECT_ROOT/firmware/shared"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║                   Protocol Test Suite                      ║${NC}"
echo -e "${YELLOW}║           Custom Binary Protocol - All Tests               ║${NC}"
echo -e "${YELLOW}╚════════════════════════════════════════════════════════════╝${NC}\n"

# Verify paths
if [ ! -f "$FIRMWARE_SHARED/src/robot_command.c" ]; then
    echo -e "${RED}Error: Could not find $FIRMWARE_SHARED/src/robot_command.c${NC}"
    exit 1
fi

# Track results
TOTAL_PASSED=0
TOTAL_FAILED=0
TESTS_RUN=0

# Function to run a test
run_test() {
    local test_name=$1
    local output_file=$2
    
    echo -e "${YELLOW}Running: $test_name${NC}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if "$output_file" > /tmp/test_output.txt 2>&1; then
        cat /tmp/test_output.txt
        echo -e "${GREEN}✓ $test_name PASSED${NC}\n"
        TOTAL_PASSED=$((TOTAL_PASSED + 1))
        return 0
    else
        cat /tmp/test_output.txt
        echo -e "${RED}✗ $test_name FAILED${NC}\n"
        TOTAL_FAILED=$((TOTAL_FAILED + 1))
        return 1
    fi
}

# Compile and run C unit tests

echo -e "${YELLOW}═════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}C Unit Tests - Protocol Encoder/Decoder${NC}"
echo -e "${YELLOW}═════════════════════════════════════════════════════════════${NC}\n"

# Test 1: Protocol encoder/decoder
echo -e "${YELLOW}[Compiling] robot_command_test...${NC}"
if gcc -I"$FIRMWARE_SHARED/inc" \
    -Wall -Wextra \
    "$TESTS_DIR/c/robot_command_test.c" \
    "$FIRMWARE_SHARED/src/robot_command.c" \
    -o /tmp/robot_command_test 2>/tmp/compile_err.txt; then
    run_test "Protocol Encoder/Decoder Tests" /tmp/robot_command_test
else
    echo -e "${RED}✗ Failed to compile robot_command_test${NC}"
    cat /tmp/compile_err.txt
    echo ""
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
fi

echo -e "${YELLOW}═════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}Integration Tests - Firmware Communication${NC}"
echo -e "${YELLOW}═════════════════════════════════════════════════════════════${NC}\n"

# Test 2: Firmware integration
echo -e "${YELLOW}[Compiling] firmware_integration_test...${NC}"
if gcc -I"$FIRMWARE_SHARED/inc" \
    -Wall -Wextra \
    "$TESTS_DIR/c/firmware_integration_test.c" \
    "$FIRMWARE_SHARED/src/robot_command.c" \
    -o /tmp/firmware_integration_test 2>/tmp/compile_err.txt; then
    run_test "Firmware Integration Tests" /tmp/firmware_integration_test
else
    echo -e "${RED}✗ Failed to compile firmware_integration_test${NC}"
    cat /tmp/compile_err.txt
    echo ""
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
fi

# Test 3: COM layer
echo -e "${YELLOW}[Compiling] com_layer_test...${NC}"
if gcc -I"$FIRMWARE_SHARED/inc" \
    -Wall -Wextra \
    "$TESTS_DIR/c/com_layer_test.c" \
    "$FIRMWARE_SHARED/src/robot_command.c" \
    -o /tmp/com_layer_test 2>/tmp/compile_err.txt; then
    run_test "COM Layer Tests" /tmp/com_layer_test
else
    echo -e "${RED}✗ Failed to compile com_layer_test${NC}"
    cat /tmp/compile_err.txt
    echo ""
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
fi

# Test 4: NAV layer
echo -e "${YELLOW}[Compiling] nav_layer_test...${NC}"
if gcc -I"$FIRMWARE_SHARED/inc" \
    -Wall -Wextra \
    "$TESTS_DIR/c/nav_layer_test.c" \
    "$FIRMWARE_SHARED/src/robot_command.c" \
    -o /tmp/nav_layer_test 2>/tmp/compile_err.txt; then
    run_test "NAV Layer Tests" /tmp/nav_layer_test
else
    echo -e "${RED}✗ Failed to compile nav_layer_test${NC}"
    cat /tmp/compile_err.txt
    echo ""
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
fi

# Print summary
echo -e "${YELLOW}═════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}Test Suite Summary${NC}"
echo -e "${YELLOW}═════════════════════════════════════════════════════════════${NC}\n"

echo "Total Tests Run: $TESTS_RUN"
echo -e "Passed: ${GREEN}$TOTAL_PASSED${NC}"
echo -e "Failed: ${RED}$TOTAL_FAILED${NC}\n"

if [ $TOTAL_FAILED -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           ✓ ALL TESTS PASSED - READY TO DEPLOY ✓           ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}\n"
    exit 0
else
    echo -e "${RED}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║                  ✗ SOME TESTS FAILED ✗                     ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════╝${NC}\n"
    exit 1
fi
