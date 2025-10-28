package models

import (
	"context"
	"fmt"
	"log"
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
func (sm *SchemaManager) InitializeSchema(ctx context.Context) error {
	log.Println("Initializing database schema...")

	// Check and create extensions
	if err := sm.ensureExtensions(ctx); err != nil {
		return fmt.Errorf("failed to ensure extensions: %w", err)
	}

	// Verify AGE graph exists
	if err := sm.ensureAGEGraph(ctx); err != nil {
		return fmt.Errorf("failed to ensure AGE graph: %w", err)
	}

	// Verify core tables exist
	if err := sm.verifyCoreTables(ctx); err != nil {
		return fmt.Errorf("failed to verify core tables: %w", err)
	}

	log.Println("Database schema initialized successfully")
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
			log.Printf("Creating extension: %s", ext)
			createQuery := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", ext)
			if _, err := sm.db.ExecContext(ctx, createQuery); err != nil {
				return fmt.Errorf("failed to create extension %s: %w (ensure you have superuser privileges or the extension is pre-installed)", ext, err)
			}
			log.Printf("Extension %s created successfully", ext)
		} else {
			log.Printf("Extension %s already exists", ext)
		}
	}

	// Load AGE into search path
	if _, err := sm.db.ExecContext(ctx, "LOAD 'age'"); err != nil {
		log.Printf("Warning: failed to load AGE (may already be loaded): %v", err)
	}

	if _, err := sm.db.ExecContext(ctx, "SET search_path = ag_catalog, \"$user\", public"); err != nil {
		log.Printf("Warning: failed to set search path: %v", err)
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
		log.Println("Creating AGE graph: code_graph")
		if _, err := sm.db.ExecContext(ctx, "SELECT ag_catalog.create_graph('code_graph')"); err != nil {
			return fmt.Errorf("failed to create graph: %w", err)
		}
		log.Println("AGE graph created successfully")
	} else {
		log.Println("AGE graph already exists")
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

	log.Println("All required tables verified")
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
	log.Printf("Creating vector similarity index with %d lists...", lists)

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

	log.Println("Vector similarity index created successfully")
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
