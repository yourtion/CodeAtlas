#!/bin/bash
# Test script to verify devcontainer setup

set -e

echo "🧪 Testing CodeAtlas DevContainer Setup"
echo "========================================"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function
test_command() {
    local description=$1
    local command=$2
    
    echo -n "Testing: $description... "
    if eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test Go installation
test_command "Go installation" "go version"
test_command "Go modules" "go mod verify"

# Test Go tools
test_command "gopls (Go language server)" "which gopls"
test_command "golangci-lint" "which golangci-lint"
test_command "delve debugger" "which dlv"

# Test Node.js and pnpm
test_command "Node.js installation" "node --version"
test_command "pnpm installation" "pnpm --version"

# Test database connection
test_command "PostgreSQL client" "which psql"
echo -n "Testing: Database connection... "
if pg_isready -h db -U codeatlas -d codeatlas > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC}"
    ((TESTS_FAILED++))
fi

# Test database schema
echo -n "Testing: Database schema... "
if psql -h db -U codeatlas -d codeatlas -c "SELECT COUNT(*) FROM repositories;" > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC}"
    ((TESTS_FAILED++))
fi

# Test database seed data
echo -n "Testing: Seed data... "
REPO_COUNT=$(psql -h db -U codeatlas -d codeatlas -t -c "SELECT COUNT(*) FROM repositories;" 2>/dev/null | xargs)
if [ "$REPO_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ ($REPO_COUNT repositories)${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⚠ (no data)${NC}"
fi

# Test build
echo -n "Testing: Project build... "
if make build > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC}"
    ((TESTS_FAILED++))
fi

# Test binaries
test_command "API binary" "test -f bin/api"
test_command "CLI binary" "test -f bin/cli"

# Summary
echo ""
echo "========================================"
echo "Test Results:"
echo -e "  Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "  Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed! DevContainer is ready.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Please check the setup.${NC}"
    exit 1
fi
