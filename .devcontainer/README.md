# CodeAtlas DevContainer

This devcontainer provides a complete development environment for CodeAtlas with all dependencies pre-configured.

## Features

- **Go 1.25** with development tools (gopls, delve, golangci-lint)
- **Node.js 20** with pnpm for frontend development
- **PostgreSQL 17** with pgvector and AGE extensions
- **Pre-seeded test data** for immediate development
- **VS Code extensions** for Go, Svelte, Docker, and PostgreSQL

## Quick Start

### Using VS Code

1. Install the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open this project in VS Code
3. Click "Reopen in Container" when prompted (or use Command Palette: `Dev Containers: Reopen in Container`)
4. Wait for the container to build and initialize (first time takes 3-5 minutes)

### Using GitHub Codespaces

1. Click "Code" → "Codespaces" → "Create codespace on main"
2. Wait for the environment to initialize
3. Start developing!

## What's Included

### Services

- **Development Container**: Full Go and Node.js development environment
- **PostgreSQL Database**: Pre-configured with extensions and sample data
  - Host: `db`
  - Port: `5432`
  - Database: `codeatlas`
  - User/Password: `codeatlas`

### Sample Data

The database is pre-seeded with:
- 3 sample repositories (Go API, Frontend, Microservice)
- Multiple code files with realistic content
- Symbols and dependencies
- Mock vector embeddings

### Ports

- `8080`: API Server
- `3000`: Frontend Dev Server
- `5432`: PostgreSQL Database

## Development Workflow

### Start API Server
```bash
make run-api
```

### Start Frontend
```bash
cd web
pnpm dev
```

### Run Tests
```bash
make test              # All tests
make test-api          # API tests only
make test-cli          # CLI tests only
```

### Database Access

Using psql:
```bash
psql -h db -U codeatlas -d codeatlas
```

Using VS Code PostgreSQL extension:
- Host: `db`
- Port: `5432`
- Database: `codeatlas`
- Username: `codeatlas`
- Password: `codeatlas`

### CLI Tool

Upload a repository:
```bash
make run-cli upload -p /path/to/repo -s http://localhost:8080
```

## Customization

### Add VS Code Extensions

Edit `.devcontainer/devcontainer.json`:
```json
"extensions": [
  "your.extension-id"
]
```

### Modify Database Seed Data

Edit `scripts/seed_data.sql` and rebuild the container.

### Environment Variables

Edit `.devcontainer/docker-compose.yml` to add or modify environment variables.

## Troubleshooting

### Database Connection Issues

Check if database is ready:
```bash
pg_isready -h db -U codeatlas -d codeatlas
```

### Rebuild Container

If you encounter issues, rebuild the container:
- Command Palette → `Dev Containers: Rebuild Container`

### View Logs

```bash
docker-compose -f .devcontainer/docker-compose.yml logs db
```

## Performance Tips

- The container uses named volumes for Go modules and pnpm store to speed up rebuilds
- Database data persists across container restarts
- First build takes longer; subsequent builds are much faster
