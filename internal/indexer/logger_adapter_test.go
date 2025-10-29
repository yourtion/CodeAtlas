package indexer

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

func TestLoggerAdapter(t *testing.T) {
	var buf bytes.Buffer
	utilsLogger := utils.NewLogger(false)
	utilsLogger.infoLog = log.New(&buf, "INFO: ", 0)
	utilsLogger.warnLog = log.New(&buf, "WARN: ", 0)
	utilsLogger.errLog = log.New(&buf, "ERROR: ", 0)
	utilsLogger.dbgLog = log.New(&buf, "DEBUG: ", 0)

	adapter := NewLoggerAdapter(utilsLogger)

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		adapter.Info("test message")
		if !strings.Contains(buf.String(), "test message") {
			t.Errorf("Info() output = %q, want to contain 'test message'", buf.String())
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		adapter.Warn("warning message")
		if !strings.Contains(buf.String(), "warning message") {
			t.Errorf("Warn() output = %q, want to contain 'warning message'", buf.String())
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		adapter.Error("error message")
		if !strings.Contains(buf.String(), "error message") {
			t.Errorf("Error() output = %q, want to contain 'error message'", buf.String())
		}
	})

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		adapter.Debug("debug message")
		// Debug won't output unless verbose is enabled
		// Just verify it doesn't panic
	})
}

func TestLoggerAdapterWithFields(t *testing.T) {
	var buf bytes.Buffer
	utilsLogger := utils.NewLogger(false)
	utilsLogger.infoLog = log.New(&buf, "INFO: ", 0)
	utilsLogger.warnLog = log.New(&buf, "WARN: ", 0)
	utilsLogger.errLog = log.New(&buf, "ERROR: ", 0)
	utilsLogger.dbgLog = log.New(&buf, "DEBUG: ", 0)

	adapter := NewLoggerAdapter(utilsLogger)

	t.Run("InfoWithFields", func(t *testing.T) {
		buf.Reset()
		adapter.InfoWithFields("test message",
			LogField{Key: "key1", Value: "value1"},
			LogField{Key: "key2", Value: 42},
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
	})

	t.Run("WarnWithFields", func(t *testing.T) {
		buf.Reset()
		adapter.WarnWithFields("warning message",
			LogField{Key: "status", Value: "degraded"},
		)
		output := buf.String()
		if !strings.Contains(output, "warning message") {
			t.Errorf("WarnWithFields() output missing message: %q", output)
		}
		if !strings.Contains(output, "status=degraded") {
			t.Errorf("WarnWithFields() output missing field: %q", output)
		}
	})

	t.Run("ErrorWithFields", func(t *testing.T) {
		buf.Reset()
		err := errors.New("test error")
		adapter.ErrorWithFields("error occurred", err,
			LogField{Key: "entity_id", Value: "123"},
		)
		output := buf.String()
		if !strings.Contains(output, "error occurred") {
			t.Errorf("ErrorWithFields() output missing message: %q", output)
		}
		if !strings.Contains(output, "entity_id=123") {
			t.Errorf("ErrorWithFields() output missing field: %q", output)
		}
		if !strings.Contains(output, "error=\"test error\"") {
			t.Errorf("ErrorWithFields() output missing error field: %q", output)
		}
	})

	t.Run("DebugWithFields", func(t *testing.T) {
		buf.Reset()
		adapter.DebugWithFields("debug message",
			LogField{Key: "key", Value: "value"},
		)
		// Debug won't output unless verbose is enabled
		// Just verify it doesn't panic
	})
}

func TestLoggerAdapterFieldConversion(t *testing.T) {
	var buf bytes.Buffer
	utilsLogger := utils.NewLogger(false)
	utilsLogger.infoLog = log.New(&buf, "INFO: ", 0)

	adapter := NewLoggerAdapter(utilsLogger)

	// Test that LogField is properly converted to utils.Field
	adapter.InfoWithFields("test",
		LogField{Key: "string", Value: "value"},
		LogField{Key: "int", Value: 42},
		LogField{Key: "bool", Value: true},
	)

	output := buf.String()
	if !strings.Contains(output, "string=value") {
		t.Errorf("Field conversion failed for string: %q", output)
	}
	if !strings.Contains(output, "int=42") {
		t.Errorf("Field conversion failed for int: %q", output)
	}
	if !strings.Contains(output, "bool=true") {
		t.Errorf("Field conversion failed for bool: %q", output)
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := &noOpLogger{}

	// Verify all methods can be called without panicking
	t.Run("all methods", func(t *testing.T) {
		logger.Info("test")
		logger.Warn("test")
		logger.Error("test")
		logger.Debug("test")
		logger.InfoWithFields("test", LogField{Key: "key", Value: "value"})
		logger.WarnWithFields("test", LogField{Key: "key", Value: "value"})
		logger.ErrorWithFields("test", errors.New("test"), LogField{Key: "key", Value: "value"})
		logger.DebugWithFields("test", LogField{Key: "key", Value: "value"})
	})
}
