package models

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/config"
)

// TestDB wraps a database connection for testing
type TestDB struct {
	*DB
	dbName string
}

// SetupTestDB creates a test database and returns a connection
// This should be called at the beginning of each test that requires database access
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Disable database logging during tests to reduce noise
	SetDBLogger(nil)

	// Generate unique database name for this test
	dbName := fmt.Sprintf("codeatlas_test_%s", uuid.New().String()[:8])

	// Connect to default postgres database to create test database
	adminCfg := &config.DatabaseConfig{
		Host:     getTestEnv("DB_HOST", "localhost"),
		Port:     getTestEnvInt("DB_PORT", 5432),
		User:     getTestEnv("DB_USER", "codeatlas"),
		Password: getTestEnv("DB_PASSWORD", "codeatlas"),
		Database: "postgres",
		SSLMode:  "disable",
	}

	adminDB, err := NewDBWithConfig(adminCfg)
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
	testCfg := &config.DatabaseConfig{
		Host:     getTestEnv("DB_HOST", "localhost"),
		Port:     getTestEnvInt("DB_PORT", 5432),
		User:     getTestEnv("DB_USER", "codeatlas"),
		Password: getTestEnv("DB_PASSWORD", "codeatlas"),
		Database: dbName,
		SSLMode:  "disable",
	}

	testDB, err := NewDBWithConfig(testCfg)
	if err != nil {
		// Clean up database if connection fails
		adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize schema
	schemaManager := NewSchemaManager(testDB)
	if err := schemaManager.InitializeSchema(ctx); err != nil {
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
	adminCfg := &config.DatabaseConfig{
		Host:     getTestEnv("DB_HOST", "localhost"),
		Port:     getTestEnvInt("DB_PORT", 5432),
		User:     getTestEnv("DB_USER", "codeatlas"),
		Password: getTestEnv("DB_PASSWORD", "codeatlas"),
		Database: "postgres",
		SSLMode:  "disable",
	}

	adminDB, err := NewDBWithConfig(adminCfg)
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

// getTestEnv retrieves environment variable or returns default value
func getTestEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getTestEnvInt retrieves environment variable as int or returns default value
func getTestEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var intValue int
	fmt.Sscanf(value, "%d", &intValue)
	return intValue
}
