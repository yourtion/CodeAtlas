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

	// Disable database logging during tests to reduce noise
	models.SetDBLogger(nil)

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

// initializeSchema 通过 goose 迁移初始化 schema（与生产/API 路径一致）。
//
// 历史实现在此内联了一份 DDL（第三套 schema），与 goose 迁移脱节，
// 导致迁移新增的列（如 BM25 的 content_tsv）在集成测试里不存在。
// 现统一调用 models.SchemaManager.InitializeSchema，走 pkg/models/migrations
// 下的真源迁移。vector 维度由迁移硬编码为 1024，CI 的 EMBEDDING_DIMENSIONS
// 须与之保持一致。
func initializeSchema(ctx context.Context, db *models.DB) error {
	sm := models.NewSchemaManager(db)
	return sm.InitializeSchema(ctx)
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
