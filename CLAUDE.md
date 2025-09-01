# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CodeAtlas is an intelligent knowledge graph platform for exploring, retrieving, and understanding codebases. It combines RAG (Retrieval-Augmented Generation), code knowledge graphs, and semantic retrieval to help developers, architects, and DevOps engineers quickly understand large codebases.

Key features include:
- Code/documentation semantic search with natural language queries
- Code knowledge graph based on static analysis and semantic parsing
- Document-code alignment to reduce understanding costs
- Incremental repository updates via CLI or Git API
- Multimodal extensions for integrating issues, PRs, and design documents

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

## Development Status

This is a new project with only documentation files committed so far. The implementation is in early stages according to the roadmap.

## Repository Structure

Currently, the repository contains only:
- README.md: Project documentation with architecture overview, technology choices, and roadmap
- LICENSE: MIT License file

As development progresses, the structure will follow the modular design described in the README.