#!/bin/bash
set -e

echo "🚀 Initializing CodeAtlas development environment..."

# Wait for database to be ready
echo "⏳ Waiting for database..."
until pg_isready -h db -U codeatlas -d codeatlas > /dev/null 2>&1; do
  sleep 1
done
echo "✅ Database is ready"

# Check if database is already initialized
if psql -h db -U codeatlas -d codeatlas -c "SELECT 1 FROM repositories LIMIT 1;" > /dev/null 2>&1; then
  echo "✅ Database already initialized with data"
else
  echo "⚠️  Database schema exists but no data found"
fi

# Build the project
echo "🔨 Building project..."
cd /workspace
make build

echo "✅ Development environment ready!"
echo ""
echo "📝 Quick start commands:"
echo "  make run-api          # Start API server (port 8080)"
echo "  cd web && pnpm dev    # Start frontend (port 3000)"
echo "  make test             # Run all tests"
echo "  make run-cli          # Run CLI tool"
echo ""
echo "🗄️  Database connection:"
echo "  Host: db"
echo "  Port: 5432"
echo "  Database: codeatlas"
echo "  User: codeatlas"
echo "  Password: codeatlas"
