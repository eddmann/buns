#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXAMPLES_DIR="$SCRIPT_DIR"

echo "========================================"
echo "  buns Examples Test Suite"
echo "========================================"
echo ""

# Helper function to run a test
run_test() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    printf "%-45s" "$name"

    # Run command and capture output
    if output=$(eval "$cmd" 2>&1); then
        # Check if expected string is in output (if provided)
        if [ -n "$expected" ]; then
            if echo "$output" | grep -qF "$expected"; then
                echo -e "${GREEN}PASS${NC}"
                ((PASS++))
                return 0
            else
                echo -e "${RED}FAIL${NC}"
                echo "  Expected: $expected"
                echo "  Got: $output"
                ((FAIL++))
                return 1
            fi
        else
            echo -e "${GREEN}PASS${NC}"
            ((PASS++))
            return 0
        fi
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Error: $output"
        ((FAIL++))
        return 1
    fi
}

# Helper for tests that should show specific blocked behavior
run_test_contains() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    printf "%-45s" "$name"

    # Run command and capture output (allow non-zero exit)
    output=$(eval "$cmd" 2>&1) || true

    if echo "$output" | grep -qF "$expected"; then
        echo -e "${GREEN}PASS${NC}"
        ((PASS++))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Expected to contain: $expected"
        echo "  Got: $output"
        ((FAIL++))
        return 1
    fi
}

echo "Basic Examples"
echo "----------------------------------------"

# 01 - Hello World
run_test "01-hello-world.ts" \
    "buns '$EXAMPLES_DIR/01-hello-world.ts'" \
    "Hello from buns!"

# 02 - Bun Version
run_test "02-bun-version.ts" \
    "buns '$EXAMPLES_DIR/02-bun-version.ts'" \
    "Bun version:"

# 03 - CLI Arguments
run_test "03-cli-arguments.ts" \
    "buns '$EXAMPLES_DIR/03-cli-arguments.ts' -- hello world" \
    "Total: 2 arguments"

echo ""
echo "Package Management"
echo "----------------------------------------"

# 04 - Single Package (chalk)
run_test "04-single-package.ts" \
    "buns '$EXAMPLES_DIR/04-single-package.ts'" \
    "Success:"

# 05 - Multiple Packages
run_test "05-multiple-packages.ts" \
    "buns '$EXAMPLES_DIR/05-multiple-packages.ts'" \
    "Current time:"

echo ""
echo "Bun Constraints"
echo "----------------------------------------"

# 06 - Bun Constraint
run_test "06-bun-constraint.ts" \
    "buns '$EXAMPLES_DIR/06-bun-constraint.ts'" \
    "Bun version:"

echo ""
echo "HTTP & Data Processing"
echo "----------------------------------------"

# 07 - HTTP Client
run_test "07-http-client.ts" \
    "buns '$EXAMPLES_DIR/07-http-client.ts'" \
    "Name:"

# 08 - JSON Processing
run_test "08-json-processing.ts" \
    "echo '{\"test\": 123}' | buns '$EXAMPLES_DIR/08-json-processing.ts'" \
    "Parsed JSON:"

# Note: 09-cli-app.ts is interactive, skip automated test
echo -e "09-cli-app.ts                                ${YELLOW}SKIP${NC} (interactive)"

echo ""
echo "Sandbox Features"
echo "----------------------------------------"

# 10 - Sandbox Basic
run_test "10-sandbox-basic.ts" \
    "buns '$EXAMPLES_DIR/10-sandbox-basic.ts' --sandbox --memory 64 --timeout 10 --cpu 5" \
    "Bun Version:"

# 11 - Sandbox Offline
run_test_contains "11-sandbox-offline.ts" \
    "buns '$EXAMPLES_DIR/11-sandbox-offline.ts' --offline" \
    "Network blocked"

# 12 - Sandbox Allow Host
run_test_contains "12-sandbox-allow-host.ts" \
    "buns '$EXAMPLES_DIR/12-sandbox-allow-host.ts' --allow-host httpbin.org" \
    "[allowed] httpbin.org"

# 13 - Sandbox Filesystem
echo "hello" > /tmp/buns-test.txt
run_test "13-sandbox-filesystem.ts" \
    "buns '$EXAMPLES_DIR/13-sandbox-filesystem.ts' --sandbox --allow-read /tmp --allow-write /tmp" \
    "Write: OK"
rm -f /tmp/buns-test.txt /tmp/buns-output.txt

# 14 - Sandbox Env
run_test_contains "14-sandbox-env.ts" \
    "API_KEY=secret123 DEBUG=1 buns '$EXAMPLES_DIR/14-sandbox-env.ts' --sandbox --allow-env API_KEY,DEBUG" \
    "API_KEY: set"

echo ""
echo "========================================"
echo -e "  Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"
echo "========================================"

if [ $FAIL -gt 0 ]; then
    exit 1
fi
