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
	go build -o ${CLI_BINARY} cmd/cli/main.go

.PHONY: build-init-db
build-init-db:
	go build -o ${INIT_DB_BINARY} scripts/init_db.go

# Test targets
.PHONY: test
test:
	go test ./...

.PHONY: test-cli
test-cli:
	go test ./tests/cli/... -v

.PHONY: test-api
test-api:
	go test ./tests/api/... -v

.PHONY: test-models
test-models:
	go test ./tests/models/... -v

# Coverage targets
.PHONY: test-coverage
test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-coverage-func
test-coverage-func:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

.PHONY: test-coverage-report
test-coverage-report:
	@if [ ! -f coverage.out ]; then \
		echo "No coverage data found. Run 'make test-coverage' first."; \
		exit 1; \
	fi
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-coverage-clean
test-coverage-clean:
	rm -f coverage.out coverage.html

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
	@echo "Usage:"
	@echo "  make build                - Build all binaries"
	@echo "  make build-api            - Build API server"
	@echo "  make build-cli            - Build CLI tool"
	@echo "  make build-init-db        - Build database initialization tool"
	@echo "  make test                 - Run all tests"
	@echo "  make test-cli             - Run CLI tests"
	@echo "  make test-api             - Run API tests"
	@echo "  make test-models          - Run database model tests"
	@echo "  make test-coverage        - Run tests with coverage and generate HTML report"
	@echo "  make test-coverage-func   - Run tests with coverage and show function-level stats"
	@echo "  make test-coverage-report - Generate HTML report from existing coverage data"
	@echo "  make test-coverage-clean  - Remove coverage files"
	@echo "  make run-api              - Build and run API server"
	@echo "  make run-cli              - Build and run CLI tool"
	@echo "  make init-db              - Initialize database schema"
	@echo "  make init-db-stats        - Initialize database and show statistics"
	@echo "  make init-db-with-index   - Initialize database with vector index"
	@echo "  make clean                - Clean build artifacts and coverage files"
	@echo "  make docker-up            - Start Docker services"
	@echo "  make docker-down          - Stop Docker services"
	@echo "  make devcontainer-build   - Build devcontainer images"
	@echo "  make devcontainer-up      - Start devcontainer environment"
	@echo "  make devcontainer-down    - Stop devcontainer environment"
	@echo "  make devcontainer-logs    - View devcontainer logs"
	@echo "  make devcontainer-clean   - Stop and remove devcontainer volumes"
	@echo "  make install              - Install dependencies"
	@echo "  make help                 - Show this help message"