package utils

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"
)

func TestLoggerInfoWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.infoLog = log.New(&buf, "INFO: ", 0)

	logger.InfoWithFields("test message",
		Field{Key: "key1", Value: "value1"},
		Field{Key: "key2", Value: 42},
	)

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("InfoWithFields() output missing message: %q", output)
	}
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("InfoWithFields() output missing field key1: %q", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("InfoWithFields() output missing field key2: %q", output)
	}
}

func TestLoggerWarnWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.warnLog = log.New(&buf, "WARN: ", 0)

	logger.WarnWithFields("warning message",
		Field{Key: "status", Value: "degraded"},
		Field{Key: "count", Value: 5},
	)

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Errorf("WarnWithFields() output missing message: %q", output)
	}
	if !strings.Contains(output, "status=degraded") {
		t.Errorf("WarnWithFields() output missing field status: %q", output)
	}
	if !strings.Contains(output, "count=5") {
		t.Errorf("WarnWithFields() output missing field count: %q", output)
	}
}

func TestLoggerErrorWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.errLog = log.New(&buf, "ERROR: ", 0)

	err := errors.New("test error")
	logger.ErrorWithFields("error occurred", err,
		Field{Key: "entity_id", Value: "123"},
		Field{Key: "file_path", Value: "test.go"},
	)

	output := buf.String()
	if !strings.Contains(output, "error occurred") {
		t.Errorf("ErrorWithFields() output missing message: %q", output)
	}
	if !strings.Contains(output, "entity_id=123") {
		t.Errorf("ErrorWithFields() output missing field entity_id: %q", output)
	}
	if !strings.Contains(output, "file_path=test.go") {
		t.Errorf("ErrorWithFields() output missing field file_path: %q", output)
	}
	if !strings.Contains(output, "error=\"test error\"") {
		t.Errorf("ErrorWithFields() output missing error field: %q", output)
	}
}

func TestLoggerDebugWithFields(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		shouldOutput bool
	}{
		{
			name:         "verbose enabled",
			verbose:      true,
			shouldOutput: true,
		},
		{
			name:         "verbose disabled",
			verbose:      false,
			shouldOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.verbose)
			logger.dbgLog = log.New(&buf, "DEBUG: ", 0)

			logger.DebugWithFields("debug message",
				Field{Key: "key", Value: "value"},
			)

			output := buf.String()
			if tt.shouldOutput {
				if output == "" {
					t.Error("DebugWithFields() did not output when verbose mode is enabled")
				}
				if !strings.Contains(output, "debug message") {
					t.Errorf("DebugWithFields() output missing message: %q", output)
				}
				if !strings.Contains(output, "key=value") {
					t.Errorf("DebugWithFields() output missing field: %q", output)
				}
			} else {
				if output != "" {
					t.Error("DebugWithFields() output when verbose mode is disabled")
				}
			}
		})
	}
}

func TestFormatWithFields(t *testing.T) {
	logger := NewLogger(false)

	tests := []struct {
		name     string
		msg      string
		fields   []Field
		contains []string
	}{
		{
			name:     "no fields",
			msg:      "simple message",
			fields:   nil,
			contains: []string{"simple message"},
		},
		{
			name: "string field",
			msg:  "message",
			fields: []Field{
				{Key: "key", Value: "value"},
			},
			contains: []string{"message", "key=value"},
		},
		{
			name: "string with spaces",
			msg:  "message",
			fields: []Field{
				{Key: "key", Value: "value with spaces"},
			},
			contains: []string{"message", "key=\"value with spaces\""},
		},
		{
			name: "integer field",
			msg:  "message",
			fields: []Field{
				{Key: "count", Value: 42},
			},
			contains: []string{"message", "count=42"},
		},
		{
			name: "boolean field",
			msg:  "message",
			fields: []Field{
				{Key: "enabled", Value: true},
			},
			contains: []string{"message", "enabled=true"},
		},
		{
			name: "duration field",
			msg:  "message",
			fields: []Field{
				{Key: "duration", Value: 5 * time.Second},
			},
			contains: []string{"message", "duration=5s"},
		},
		{
			name: "multiple fields",
			msg:  "message",
			fields: []Field{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: 42},
				{Key: "key3", Value: true},
			},
			contains: []string{"message", "key1=value1", "key2=42", "key3=true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := logger.formatWithFields(tt.msg, tt.fields...)
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("formatWithFields() output = %q, want to contain %q", output, expected)
				}
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	logger := NewLogger(false)

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "simple string",
			value:    "test",
			expected: "test",
		},
		{
			name:     "string with spaces",
			value:    "test value",
			expected: "\"test value\"",
		},
		{
			name:     "integer",
			value:    42,
			expected: "42",
		},
		{
			name:     "boolean true",
			value:    true,
			expected: "true",
		},
		{
			name:     "boolean false",
			value:    false,
			expected: "false",
		},
		{
			name:     "duration",
			value:    5 * time.Second,
			expected: "5s",
		},
		{
			name:     "error",
			value:    errors.New("test error"),
			expected: "\"test error\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := logger.formatValue(tt.value)
			if output != tt.expected {
				t.Errorf("formatValue() = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestErrorWithFieldsNilError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.errLog = log.New(&buf, "ERROR: ", 0)

	logger.ErrorWithFields("error occurred", nil,
		Field{Key: "entity_id", Value: "123"},
	)

	output := buf.String()
	if !strings.Contains(output, "error occurred") {
		t.Errorf("ErrorWithFields() output missing message: %q", output)
	}
	if !strings.Contains(output, "entity_id=123") {
		t.Errorf("ErrorWithFields() output missing field entity_id: %q", output)
	}
	// Should not contain error field when err is nil
	if strings.Contains(output, "error=") {
		t.Errorf("ErrorWithFields() output should not contain error field when err is nil: %q", output)
	}
}

func TestLoggerWithFieldsNoFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.infoLog = log.New(&buf, "INFO: ", 0)

	logger.InfoWithFields("message only")

	output := buf.String()
	if !strings.Contains(output, "message only") {
		t.Errorf("InfoWithFields() output missing message: %q", output)
	}
	// Should not have any key=value pairs
	if strings.Contains(output, "=") {
		t.Errorf("InfoWithFields() output should not contain fields: %q", output)
	}
}
