# CodeAtlas Quick Start Guide

Get CodeAtlas up and running in 5 minutes.

## Prerequisites

- Docker and Docker Compose installed
- 4GB+ RAM available
- 20GB+ disk space

## Quick Start (Docker)

### 1. Clone and Configure

```bash
# Clone repository
git clone https://github.com/yourusername/codeatlas.git
cd codeatlas

# Navigate to deployments
cd deployments

# Copy environment file
cp .env.example .env

# Edit configuration (optional for testing)
nano .env
```

### 2. Deploy

```bash
# Run deployment script
./scripts/deploy.sh docker

# Or manually
docker-compose -f docker-compose.prod.yml up -d
```

### 3. Verify

```bash
# Check health
./scripts/health-check.sh docker

# Or manually
curl http://localhost:8080/health
```

### 4. Use

```bash
# Index a repository
cd ..
make build-cli
./bin/cli index \
  --path /path/to/your/repo \
  --name "my-project" \
  --api-url http://localhost:8080

# Search code
./bin/cli search \
  --query "authentication function" \
  --api-url http://localhost:8080
```

## Quick Start (Systemd)

### 1. Install

```bash
# Run as root
sudo ./scripts/deploy.sh systemd

# Configure
sudo nano /etc/codeatlas/api.env

# Restart
sudo systemctl restart codeatlas-api
```

### 2. Verify

```bash
# Check status
sudo systemctl status codeatlas-api

# Check health
./scripts/health-check.sh systemd
```

## Common Commands

### Docker

```bash
# View logs
docker-compose -f deployments/docker-compose.prod.yml logs -f api

# Restart
docker-compose -f deployments/docker-compose.prod.yml restart

# Stop
docker-compose -f deployments/docker-compose.prod.yml down

# Backup
./deployments/scripts/backup.sh docker
```

### Systemd

```bash
# View logs
sudo journalctl -u codeatlas-api -f

# Restart
sudo systemctl restart codeatlas-api

# Stop
sudo systemctl stop codeatlas-api

# Backup
./deployments/scripts/backup.sh direct
```

## Troubleshooting

### API won't start

```bash
# Check logs
docker-compose -f deployments/docker-compose.prod.yml logs api

# Check database
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "SELECT 1;"
```

### Database connection error

```bash
# Verify database is running
docker-compose -f deployments/docker-compose.prod.yml ps db

# Check credentials in .env
cat deployments/.env | grep DB_
```

### Extensions not found

```bash
# Install extensions
docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS vector;"

docker-compose -f deployments/docker-compose.prod.yml exec db \
  psql -U codeatlas -d codeatlas -c "CREATE EXTENSION IF NOT EXISTS age;"
```

## Next Steps

- Read full [Deployment Guide](README.md)
- Configure [embedding models](README.md#embedding-configuration)
- Set up [automated backups](README.md#backup-and-restore)
- Enable [monitoring](README.md#monitoring-and-maintenance)
- Review [security checklist](README.md#production-checklist)

## Getting Help

- Documentation: [deployments/README.md](README.md)
- Issues: https://github.com/yourusername/codeatlas/issues
- Health check: `./scripts/health-check.sh`
