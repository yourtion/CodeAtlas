package indexer

import (
	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

// LoggerAdapter adapts utils.Logger to IndexerLogger interface
type LoggerAdapter struct {
	logger *utils.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger *utils.Logger) *LoggerAdapter {
	return &LoggerAdapter{
		logger: logger,
	}
}

// Info logs an informational message
func (a *LoggerAdapter) Info(msg string, args ...interface{}) {
	a.logger.Info(msg, args...)
}

// Warn logs a warning message
func (a *LoggerAdapter) Warn(msg string, args ...interface{}) {
	a.logger.Warn(msg, args...)
}

// Error logs an error message
func (a *LoggerAdapter) Error(msg string, args ...interface{}) {
	a.logger.Error(msg, args...)
}

// Debug logs a debug message
func (a *LoggerAdapter) Debug(msg string, args ...interface{}) {
	a.logger.Debug(msg, args...)
}

// InfoWithFields logs an informational message with structured fields
func (a *LoggerAdapter) InfoWithFields(msg string, fields ...LogField) {
	utilsFields := make([]utils.Field, len(fields))
	for i, f := range fields {
		utilsFields[i] = utils.Field{Key: f.Key, Value: f.Value}
	}
	a.logger.InfoWithFields(msg, utilsFields...)
}

// WarnWithFields logs a warning message with structured fields
func (a *LoggerAdapter) WarnWithFields(msg string, fields ...LogField) {
	utilsFields := make([]utils.Field, len(fields))
	for i, f := range fields {
		utilsFields[i] = utils.Field{Key: f.Key, Value: f.Value}
	}
	a.logger.WarnWithFields(msg, utilsFields...)
}

// ErrorWithFields logs an error message with structured fields
func (a *LoggerAdapter) ErrorWithFields(msg string, err error, fields ...LogField) {
	utilsFields := make([]utils.Field, len(fields))
	for i, f := range fields {
		utilsFields[i] = utils.Field{Key: f.Key, Value: f.Value}
	}
	a.logger.ErrorWithFields(msg, err, utilsFields...)
}

// DebugWithFields logs a debug message with structured fields
func (a *LoggerAdapter) DebugWithFields(msg string, fields ...LogField) {
	utilsFields := make([]utils.Field, len(fields))
	for i, f := range fields {
		utilsFields[i] = utils.Field{Key: f.Key, Value: f.Value}
	}
	a.logger.DebugWithFields(msg, utilsFields...)
}
