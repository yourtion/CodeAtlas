# Makefile for CodeAtlas

#==============================================================================
# Variables
#==============================================================================
API_BINARY=bin/api
CLI_BINARY=bin/cli
TEST_PACKAGES=$(shell go list ./... | grep -v /scripts | grep -v /test-repo)

#==============================================================================
# Build & Run
#==============================================================================
.PHONY: build build-api build-cli run-api run-cli clean

build: build-api build-cli

build-api:
	@go build -o ${API_BINARY} cmd/api/main.go

build-cli:
	@go build -o ${CLI_BINARY} cmd/cli/*.go

run-api: build-api
	@./${API_BINARY}

run-cli: build-cli
	@./${CLI_BINARY}

clean:
	@rm -f bin/* coverage*.out coverage*.html test_report_*.json
	@echo "✓ Cleaned"

#==============================================================================
# Test
#==============================================================================
.PHONY: test test-integration test-coverage verify clean-test-dbs

# Fast unit tests (no database required)
test:
	@go test -short ${TEST_PACKAGES} -v

# Integration tests (requires database)
test-integration:
	@echo "Make sure database is running: make db"
	@go test ${TEST_PACKAGES} -v

# Coverage report
test-coverage:
	@go test -short ${TEST_PACKAGES} -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage: coverage.html"

# Complete test verification
verify:
	@bash scripts/verify_test_setup.sh

# Clean test databases
clean-test-dbs:
	@bash scripts/cleanup_test_databases.sh

#==============================================================================
# Database
#==============================================================================
.PHONY: db db-init db-stop db-logs

# Start database
db:
	@docker-compose up -d db

# Initialize database schema
db-init:
	@go run scripts/init_db.go -stats

# Stop database
db-stop:
	@docker-compose stop db

# View database logs
db-logs:
	@docker-compose logs -f db

#==============================================================================
# Docker
#==============================================================================
.PHONY: up down logs

# Start all services
up:
	@docker-compose up -d

# Stop all services
down:
	@docker-compose down

# View logs
logs:
	@docker-compose logs -f

#==============================================================================
# Development
#==============================================================================
.PHONY: install fmt lint

install:
	@go mod tidy

fmt:
	@gofmt -w .

lint:
	@golangci-lint run ./...

#==============================================================================
# Help
#==============================================================================
.PHONY: help

help:
	@echo "CodeAtlas Makefile"
	@echo ""
	@echo "BUILD:"
	@echo "  make build       Build all binaries"
	@echo "  make build-api   Build API server"
	@echo "  make build-cli   Build CLI tool"
	@echo "  make run-api     Run API server"
	@echo "  make run-cli     Run CLI tool"
	@echo "  make clean       Clean build artifacts"
	@echo ""
	@echo "TEST:"
	@echo "  make test              Fast unit tests (no database)"
	@echo "  make test-integration  All tests (requires database)"
	@echo "  make test-coverage     Generate coverage report"
	@echo "  make verify            Complete test verification"
	@echo "  make clean-test-dbs    Clean test databases"
	@echo ""
	@echo "DATABASE:"
	@echo "  make db          Start database"
	@echo "  make db-init     Initialize database schema"
	@echo "  make db-stop     Stop database"
	@echo "  make db-logs     View database logs"
	@echo ""
	@echo "DOCKER:"
	@echo "  make up          Start all services"
	@echo "  make down        Stop all services"
	@echo "  make logs        View logs"
	@echo ""
	@echo "DEVELOPMENT:"
	@echo "  make install     Install dependencies"
	@echo "  make fmt         Format code"
	@echo "  make lint        Run linter"
	@echo ""
	@echo "QUICK START:"
	@echo "  make db          # Start database"
	@echo "  make db-init     # Initialize schema"
	@echo "  make test        # Run tests"
	@echo "  make run-api     # Start API"

.DEFAULT_GOAL := help
