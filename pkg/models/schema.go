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

// CreateSchema applies all pending database migrations via goose.
// 迁移 SQL 的唯一真源为 pkg/models/migrations/*.sql，由 go:embed 嵌入二进制。
// 历史实现在此内联了一份 DDL（与 docker/initdb、deployments/migrations 三套并行，
// 且外键 CASCADE/SET NULL、向量索引等存在不一致），现已统一到迁移文件。
func (sm *SchemaManager) CreateSchema(ctx context.Context) error {
	return RunMigrations(ctx, sm.db.DB)
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

// CreateVectorIndex 已废弃：向量相似度索引现在由迁移
// 20260101000002_vector_hnsw.sql 以 HNSW 方式创建（召回率更高、增量友好）。
// 保留此方法仅为向后兼容 scripts/init_db.go 的 -create-vector-index 标志；
// 调用时为空操作并记录提示，不再创建 IVFFlat 索引。
func (sm *SchemaManager) CreateVectorIndex(ctx context.Context, lists int) error {
	if dbLogger != nil {
		dbLogger.Debug("CreateVectorIndex is deprecated; HNSW index is created by migration 20260101000002_vector_hnsw.sql (no-op)")
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
