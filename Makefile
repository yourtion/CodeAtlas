# Makefile for CodeAtlas

# Variables
BINARY_NAME=codeatlas
API_BINARY=bin/api
CLI_BINARY=bin/cli
INIT_DB_BINARY=bin/init-db

# Default target
.PHONY: all
all: build

# Build targets
.PHONY: build
build: build-api build-cli

.PHONY: build-api
build-api:
	go build -o ${API_BINARY} cmd/api/main.go

.PHONY: build-cli
build-cli:
	go build -o ${CLI_BINARY} cmd/cli/*.go

.PHONY: build-init-db
build-init-db:
	go build -o ${INIT_DB_BINARY} scripts/init_db.go

# Test targets
.PHONY: test
test: test-unit

# Unit tests only (no external dependencies - fast)
.PHONY: test-unit
test-unit:
	@echo "Running unit tests (no external dependencies)..."
	go test -short $(shell go list ./... | grep -v /scripts | grep -v /test-repo) -v

# Integration tests only (requires database and external services)
.PHONY: test-integration
test-integration:
	@echo "Running integration tests (requires database)..."
	@echo "Make sure database is running: make docker-up"
	go test ./tests/integration/... -v -timeout 5m

# Integration tests with short flag (skips actual database tests)
.PHONY: test-integration-short
test-integration-short:
	@echo "Running integration tests in short mode (skips database)..."
	go test ./tests/integration/... -v -short -timeout 30s

# Legacy integration tests
.PHONY: test-integration-legacy
test-integration-legacy:
	@echo "Running legacy integration tests (requires database)..."
	@echo "Make sure database is running: make docker-up"
	go test ./pkg/models/... ./internal/indexer/... -v -run Integration || \
	go test ./pkg/models/... ./internal/indexer/... -v

# Integration tests with build tags
.PHONY: test-integration-tagged
test-integration-tagged:
	@echo "Running integration tests with build tags..."
	go test -tags=integration $(shell go list ./... | grep -v /scripts | grep -v /test-repo) -v

# CLI integration tests (requires built binary)
.PHONY: test-cli-integration
test-cli-integration: build-cli
	@echo "Running CLI integration tests..."
	go test -tags=parse_tests ./tests/cli/... -v

# All tests (unit + integration)
.PHONY: test-all
test-all:
	@echo "Running all tests (unit + integration)..."
	@echo "Make sure database is running: make docker-up"
	go test $(shell go list ./... | grep -v /scripts | grep -v /test-repo) -v

# Specific test suites
.PHONY: test-cli
test-cli:
	@echo "Running CLI unit tests..."
	go test ./cmd/cli/... ./tests/cli/... -short -v

.PHONY: test-api
test-api:
	@echo "Running API unit tests..."
	go test ./cmd/api/... ./internal/api/... ./tests/api/... -short -v

.PHONY: test-models
test-models:
	@echo "Running model tests (requires database)..."
	go test ./pkg/models/... -v

.PHONY: test-parser
test-parser:
	@echo "Running parser unit tests..."
	go test ./internal/parser/... -short -v

.PHONY: test-indexer
test-indexer:
	@echo "Running indexer tests (requires database)..."
	go test ./internal/indexer/... -v

# Coverage targets
.PHONY: test-coverage
test-coverage: test-coverage-unit

# Unit test coverage only (fast, no external dependencies)
.PHONY: test-coverage-unit
test-coverage-unit:
	@echo "Generating unit test coverage report..."
	go test -short $(shell go list ./... | grep -v /scripts | grep -v /test-repo) -coverprofile=coverage_unit.out -covermode=atomic
	go tool cover -html=coverage_unit.out -o coverage_unit.html
	@echo "Unit test coverage: $$(go tool cover -func=coverage_unit.out | tail -1 | awk '{print $$3}')"
	@echo "Coverage report generated: coverage_unit.html"

# Integration test coverage (requires database)
.PHONY: test-coverage-integration
test-coverage-integration:
	@echo "Generating integration test coverage report..."
	@echo "Make sure database is running: make docker-up"
	go test ./pkg/models/... ./internal/indexer/... -coverprofile=coverage_integration.out -covermode=atomic
	go tool cover -html=coverage_integration.out -o coverage_integration.html
	@echo "Integration test coverage: $$(go tool cover -func=coverage_integration.out | tail -1 | awk '{print $$3}')"
	@echo "Coverage report generated: coverage_integration.html"

# Combined coverage (unit + integration)
.PHONY: test-coverage-all
test-coverage-all:
	@echo "Generating combined test coverage report..."
	@echo "Make sure database is running: make docker-up"
	go test $(shell go list ./... | grep -v /scripts | grep -v /test-repo) -coverprofile=coverage_all.out -covermode=atomic
	go tool cover -html=coverage_all.out -o coverage_all.html
	@echo "Total coverage: $$(go tool cover -func=coverage_all.out | tail -1 | awk '{print $$3}')"
	@echo "Coverage report generated: coverage_all.html"

# Show coverage statistics
.PHONY: test-coverage-func
test-coverage-func:
	@if [ -f coverage_unit.out ]; then \
		echo "=== Unit Test Coverage ==="; \
		go tool cover -func=coverage_unit.out | tail -20; \
	fi
	@if [ -f coverage_integration.out ]; then \
		echo "\n=== Integration Test Coverage ==="; \
		go tool cover -func=coverage_integration.out | tail -20; \
	fi
	@if [ -f coverage_all.out ]; then \
		echo "\n=== Total Coverage ==="; \
		go tool cover -func=coverage_all.out | tail -20; \
	fi

# Generate HTML report from existing coverage data
.PHONY: test-coverage-report
test-coverage-report:
	@if [ -f coverage_unit.out ]; then \
		go tool cover -html=coverage_unit.out -o coverage_unit.html; \
		echo "Unit coverage report: coverage_unit.html"; \
	fi
	@if [ -f coverage_integration.out ]; then \
		go tool cover -html=coverage_integration.out -o coverage_integration.html; \
		echo "Integration coverage report: coverage_integration.html"; \
	fi
	@if [ -f coverage_all.out ]; then \
		go tool cover -html=coverage_all.out -o coverage_all.html; \
		echo "Total coverage report: coverage_all.html"; \
	fi

# Clean coverage files
.PHONY: test-coverage-clean
test-coverage-clean:
	rm -f coverage*.out coverage*.html

# Run targets
.PHONY: run-api
run-api: build-api
	./${API_BINARY}

.PHONY: run-cli
run-cli: build-cli
	./${CLI_BINARY}

# Database initialization
.PHONY: init-db
init-db: build-init-db
	./${INIT_DB_BINARY}

.PHONY: init-db-stats
init-db-stats: build-init-db
	./${INIT_DB_BINARY} -stats

.PHONY: init-db-with-index
init-db-with-index: build-init-db
	./${INIT_DB_BINARY} -create-vector-index -vector-index-lists 100

# Vector dimension management
.PHONY: alter-vector-dimension
alter-vector-dimension:
	@if [ -z "$(VECTOR_DIM)" ]; then \
		echo "Error: VECTOR_DIM not specified"; \
		echo "Usage: make alter-vector-dimension VECTOR_DIM=1536"; \
		echo "Or set EMBEDDING_DIMENSIONS environment variable"; \
		exit 1; \
	fi
	go run scripts/alter_vector_dimension.go -dimension $(VECTOR_DIM)

.PHONY: alter-vector-dimension-force
alter-vector-dimension-force:
	@if [ -z "$(VECTOR_DIM)" ]; then \
		echo "Error: VECTOR_DIM not specified"; \
		echo "Usage: make alter-vector-dimension-force VECTOR_DIM=1536"; \
		exit 1; \
	fi
	go run scripts/alter_vector_dimension.go -dimension $(VECTOR_DIM) -force

.PHONY: alter-vector-dimension-from-env
alter-vector-dimension-from-env:
	@if [ -z "$$EMBEDDING_DIMENSIONS" ]; then \
		echo "Error: EMBEDDING_DIMENSIONS environment variable not set"; \
		echo "Usage: EMBEDDING_DIMENSIONS=1536 make alter-vector-dimension-from-env"; \
		exit 1; \
	fi
	go run scripts/alter_vector_dimension.go

# Clean targets
.PHONY: clean
clean:
	rm -f ${API_BINARY} ${CLI_BINARY} ${INIT_DB_BINARY}
	rm -f coverage.out coverage.html

# Docker targets
.PHONY: docker-up
docker-up:
	docker-compose up -d

.PHONY: docker-down
docker-down:
	docker-compose down

# DevContainer targets
.PHONY: devcontainer-build
devcontainer-build:
	docker-compose -f .devcontainer/docker-compose.yml build

.PHONY: devcontainer-up
devcontainer-up:
	docker-compose -f .devcontainer/docker-compose.yml up -d

.PHONY: devcontainer-down
devcontainer-down:
	docker-compose -f .devcontainer/docker-compose.yml down

.PHONY: devcontainer-logs
devcontainer-logs:
	docker-compose -f .devcontainer/docker-compose.yml logs -f

.PHONY: devcontainer-clean
devcontainer-clean:
	docker-compose -f .devcontainer/docker-compose.yml down -v

# Install dependencies
.PHONY: install
install:
	go mod tidy

# Help target
.PHONY: help
help:
	@echo "Makefile for CodeAtlas"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build                     - Build all binaries"
	@echo "  make build-api                 - Build API server"
	@echo "  make build-cli                 - Build CLI tool"
	@echo "  make build-init-db             - Build database initialization tool"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test                      - Run unit tests (default, fast)"
	@echo "  make test-unit                 - Run unit tests only (no external dependencies)"
	@echo "  make test-integration          - Run integration tests (requires database)"
	@echo "  make test-integration-short    - Run integration tests in short mode (no database)"
	@echo "  make test-integration-legacy   - Run legacy integration tests (requires database)"
	@echo "  make test-integration-tagged   - Run integration tests with build tags"
	@echo "  make test-cli-integration      - Run CLI integration tests (requires built binary)"
	@echo "  make test-all                  - Run all tests (unit + integration)"
	@echo "  make test-cli                  - Run CLI unit tests"
	@echo "  make test-api                  - Run API unit tests"
	@echo "  make test-models               - Run model tests (requires database)"
	@echo "  make test-parser               - Run parser unit tests"
	@echo "  make test-indexer              - Run indexer tests (requires database)"
	@echo ""
	@echo "Coverage Commands:"
	@echo "  make test-coverage             - Generate unit test coverage report (default)"
	@echo "  make test-coverage-unit        - Generate unit test coverage report"
	@echo "  make test-coverage-integration - Generate integration test coverage report"
	@echo "  make test-coverage-all         - Generate combined coverage report (unit + integration)"
	@echo "  make test-coverage-func        - Show function-level coverage statistics"
	@echo "  make test-coverage-report      - Generate HTML reports from existing coverage data"
	@echo "  make test-coverage-clean       - Remove all coverage files"
	@echo ""
	@echo "Run Commands:"
	@echo "  make run-api                   - Build and run API server"
	@echo "  make run-cli                   - Build and run CLI tool"
	@echo ""
	@echo "Database Commands:"
	@echo "  make init-db                   - Initialize database schema"
	@echo "  make init-db-stats             - Initialize database and show statistics"
	@echo "  make init-db-with-index        - Initialize database with vector index"
	@echo "  make alter-vector-dimension VECTOR_DIM=<dim> - Change vector dimension"
	@echo "  make alter-vector-dimension-force VECTOR_DIM=<dim> - Change dimension (truncate data)"
	@echo "  make alter-vector-dimension-from-env - Change dimension from EMBEDDING_DIMENSIONS env"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-up                 - Start Docker services"
	@echo "  make docker-down               - Stop Docker services"
	@echo ""
	@echo "DevContainer Commands:"
	@echo "  make devcontainer-build        - Build devcontainer images"
	@echo "  make devcontainer-up           - Start devcontainer environment"
	@echo "  make devcontainer-down         - Stop devcontainer environment"
	@echo "  make devcontainer-logs         - View devcontainer logs"
	@echo "  make devcontainer-clean        - Stop and remove devcontainer volumes"
	@echo ""
	@echo "Other Commands:"
	@echo "  make clean                     - Clean build artifacts and coverage files"
	@echo "  make install                   - Install dependencies"
	@echo "  make help                      - Show this help message"
	@echo ""
	@echo "Test Workflow:"
	@echo "  1. Fast unit tests:            make test-unit"
	@echo "  2. With database:              make docker-up && make test-all"
	@echo "  3. Coverage report:            make test-coverage-all"