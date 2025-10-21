package models

import (
	"os"
	"testing"
)

// Unit tests for database utility functions

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_VAR_SET",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "environment variable not set",
			key:          "TEST_VAR_NOT_SET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "empty default value",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "",
			envValue:     "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%s, %s) = %s, want %s", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

// Integration tests for database connection

func TestNewDB_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test that we can ping the database
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestNewDB_WithCustomEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Save original env vars
	originalHost := os.Getenv("DB_HOST")
	originalPort := os.Getenv("DB_PORT")
	originalUser := os.Getenv("DB_USER")
	originalPassword := os.Getenv("DB_PASSWORD")
	originalDBName := os.Getenv("DB_NAME")

	// Restore original env vars after test
	defer func() {
		if originalHost != "" {
			os.Setenv("DB_HOST", originalHost)
		} else {
			os.Unsetenv("DB_HOST")
		}
		if originalPort != "" {
			os.Setenv("DB_PORT", originalPort)
		} else {
			os.Unsetenv("DB_PORT")
		}
		if originalUser != "" {
			os.Setenv("DB_USER", originalUser)
		} else {
			os.Unsetenv("DB_USER")
		}
		if originalPassword != "" {
			os.Setenv("DB_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("DB_PASSWORD")
		}
		if originalDBName != "" {
			os.Setenv("DB_NAME", originalDBName)
		} else {
			os.Unsetenv("DB_NAME")
		}
	}()

	// Set custom env vars (using same values as default for this test)
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "codeatlas")
	os.Setenv("DB_PASSWORD", "codeatlas")
	os.Setenv("DB_NAME", "codeatlas")

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database with custom env: %v", err)
	}
	defer db.Close()

	// Test that we can ping the database
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestNewDB_InvalidConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Save original env vars
	originalHost := os.Getenv("DB_HOST")
	originalPort := os.Getenv("DB_PORT")

	// Restore original env vars after test
	defer func() {
		if originalHost != "" {
			os.Setenv("DB_HOST", originalHost)
		} else {
			os.Unsetenv("DB_HOST")
		}
		if originalPort != "" {
			os.Setenv("DB_PORT", originalPort)
		} else {
			os.Unsetenv("DB_PORT")
		}
	}()

	// Set invalid connection parameters
	os.Setenv("DB_HOST", "invalid-host")
	os.Setenv("DB_PORT", "9999")

	_, err := NewDB()
	if err == nil {
		t.Error("Expected error when connecting to invalid database, got nil")
	}
}

func TestDB_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Close the database
	err = db.Close()
	if err != nil {
		t.Errorf("Failed to close database: %v", err)
	}

	// Verify that we can't ping after closing
	err = db.Ping()
	if err == nil {
		t.Error("Expected error when pinging closed database, got nil")
	}
}

func TestDB_MultipleConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create multiple connections
	db1, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to create first connection: %v", err)
	}
	defer db1.Close()

	db2, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to create second connection: %v", err)
	}
	defer db2.Close()

	// Both should be able to ping
	if err := db1.Ping(); err != nil {
		t.Errorf("First connection failed to ping: %v", err)
	}

	if err := db2.Ping(); err != nil {
		t.Errorf("Second connection failed to ping: %v", err)
	}
}
