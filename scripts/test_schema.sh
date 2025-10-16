#!/bin/bash
# Test script to verify database schema initialization

set -e

echo "Testing database schema initialization..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "Error: docker-compose is not installed"
    exit 1
fi

# Start database
echo "Starting database..."
docker-compose up -d db

# Wait for database to be ready
echo "Waiting for database to be ready..."
max_attempts=30
attempt=0
until docker-compose exec -T db pg_isready -U codeatlas > /dev/null 2>&1; do
    attempt=$((attempt + 1))
    if [ $attempt -ge $max_attempts ]; then
        echo "Error: Database failed to start after $max_attempts attempts"
        docker-compose logs postgres
        exit 1
    fi
    echo "Waiting for database... (attempt $attempt/$max_attempts)"
    sleep 2
done

echo "Database is ready!"

# Verify extensions
echo "Verifying extensions..."
docker-compose exec -T db psql -U codeatlas -d codeatlas -c "SELECT extname FROM pg_extension WHERE extname IN ('vector', 'age');"

# Verify tables
echo "Verifying tables..."
docker-compose exec -T db psql -U codeatlas -d codeatlas -c "\dt"

# Verify AGE graph
echo "Verifying AGE graph..."
docker-compose exec -T db psql -U codeatlas -d codeatlas -c "SELECT * FROM ag_catalog.ag_graph WHERE name = 'code_graph';"

# Get table counts
echo "Getting table counts..."
docker-compose exec -T db psql -U codeatlas -d codeatlas -c "
SELECT 
    'repositories' as table_name, COUNT(*) as count FROM repositories
UNION ALL
SELECT 'files', COUNT(*) FROM files
UNION ALL
SELECT 'symbols', COUNT(*) FROM symbols
UNION ALL
SELECT 'ast_nodes', COUNT(*) FROM ast_nodes
UNION ALL
SELECT 'edges', COUNT(*) FROM edges
UNION ALL
SELECT 'vectors', COUNT(*) FROM vectors
UNION ALL
SELECT 'docstrings', COUNT(*) FROM docstrings
UNION ALL
SELECT 'summaries', COUNT(*) FROM summaries;
"

echo "✓ Schema initialization test completed successfully!"
