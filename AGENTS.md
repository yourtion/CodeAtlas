# AGENTS.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CodeAtlas is an intelligent knowledge graph platform for exploring, retrieving, and understanding codebases. It combines RAG (Retrieval-Augmented Generation), code knowledge graphs, and semantic retrieval to help developers, architects, and DevOps engineers quickly understand large codebases.

## Technology Stack

- Backend Service: Go (Gin/Fiber) for high-performance API service
- Parsing Engine: Go + Tree-sitter + LLM for code syntax parsing and AI enhancement
- Vector Storage: PostgreSQL + pgvector for semantic retrieval
- Graph Storage: PostgreSQL AGE for dependency relationships and path queries
- Frontend: Svelte + Rsbuild for a lightweight modern frontend framework
- Containerization: Docker + Docker Compose for consistent local and production environments
- CLI Tool: Go for lightweight cross-platform sync tool

## Architecture Modules

1. CLI Tool: Synchronizes local repository structure and Git information to the server
2. Parsing Engine: Performs syntax parsing, semantic enhancement, and vectorization of code
3. Graph Service: Builds and maintains repository-level knowledge graphs
4. Retrieval & QA: Intelligent RAG engine based on vector retrieval + graph reasoning
5. Web Frontend: Visualizes code navigation, graph queries, and QA interface

## Development Guidelines

### Backend Development (Go)
- Follow Go project structure conventions with cmd, internal, pkg directories
- Use Gin/Fiber for REST API implementation
- Implement proper error handling and logging
- Write unit tests for all business logic
- Use dependency injection for better testability

### Parsing Engine
- Use Tree-sitter for accurate code parsing
- Integrate with LLM APIs for semantic enhancement
- Implement incremental parsing for performance
- Handle multiple programming languages

### Database Design
- Use PostgreSQL with pgvector extension for vector storage
- Use PostgreSQL AGE for graph database functionality
- Design efficient schemas for code entities and relationships
- Implement proper indexing for query performance

### Frontend Development (Svelte)
- Use Svelte for reactive UI components
- Follow modern CSS practices
- Implement responsive design
- Use Rsbuild for fast compilation

### Containerization
- Use Docker for service containerization
- Use Docker Compose for multi-service orchestration
- Optimize container images for size and security
- Implement health checks and monitoring

## Common Development Tasks

### Building the Project
```bash
# Build backend services
go build -o bin/api cmd/api/main.go

# Build CLI tool
go build -o bin/codeatlas cmd/cli/main.go

# Build frontend
pnpm run build
```

### Running Tests
```bash
# Run Go unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...
```

### Running Services Locally
```bash
# Start all services with Docker Compose
docker-compose up

# Run backend API server
go run cmd/api/main.go

# Run CLI tool
go run cmd/cli/main.go
```

### Development Environment
- Install Go 1.21+
- Install Node.js 18+
- Install Docker and Docker Compose
- Set up PostgreSQL with pgvector and AGE extensions
- Configure environment variables in .env file

## Repository Structure (Planned)
```
.
├── cmd/
│   ├── api/          # API server entry point
│   └── cli/          # CLI tool entry point
├── internal/
│   ├── api/          # API service implementation
│   ├── parser/       # Code parsing engine
│   ├── graph/        # Knowledge graph service
│   ├── retrieval/    # Vector retrieval service
│   └── qa/           # QA engine implementation
├── pkg/
│   ├── models/       # Shared data models
│   └── utils/        # Utility functions
├── web/              # Svelte frontend
│   ├── src/
│   └── public/
├── configs/          # Configuration files
├── scripts/          # Development scripts
├── deployments/      # Docker and Kubernetes manifests
├── docs/             # Documentation
├── tests/            # Integration tests
├── go.mod            # Go module definition
├── go.sum            # Go dependencies
├── package.json      # Frontend dependencies
├── docker-compose.yml # Development environment
└── README.md         # Project documentation
```

## Implementation Roadmap

### Phase 1 - Foundation
1. Implement CLI tool for repository upload
2. Build basic API server with Gin/Fiber
3. Set up database schema and connections
4. Implement basic parsing engine with Tree-sitter
5. Create simple frontend with Svelte

### Phase 2 - Core Features
1. Implement vector storage with pgvector
2. Build knowledge graph with AGE
3. Add semantic enhancement with LLM
4. Implement basic RAG functionality
5. Enhance frontend with search UI

### Phase 3 - Advanced Features
1. Add Git history tracking
2. Implement complex graph queries
3. Add multi-repository support
4. Enhance QA engine with agentic pipeline
5. Add enterprise integration features

## Code Quality Standards

**Important!!! Write test after code written !!! Run and pass all test then commit !!!**

**Check all docs and update with code implement !!!**

- Write clean, readable, and maintainable code
- Follow language-specific style guides (Go, JavaScript)
- Implement comprehensive error handling
- Write unit tests for all new functionality
- Document public APIs and complex logic
- Use meaningful variable and function names
- Keep functions small and focused
- Avoid code duplication
- Avoid `any` type in web project
