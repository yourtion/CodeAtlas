#!/bin/bash
set -e

# CodeAtlas Database Migration Script
# This script runs database migrations in order
# Usage: ./migrate.sh [docker|direct] [migration_version]

DEPLOYMENT_TYPE="${1:-docker}"
MIGRATION_VERSION="${2:-all}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/deployments/migrations"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Run migration via Docker
run_migration_docker() {
    local migration_file=$1
    log_info "Running migration: $(basename $migration_file)"
    
    docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.prod.yml" exec -T db \
        psql -U "${DB_USER:-codeatlas}" -d "${DB_NAME:-codeatlas}" -f "/docker-entrypoint-initdb.d/$(basename $migration_file)"
}

# Run migration directly
run_migration_direct() {
    local migration_file=$1
    log_info "Running migration: $(basename $migration_file)"
    
    PGPASSWORD="${DB_PASSWORD}" psql \
        -h "${DB_HOST:-localhost}" \
        -p "${DB_PORT:-5432}" \
        -U "${DB_USER:-codeatlas}" \
        -d "${DB_NAME:-codeatlas}" \
        -f "$migration_file"
}

# Check if migration was already applied
check_migration_applied() {
    local version=$1
    local result
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        result=$(docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.prod.yml" exec -T db \
            psql -U "${DB_USER:-codeatlas}" -d "${DB_NAME:-codeatlas}" -t -c \
            "SELECT COUNT(*) FROM schema_migrations WHERE version = '$version';" 2>/dev/null || echo "0")
    else
        result=$(PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST:-localhost}" \
            -p "${DB_PORT:-5432}" \
            -U "${DB_USER:-codeatlas}" \
            -d "${DB_NAME:-codeatlas}" \
            -t -c "SELECT COUNT(*) FROM schema_migrations WHERE version = '$version';" 2>/dev/null || echo "0")
    fi
    
    [ "$(echo $result | tr -d ' ')" -gt 0 ]
}

# Main migration logic
main() {
    log_info "CodeAtlas Database Migration Script"
    log_info "Deployment type: $DEPLOYMENT_TYPE"
    log_info "Migration version: $MIGRATION_VERSION"
    log_info ""
    
    # Load environment variables if .env exists
    if [ -f "$PROJECT_ROOT/deployments/.env" ]; then
        log_info "Loading environment from .env"
        export $(cat "$PROJECT_ROOT/deployments/.env" | grep -v '^#' | xargs)
    fi
    
    # Get list of migrations
    if [ "$MIGRATION_VERSION" = "all" ]; then
        migrations=$(ls -1 "$MIGRATIONS_DIR"/*.sql | sort)
    else
        migrations="$MIGRATIONS_DIR/${MIGRATION_VERSION}.sql"
        if [ ! -f "$migrations" ]; then
            log_error "Migration file not found: $migrations"
            exit 1
        fi
    fi
    
    # Run migrations
    for migration in $migrations; do
        version=$(basename "$migration" .sql)
        
        # Check if already applied
        if check_migration_applied "$version"; then
            log_info "Migration $version already applied, skipping..."
            continue
        fi
        
        # Run migration
        if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
            run_migration_docker "$migration"
        else
            run_migration_direct "$migration"
        fi
        
        if [ $? -eq 0 ]; then
            log_info "Migration $version completed successfully"
        else
            log_error "Migration $version failed"
            exit 1
        fi
    done
    
    log_info ""
    log_info "All migrations completed successfully!"
}

main
