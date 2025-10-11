package utils

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "verbose mode enabled",
			verbose: true,
		},
		{
			name:    "verbose mode disabled",
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.verbose)
			if logger == nil {
				t.Error("NewLogger() returned nil")
			}
			if logger.verbose != tt.verbose {
				t.Errorf("NewLogger() verbose = %v, want %v", logger.verbose, tt.verbose)
			}
			if logger.infoLog == nil || logger.warnLog == nil || logger.errLog == nil || logger.dbgLog == nil {
				t.Error("NewLogger() did not initialize all log instances")
			}
		})
	}
}

func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.infoLog = log.New(&buf, "INFO: ", 0)

	tests := []struct {
		name     string
		msg      string
		args     []interface{}
		contains string
	}{
		{
			name:     "simple message",
			msg:      "test message",
			args:     nil,
			contains: "test message",
		},
		{
			name:     "formatted message",
			msg:      "test %s %d",
			args:     []interface{}{"message", 42},
			contains: "test message 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Info(tt.msg, tt.args...)
			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Info() output = %q, want to contain %q", output, tt.contains)
			}
		})
	}
}

func TestLoggerWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.warnLog = log.New(&buf, "WARN: ", 0)

	tests := []struct {
		name     string
		msg      string
		args     []interface{}
		contains string
	}{
		{
			name:     "simple warning",
			msg:      "warning message",
			args:     nil,
			contains: "warning message",
		},
		{
			name:     "formatted warning",
			msg:      "warning %s",
			args:     []interface{}{"test"},
			contains: "warning test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Warn(tt.msg, tt.args...)
			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Warn() output = %q, want to contain %q", output, tt.contains)
			}
		})
	}
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.errLog = log.New(&buf, "ERROR: ", 0)

	tests := []struct {
		name     string
		msg      string
		args     []interface{}
		contains string
	}{
		{
			name:     "simple error",
			msg:      "error message",
			args:     nil,
			contains: "error message",
		},
		{
			name:     "formatted error",
			msg:      "error: %v",
			args:     []interface{}{"failed"},
			contains: "error: failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.Error(tt.msg, tt.args...)
			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Error() output = %q, want to contain %q", output, tt.contains)
			}
		})
	}
}

func TestLoggerDebug(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		msg          string
		args         []interface{}
		shouldOutput bool
	}{
		{
			name:         "debug with verbose enabled",
			verbose:      true,
			msg:          "debug message",
			args:         nil,
			shouldOutput: true,
		},
		{
			name:         "debug with verbose disabled",
			verbose:      false,
			msg:          "debug message",
			args:         nil,
			shouldOutput: false,
		},
		{
			name:         "formatted debug with verbose enabled",
			verbose:      true,
			msg:          "debug %s",
			args:         []interface{}{"test"},
			shouldOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.verbose)
			logger.dbgLog = log.New(&buf, "DEBUG: ", 0)

			logger.Debug(tt.msg, tt.args...)
			output := buf.String()

			if tt.shouldOutput && output == "" {
				t.Error("Debug() did not output when verbose mode is enabled")
			}
			if !tt.shouldOutput && output != "" {
				t.Error("Debug() output when verbose mode is disabled")
			}
		})
	}
}

func TestLoggerInfof(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.infoLog = log.New(&buf, "INFO: ", 0)

	logger.Infof("formatted %s %d", "message", 123)
	output := buf.String()

	if !strings.Contains(output, "formatted message 123") {
		t.Errorf("Infof() output = %q, want to contain 'formatted message 123'", output)
	}
}

func TestLoggerWarnf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.warnLog = log.New(&buf, "WARN: ", 0)

	logger.Warnf("warning %s", "test")
	output := buf.String()

	if !strings.Contains(output, "warning test") {
		t.Errorf("Warnf() output = %q, want to contain 'warning test'", output)
	}
}

func TestLoggerErrorf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(false)
	logger.errLog = log.New(&buf, "ERROR: ", 0)

	logger.Errorf("error code: %d", 500)
	output := buf.String()

	if !strings.Contains(output, "error code: 500") {
		t.Errorf("Errorf() output = %q, want to contain 'error code: 500'", output)
	}
}

func TestLoggerDebugf(t *testing.T) {
	tests := []struct {
		name         string
		verbose      bool
		format       string
		args         []interface{}
		shouldOutput bool
		contains     string
	}{
		{
			name:         "debugf with verbose enabled",
			verbose:      true,
			format:       "debug %s %d",
			args:         []interface{}{"test", 42},
			shouldOutput: true,
			contains:     "debug test 42",
		},
		{
			name:         "debugf with verbose disabled",
			verbose:      false,
			format:       "debug %s",
			args:         []interface{}{"test"},
			shouldOutput: false,
			contains:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.verbose)
			logger.dbgLog = log.New(&buf, "DEBUG: ", 0)

			logger.Debugf(tt.format, tt.args...)
			output := buf.String()

			if tt.shouldOutput {
				if output == "" {
					t.Error("Debugf() did not output when verbose mode is enabled")
				}
				if !strings.Contains(output, tt.contains) {
					t.Errorf("Debugf() output = %q, want to contain %q", output, tt.contains)
				}
			} else {
				if output != "" {
					t.Error("Debugf() output when verbose mode is disabled")
				}
			}
		})
	}
}

func TestLoggerVerboseMode(t *testing.T) {
	var buf bytes.Buffer

	// Test with verbose disabled
	logger := NewLogger(false)
	logger.dbgLog = log.New(&buf, "DEBUG: ", 0)
	logger.Debug("should not appear")

	if buf.String() != "" {
		t.Error("Debug message appeared when verbose mode is disabled")
	}

	// Test with verbose enabled
	buf.Reset()
	logger = NewLogger(true)
	logger.dbgLog = log.New(&buf, "DEBUG: ", 0)
	logger.Debug("should appear")

	if buf.String() == "" {
		t.Error("Debug message did not appear when verbose mode is enabled")
	}
}
