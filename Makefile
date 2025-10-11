# Makefile for CodeAtlas

# Variables
BINARY_NAME=codeatlas
API_BINARY=bin/api
CLI_BINARY=bin/cli

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

# Run targets
.PHONY: run-api
run-api: build-api
	./${API_BINARY}

.PHONY: run-cli
run-cli: build-cli
	./${CLI_BINARY}

# Clean targets
.PHONY: clean
clean:
	rm -f ${API_BINARY} ${CLI_BINARY}

# Docker targets
.PHONY: docker-up
docker-up:
	docker-compose up -d

.PHONY: docker-down
docker-down:
	docker-compose down

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
	@echo "  make build          - Build all binaries"
	@echo "  make build-api      - Build API server"
	@echo "  make build-cli      - Build CLI tool"
	@echo "  make test           - Run all tests"
	@echo "  make test-cli       - Run CLI tests"
	@echo "  make test-api       - Run API tests"
	@echo "  make test-models    - Run database model tests"
	@echo "  make run-api        - Build and run API server"
	@echo "  make run-cli        - Build and run CLI tool"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make docker-up      - Start Docker services"
	@echo "  make docker-down    - Stop Docker services"
	@echo "  make install        - Install dependencies"
	@echo "  make help           - Show this help message"