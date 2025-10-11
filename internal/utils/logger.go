package utils

import (
	"fmt"
	"log"
	"os"
)

// Logger provides structured logging with different levels
type Logger struct {
	verbose bool
	infoLog *log.Logger
	warnLog *log.Logger
	errLog  *log.Logger
	dbgLog  *log.Logger
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
