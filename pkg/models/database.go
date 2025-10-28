package models

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/yourtionguo/CodeAtlas/internal/config"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// NewDB creates a new database connection using default configuration
func NewDB() (*DB, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return NewDBWithConfig(&cfg.Database)
}

// NewDBWithConfig creates a new database connection with provided configuration
func NewDBWithConfig(cfg *config.DatabaseConfig) (*DB, error) {
	// Create connection string
	connStr := cfg.ConnectionString()

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to database at %s:%d (pool: %d max, %d idle)",
		cfg.Host, cfg.Port, cfg.MaxOpenConns, cfg.MaxIdleConns)
	return &DB{db}, nil
}

// getEnv retrieves environment variable or returns default value
// Deprecated: Use config.LoadConfig() instead
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// WaitForDatabase waits for database to be ready with retries
func WaitForDatabase(maxRetries int, retryDelay time.Duration) (*DB, error) {
	var db *DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = NewDB()
		if err == nil {
			return db, nil
		}

		log.Printf("Database connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}
