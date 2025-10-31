# CodeAtlas Deployment Guide

This directory contains all deployment configurations and scripts for the CodeAtlas Knowledge Graph Indexer.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
  - [Docker Deployment](#docker-deployment)
  - [Systemd Deployment](#systemd-deployment)
- [Database Migrations](#database-migrations)
- [Configuration](#configuration)
- [Monitoring and Maintenance](#monitoring-and-maintenance)
- [Troubleshooting](#troubleshooting)

## Overview

CodeAtlas can be deployed in two ways:

1. **Docker Deployment**: Containerized deployment using Docker Compose (recommended for development and small-scale production)
2. **Systemd Deployment**: Native binary deployment with systemd service management (recommended for large-scale production)

## Prerequisites

### Common Requirements

- PostgreSQL 17+ with extensions:
  - pgvector (for semantic search)
  - Apache AGE (for graph queries)
- Go 1.21+ (for building from source)
- Network access to embedding API (OpenAI or local server)

### Docker Deployment Requirements

- Docker 20.10+
- Docker Compose 2.0+
- 4GB+ RAM
- 20GB+ disk space

### Systemd Deployment Requirements

- Linux system with systemd
- Root access for installation
- PostgreSQL installed and running
- 8GB+ RAM (recommended)
- 50GB+ disk space

## Deployment Options

### Docker Deployment

Docker deployment is the easiest way to get started with CodeAtlas. It includes PostgreSQL with all required extensions pre-configured.

#### Quick Start

```bash
# 1. Navigate to deployments directory
cd deployments

# 2. Copy environment configuration
cp .env.example .env

# 3. Edit .env with your configuration
nano .env

# 4. Run deployment script
./scripts/deploy.sh docker
```

#### Manual Docker Deployment

```bash
# 1. Build images
docker-compose -f deployments/docker-compose.prod.yml build

# 2. Start services
docker-compose -f deployments/docker-compose.prod.yml up -d

# 3. Check status
docker-compose -f deployments/docker-compose.prod.yml ps

# 4. View logs
docker-compose -f deployments/docker-compose.prod.yml logs -f api
```

#### Docker Management Commands

```bash
# Stop services
docker-compose -f deployments/docker-compose.prod.yml down

# Restart services
docker-compose -f deployments/docker-compose.prod.yml restart

# View API logs
docker-compose -f deployments/docker-compose.prod.yml logs -f api

# View database logs
docker-compose -f deployments/docker-compose.prod.yml logs -f db

# Execute command in API container
docker-compose -f deployments/docker-compose.prod.yml exec api /bin/sh

# Access database
docker-compose -f deployments/docker-compose.prod.yml exec db psql -U codeatlas -d codeatlas

# Backup database
docker-compose -f deployments/docker-compose.prod.yml exec db pg_dump -U codeatlas codeatlas > backup.sql

# Restore database
docker-compose -f deployments/docker-compose.prod.yml exec -T db psql -U codeatlas codeatlas < backup.sql
```

### Systemd Deployment

Systemd deployment runs CodeAtlas as a native Linux service, providing better performance and resource management for production environments.

#### Installation Steps

```bash
# 1. Run deployment script as root
sudo ./scripts/deploy.sh systemd

# 2. Update environment configuration
sudo nano /etc/codeatlas/api.env

# 3. Restart service
sudo systemctl restart codeatlas-api
```

#### Manual Systemd Deployment

```bash
# 1. Create codeatlas user
sudo useradd -r -s /bin/false -d /opt/codeatlas codeatlas

# 2. Create directories
sudo mkdir -p /opt/codeatlas/bin
sudo mkdir -p /etc/codeatlas
sudo mkdir -p /var/log/codeatlas

# 3. Build binary
cd /path/to/codeatlas
go build -o /opt/codeatlas/bin/codeatlas-api cmd/api/main.go

# 4. Copy configuration
sudo cp deployments/systemd/api.env.example /etc/codeatlas/api.env
sudo nano /etc/codeatlas/api.env

# 5. Install systemd service
sudo cp deployments/systemd/codeatlas-api.service /etc/systemd/system/
sudo systemctl daemon-reload

# 6. Set permissions
sudo chown -R codeatlas:codeatlas /opt/codeatlas
sudo chown -R codeatlas:codeatlas /var/log/codeatlas
sudo chmod 600 /etc/codeatlas/api.env
sudo chmod 755 /opt/codeatlas/bin/codeatlas-api

# 7. Enable and start service
sudo systemctl enable codeatlas-api
sudo systemctl start codeatlas-api
```

#### Systemd Management Commands

```bash
# Check service status
sudo systemctl status codeatlas-api

# Start service
sudo systemctl start codeatlas-api

# Stop service
sudo systemctl stop codeatlas-api

# Restart service
sudo systemctl restart codeatlas-api

# View logs
sudo journalctl -u codeatlas-api -f

# View recent logs
sudo journalctl -u codeatlas-api -n 100

# Enable service on boot
sudo systemctl enable codeatlas-api

# Disable service on boot
sudo systemctl disable codeatlas-api
```

## Database Migrations

Database migrations are managed through SQL scripts in the `migrations/` directory.

### Migration Files

- `01_init_schema.sql`: Initial database schema with all tables and extensions
- `02_performance_indexes.sql`: Performance indexes for vector similarity search

### Running Migrations

#### Docker Environment

```bash
# Run all migrations
./scripts/migrate.sh docker all

# Run specific migration
./scripts/migrate.sh docker 01_init_schema
```

#### Direct Database Connection

```bash
# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=codeatlas
export DB_PASSWORD=your_password
export DB_NAME=codeatlas

# Run all migrations
./scripts/migrate.sh direct all

# Run specific migration
./scripts/migrate.sh direct 01_init_schema
```

#### Manual Migration

```bash
# Docker
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -f /docker-entrypoint-initdb.d/01_init_schema.sql

# Direct
psql -h localhost -U codeatlas -d codeatlas -f deployments/migrations/01_init_schema.sql
```

### Creating New Migrations

1. Create a new SQL file in `deployments/migrations/` with format: `XX_description.sql`
2. Include migration tracking at the end:

```sql
INSERT INTO schema_migrations (version, description)
VALUES ('XX_description', 'Description of migration')
ON CONFLICT (version) DO NOTHING;
```

3. Test migration on development database
4. Run migration script to apply

## Configuration

### Environment Variables

All configuration is done through environment variables. See `.env.example` or `systemd/api.env.example` for complete reference.

#### Database Configuration

```bash
DB_HOST=localhost              # Database host
DB_PORT=5432                   # Database port
DB_USER=codeatlas              # Database user
DB_PASSWORD=secure_password    # Database password (CHANGE IN PRODUCTION!)
DB_NAME=codeatlas              # Database name
DB_SSLMODE=require             # SSL mode (disable, require, verify-full)
DB_MAX_CONNECTIONS=20          # Maximum database connections
```

#### API Server Configuration

```bash
API_HOST=0.0.0.0               # API server host
API_PORT=8080                  # API server port
API_ENABLE_AUTH=true           # Enable authentication
API_CORS_ORIGINS=*             # Allowed CORS origins (comma-separated)
API_REQUEST_TIMEOUT=5m         # Request timeout
```

#### Indexer Configuration

```bash
INDEXER_BATCH_SIZE=100         # Batch size for processing
INDEXER_WORKER_COUNT=4         # Number of worker threads
INDEXER_GRAPH_NAME=code_graph  # AGE graph name
```

#### Embedding Configuration

```bash
# OpenAI API
EMBEDDING_MODEL=openai/text-embedding-3-small
EMBEDDING_API_KEY=sk-your-key-here
EMBEDDING_DIMENSIONS=768
EMBEDDING_BATCH_SIZE=50

# Local embedding server
EMBEDDING_API_URL=http://localhost:1234/v1/embeddings
EMBEDDING_MODEL=text-embedding-qwen3-embedding-0.6b
```

#### Logging Configuration

```bash
LOG_LEVEL=info                 # Log level (debug, info, warn, error)
LOG_FORMAT=json                # Log format (json, text)
```

### Security Considerations

1. **Change default passwords**: Update `DB_PASSWORD` in production
2. **Enable SSL**: Set `DB_SSLMODE=require` for production
3. **Restrict CORS**: Set specific origins instead of `*`
4. **Enable authentication**: Set `API_ENABLE_AUTH=true`
5. **Secure environment files**: Set permissions to 600
6. **Use secrets management**: Consider using HashiCorp Vault or AWS Secrets Manager

## Monitoring and Maintenance

### Health Checks

```bash
# Check API health
curl http://localhost:8080/health

# Check database connection
docker-compose -f deployments/docker-compose.prod.yml exec db \
  pg_isready -U codeatlas -d codeatlas
```

### Performance Monitoring

```bash
# View API metrics (if enabled)
curl http://localhost:8080/metrics

# Database statistics
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT * FROM pg_stat_database WHERE datname = 'codeatlas';"

# Active connections
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT count(*) FROM pg_stat_activity;"
```

### Backup and Restore

#### Database Backup

```bash
# Docker
docker-compose -f deployments/docker-compose.prod.yml exec db \
  pg_dump -U codeatlas codeatlas > backup_$(date +%Y%m%d_%H%M%S).sql

# Direct
pg_dump -h localhost -U codeatlas codeatlas > backup_$(date +%Y%m%d_%H%M%S).sql
```

#### Database Restore

```bash
# Docker
docker-compose -f deployments/docker-compose.prod.yml exec -T db \
  psql -U codeatlas codeatlas < backup.sql

# Direct
psql -h localhost -U codeatlas codeatlas < backup.sql
```

#### Automated Backups

Add to crontab for daily backups:

```bash
# Daily backup at 2 AM
0 2 * * * /path/to/backup_script.sh
```

### Log Rotation

#### Docker Logs

Configure in `docker-compose.prod.yml`:

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "5"
```

#### Systemd Logs

```bash
# Configure journald
sudo nano /etc/systemd/journald.conf

# Set limits
SystemMaxUse=1G
SystemMaxFileSize=100M
```

## Troubleshooting

### Common Issues

#### API Server Won't Start

```bash
# Check logs
docker-compose -f deployments/docker-compose.prod.yml logs api
# or
sudo journalctl -u codeatlas-api -n 100

# Check database connection
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT 1;"

# Verify environment variables
docker-compose -f deployments/docker-compose.prod.yml exec api env | grep DB_
```

#### Database Connection Errors

```bash
# Check PostgreSQL is running
docker-compose -f deployments/docker-compose.prod.yml ps db
# or
sudo systemctl status postgresql

# Check network connectivity
docker-compose -f deployments/docker-compose.prod.yml exec api ping db

# Verify credentials
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT current_user;"
```

#### Extension Not Found

```bash
# Check extensions
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT * FROM pg_extension;"

# Install pgvector
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS vector;"

# Install AGE
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS age;"
```

#### Performance Issues

```bash
# Check database statistics
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT * FROM pg_stat_user_tables;"

# Analyze tables
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "ANALYZE;"

# Check slow queries
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"
```

### Getting Help

- Check logs first: `docker-compose logs` or `journalctl`
- Review configuration: Verify environment variables
- Test connectivity: Ensure database and API are reachable
- Check resources: Monitor CPU, memory, and disk usage
- Consult documentation: Review API and database docs

## Upgrading

### Docker Upgrade

```bash
# 1. Pull latest changes
git pull origin main

# 2. Rebuild images
docker-compose -f deployments/docker-compose.prod.yml build

# 3. Stop services
docker-compose -f deployments/docker-compose.prod.yml down

# 4. Run migrations
./scripts/migrate.sh docker all

# 5. Start services
docker-compose -f deployments/docker-compose.prod.yml up -d
```

### Systemd Upgrade

```bash
# 1. Pull latest changes
git pull origin main

# 2. Build new binary
go build -o /tmp/codeatlas-api cmd/api/main.go

# 3. Stop service
sudo systemctl stop codeatlas-api

# 4. Replace binary
sudo mv /tmp/codeatlas-api /opt/codeatlas/bin/codeatlas-api
sudo chmod 755 /opt/codeatlas/bin/codeatlas-api

# 5. Run migrations
./scripts/migrate.sh direct all

# 6. Start service
sudo systemctl start codeatlas-api
```

## Production Checklist

- [ ] Change default database password
- [ ] Enable SSL for database connections
- [ ] Configure CORS with specific origins
- [ ] Enable API authentication
- [ ] Set up automated backups
- [ ] Configure log rotation
- [ ] Set up monitoring and alerts
- [ ] Review and adjust resource limits
- [ ] Test disaster recovery procedures
- [ ] Document custom configuration
- [ ] Set up firewall rules
- [ ] Configure reverse proxy (nginx/traefik)
- [ ] Enable HTTPS with valid certificates
- [ ] Review security hardening settings

## Support

For issues and questions:
- GitHub Issues: https://github.com/yourusername/codeatlas/issues
- Documentation: https://github.com/yourusername/codeatlas/docs
- Email: support@yourdomain.com
