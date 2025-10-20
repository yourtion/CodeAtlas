package indexer

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexerError(t *testing.T) {
	cause := errors.New("underlying error")

	err := NewValidationError("invalid data", "entity-123", "file.go", cause)

	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "invalid data", err.Message)
	assert.Equal(t, "entity-123", err.EntityID)
	assert.Equal(t, "file.go", err.FilePath)
	assert.Equal(t, cause, err.Cause)
	assert.False(t, err.Retryable)

	expectedMsg := "validation error for entity entity-123 in file file.go: invalid data"
	assert.Equal(t, expectedMsg, err.Error())

	assert.Equal(t, cause, err.Unwrap())
}

func TestIndexerErrorFormats(t *testing.T) {
	tests := []struct {
		name     string
		err      *IndexerError
		expected string
	}{
		{
			name: "with entity and file",
			err: &IndexerError{
				Type:     ErrorTypeDatabase,
				Message:  "connection failed",
				EntityID: "entity-123",
				FilePath: "file.go",
			},
			expected: "database error for entity entity-123 in file file.go: connection failed",
		},
		{
			name: "with entity only",
			err: &IndexerError{
				Type:     ErrorTypeGraph,
				Message:  "node not found",
				EntityID: "entity-123",
			},
			expected: "graph error for entity entity-123: node not found",
		},
		{
			name: "with file only",
			err: &IndexerError{
				Type:     ErrorTypeEmbedding,
				Message:  "API error",
				FilePath: "file.go",
			},
			expected: "embedding error in file file.go: API error",
		},
		{
			name: "minimal",
			err: &IndexerError{
				Type:    ErrorTypeTimeout,
				Message: "operation timed out",
			},
			expected: "timeout error: operation timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	cause := errors.New("cause")

	tests := []struct {
		name      string
		err       *IndexerError
		errorType IndexerErrorType
		retryable bool
	}{
		{
			name:      "validation error",
			err:       NewValidationError("msg", "id", "file", cause),
			errorType: ErrorTypeValidation,
			retryable: false,
		},
		{
			name:      "database error retryable",
			err:       NewDatabaseError("msg", "id", "file", cause, true),
			errorType: ErrorTypeDatabase,
			retryable: true,
		},
		{
			name:      "database error non-retryable",
			err:       NewDatabaseError("msg", "id", "file", cause, false),
			errorType: ErrorTypeDatabase,
			retryable: false,
		},
		{
			name:      "graph error",
			err:       NewGraphError("msg", "id", "file", cause),
			errorType: ErrorTypeGraph,
			retryable: true,
		},
		{
			name:      "embedding error retryable",
			err:       NewEmbeddingError("msg", "id", "file", cause, true),
			errorType: ErrorTypeEmbedding,
			retryable: true,
		},
		{
			name:      "transaction error",
			err:       NewTransactionError("msg", "id", "file", cause),
			errorType: ErrorTypeTransaction,
			retryable: false,
		},
		{
			name:      "not found error",
			err:       NewNotFoundError("msg", "id", "file", cause),
			errorType: ErrorTypeNotFound,
			retryable: false,
		},
		{
			name:      "conflict error",
			err:       NewConflictError("msg", "id", "file", cause),
			errorType: ErrorTypeConflict,
			retryable: false,
		},
		{
			name:      "timeout error",
			err:       NewTimeoutError("msg", "id", "file", cause),
			errorType: ErrorTypeTimeout,
			retryable: true,
		},
		{
			name:      "connection error",
			err:       NewConnectionError("msg", "id", "file", cause),
			errorType: ErrorTypeConnection,
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.errorType, tt.err.Type)
			assert.Equal(t, tt.retryable, tt.err.Retryable)
			assert.Equal(t, cause, tt.err.Cause)
		})
	}
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()

	// Test empty collector
	assert.False(t, collector.HasErrors())
	assert.Equal(t, 0, collector.Count())
	assert.Nil(t, collector.First())
	assert.Equal(t, "no errors", collector.Error())

	// Add errors
	err1 := NewValidationError("validation failed", "id1", "file1", nil)
	err2 := NewDatabaseError("db error", "id2", "file2", nil, true)
	err3 := errors.New("generic error")

	collector.Add(err1)
	collector.Add(err2)
	collector.Add(err3)

	// Test collector with errors
	assert.True(t, collector.HasErrors())
	assert.Equal(t, 3, collector.Count())
	assert.Equal(t, err1, collector.First())
	assert.Contains(t, collector.Error(), "multiple errors (3)")

	errors := collector.Errors()
	assert.Len(t, errors, 3)
	assert.Equal(t, err1, errors[0])
	assert.Equal(t, err2, errors[1])
	assert.Equal(t, err3, errors[2])
}

func TestErrorCollectorFiltering(t *testing.T) {
	collector := NewErrorCollector()

	retryableErr := NewDatabaseError("retryable", "id1", "file1", nil, true)
	nonRetryableErr := NewValidationError("non-retryable", "id2", "file2", nil)
	genericErr := errors.New("generic")

	collector.Add(retryableErr)
	collector.Add(nonRetryableErr)
	collector.Add(genericErr)

	// Test retryable filtering
	retryable := collector.FilterRetryable()
	assert.Len(t, retryable, 1)
	assert.Equal(t, retryableErr, retryable[0])

	// Test non-retryable filtering
	nonRetryable := collector.FilterNonRetryable()
	assert.Len(t, nonRetryable, 2)
	assert.Contains(t, nonRetryable, nonRetryableErr)
	assert.Contains(t, nonRetryable, genericErr)
}

func TestErrorCollectorGrouping(t *testing.T) {
	collector := NewErrorCollector()

	validationErr := NewValidationError("validation", "id1", "file1", nil)
	databaseErr := NewDatabaseError("database", "id2", "file2", nil, true)
	anotherValidationErr := NewValidationError("another validation", "id3", "file3", nil)
	genericErr := errors.New("generic")

	collector.Add(validationErr)
	collector.Add(databaseErr)
	collector.Add(anotherValidationErr)
	collector.Add(genericErr)

	// Test grouping by type
	groups := collector.GroupByType()
	assert.Len(t, groups, 3) // validation, database, unknown

	assert.Len(t, groups[ErrorTypeValidation], 2)
	assert.Contains(t, groups[ErrorTypeValidation], validationErr)
	assert.Contains(t, groups[ErrorTypeValidation], anotherValidationErr)

	assert.Len(t, groups[ErrorTypeDatabase], 1)
	assert.Contains(t, groups[ErrorTypeDatabase], databaseErr)

	assert.Len(t, groups["unknown"], 1)
	assert.Contains(t, groups["unknown"], genericErr)
}

func TestErrorCollectorSummary(t *testing.T) {
	collector := NewErrorCollector()

	collector.Add(NewValidationError("validation1", "id1", "file1", nil))
	collector.Add(NewValidationError("validation2", "id2", "file2", nil))
	collector.Add(NewDatabaseError("database", "id3", "file3", nil, true))
	collector.Add(errors.New("generic"))

	summary := collector.Summary()
	assert.Equal(t, 2, summary["validation"])
	assert.Equal(t, 1, summary["database"])
	assert.Equal(t, 1, summary["unknown"])
}

func TestErrorCollectorClear(t *testing.T) {
	collector := NewErrorCollector()

	collector.Add(NewValidationError("validation", "id1", "file1", nil))
	collector.Add(NewDatabaseError("database", "id2", "file2", nil, true))

	assert.True(t, collector.HasErrors())
	assert.Equal(t, 2, collector.Count())

	collector.Clear()

	assert.False(t, collector.HasErrors())
	assert.Equal(t, 0, collector.Count())
	assert.Nil(t, collector.First())
}

func TestErrorCollectorSingleError(t *testing.T) {
	collector := NewErrorCollector()

	err := NewValidationError("single error", "id1", "file1", nil)
	collector.Add(err)

	assert.Equal(t, err.Error(), collector.Error())
}

func TestErrorCollectorAddNil(t *testing.T) {
	collector := NewErrorCollector()

	collector.Add(nil)

	assert.False(t, collector.HasErrors())
	assert.Equal(t, 0, collector.Count())
}
