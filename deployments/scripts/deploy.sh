#!/bin/bash
set -e

# CodeAtlas Deployment Script
# This script automates the deployment of CodeAtlas Knowledge Graph Indexer
# Usage: ./deploy.sh [docker|systemd]

DEPLOYMENT_TYPE="${1:-docker}"
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

# Check if running as root for systemd deployment
check_root() {
    if [ "$DEPLOYMENT_TYPE" = "systemd" ] && [ "$EUID" -ne 0 ]; then
        log_error "Systemd deployment requires root privileges"
        exit 1
    fi
}

# Deploy using Docker Compose
deploy_docker() {
    log_info "Starting Docker deployment..."
    
    cd "$PROJECT_ROOT/deployments"
    
    # Check if .env exists
    if [ ! -f .env ]; then
        log_warn ".env file not found, copying from .env.example"
        cp .env.example .env
        log_warn "Please update .env with your production values before continuing"
        read -p "Press enter to continue after updating .env..."
    fi
    
    # Build images
    log_info "Building Docker images..."
    docker-compose -f docker-compose.prod.yml build
    
    # Start services
    log_info "Starting services..."
    docker-compose -f docker-compose.prod.yml up -d
    
    # Wait for database to be ready
    log_info "Waiting for database to be ready..."
    sleep 10
    
    # Check health
    log_info "Checking service health..."
    docker-compose -f docker-compose.prod.yml ps
    
    # Run migrations
    log_info "Running database migrations..."
    docker-compose -f docker-compose.prod.yml exec -T db psql -U codeatlas -d codeatlas -f /docker-entrypoint-initdb.d/01_init_schema.sql || true
    
    log_info "Docker deployment completed successfully!"
    log_info "API server is running at http://localhost:8080"
    log_info "Database is running at localhost:5432"
    log_info ""
    log_info "Useful commands:"
    log_info "  View logs: docker-compose -f deployments/docker-compose.prod.yml logs -f"
    log_info "  Stop services: docker-compose -f deployments/docker-compose.prod.yml down"
    log_info "  Restart services: docker-compose -f deployments/docker-compose.prod.yml restart"
}

# Deploy using systemd
deploy_systemd() {
    log_info "Starting systemd deployment..."
    
    # Create codeatlas user if not exists
    if ! id -u codeatlas > /dev/null 2>&1; then
        log_info "Creating codeatlas user..."
        useradd -r -s /bin/false -d /opt/codeatlas codeatlas
    fi
    
    # Create directories
    log_info "Creating directories..."
    mkdir -p /opt/codeatlas/bin
    mkdir -p /etc/codeatlas
    mkdir -p /var/log/codeatlas
    
    # Build binary
    log_info "Building API server binary..."
    cd "$PROJECT_ROOT"
    go build -o /opt/codeatlas/bin/codeatlas-api cmd/api/main.go
    
    # Copy environment file
    if [ ! -f /etc/codeatlas/api.env ]; then
        log_info "Copying environment configuration..."
        cp "$PROJECT_ROOT/deployments/systemd/api.env.example" /etc/codeatlas/api.env
        log_warn "Please update /etc/codeatlas/api.env with your production values"
    fi
    
    # Copy systemd service file
    log_info "Installing systemd service..."
    cp "$PROJECT_ROOT/deployments/systemd/codeatlas-api.service" /etc/systemd/system/
    
    # Set permissions
    chown -R codeatlas:codeatlas /opt/codeatlas
    chown -R codeatlas:codeatlas /var/log/codeatlas
    chmod 600 /etc/codeatlas/api.env
    chmod 755 /opt/codeatlas/bin/codeatlas-api
    
    # Reload systemd
    log_info "Reloading systemd..."
    systemctl daemon-reload
    
    # Enable and start service
    log_info "Enabling and starting service..."
    systemctl enable codeatlas-api.service
    systemctl start codeatlas-api.service
    
    # Check status
    sleep 2
    systemctl status codeatlas-api.service --no-pager
    
    log_info "Systemd deployment completed successfully!"
    log_info ""
    log_info "Useful commands:"
    log_info "  View logs: journalctl -u codeatlas-api -f"
    log_info "  Stop service: systemctl stop codeatlas-api"
    log_info "  Restart service: systemctl restart codeatlas-api"
    log_info "  Check status: systemctl status codeatlas-api"
}

# Main deployment logic
main() {
    log_info "CodeAtlas Deployment Script"
    log_info "Deployment type: $DEPLOYMENT_TYPE"
    log_info ""
    
    check_root
    
    case "$DEPLOYMENT_TYPE" in
        docker)
            deploy_docker
            ;;
        systemd)
            deploy_systemd
            ;;
        *)
            log_error "Invalid deployment type: $DEPLOYMENT_TYPE"
            log_error "Usage: $0 [docker|systemd]"
            exit 1
            ;;
    esac
}

main
