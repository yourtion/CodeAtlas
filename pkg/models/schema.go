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

// CreateSchema applies all pending database migrations via goose.
// 迁移 SQL 的唯一真源为 pkg/models/migrations/*.sql，由 go:embed 嵌入二进制。
// 历史实现在此内联了一份 DDL（与 docker/initdb、deployments/migrations 三套并行，
// 且外键 CASCADE/SET NULL、向量索引等存在不一致），现已统一到迁移文件。
func (sm *SchemaManager) CreateSchema(ctx context.Context) error {
	return RunMigrations(ctx, sm.db.DB)
}

// ensureExtensions checks and creates required PostgreSQL extensions
func (sm *SchemaManager) ensureExtensions(ctx context.Context) error {
	extensions := []string{"vector"}

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

// GetSchemaVersion returns the current goose migration version applied to the database.
// 历史实现仅按表数量返回伪版本字符串；现在查询 goose_db_version 表返回真实版本。
func (sm *SchemaManager) GetSchemaVersion(ctx context.Context) (string, error) {
	version, err := MigrationStatus(ctx, sm.db.DB)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d", version), nil
}

// HealthCheck performs a comprehensive health check of the database
func (sm *SchemaManager) HealthCheck(ctx context.Context) error {
	// Check database connection
	if err := sm.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Check extensions
	extensions := []string{"vector"}
	for _, ext := range extensions {
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)`
		err := sm.db.QueryRowContext(ctx, query, ext).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("extension %s not available", ext)
		}
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
