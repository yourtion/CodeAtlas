package indexer

import (
	"errors"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

func TestLoggerAdapter(t *testing.T) {
	utilsLogger := utils.NewLogger(false)
	adapter := NewLoggerAdapter(utilsLogger)

	t.Run("Info", func(t *testing.T) {
		// Just verify it doesn't panic
		adapter.Info("test message")
	})

	t.Run("Warn", func(t *testing.T) {
		// Just verify it doesn't panic
		adapter.Warn("warning message")
	})

	t.Run("Error", func(t *testing.T) {
		// Just verify it doesn't panic
		adapter.Error("error message")
	})

	t.Run("Debug", func(t *testing.T) {
		// Just verify it doesn't panic
		adapter.Debug("debug message")
	})
}

func TestLoggerAdapterWithFields(t *testing.T) {
	utilsLogger := utils.NewLogger(false)
	adapter := NewLoggerAdapter(utilsLogger)

	t.Run("InfoWithFields", func(t *testing.T) {
		// Just verify it doesn't panic
		adapter.InfoWithFields("test message",
			LogField{Key: "key1", Value: "value1"},
			LogField{Key: "key2", Value: 42},
		)
	})

	t.Run("WarnWithFields", func(t *testing.T) {
		// Just verify it doesn't panic
		adapter.WarnWithFields("warning message",
			LogField{Key: "status", Value: "degraded"},
		)
	})

	t.Run("ErrorWithFields", func(t *testing.T) {
		// Just verify it doesn't panic
		err := errors.New("test error")
		adapter.ErrorWithFields("error occurred", err,
			LogField{Key: "entity_id", Value: "123"},
		)
	})

	t.Run("DebugWithFields", func(t *testing.T) {
		// Just verify it doesn't panic
		adapter.DebugWithFields("debug message",
			LogField{Key: "key", Value: "value"},
		)
	})
}

func TestLoggerAdapterFieldConversion(t *testing.T) {
	utilsLogger := utils.NewLogger(false)
	adapter := NewLoggerAdapter(utilsLogger)

	// Test that LogField is properly converted to utils.Field - just verify no panic
	adapter.InfoWithFields("test",
		LogField{Key: "string", Value: "value"},
		LogField{Key: "int", Value: 42},
		LogField{Key: "bool", Value: true},
	)
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
