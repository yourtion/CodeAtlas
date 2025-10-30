package models

import (
	"context"
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

	// Configure connection pool with optimized settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to database at %s:%d (pool: %d max, %d idle, lifetime: %s)",
		cfg.Host, cfg.Port, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)
	
	// Log connection pool statistics
	stats := db.Stats()
	log.Printf("Initial connection pool stats - Open: %d, InUse: %d, Idle: %d",
		stats.OpenConnections, stats.InUse, stats.Idle)
	
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

// GetPoolStats returns current connection pool statistics
func (db *DB) GetPoolStats() sql.DBStats {
	return db.Stats()
}

// LogPoolStats logs current connection pool statistics
func (db *DB) LogPoolStats() {
	stats := db.Stats()
	log.Printf("Connection pool stats - Open: %d, InUse: %d, Idle: %d, WaitCount: %d, WaitDuration: %s, MaxIdleClosed: %d, MaxLifetimeClosed: %d",
		stats.OpenConnections,
		stats.InUse,
		stats.Idle,
		stats.WaitCount,
		stats.WaitDuration,
		stats.MaxIdleClosed,
		stats.MaxLifetimeClosed,
	)
}

// OptimizeForBulkInserts configures the database connection for bulk insert operations
func (db *DB) OptimizeForBulkInserts(ctx context.Context) error {
	// Increase work_mem for better sort performance during bulk inserts
	_, err := db.ExecContext(ctx, "SET work_mem = '256MB'")
	if err != nil {
		return fmt.Errorf("failed to set work_mem: %w", err)
	}

	// Increase maintenance_work_mem for index creation
	_, err = db.ExecContext(ctx, "SET maintenance_work_mem = '512MB'")
	if err != nil {
		return fmt.Errorf("failed to set maintenance_work_mem: %w", err)
	}

	// Disable synchronous commit for better performance (trade-off: potential data loss on crash)
	// Only use this for bulk operations, not for critical data
	_, err = db.ExecContext(ctx, "SET synchronous_commit = 'off'")
	if err != nil {
		return fmt.Errorf("failed to set synchronous_commit: %w", err)
	}

	log.Println("Database optimized for bulk insert operations")
	return nil
}

// ResetOptimizations resets database optimizations to defaults
func (db *DB) ResetOptimizations(ctx context.Context) error {
	_, err := db.ExecContext(ctx, "RESET work_mem")
	if err != nil {
		return fmt.Errorf("failed to reset work_mem: %w", err)
	}

	_, err = db.ExecContext(ctx, "RESET maintenance_work_mem")
	if err != nil {
		return fmt.Errorf("failed to reset maintenance_work_mem: %w", err)
	}

	_, err = db.ExecContext(ctx, "RESET synchronous_commit")
	if err != nil {
		return fmt.Errorf("failed to reset synchronous_commit: %w", err)
	}

	log.Println("Database optimizations reset to defaults")
	return nil
}

// AnalyzeTables runs ANALYZE on all tables to update query planner statistics
func (db *DB) AnalyzeTables(ctx context.Context) error {
	tables := []string{
		"repositories",
		"files",
		"symbols",
		"ast_nodes",
		"edges",
		"vectors",
		"docstrings",
		"summaries",
	}

	for _, table := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf("ANALYZE %s", table))
		if err != nil {
			log.Printf("Warning: failed to analyze table %s: %v", table, err)
			// Continue with other tables
		}
	}

	log.Println("Table statistics updated with ANALYZE")
	return nil
}

// VacuumTables runs VACUUM on all tables to reclaim storage and update statistics
func (db *DB) VacuumTables(ctx context.Context) error {
	tables := []string{
		"repositories",
		"files",
		"symbols",
		"ast_nodes",
		"edges",
		"vectors",
		"docstrings",
		"summaries",
	}

	for _, table := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf("VACUUM ANALYZE %s", table))
		if err != nil {
			log.Printf("Warning: failed to vacuum table %s: %v", table, err)
			// Continue with other tables
		}
	}

	log.Println("Tables vacuumed and analyzed")
	return nil
}
