# CodeAtlas Documentation

Welcome to the CodeAtlas documentation. This directory contains comprehensive guides for using CodeAtlas.

## Table of Contents

### CLI Tools

#### Parse Command
The `parse` command analyzes source code and outputs structured JSON AST representations.

- **[Complete Documentation](./cli/cli-parse-command.md)** - Full reference guide with all features and options
- **[Quick Reference](./cli/parse-command-quick-reference.md)** - Common commands and quick examples
- **[Troubleshooting Guide](./cli/parse-troubleshooting.md)** - Solutions to common problems
- **[Environment Variables](./cli/parse-environment-variables.md)** - Configuration via environment variables
- **[Example Output](./examples/parse-output-example.json)** - Sample JSON output

#### Index Command
The `index` command indexes parsed code into the knowledge graph for semantic search and relationship queries.

- **[Complete Documentation](./indexer/cli-index-command.md)** - Full CLI reference
- **[Quick Start Guide](./indexer/quick-start.md)** - Get started in minutes
- **[Troubleshooting Guide](./indexer/troubleshooting.md)** - Common issues and solutions

### Knowledge Graph Indexer

The indexer transforms parsed code into a queryable knowledge base with semantic search and graph traversal.

- **[Overview](./indexer/README.md)** - Introduction and features
- **[Quick Start](./indexer/quick-start.md)** - Get up and running quickly
- **[Architecture](./indexer/architecture.md)** - System design and components
- **[API Reference](./indexer/api-reference.md)** - Complete API documentation
- **[Configuration](./indexer/configuration.md)** - All configuration options
- **[Troubleshooting](./indexer/troubleshooting.md)** - Common issues and solutions

### API Server

The API server provides HTTP endpoints for indexing and querying code.

- **[API Documentation](./api/README.md)** - Complete API guide
- **[Quick Start](./api/quick-start.md)** - Get started with the API
- **[API Reference](./api/api-reference.md)** - Endpoint documentation
- **[Search and Relationships](./api/search-and-relationships.md)** - Advanced queries

### Testing

- **[Testing Coverage](./testing/testing-coverage.md)** - Comprehensive testing guide
- **[Coverage Summary](./testing/coverage-summary.md)** - Test coverage statistics
- **[Coverage Quick Reference](./testing/coverage-quick-reference.md)** - Quick testing commands
- **[Test Template](./testing/test-template.md)** - Template for writing tests

### Architecture

- **[Schema Documentation](./schema.md)** - Database schema and data models
- **[Error Handling Implementation](./error-handling-implementation.md)** - Error handling patterns

## Quick Links

### Getting Started

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Basic Usage](#basic-usage)

### Common Tasks

- [Parse a repository](./cli/cli-parse-command.md#basic-usage)
- [Parse a single file](./cli/cli-parse-command.md#parse-a-single-file)
- [Filter by language](./cli/cli-parse-command.md#language-specific-parsing)
- [Optimize performance](./cli/cli-parse-command.md#performance-tips)
- [Troubleshoot issues](./cli/parse-troubleshooting.md)

### Advanced Topics

- [Custom ignore patterns](./cli/cli-parse-command.md#file-filtering-and-ignore-patterns)
- [Semantic enhancement with LLM](./cli/cli-parse-command.md#semantic-enhancement)
- [Concurrent processing](./cli/cli-parse-command.md#performance-optimization)
- [Output processing with jq](./cli/parse-command-quick-reference.md#output-processing-with-jq)

## Installation

Build the CLI tool:

```bash
make build-cli
```

The binary will be available at `bin/cli`.

## Quick Start

### Parse a Repository

```bash
codeatlas parse --path /path/to/repository --output result.json
```

### Parse a Single File

```bash
codeatlas parse --file src/main.go
```

### Parse with Verbose Output

```bash
codeatlas parse --path /path/to/repository --verbose
```

## Basic Usage

### Parse Command

```bash
# Parse entire repository
codeatlas parse --path ./myproject

# Parse only Go files
codeatlas parse --path ./myproject --language go

# Use 8 concurrent workers
codeatlas parse --path ./myproject --workers 8

# Enable verbose logging
codeatlas parse --path ./myproject --verbose
```

For more examples, see the [Quick Reference Guide](./parse-command-quick-reference.md).

## Documentation Structure

```
docs/
├── README.md                               # This file
├── cli/
    ├── cli-parse-command.md                # Complete parse command documentation
    ├── parse-command-quick-reference.md    # Quick reference guide
    ├── parse-troubleshooting.md            # Troubleshooting guide
    └──parse-environment-variables.md       # Environment variables reference
├── schema.md                               # Database schema
├── error-handling-implementation.md    # Error handling patterns
├── examples/
│   └── parse-output-example.json       # Example JSON output
└── testing/
    ├── testing-coverage.md             # Testing guide
    ├── coverage-summary.md             # Coverage statistics
    ├── coverage-quick-reference.md     # Quick testing commands
    └── test-template.md                # Test template
```

## Contributing

When adding new documentation:

1. Follow the existing structure and style
2. Include practical examples
3. Add troubleshooting sections for common issues
4. Update this index file with links to new documents
5. Cross-reference related documentation

## Support

- **Issues**: [GitHub Issues](https://github.com/your-org/codeatlas/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/codeatlas/discussions)
- **Documentation**: This directory

## License

MIT License - See [LICENSE](../LICENSE) for details.
