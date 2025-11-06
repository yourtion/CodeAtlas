#!/bin/bash

# CI-friendly test runner with JSON output and failure extraction
# Designed for CI/CD pipelines with structured output

set -o pipefail

# Temporary files
TEMP_OUTPUT=$(mktemp)
JSON_OUTPUT=$(mktemp)

# Cleanup on exit
trap "rm -f $TEMP_OUTPUT $JSON_OUTPUT" EXIT

# Parse command line arguments
TEST_CMD="$@"
if [ -z "$TEST_CMD" ]; then
    echo "Usage: $0 <go test command>"
    exit 1
fi

# Print CI header
echo "========================================="
echo "CodeAtlas CI Test Runner"
echo "========================================="
echo "Command: $TEST_CMD"
echo ""

# Run tests with JSON output
START_TIME=$(date +%s)
$TEST_CMD -json 2>&1 | tee $TEMP_OUTPUT
TEST_EXIT_CODE=${PIPESTATUS[0]}
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Parse JSON output for statistics
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Arrays to store failed test info
declare -a FAILED_TEST_NAMES
declare -a FAILED_TEST_PACKAGES
declare -a FAILED_TEST_OUTPUTS

# Process JSON output
while IFS= read -r line; do
    if echo "$line" | grep -q '"Action":"pass"'; then
        if echo "$line" | grep -q '"Test":'; then
            ((PASSED_TESTS++))
            ((TOTAL_TESTS++))
        fi
    elif echo "$line" | grep -q '"Action":"fail"'; then
        if echo "$line" | grep -q '"Test":'; then
            ((FAILED_TESTS++))
            ((TOTAL_TESTS++))
            
            # Extract test name and package
            TEST_NAME=$(echo "$line" | grep -o '"Test":"[^"]*"' | cut -d'"' -f4)
            PACKAGE=$(echo "$line" | grep -o '"Package":"[^"]*"' | cut -d'"' -f4)
            
            FAILED_TEST_NAMES+=("$TEST_NAME")
            FAILED_TEST_PACKAGES+=("$PACKAGE")
        fi
    elif echo "$line" | grep -q '"Action":"skip"'; then
        if echo "$line" | grep -q '"Test":'; then
            ((SKIPPED_TESTS++))
        fi
    fi
done < $TEMP_OUTPUT

# Calculate pass rate
PASS_RATE=0
if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$(awk "BEGIN {printf \"%.1f\", ($PASSED_TESTS/$TOTAL_TESTS)*100}")
fi

# Print summary
echo ""
echo "========================================="
echo "Test Results Summary"
echo "========================================="
echo "Total Tests:   $TOTAL_TESTS"
echo "Passed:        $PASSED_TESTS"
echo "Failed:        $FAILED_TESTS"
echo "Skipped:       $SKIPPED_TESTS"
echo "Pass Rate:     ${PASS_RATE}%"
echo "Duration:      ${DURATION}s"
echo ""

# Print failed tests if any
if [ $FAILED_TESTS -gt 0 ]; then
    echo "========================================="
    echo "Failed Tests"
    echo "========================================="
    
    for i in "${!FAILED_TEST_NAMES[@]}"; do
        echo "[$((i+1))] ${FAILED_TEST_PACKAGES[$i]}"
        echo "    Test: ${FAILED_TEST_NAMES[$i]}"
        echo ""
    done
    
    echo "========================================="
    echo "Failure Details"
    echo "========================================="
    
    # Extract failure messages from output
    grep -A 20 "FAIL:" $TEMP_OUTPUT | grep -E "(Error|panic|fatal|expected|actual|got|want)" | head -50
    echo ""
fi

# Generate JSON report for CI systems
cat > $JSON_OUTPUT <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "duration_seconds": $DURATION,
  "total_tests": $TOTAL_TESTS,
  "passed": $PASSED_TESTS,
  "failed": $FAILED_TESTS,
  "skipped": $SKIPPED_TESTS,
  "pass_rate": $PASS_RATE,
  "exit_code": $TEST_EXIT_CODE,
  "failed_tests": [
EOF

# Add failed test details to JSON
for i in "${!FAILED_TEST_NAMES[@]}"; do
    if [ $i -gt 0 ]; then
        echo "," >> $JSON_OUTPUT
    fi
    cat >> $JSON_OUTPUT <<EOF
    {
      "package": "${FAILED_TEST_PACKAGES[$i]}",
      "test": "${FAILED_TEST_NAMES[$i]}"
    }
EOF
done

cat >> $JSON_OUTPUT <<EOF
  ]
}
EOF

# Save JSON report
REPORT_FILE="test_report_$(date +%Y%m%d_%H%M%S).json"
cp $JSON_OUTPUT "$REPORT_FILE"
echo "JSON report saved to: $REPORT_FILE"
echo ""

# Print final status
echo "========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✓ ALL TESTS PASSED"
else
    echo "✗ TESTS FAILED"
    echo ""
    echo "Quick Summary:"
    echo "  - $FAILED_TESTS test(s) failed"
    echo "  - Check details above"
    echo "  - Full report: $REPORT_FILE"
fi
echo "========================================="

# Exit with the same code as the test command
exit $TEST_EXIT_CODE
