#!/bin/bash

# Cleanup all test databases created during testing
# This script helps verify that tests can create their own databases from scratch

set -e

# Database connection parameters
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-codeatlas}"
DB_PASSWORD="${DB_PASSWORD:-codeatlas}"

echo "🧹 Cleaning up test databases..."
echo "Host: $DB_HOST:$DB_PORT"
echo "User: $DB_USER"
echo ""

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
            echo "❌ Error: psql command not found and postgres container is not running"
            echo "Please either:"
            echo "  1. Install PostgreSQL client tools (psql)"
            echo "  2. Start the postgres container: make docker-db"
            exit 1
        fi
    else
        # Use local psql
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -t -c "$query"
    fi
}

# Get list of all test databases
TEST_DBS=$(run_psql "SELECT datname FROM pg_database WHERE datname LIKE 'codeatlas_test_%';")

if [ -z "$TEST_DBS" ]; then
    echo "✅ No test databases found. All clean!"
    exit 0
fi

echo "Found test databases:"
echo "$TEST_DBS"
echo ""

# Count databases
DB_COUNT=$(echo "$TEST_DBS" | wc -l | tr -d ' ')
echo "Total: $DB_COUNT test database(s)"
echo ""

# Ask for confirmation
read -p "⚠️  Delete all test databases? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Cancelled"
    exit 1
fi

# Drop each test database
DROPPED=0
FAILED=0

for db in $TEST_DBS; do
    # Trim whitespace
    db=$(echo "$db" | xargs)
    
    if [ -n "$db" ]; then
        echo "Dropping: $db"
        
        # Terminate all connections to the database first
        run_psql "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$db';" \
            > /dev/null 2>&1 || true
        
        # Drop the database
        if run_psql "DROP DATABASE IF EXISTS $db;" > /dev/null 2>&1; then
            echo "  ✅ Dropped"
            ((DROPPED++))
        else
            echo "  ❌ Failed"
            ((FAILED++))
        fi
    fi
done

echo ""
echo "📊 Summary:"
echo "  Dropped: $DROPPED"
echo "  Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "✅ All test databases cleaned up successfully!"
else
    echo "⚠️  Some databases could not be dropped. They may be in use."
    exit 1
fi
