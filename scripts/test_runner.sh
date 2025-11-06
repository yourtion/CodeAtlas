#!/bin/bash

# Test runner with enhanced output formatting
# Highlights failures and provides statistics

set -o pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Temporary files
TEMP_OUTPUT=$(mktemp)
FAILURES_FILE=$(mktemp)
STATS_FILE=$(mktemp)

# Cleanup on exit
trap "rm -f $TEMP_OUTPUT $FAILURES_FILE $STATS_FILE" EXIT

# Parse command line arguments
TEST_CMD="$@"
if [ -z "$TEST_CMD" ]; then
    echo "Usage: $0 <go test command>"
    echo "Example: $0 go test ./... -v"
    exit 1
fi

# Print header
echo -e "${BOLD}${BLUE}========================================${NC}"
echo -e "${BOLD}${BLUE}  CodeAtlas Test Runner${NC}"
echo -e "${BOLD}${BLUE}========================================${NC}"
echo -e "Command: ${YELLOW}$TEST_CMD${NC}\n"

# Run tests and capture output
START_TIME=$(date +%s)
$TEST_CMD 2>&1 | tee $TEMP_OUTPUT
TEST_EXIT_CODE=${PIPESTATUS[0]}
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Parse test output for statistics
TOTAL_TESTS=$(grep -E "^(PASS|FAIL):" $TEMP_OUTPUT | wc -l | tr -d ' ')
PASSED_TESTS=$(grep -E "^PASS:" $TEMP_OUTPUT | wc -l | tr -d ' ')
FAILED_TESTS=$(grep -E "^FAIL:" $TEMP_OUTPUT | wc -l | tr -d ' ')
SKIPPED_TESTS=$(grep -c "SKIP" $TEMP_OUTPUT || echo "0")

# Extract failed test details
grep -B 5 -A 10 "FAIL:" $TEMP_OUTPUT > $FAILURES_FILE 2>/dev/null || true
grep -B 2 "--- FAIL:" $TEMP_OUTPUT >> $FAILURES_FILE 2>/dev/null || true

# Print separator
echo -e "\n${BOLD}${BLUE}========================================${NC}"
echo -e "${BOLD}${BLUE}  Test Results Summary${NC}"
echo -e "${BOLD}${BLUE}========================================${NC}\n"

# Print statistics
echo -e "${BOLD}Statistics:${NC}"
echo -e "  Total Tests:   ${BOLD}$TOTAL_TESTS${NC}"
echo -e "  ${GREEN}✓ Passed:${NC}      ${GREEN}$PASSED_TESTS${NC}"
echo -e "  ${RED}✗ Failed:${NC}      ${RED}$FAILED_TESTS${NC}"
echo -e "  ${YELLOW}⊘ Skipped:${NC}     ${YELLOW}$SKIPPED_TESTS${NC}"
echo -e "  Duration:      ${BOLD}${DURATION}s${NC}\n"

# Calculate pass rate
if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$(awk "BEGIN {printf \"%.1f\", ($PASSED_TESTS/$TOTAL_TESTS)*100}")
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "  Pass Rate:     ${GREEN}${BOLD}${PASS_RATE}%${NC} ${GREEN}✓${NC}"
    else
        echo -e "  Pass Rate:     ${YELLOW}${BOLD}${PASS_RATE}%${NC}"
    fi
    echo ""
fi

# Print failed tests if any
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${BOLD}${RED}========================================${NC}"
    echo -e "${BOLD}${RED}  Failed Tests Details${NC}"
    echo -e "${BOLD}${RED}========================================${NC}\n"
    
    # Extract and format failed test names
    grep -E "--- FAIL:" $TEMP_OUTPUT | while read -r line; do
        TEST_NAME=$(echo "$line" | sed 's/--- FAIL: //')
        echo -e "${RED}✗${NC} ${BOLD}$TEST_NAME${NC}"
    done
    
    echo -e "\n${BOLD}${RED}Failure Details:${NC}\n"
    
    # Print failure context
    cat $FAILURES_FILE | grep -E "(FAIL:|Error|panic|fatal|expected|actual|got|want)" | \
        sed "s/FAIL:/${RED}FAIL:${NC}/g" | \
        sed "s/Error/${RED}Error${NC}/g" | \
        sed "s/panic/${RED}panic${NC}/g" | \
        sed "s/fatal/${RED}fatal${NC}/g"
    
    echo ""
fi

# Print coverage info if available
if grep -q "coverage:" $TEMP_OUTPUT; then
    echo -e "${BOLD}${BLUE}========================================${NC}"
    echo -e "${BOLD}${BLUE}  Coverage Information${NC}"
    echo -e "${BOLD}${BLUE}========================================${NC}\n"
    grep "coverage:" $TEMP_OUTPUT | tail -5
    echo ""
fi

# Print final status
echo -e "${BOLD}${BLUE}========================================${NC}"
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}${BOLD}  ✓ ALL TESTS PASSED${NC}"
else
    echo -e "${RED}${BOLD}  ✗ TESTS FAILED${NC}"
fi
echo -e "${BOLD}${BLUE}========================================${NC}\n"

# Exit with the same code as the test command
exit $TEST_EXIT_CODE
