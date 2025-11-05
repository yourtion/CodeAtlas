package models

import (
	"context"
	"fmt"
)

// SchemaManager handles database schema initialization and validation
type SchemaManager struct {
	db *DB
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(db *DB) *SchemaManager {
	return &SchemaManager{db: db}
}

// InitializeSchema ensures all required extensions and schema are set up
// It creates tables if they don't exist, making it safe for test environments
func (sm *SchemaManager) InitializeSchema(ctx context.Context) error {
	if dbLogger != nil {
		dbLogger.Debug("Initializing database schema...")
	}

	// Check and create extensions
	if err := sm.ensureExtensions(ctx); err != nil {
		return fmt.Errorf("failed to ensure extensions: %w", err)
	}

	// Verify AGE graph exists
	if err := sm.ensureAGEGraph(ctx); err != nil {
		return fmt.Errorf("failed to ensure AGE graph: %w", err)
	}

	// Try to verify tables exist, if not create them
	if err := sm.verifyCoreTables(ctx); err != nil {
		if dbLogger != nil {
			dbLogger.Debug("Core tables don't exist, creating schema...")
		}
		if err := sm.CreateSchema(ctx); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
		// Verify again after creation
		if err := sm.verifyCoreTables(ctx); err != nil {
			return fmt.Errorf("failed to verify tables after creation: %w", err)
		}
	}

	if dbLogger != nil {
		dbLogger.Debug("Database schema initialized successfully")
	}
	return nil
}

// CreateSchema creates all required database tables
func (sm *SchemaManager) CreateSchema(ctx context.Context) error {
	// Get vector dimension from environment or use default
	vectorDim := getEnvInt("EMBEDDING_DIMENSIONS", 1024)

	schema := fmt.Sprintf(`
		-- Repositories
		CREATE TABLE IF NOT EXISTS repositories (
			repo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			url TEXT,
			branch VARCHAR(255) DEFAULT 'main',
			commit_hash VARCHAR(64),
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		-- Files
		CREATE TABLE IF NOT EXISTS files (
			file_id UUID PRIMARY KEY,
			repo_id UUID NOT NULL REFERENCES repositories(repo_id) ON DELETE CASCADE,
			path TEXT NOT NULL,
			language VARCHAR(50) NOT NULL,
			size BIGINT NOT NULL,
			checksum VARCHAR(64) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(repo_id, path)
		);
		CREATE INDEX IF NOT EXISTS idx_files_repo ON files(repo_id);
		CREATE INDEX IF NOT EXISTS idx_files_checksum ON files(checksum);

		-- Symbols
		CREATE TABLE IF NOT EXISTS symbols (
			symbol_id UUID PRIMARY KEY,
			file_id UUID NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			kind VARCHAR(50) NOT NULL,
			signature TEXT,
			start_line INT NOT NULL,
			end_line INT NOT NULL,
			start_byte INT NOT NULL,
			end_byte INT NOT NULL,
			docstring TEXT,
			semantic_summary TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(file_id, name, start_line, start_byte)
		);
		CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
		CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
		CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind);

		-- AST Nodes
		CREATE TABLE IF NOT EXISTS ast_nodes (
			node_id UUID PRIMARY KEY,
			file_id UUID NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
			type VARCHAR(100) NOT NULL,
			parent_id UUID REFERENCES ast_nodes(node_id) ON DELETE CASCADE,
			start_line INT NOT NULL,
			end_line INT NOT NULL,
			start_byte INT NOT NULL,
			end_byte INT NOT NULL,
			text TEXT,
			attributes JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_ast_nodes_file ON ast_nodes(file_id);
		CREATE INDEX IF NOT EXISTS idx_ast_nodes_parent ON ast_nodes(parent_id);
		CREATE INDEX IF NOT EXISTS idx_ast_nodes_type ON ast_nodes(type);

		-- Dependency Edges
		CREATE TABLE IF NOT EXISTS edges (
			edge_id UUID PRIMARY KEY,
			source_id UUID NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
			target_id UUID REFERENCES symbols(symbol_id) ON DELETE CASCADE,
			edge_type VARCHAR(50) NOT NULL,
			source_file TEXT NOT NULL,
			target_file TEXT,
			target_module TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
		CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
		CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(edge_type);

		-- Vectors (pgvector)
		CREATE TABLE IF NOT EXISTS vectors (
			vector_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_id UUID NOT NULL,
			entity_type VARCHAR(50) NOT NULL,
			embedding vector(%d),
			content TEXT NOT NULL,
			model VARCHAR(100) NOT NULL,
			chunk_index INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_vectors_entity ON vectors(entity_id, entity_type);
		CREATE INDEX IF NOT EXISTS idx_vectors_embedding ON vectors USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

		-- Docstrings
		CREATE TABLE IF NOT EXISTS docstrings (
			doc_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			symbol_id UUID NOT NULL REFERENCES symbols(symbol_id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_docstrings_symbol ON docstrings(symbol_id);

		-- Summaries
		CREATE TABLE IF NOT EXISTS summaries (
			summary_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_id UUID NOT NULL,
			entity_type VARCHAR(50) NOT NULL,
			summary_type VARCHAR(50) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_summaries_entity ON summaries(entity_id, entity_type);
	`, vectorDim)

	if _, err := sm.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	if dbLogger != nil {
		dbLogger.Debug("Database schema created successfully")
	}
	return nil
}

// ensureExtensions checks and creates required PostgreSQL extensions
func (sm *SchemaManager) ensureExtensions(ctx context.Context) error {
	extensions := []string{"vector", "age"}

	for _, ext := range extensions {
		// Check if extension exists
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)`
		err := sm.db.QueryRowContext(ctx, query, ext).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check extension %s: %w", ext, err)
		}

		if !exists {
			if dbLogger != nil {
				dbLogger.Debugf("Creating extension: %s", ext)
			}
			createQuery := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", ext)
			if _, err := sm.db.ExecContext(ctx, createQuery); err != nil {
				return fmt.Errorf("failed to create extension %s: %w (ensure you have superuser privileges or the extension is pre-installed)", ext, err)
			}
			if dbLogger != nil {
				dbLogger.Debugf("Extension %s created successfully", ext)
			}
		} else {
			if dbLogger != nil {
				dbLogger.Debugf("Extension %s already exists", ext)
			}
		}
	}

	// Load AGE into search path
	if _, err := sm.db.ExecContext(ctx, "LOAD 'age'"); err != nil {
		if dbLogger != nil {
			dbLogger.Debugf("Failed to load AGE (may already be loaded): %v", err)
		}
	}

	if _, err := sm.db.ExecContext(ctx, "SET search_path = ag_catalog, \"$user\", public"); err != nil {
		if dbLogger != nil {
			dbLogger.Debugf("Failed to set search path: %v", err)
		}
	}

	return nil
}

// ensureAGEGraph ensures the code_graph exists in AGE
func (sm *SchemaManager) ensureAGEGraph(ctx context.Context) error {
	// Check if graph exists
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM ag_catalog.ag_graph WHERE name = 'code_graph')`
	err := sm.db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}

	if !exists {
		if dbLogger != nil {
			dbLogger.Debug("Creating AGE graph: code_graph")
		}
		if _, err := sm.db.ExecContext(ctx, "SELECT ag_catalog.create_graph('code_graph')"); err != nil {
			return fmt.Errorf("failed to create graph: %w", err)
		}
		if dbLogger != nil {
			dbLogger.Debug("AGE graph created successfully")
		}
	} else {
		if dbLogger != nil {
			dbLogger.Debug("AGE graph already exists")
		}
	}

	return nil
}

// verifyCoreTables checks that all required tables exist
func (sm *SchemaManager) verifyCoreTables(ctx context.Context) error {
	requiredTables := []string{
		"repositories",
		"files",
		"symbols",
		"ast_nodes",
		"edges",
		"vectors",
		"docstrings",
		"summaries",
	}

	for _, table := range requiredTables {
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`
		err := sm.db.QueryRowContext(ctx, query, table).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}

		if !exists {
			return fmt.Errorf("required table %s does not exist - please run database migration scripts", table)
		}
	}

	if dbLogger != nil {
		dbLogger.Debug("All required tables verified")
	}
	return nil
}

// GetSchemaVersion returns the current schema version
func (sm *SchemaManager) GetSchemaVersion(ctx context.Context) (string, error) {
	// For now, we'll use a simple version check based on table existence
	// In production, you'd want a proper migrations table
	var count int
	query := `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'`
	err := sm.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("v1.0.0 (%d tables)", count), nil
}

// HealthCheck performs a comprehensive health check of the database
func (sm *SchemaManager) HealthCheck(ctx context.Context) error {
	// Check database connection
	if err := sm.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Check extensions
	extensions := []string{"vector", "age"}
	for _, ext := range extensions {
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)`
		err := sm.db.QueryRowContext(ctx, query, ext).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("extension %s not available", ext)
		}
	}

	// Check AGE graph
	var graphExists bool
	query := `SELECT EXISTS(SELECT 1 FROM ag_catalog.ag_graph WHERE name = 'code_graph')`
	err := sm.db.QueryRowContext(ctx, query).Scan(&graphExists)
	if err != nil || !graphExists {
		return fmt.Errorf("AGE graph 'code_graph' not available")
	}

	return nil
}

// CreateVectorIndex creates the IVFFlat index for vector similarity search
// This should be called after initial data is loaded for better performance
func (sm *SchemaManager) CreateVectorIndex(ctx context.Context, lists int) error {
	if dbLogger != nil {
		dbLogger.Debugf("Creating vector similarity index with %d lists...", lists)
	}

	// Drop existing index if it exists
	dropQuery := `DROP INDEX IF EXISTS idx_vectors_embedding`
	if _, err := sm.db.ExecContext(ctx, dropQuery); err != nil {
		return fmt.Errorf("failed to drop existing index: %w", err)
	}

	// Create new index
	createQuery := fmt.Sprintf(`
		CREATE INDEX idx_vectors_embedding ON vectors 
		USING ivfflat (embedding vector_cosine_ops) 
		WITH (lists = %d)
	`, lists)

	if _, err := sm.db.ExecContext(ctx, createQuery); err != nil {
		return fmt.Errorf("failed to create vector index: %w", err)
	}

	if dbLogger != nil {
		dbLogger.Debug("Vector similarity index created successfully")
	}
	return nil
}

// GetDatabaseStats returns statistics about the database
func (sm *SchemaManager) GetDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Count repositories
	if err := sm.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM repositories").Scan(&stats.RepositoryCount); err != nil {
		return nil, err
	}

	// Count files
	if err := sm.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files").Scan(&stats.FileCount); err != nil {
		return nil, err
	}

	// Count symbols
	if err := sm.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM symbols").Scan(&stats.SymbolCount); err != nil {
		return nil, err
	}

	// Count edges
	if err := sm.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM edges").Scan(&stats.EdgeCount); err != nil {
		return nil, err
	}

	// Count vectors
	if err := sm.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vectors").Scan(&stats.VectorCount); err != nil {
		return nil, err
	}

	// Get database size
	query := `SELECT pg_size_pretty(pg_database_size(current_database()))`
	if err := sm.db.QueryRowContext(ctx, query).Scan(&stats.DatabaseSize); err != nil {
		return nil, err
	}

	return stats, nil
}

// DatabaseStats holds statistics about the database
type DatabaseStats struct {
	RepositoryCount int64
	FileCount       int64
	SymbolCount     int64
	EdgeCount       int64
	VectorCount     int64
	DatabaseSize    string
}

// getEnvInt retrieves environment variable as int or returns default value
func getEnvInt(key string, defaultValue int) int {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	var intValue int
	fmt.Sscanf(value, "%d", &intValue)
	return intValue
}
