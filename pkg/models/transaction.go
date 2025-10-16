package models

import (
	"context"
	"database/sql"
	"fmt"
)

// TransactionManager provides utilities for managing database transactions
type TransactionManager struct {
	db *DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				fmt.Printf("Failed to rollback transaction after panic: %v\n", rollbackErr)
			}
			panic(p) // Re-panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("failed to rollback transaction: %v (original error: %w)", rollbackErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTransactionRetry executes a function within a database transaction with retry logic
func (tm *TransactionManager) WithTransactionRetry(ctx context.Context, maxRetries int, fn func(*sql.Tx) error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := tm.WithTransaction(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable (e.g., serialization failure, deadlock)
		if !isRetryableError(err) {
			return err
		}

		if attempt < maxRetries {
			// Could add exponential backoff here if needed
			continue
		}
	}

	return fmt.Errorf("transaction failed after %d attempts: %w", maxRetries+1, lastErr)
}

// BeginTx starts a new transaction
func (tm *TransactionManager) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return tm.db.BeginTx(ctx, nil)
}

// BeginTxWithOptions starts a new transaction with specific options
func (tm *TransactionManager) BeginTxWithOptions(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return tm.db.BeginTx(ctx, opts)
}

// CommitTx commits a transaction
func (tm *TransactionManager) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx rolls back a transaction
func (tm *TransactionManager) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}

// BatchOperation represents a batch operation that can be executed within a transaction
type BatchOperation struct {
	Name string
	Fn   func(*sql.Tx) error
}

// ExecuteBatch executes multiple operations within a single transaction
func (tm *TransactionManager) ExecuteBatch(ctx context.Context, operations []BatchOperation) error {
	return tm.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, op := range operations {
			if err := op.Fn(tx); err != nil {
				return fmt.Errorf("batch operation '%s' failed: %w", op.Name, err)
			}
		}
		return nil
	})
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for PostgreSQL-specific retryable errors
	errStr := err.Error()
	
	// Serialization failure
	if contains(errStr, "serialization failure") {
		return true
	}
	
	// Deadlock detected
	if contains(errStr, "deadlock detected") {
		return true
	}
	
	// Connection errors that might be temporary
	if contains(errStr, "connection refused") || contains(errStr, "connection reset") {
		return true
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TransactionStats holds statistics about transaction operations
type TransactionStats struct {
	TotalTransactions int64
	CommittedTx       int64
	RolledBackTx      int64
	FailedTx          int64
}

// GetTransactionStats returns transaction statistics (placeholder implementation)
func (tm *TransactionManager) GetTransactionStats(ctx context.Context) (*TransactionStats, error) {
	// This would typically query PostgreSQL's pg_stat_database or similar
	// For now, return empty stats
	return &TransactionStats{}, nil
}