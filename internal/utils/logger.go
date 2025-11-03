package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Logger provides structured logging with different levels
type Logger struct {
	verbose bool
	infoLog *log.Logger
	warnLog *log.Logger
	errLog  *log.Logger
	dbgLog  *log.Logger
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value interface{}
}

// NewLogger creates a new Logger instance
func NewLogger(verbose bool) *Logger {
	return &Logger{
		verbose: verbose,
		infoLog: log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
		warnLog: log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime),
		errLog:  log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime),
		dbgLog:  log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// NewSilentLogger creates a logger that discards all output (useful for tests)
func NewSilentLogger() *Logger {
	discard := log.New(io.Discard, "", 0)
	return &Logger{
		verbose: false,
		infoLog: discard,
		warnLog: discard,
		errLog:  discard,
		dbgLog:  discard,
	}
}

// Info logs an informational message
func (l *Logger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.infoLog.Printf(msg, args...)
	} else {
		l.infoLog.Println(msg)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.warnLog.Printf(msg, args...)
	} else {
		l.warnLog.Println(msg)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.errLog.Printf(msg, args...)
	} else {
		l.errLog.Println(msg)
	}
}

// Debug logs a debug message (only if verbose mode is enabled)
func (l *Logger) Debug(msg string, args ...interface{}) {
	if !l.verbose {
		return
	}
	if len(args) > 0 {
		l.dbgLog.Printf(msg, args...)
	} else {
		l.dbgLog.Println(msg)
	}
}

// Infof logs a formatted informational message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.infoLog.Println(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.warnLog.Println(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.errLog.Println(fmt.Sprintf(format, args...))
}

// Debugf logs a formatted debug message (only if verbose mode is enabled)
func (l *Logger) Debugf(format string, args ...interface{}) {
	if !l.verbose {
		return
	}
	l.dbgLog.Println(fmt.Sprintf(format, args...))
}

// InfoWithFields logs an informational message with structured fields
func (l *Logger) InfoWithFields(msg string, fields ...Field) {
	l.infoLog.Println(l.formatWithFields(msg, fields...))
}

// WarnWithFields logs a warning message with structured fields
func (l *Logger) WarnWithFields(msg string, fields ...Field) {
	l.warnLog.Println(l.formatWithFields(msg, fields...))
}

// ErrorWithFields logs an error message with structured fields
func (l *Logger) ErrorWithFields(msg string, err error, fields ...Field) {
	if err != nil {
		fields = append(fields, Field{Key: "error", Value: err.Error()})
	}
	l.errLog.Println(l.formatWithFields(msg, fields...))
}

// DebugWithFields logs a debug message with structured fields (only if verbose mode is enabled)
func (l *Logger) DebugWithFields(msg string, fields ...Field) {
	if !l.verbose {
		return
	}
	l.dbgLog.Println(l.formatWithFields(msg, fields...))
}

// formatWithFields formats a message with structured fields
func (l *Logger) formatWithFields(msg string, fields ...Field) string {
	if len(fields) == 0 {
		return msg
	}

	var parts []string
	parts = append(parts, msg)

	for _, field := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", field.Key, l.formatValue(field.Value)))
	}

	return strings.Join(parts, " ")
}

// formatValue formats a field value for logging
func (l *Logger) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Quote strings if they contain spaces
		if strings.Contains(v, " ") {
			return fmt.Sprintf("%q", v)
		}
		return v
	case time.Duration:
		return v.String()
	case time.Time:
		return v.Format(time.RFC3339)
	case error:
		return fmt.Sprintf("%q", v.Error())
	default:
		return fmt.Sprintf("%v", v)
	}
}
