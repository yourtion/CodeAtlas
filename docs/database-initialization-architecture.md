# Database Initialization Architecture

## Overview

The database initialization system is designed with separation of concerns:
- **API Server**: Automatically initializes database on startup
- **Docker Init Scripts**: SQL-based initialization for containerized deployments
- **Standalone Tool**: Manual maintenance and administration

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Database Initialization                   │
└─────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
                ▼             ▼             ▼
        ┌──────────┐  ┌──────────┐  ┌──────────┐
        │   API    │  │  Docker  │  │ Standalone│
        │  Server  │  │   Init   │  │   Tool   │
        └──────────┘  └──────────┘  └──────────┘
                │             │             │
                └─────────────┼─────────────┘
                              ▼
                    ┌──────────────────┐
                    │  SchemaManager   │
                    │  (pkg/models)    │
                    └──────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   PostgreSQL     │
                    │  + pgvector      │
                    │  + Apache AGE    │
                    └──────────────────┘
```

## Components

### 1. API Server (cmd/api/main.go)

**Purpose**: Production deployment - automatic initialization on startup

**Behavior**:
- Waits for database to be ready (30 retries, 2s delay)
- Initializes schema (extensions, tables, indexes, graph)
- Runs health check
- Logs database statistics
- Starts API server

**Usage**:
```bash
# Start with Docker Compose
docker-compose up -d

# Or run directly
make run-api
```

**Advantages**:
- ✅ Zero manual intervention required
- ✅ Automatic retry logic for database readiness
- ✅ Health check before accepting requests
- ✅ Idempotent - safe to restart

**When to use**:
- Production deployments
- Development with Docker Compose
- CI/CD pipelines
- Kubernetes deployments

### 2. Docker Init Scripts (docker/initdb/)

**Purpose**: First-time database setup in containerized environments

**Files**:
- `init.sql`: Basic extension creation (legacy)
- `01_init_schema.sql`: Complete schema initialization

**Behavior**:
- Runs automatically on first database container startup
- Executes SQL scripts in alphabetical order
- Only runs if database is empty

**Usage**:
```bash
# Start database (scripts run automatically)
docker-compose up -d db
```

**Advantages**:
- ✅ Native PostgreSQL initialization
- ✅ No application dependencies
- ✅ Fast execution
- ✅ Standard Docker pattern

**When to use**:
- Fresh database deployments
- Development environment setup
- Database-only containers

### 3. Standalone Tool (scripts/init_db.go)

**Purpose**: Manual database administration and maintenance

**Features**:
- Schema initialization
- Health checks
- Database statistics
- Vector index creation
- Configurable retry logic

**Usage**:
```bash
# Build the tool
make build-init-db

# Basic initialization
make init-db

# With statistics
make init-db-stats

# Create vector index
make init-db-with-index

# Custom options
./bin/init-db -max-retries 20 -retry-delay 3 -stats
./bin/init-db -create-vector-index -vector-index-lists 100
```

**Flags**:
- `-max-retries`: Connection retry attempts (default: 10)
- `-retry-delay`: Delay between retries in seconds (default: 2)
- `-create-vector-index`: Create IVFFlat vector index
- `-vector-index-lists`: Number of lists for index (default: 100)
- `-stats`: Show database statistics

**Advantages**:
- ✅ Independent of API server
- ✅ Useful for database migrations
- ✅ Can create vector index after data load
- ✅ Detailed logging and statistics

**When to use**:
- Database maintenance
- Schema migrations
- Vector index creation (after data load)
- Troubleshooting
- Manual administration

### 4. SchemaManager (pkg/models/schema.go)

**Purpose**: Core database initialization logic (shared by all components)

**Methods**:
- `InitializeSchema()`: Main orchestrator
- `ensureExtensions()`: Check/create pgvector and AGE
- `ensureAGEGraph()`: Create code_graph
- `verifyCoreTables()`: Validate schema
- `HealthCheck()`: Comprehensive verification
- `GetDatabaseStats()`: Database metrics
- `CreateVectorIndex()`: IVFFlat index creation
- `WaitForDatabase()`: Connection retry logic

**Usage in code**:
```go
import "github.com/yourtionguo/CodeAtlas/pkg/models"

// Wait for database
db, err := models.WaitForDatabase(10, 2*time.Second)

// Initialize schema
sm := models.NewSchemaManager(db)
err = sm.InitializeSchema(context.Background())

// Health check
err = sm.HealthCheck(context.Background())

// Get statistics
stats, err := sm.GetDatabaseStats(context.Background())
```

## Design Decisions

### Why CLI doesn't initialize database?

**Problem**: CLI tool runs on client machines, not server infrastructure

**Reasons**:
1. **Security**: Client shouldn't have direct database access
2. **Architecture**: CLI communicates with API server via HTTP
3. **Deployment**: Database credentials shouldn't be on client machines
4. **Separation**: CLI is for repository operations, not database admin

**Solution**: 
- API server initializes database on startup
- Standalone tool for manual administration (runs on server)

### Why three initialization methods?

**Different use cases require different approaches**:

1. **API Server**: Production deployments need automatic initialization
2. **Docker Scripts**: Container-first deployments need native PostgreSQL init
3. **Standalone Tool**: Administrators need manual control for maintenance

### Idempotency

All initialization methods are idempotent:
- Extensions: `CREATE EXTENSION IF NOT EXISTS`
- Tables: `CREATE TABLE IF NOT EXISTS`
- Indexes: `CREATE INDEX IF NOT EXISTS`
- Graph: Checks existence before creation

Safe to run multiple times without errors.

## Deployment Scenarios

### Scenario 1: Docker Compose Development

```bash
# Start all services
docker-compose up -d

# Database initializes via SQL scripts
# API server verifies and completes initialization
# Ready to use
```

**Initialization flow**:
1. Docker starts `db` service
2. SQL scripts in `docker/initdb/` run automatically
3. API server starts, waits for database
4. API server runs `InitializeSchema()` (idempotent)
5. API server runs health check
6. System ready

### Scenario 2: Kubernetes Production

```yaml
# API deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codeatlas-api
spec:
  template:
    spec:
      containers:
      - name: api
        image: codeatlas/api:latest
        env:
        - name: DB_HOST
          value: postgres-service
```

**Initialization flow**:
1. PostgreSQL deployed separately (managed service or StatefulSet)
2. API pods start
3. Each pod runs `InitializeSchema()` on startup (idempotent)
4. Health check passes
5. Ready to serve traffic

### Scenario 3: Manual Database Setup

```bash
# 1. Start database
docker-compose up -d db

# 2. Run initialization tool
make init-db-stats

# 3. Load data
# ... (data loading process)

# 4. Create vector index
make init-db-with-index

# 5. Start API server
make run-api
```

### Scenario 4: Database Migration

```bash
# 1. Backup database
pg_dump codeatlas > backup.sql

# 2. Run initialization tool to verify/update schema
./bin/init-db -stats

# 3. Check for issues
# Review logs and statistics

# 4. Create vector index if needed
./bin/init-db -create-vector-index
```

## Environment Variables

All components use the same environment variables:

```bash
export DB_HOST=localhost      # Database host
export DB_PORT=5432           # Database port
export DB_USER=codeatlas      # Database user
export DB_PASSWORD=codeatlas  # Database password
export DB_NAME=codeatlas      # Database name
```

## Error Handling

### Connection Failures

All components implement retry logic:
- API Server: 30 retries, 2s delay
- Standalone Tool: Configurable (default: 10 retries, 2s delay)

### Extension Missing

If pgvector or AGE extensions are not available:
1. Attempt to create with `CREATE EXTENSION`
2. If fails, provide clear error message
3. Log installation instructions

### Schema Conflicts

If schema already exists:
- All operations are idempotent
- Existing objects are preserved
- No errors thrown

## Testing

### Integration Tests

```bash
# Run all schema tests
go test -v ./pkg/models -run TestSchemaManager

# Specific tests
go test -v ./pkg/models -run TestSchemaManager_InitializeSchema
go test -v ./pkg/models -run TestSchemaManager_HealthCheck
```

### Manual Testing

```bash
# 1. Start fresh database
docker-compose down -v
docker-compose up -d db

# 2. Test standalone tool
make init-db-stats

# 3. Test API server
make run-api

# 4. Verify schema
psql -h localhost -U codeatlas -d codeatlas -c "\dt"
```

## Monitoring

### Health Check Endpoint

API server provides health check:

```bash
curl http://localhost:8080/health
```

Returns:
- Database connection status
- Extension availability
- Schema validation

### Logs

All components log initialization progress:
- Connection attempts
- Extension creation
- Table verification
- Health check results
- Database statistics

## Best Practices

1. **Always use API server for production**: Automatic initialization and health checks
2. **Use Docker scripts for fresh deployments**: Fast and native PostgreSQL
3. **Use standalone tool for maintenance**: Manual control and detailed output
4. **Create vector index after data load**: Better performance with populated tables
5. **Monitor initialization logs**: Catch issues early
6. **Set appropriate retry values**: Balance between startup time and reliability

## Troubleshooting

### API Server Won't Start

```bash
# Check database connectivity
docker-compose logs db

# Test with standalone tool
make init-db-stats

# Check environment variables
env | grep DB_
```

### Extensions Not Available

```bash
# Check PostgreSQL version
docker-compose exec db psql -U codeatlas -c "SELECT version();"

# Check available extensions
docker-compose exec db psql -U codeatlas -c "SELECT * FROM pg_available_extensions WHERE name IN ('vector', 'age');"

# Rebuild database image
docker-compose build db
docker-compose up -d db
```

### Schema Initialization Fails

```bash
# Check database logs
docker-compose logs db

# Run standalone tool with verbose output
./bin/init-db -stats

# Manually check schema
psql -h localhost -U codeatlas -d codeatlas -c "\dt"
```

## Future Enhancements

1. **Migration System**: Track schema versions and apply migrations
2. **Rollback Support**: Ability to rollback schema changes
3. **Schema Validation**: Verify schema matches expected state
4. **Performance Metrics**: Track initialization time and performance
5. **Backup Integration**: Automatic backup before schema changes

## References

- [Database Schema Documentation](./database-schema-initialization.md)
- [API Server Documentation](../README.md)
- [Docker Deployment Guide](../deployments/README.md)
