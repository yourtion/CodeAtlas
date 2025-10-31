#!/bin/bash

# CodeAtlas Health Check Script
# This script checks the health of all CodeAtlas components
# Usage: ./health-check.sh [docker|systemd]

DEPLOYMENT_TYPE="${1:-docker}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Status tracking
OVERALL_STATUS=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[!]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    OVERALL_STATUS=1
}

# Load environment variables if .env exists
if [ -f "$PROJECT_ROOT/deployments/.env" ]; then
    export $(cat "$PROJECT_ROOT/deployments/.env" | grep -v '^#' | xargs 2>/dev/null)
fi

# Check API health
check_api_health() {
    log_info "Checking API server health..."
    
    local api_url="${API_URL:-http://localhost:8080}"
    local response=$(curl -s -o /dev/null -w "%{http_code}" "$api_url/health" 2>/dev/null)
    
    if [ "$response" = "200" ]; then
        log_success "API server is healthy (HTTP $response)"
        return 0
    else
        log_error "API server is unhealthy (HTTP $response)"
        return 1
    fi
}

# Check database health
check_database_health() {
    log_info "Checking database health..."
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        local result=$(docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.prod.yml" exec -T db \
            pg_isready -U "${DB_USER:-codeatlas}" -d "${DB_NAME:-codeatlas}" 2>&1)
    else
        local result=$(PGPASSWORD="${DB_PASSWORD}" pg_isready \
            -h "${DB_HOST:-localhost}" \
            -p "${DB_PORT:-5432}" \
            -U "${DB_USER:-codeatlas}" \
            -d "${DB_NAME:-codeatlas}" 2>&1)
    fi
    
    if [ $? -eq 0 ]; then
        log_success "Database is healthy"
        return 0
    else
        log_error "Database is unhealthy: $result"
        return 1
    fi
}

# Check database extensions
check_database_extensions() {
    log_info "Checking database extensions..."
    
    local extensions=("vector" "age")
    local all_ok=true
    
    for ext in "${extensions[@]}"; do
        if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
            local result=$(docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.prod.yml" exec -T db \
                psql -U "${DB_USER:-codeatlas}" -d "${DB_NAME:-codeatlas}" -t -c \
                "SELECT COUNT(*) FROM pg_extension WHERE extname = '$ext';" 2>/dev/null | tr -d ' ')
        else
            local result=$(PGPASSWORD="${DB_PASSWORD}" psql \
                -h "${DB_HOST:-localhost}" \
                -p "${DB_PORT:-5432}" \
                -U "${DB_USER:-codeatlas}" \
                -d "${DB_NAME:-codeatlas}" \
                -t -c "SELECT COUNT(*) FROM pg_extension WHERE extname = '$ext';" 2>/dev/null | tr -d ' ')
        fi
        
        if [ "$result" = "1" ]; then
            log_success "Extension '$ext' is installed"
        else
            log_error "Extension '$ext' is not installed"
            all_ok=false
        fi
    done
    
    if [ "$all_ok" = true ]; then
        return 0
    else
        return 1
    fi
}

# Check database tables
check_database_tables() {
    log_info "Checking database tables..."
    
    local tables=("repositories" "files" "symbols" "ast_nodes" "edges" "vectors" "docstrings" "summaries")
    local all_ok=true
    
    for table in "${tables[@]}"; do
        if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
            local result=$(docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.prod.yml" exec -T db \
                psql -U "${DB_USER:-codeatlas}" -d "${DB_NAME:-codeatlas}" -t -c \
                "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '$table';" 2>/dev/null | tr -d ' ')
        else
            local result=$(PGPASSWORD="${DB_PASSWORD}" psql \
                -h "${DB_HOST:-localhost}" \
                -p "${DB_PORT:-5432}" \
                -U "${DB_USER:-codeatlas}" \
                -d "${DB_NAME:-codeatlas}" \
                -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '$table';" 2>/dev/null | tr -d ' ')
        fi
        
        if [ "$result" = "1" ]; then
            log_success "Table '$table' exists"
        else
            log_error "Table '$table' does not exist"
            all_ok=false
        fi
    done
    
    if [ "$all_ok" = true ]; then
        return 0
    else
        return 1
    fi
}

# Check Docker containers (if Docker deployment)
check_docker_containers() {
    if [ "$DEPLOYMENT_TYPE" != "docker" ]; then
        return 0
    fi
    
    log_info "Checking Docker containers..."
    
    local containers=("codeatlas-api" "codeatlas-db")
    local all_ok=true
    
    for container in "${containers[@]}"; do
        local status=$(docker inspect -f '{{.State.Status}}' "$container" 2>/dev/null)
        
        if [ "$status" = "running" ]; then
            log_success "Container '$container' is running"
        else
            log_error "Container '$container' is not running (status: $status)"
            all_ok=false
        fi
    done
    
    if [ "$all_ok" = true ]; then
        return 0
    else
        return 1
    fi
}

# Check systemd service (if systemd deployment)
check_systemd_service() {
    if [ "$DEPLOYMENT_TYPE" != "systemd" ]; then
        return 0
    fi
    
    log_info "Checking systemd service..."
    
    local status=$(systemctl is-active codeatlas-api 2>/dev/null)
    
    if [ "$status" = "active" ]; then
        log_success "Service 'codeatlas-api' is active"
        return 0
    else
        log_error "Service 'codeatlas-api' is not active (status: $status)"
        return 1
    fi
}

# Check disk space
check_disk_space() {
    log_info "Checking disk space..."
    
    local usage=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
    
    if [ "$usage" -lt 80 ]; then
        log_success "Disk usage is healthy ($usage%)"
        return 0
    elif [ "$usage" -lt 90 ]; then
        log_warn "Disk usage is high ($usage%)"
        return 0
    else
        log_error "Disk usage is critical ($usage%)"
        return 1
    fi
}

# Check database connections
check_database_connections() {
    log_info "Checking database connections..."
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        local count=$(docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.prod.yml" exec -T db \
            psql -U "${DB_USER:-codeatlas}" -d "${DB_NAME:-codeatlas}" -t -c \
            "SELECT count(*) FROM pg_stat_activity WHERE datname = '${DB_NAME:-codeatlas}';" 2>/dev/null | tr -d ' ')
    else
        local count=$(PGPASSWORD="${DB_PASSWORD}" psql \
            -h "${DB_HOST:-localhost}" \
            -p "${DB_PORT:-5432}" \
            -U "${DB_USER:-codeatlas}" \
            -d "${DB_NAME:-codeatlas}" \
            -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname = '${DB_NAME:-codeatlas}';" 2>/dev/null | tr -d ' ')
    fi
    
    if [ -n "$count" ]; then
        log_success "Active database connections: $count"
        return 0
    else
        log_error "Could not retrieve database connection count"
        return 1
    fi
}

# Main health check logic
main() {
    echo ""
    echo "=========================================="
    echo "  CodeAtlas Health Check"
    echo "  Deployment: $DEPLOYMENT_TYPE"
    echo "  Time: $(date)"
    echo "=========================================="
    echo ""
    
    # Run all checks
    check_docker_containers
    check_systemd_service
    check_api_health
    check_database_health
    check_database_extensions
    check_database_tables
    check_database_connections
    check_disk_space
    
    echo ""
    echo "=========================================="
    if [ $OVERALL_STATUS -eq 0 ]; then
        log_success "All health checks passed!"
    else
        log_error "Some health checks failed!"
    fi
    echo "=========================================="
    echo ""
    
    exit $OVERALL_STATUS
}

main
