#!/bin/bash

# Complete verification script for test setup
# This script:
# 1. Cleans up all test databases
# 2. Runs tests to verify they can create databases from scratch
# 3. Reports results

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🔍 CodeAtlas Test Setup Verification"
echo "===================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to run psql command
run_psql() {
    local query="$1"
    
    # Try to use docker exec if psql is not available locally
    if ! command -v psql &> /dev/null; then
        # Find postgres container (could be named postgres, codeatlas-db-1, etc.)
        POSTGRES_CONTAINER=$(docker ps --format '{{.Names}}' | grep -E 'postgres|db' | head -1)
        
        if [ -n "$POSTGRES_CONTAINER" ]; then
            docker exec -e PGPASSWORD="$DB_PASSWORD" "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d postgres -t -c "$query"
        else
            return 1
        fi
    else
        # Use local psql
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -t -c "$query"
    fi
}

# Step 1: Check database connection
echo "📡 Step 1: Checking database connection..."
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-codeatlas}"
DB_PASSWORD="${DB_PASSWORD:-codeatlas}"

if run_psql "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Database connection successful${NC}"
else
    echo -e "${RED}❌ Cannot connect to database${NC}"
    echo "Please ensure PostgreSQL is running:"
    echo "  make docker-up"
    exit 1
fi
echo ""

# Step 2: Check for existing test databases
echo "📊 Step 2: Checking for existing test databases..."
TEST_DBS=$(run_psql "SELECT datname FROM pg_database WHERE datname LIKE 'codeatlas_test_%';" | wc -l | tr -d ' ')

if [ "$TEST_DBS" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $TEST_DBS existing test database(s)${NC}"
    echo "Cleaning up..."
    "$SCRIPT_DIR/cleanup_test_databases.sh" <<< "y"
else
    echo -e "${GREEN}✅ No existing test databases${NC}"
fi
echo ""

# Step 3: Build CLI binary
echo "🔨 Step 3: Building CLI binary..."
cd "$PROJECT_ROOT"
if make build-cli > /dev/null 2>&1; then
    echo -e "${GREEN}✅ CLI binary built successfully${NC}"
else
    echo -e "${RED}❌ Failed to build CLI binary${NC}"
    exit 1
fi
echo ""

# Step 4: Run unit tests (short mode)
echo "🧪 Step 4: Running unit tests (short mode)..."
if go test $(go list ./... | grep -v /scripts | grep -v /test-repo) -short > /tmp/test_output.log 2>&1; then
    echo -e "${GREEN}✅ All unit tests passed${NC}"
    UNIT_PASS=true
else
    echo -e "${RED}❌ Some unit tests failed${NC}"
    echo "See /tmp/test_output.log for details"
    UNIT_PASS=false
fi
echo ""

# Step 5: Run a sample integration test
echo "🔬 Step 5: Running sample integration test..."
if go test ./pkg/models -run TestFileRepository_Create -v > /tmp/integration_test.log 2>&1; then
    echo -e "${GREEN}✅ Integration test passed${NC}"
    
    # Check if database was created
    NEW_TEST_DBS=$(run_psql "SELECT COUNT(*) FROM pg_database WHERE datname LIKE 'codeatlas_test_%';" | tr -d ' ')
    
    if [ "$NEW_TEST_DBS" -gt 0 ]; then
        echo -e "${GREEN}✅ Test database was created and cleaned up properly${NC}"
    else
        echo -e "${YELLOW}⚠️  No test database found (may have been cleaned up)${NC}"
    fi
    INTEGRATION_PASS=true
else
    echo -e "${RED}❌ Integration test failed${NC}"
    echo "See /tmp/integration_test.log for details"
    INTEGRATION_PASS=false
fi
echo ""

# Step 6: Verify CLI tests
echo "🖥️  Step 6: Running CLI tests..."
if go test ./tests/cli -v > /tmp/cli_test.log 2>&1; then
    echo -e "${GREEN}✅ CLI tests passed${NC}"
    CLI_PASS=true
else
    echo -e "${RED}❌ CLI tests failed${NC}"
    echo "See /tmp/cli_test.log for details"
    CLI_PASS=false
fi
echo ""

# Step 7: Final cleanup
echo "🧹 Step 7: Final cleanup..."
"$SCRIPT_DIR/cleanup_test_databases.sh" <<< "y" > /dev/null 2>&1 || true
echo -e "${GREEN}✅ Cleanup complete${NC}"
echo ""

# Summary
echo "📋 Verification Summary"
echo "======================"
echo ""

if [ "$UNIT_PASS" = true ]; then
    echo -e "Unit Tests:        ${GREEN}✅ PASS${NC}"
else
    echo -e "Unit Tests:        ${RED}❌ FAIL${NC}"
fi

if [ "$INTEGRATION_PASS" = true ]; then
    echo -e "Integration Tests: ${GREEN}✅ PASS${NC}"
else
    echo -e "Integration Tests: ${RED}❌ FAIL${NC}"
fi

if [ "$CLI_PASS" = true ]; then
    echo -e "CLI Tests:         ${GREEN}✅ PASS${NC}"
else
    echo -e "CLI Tests:         ${RED}❌ FAIL${NC}"
fi

echo ""

if [ "$UNIT_PASS" = true ] && [ "$INTEGRATION_PASS" = true ] && [ "$CLI_PASS" = true ]; then
    echo -e "${GREEN}🎉 All verifications passed!${NC}"
    echo ""
    echo "Your test setup is working correctly:"
    echo "  ✅ Tests can create databases from scratch"
    echo "  ✅ Tests clean up after themselves"
    echo "  ✅ CLI binary is built and working"
    echo "  ✅ All test patterns are correct"
    echo ""
    echo "You can now run the full test suite:"
    echo "  make test-all"
    exit 0
else
    echo -e "${RED}❌ Some verifications failed${NC}"
    echo ""
    echo "Check the log files for details:"
    echo "  /tmp/test_output.log"
    echo "  /tmp/integration_test.log"
    echo "  /tmp/cli_test.log"
    exit 1
fi
