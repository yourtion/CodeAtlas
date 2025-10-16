package indexer

import (
	"fmt"
)

// IndexerErrorType represents the category of indexer error
type IndexerErrorType string

const (
	ErrorTypeValidation  IndexerErrorType = "validation"
	ErrorTypeDatabase    IndexerErrorType = "database"
	ErrorTypeGraph       IndexerErrorType = "graph"
	ErrorTypeEmbedding   IndexerErrorType = "embedding"
	ErrorTypeTransaction IndexerErrorType = "transaction"
	ErrorTypeNotFound    IndexerErrorType = "not_found"
	ErrorTypeConflict    IndexerErrorType = "conflict"
	ErrorTypeTimeout     IndexerErrorType = "timeout"
	ErrorTypeConnection  IndexerErrorType = "connection"
)

// IndexerError represents an error that occurred during indexing
type IndexerError struct {
	Type      IndexerErrorType `json:"type"`
	Message   string           `json:"message"`
	EntityID  string           `json:"entity_id,omitempty"`
	FilePath  string           `json:"file_path,omitempty"`
	Cause     error            `json:"-"`
	Retryable bool             `json:"retryable"`
}

// Error implements the error interface
func (e *IndexerError) Error() string {
	if e.EntityID != "" && e.FilePath != "" {
		return fmt.Sprintf("%s error for entity %s in file %s: %s", e.Type, e.EntityID, e.FilePath, e.Message)
	} else if e.EntityID != "" {
		return fmt.Sprintf("%s error for entity %s: %s", e.Type, e.EntityID, e.Message)
	} else if e.FilePath != "" {
		return fmt.Sprintf("%s error in file %s: %s", e.Type, e.FilePath, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause error
func (e *IndexerError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(message, entityID, filePath string, cause error) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeValidation,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: false,
	}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(message, entityID, filePath string, cause error, retryable bool) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeDatabase,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: retryable,
	}
}

// NewGraphError creates a new graph error
func NewGraphError(message, entityID, filePath string, cause error) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeGraph,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: true, // Graph operations are generally retryable
	}
}

// NewEmbeddingError creates a new embedding error
func NewEmbeddingError(message, entityID, filePath string, cause error, retryable bool) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeEmbedding,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: retryable,
	}
}

// NewTransactionError creates a new transaction error
func NewTransactionError(message, entityID, filePath string, cause error) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeTransaction,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: false, // Transaction errors are not retryable
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message, entityID, filePath string, cause error) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeNotFound,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: false,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(message, entityID, filePath string, cause error) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeConflict,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: false,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message, entityID, filePath string, cause error) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeTimeout,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: true,
	}
}

// NewConnectionError creates a new connection error
func NewConnectionError(message, entityID, filePath string, cause error) *IndexerError {
	return &IndexerError{
		Type:      ErrorTypeConnection,
		Message:   message,
		EntityID:  entityID,
		FilePath:  filePath,
		Cause:     cause,
		Retryable: true,
	}
}

// ErrorCollector collects multiple errors during batch operations
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Count returns the number of errors
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// First returns the first error, or nil if no errors
func (ec *ErrorCollector) First() error {
	if len(ec.errors) == 0 {
		return nil
	}
	return ec.errors[0]
}

// Error implements the error interface, combining all errors
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return "no errors"
	}
	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}
	return fmt.Sprintf("multiple errors (%d): %s", len(ec.errors), ec.errors[0].Error())
}

// Clear removes all errors from the collector
func (ec *ErrorCollector) Clear() {
	ec.errors = ec.errors[:0]
}

// FilterRetryable returns only retryable errors
func (ec *ErrorCollector) FilterRetryable() []error {
	var retryable []error
	for _, err := range ec.errors {
		if indexerErr, ok := err.(*IndexerError); ok && indexerErr.Retryable {
			retryable = append(retryable, err)
		}
	}
	return retryable
}

// FilterNonRetryable returns only non-retryable errors
func (ec *ErrorCollector) FilterNonRetryable() []error {
	var nonRetryable []error
	for _, err := range ec.errors {
		if indexerErr, ok := err.(*IndexerError); ok && !indexerErr.Retryable {
			nonRetryable = append(nonRetryable, err)
		} else if _, ok := err.(*IndexerError); !ok {
			// Non-IndexerError types are considered non-retryable
			nonRetryable = append(nonRetryable, err)
		}
	}
	return nonRetryable
}

// GroupByType groups errors by their type
func (ec *ErrorCollector) GroupByType() map[IndexerErrorType][]error {
	groups := make(map[IndexerErrorType][]error)
	for _, err := range ec.errors {
		if indexerErr, ok := err.(*IndexerError); ok {
			groups[indexerErr.Type] = append(groups[indexerErr.Type], err)
		} else {
			// Unknown error type
			groups["unknown"] = append(groups["unknown"], err)
		}
	}
	return groups
}

// Summary returns a summary of errors by type
func (ec *ErrorCollector) Summary() map[string]int {
	summary := make(map[string]int)
	for _, err := range ec.errors {
		if indexerErr, ok := err.(*IndexerError); ok {
			summary[string(indexerErr.Type)]++
		} else {
			summary["unknown"]++
		}
	}
	return summary
}