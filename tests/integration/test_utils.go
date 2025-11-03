package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/config"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestDB wraps a database connection for testing
type TestDB struct {
	*models.DB
	dbName string
}

// SetupTestDB creates a test database and returns a connection
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Generate unique database name for this test
	dbName := fmt.Sprintf("codeatlas_test_%s", uuid.New().String()[:8])

	// Connect to default postgres database to create test database
	cfg := &config.DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "codeatlas"),
		Password:        getEnv("DB_PASSWORD", "codeatlas"),
		Database:        "postgres",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	adminDB, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres database: %v", err)
	}
	defer adminDB.Close()

	// Create test database
	ctx := context.Background()
	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to test database
	cfg.Database = dbName
	testDB, err := models.NewDBWithConfig(cfg)
	if err != nil {
		// Clean up database if connection fails
		adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize schema
	if err := initializeSchema(ctx, testDB); err != nil {
		testDB.Close()
		adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return &TestDB{
		DB:     testDB,
		dbName: dbName,
	}
}

// TeardownTestDB drops the test database and closes the connection
func (tdb *TestDB) TeardownTestDB(t *testing.T) {
	t.Helper()

	dbName := tdb.dbName
	tdb.Close()

	// Connect to default postgres database to drop test database
	cfg := &config.DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "codeatlas"),
		Password: getEnv("DB_PASSWORD", "codeatlas"),
		Database: "postgres",
		SSLMode:  "disable",
	}

	adminDB, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Logf("Warning: Failed to connect to postgres database for cleanup: %v", err)
		return
	}
	defer adminDB.Close()

	// Drop test database
	ctx := context.Background()
	_, err = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		t.Logf("Warning: Failed to drop test database: %v", err)
	}
}

// initializeSchema creates all required tables and extensions
func initializeSchema(ctx context.Context, db *models.DB) error {
	// Create extensions
	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
		"CREATE EXTENSION IF NOT EXISTS vector",
	}

	for _, ext := range extensions {
		if _, err := db.ExecContext(ctx, ext); err != nil {
			return fmt.Errorf("failed to create extension: %w", err)
		}
	}

	// Create tables
	schema := `
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
		-- Using 1024 dimensions to match default embedder (text-embedding-qwen3-embedding-0.6b)
		CREATE TABLE IF NOT EXISTS vectors (
			vector_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_id UUID NOT NULL,
			entity_type VARCHAR(50) NOT NULL,
			embedding vector(1024),
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
	`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CleanupTables truncates all tables for test isolation
func CleanupTables(ctx context.Context, db *models.DB) error {
	tables := []string{
		"summaries",
		"docstrings",
		"vectors",
		"edges",
		"ast_nodes",
		"symbols",
		"files",
		"repositories",
	}

	for _, table := range tables {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// VerifyReferentialIntegrity checks that all foreign key relationships are valid
func VerifyReferentialIntegrity(ctx context.Context, db *models.DB) error {
	checks := []struct {
		name  string
		query string
	}{
		{
			name: "files reference valid repositories",
			query: `
				SELECT COUNT(*) FROM files f
				LEFT JOIN repositories r ON f.repo_id = r.repo_id
				WHERE r.repo_id IS NULL
			`,
		},
		{
			name: "symbols reference valid files",
			query: `
				SELECT COUNT(*) FROM symbols s
				LEFT JOIN files f ON s.file_id = f.file_id
				WHERE f.file_id IS NULL
			`,
		},
		{
			name: "ast_nodes reference valid files",
			query: `
				SELECT COUNT(*) FROM ast_nodes a
				LEFT JOIN files f ON a.file_id = f.file_id
				WHERE f.file_id IS NULL
			`,
		},
		{
			name: "edges reference valid source symbols",
			query: `
				SELECT COUNT(*) FROM edges e
				LEFT JOIN symbols s ON e.source_id = s.symbol_id
				WHERE s.symbol_id IS NULL
			`,
		},
		{
			name: "vectors reference valid entities",
			query: `
				SELECT COUNT(*) FROM vectors v
				WHERE v.entity_type = 'symbol'
				AND NOT EXISTS (SELECT 1 FROM symbols s WHERE s.symbol_id = v.entity_id)
			`,
		},
	}

	for _, check := range checks {
		var count int
		if err := db.QueryRowContext(ctx, check.query).Scan(&count); err != nil {
			return fmt.Errorf("failed to run integrity check '%s': %w", check.name, err)
		}
		if count > 0 {
			return fmt.Errorf("referential integrity violation: %s (found %d orphaned records)", check.name, count)
		}
	}

	return nil
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt retrieves environment variable as int or returns default value
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var intValue int
	fmt.Sscanf(value, "%d", &intValue)
	return intValue
}
