#!/bin/bash
set -e

# CodeAtlas Database Backup Script
# This script creates backups of the CodeAtlas database
# Usage: ./backup.sh [docker|direct] [backup_dir]

DEPLOYMENT_TYPE="${1:-docker}"
BACKUP_DIR="${2:-/var/backups/codeatlas}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

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

# Create backup directory if not exists
mkdir -p "$BACKUP_DIR"

# Load environment variables if .env exists
if [ -f "$PROJECT_ROOT/deployments/.env" ]; then
    log_info "Loading environment from .env"
    export $(cat "$PROJECT_ROOT/deployments/.env" | grep -v '^#' | xargs)
fi

# Backup via Docker
backup_docker() {
    log_info "Creating Docker backup..."
    
    local backup_file="$BACKUP_DIR/codeatlas_backup_${TIMESTAMP}.sql"
    
    docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.prod.yml" exec -T db \
        pg_dump -U "${DB_USER:-codeatlas}" "${DB_NAME:-codeatlas}" > "$backup_file"
    
    if [ $? -eq 0 ]; then
        log_info "Backup created successfully: $backup_file"
        
        # Compress backup
        log_info "Compressing backup..."
        gzip "$backup_file"
        log_info "Compressed backup: ${backup_file}.gz"
        
        # Calculate size
        local size=$(du -h "${backup_file}.gz" | cut -f1)
        log_info "Backup size: $size"
        
        return 0
    else
        log_error "Backup failed"
        return 1
    fi
}

# Backup directly
backup_direct() {
    log_info "Creating direct backup..."
    
    local backup_file="$BACKUP_DIR/codeatlas_backup_${TIMESTAMP}.sql"
    
    PGPASSWORD="${DB_PASSWORD}" pg_dump \
        -h "${DB_HOST:-localhost}" \
        -p "${DB_PORT:-5432}" \
        -U "${DB_USER:-codeatlas}" \
        "${DB_NAME:-codeatlas}" > "$backup_file"
    
    if [ $? -eq 0 ]; then
        log_info "Backup created successfully: $backup_file"
        
        # Compress backup
        log_info "Compressing backup..."
        gzip "$backup_file"
        log_info "Compressed backup: ${backup_file}.gz"
        
        # Calculate size
        local size=$(du -h "${backup_file}.gz" | cut -f1)
        log_info "Backup size: $size"
        
        return 0
    else
        log_error "Backup failed"
        return 1
    fi
}

# Clean old backups (keep last 7 days)
cleanup_old_backups() {
    log_info "Cleaning up old backups (keeping last 7 days)..."
    
    find "$BACKUP_DIR" -name "codeatlas_backup_*.sql.gz" -mtime +7 -delete
    
    local remaining=$(ls -1 "$BACKUP_DIR"/codeatlas_backup_*.sql.gz 2>/dev/null | wc -l)
    log_info "Remaining backups: $remaining"
}

# Main backup logic
main() {
    log_info "CodeAtlas Database Backup Script"
    log_info "Deployment type: $DEPLOYMENT_TYPE"
    log_info "Backup directory: $BACKUP_DIR"
    log_info "Timestamp: $TIMESTAMP"
    log_info ""
    
    case "$DEPLOYMENT_TYPE" in
        docker)
            backup_docker
            ;;
        direct)
            backup_direct
            ;;
        *)
            log_error "Invalid deployment type: $DEPLOYMENT_TYPE"
            log_error "Usage: $0 [docker|direct] [backup_dir]"
            exit 1
            ;;
    esac
    
    if [ $? -eq 0 ]; then
        cleanup_old_backups
        log_info ""
        log_info "Backup completed successfully!"
    else
        log_error "Backup failed!"
        exit 1
    fi
}

main
