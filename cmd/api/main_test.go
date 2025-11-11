package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/config"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestConfigurationLoading(t *testing.T) {
	// Test configuration loading with environment variables
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "default configuration",
			envVars: map[string]string{
				"DB_HOST":     "localhost",
				"DB_PORT":     "5432",
				"DB_USER":     "codeatlas",
				"DB_PASSWORD": "codeatlas",
				"DB_NAME":     "codeatlas",
			},
			wantErr: false,
		},
		{
			name: "custom API port",
			envVars: map[string]string{
				"API_PORT":    "9090",
				"DB_HOST":     "localhost",
				"DB_PORT":     "5432",
				"DB_USER":     "codeatlas",
				"DB_PASSWORD": "codeatlas",
				"DB_NAME":     "codeatlas",
			},
			wantErr: false,
		},
		{
			name: "with authentication",
			envVars: map[string]string{
				"ENABLE_AUTH": "true",
				"AUTH_TOKENS": "token1,token2,token3",
				"DB_HOST":     "localhost",
				"DB_PORT":     "5432",
				"DB_USER":     "codeatlas",
				"DB_PASSWORD": "codeatlas",
				"DB_NAME":     "codeatlas",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				// Clean up environment variables
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			// Load configuration
			cfg, err := config.LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify configuration was loaded
				if cfg == nil {
					t.Error("Expected configuration to be loaded, got nil")
					return
				}

				// Verify database configuration
				if cfg.Database.Host != tt.envVars["DB_HOST"] {
					t.Errorf("Expected DB_HOST %s, got %s", tt.envVars["DB_HOST"], cfg.Database.Host)
				}

				// Verify API configuration if set
				if apiPort, ok := tt.envVars["API_PORT"]; ok {
					if cfg.API.Port != 9090 {
						t.Errorf("Expected API_PORT 9090, got %d", cfg.API.Port)
					}
					_ = apiPort
				}

				// Verify authentication configuration if set
				if enableAuth, ok := tt.envVars["ENABLE_AUTH"]; ok && enableAuth == "true" {
					if !cfg.API.EnableAuth {
						t.Error("Expected EnableAuth to be true")
					}
					if len(cfg.API.AuthTokens) != 3 {
						t.Errorf("Expected 3 auth tokens, got %d", len(cfg.API.AuthTokens))
					}
				}
			}
		})
	}
}

func TestDatabaseConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database connection test in short mode")
	}

	// Test database connection with valid configuration
	cfg := &config.DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "codeatlas"),
		Password: getEnv("DB_PASSWORD", "codeatlas"),
		Database: getEnv("DB_NAME", "codeatlas"),
		SSLMode:  "disable",
	}

	db, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return
	}
	defer db.Close()

	// Test connection is alive
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

func TestDatabaseSchemaInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database schema initialization test in short mode")
	}

	// Connect to database
	cfg := &config.DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "codeatlas"),
		Password: getEnv("DB_PASSWORD", "codeatlas"),
		Database: getEnv("DB_NAME", "codeatlas"),
		SSLMode:  "disable",
	}

	db, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return
	}
	defer db.Close()

	// Initialize schema
	sm := models.NewSchemaManager(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = sm.InitializeSchema(ctx)
	if err != nil {
		t.Errorf("Failed to initialize schema: %v", err)
		return
	}

	// Verify schema was initialized by running health check
	err = sm.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed after schema initialization: %v", err)
	}

	// Verify tables exist
	tables := []string{"repositories", "files", "symbols", "edges"}
	for _, table := range tables {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)
		`
		err := db.QueryRowContext(ctx, query, table).Scan(&exists)
		if err != nil {
			t.Errorf("Failed to check if table %s exists: %v", table, err)
			continue
		}
		if !exists {
			t.Errorf("Table %s does not exist after schema initialization", table)
		}
	}
}

func TestDatabaseHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database health check test in short mode")
	}

	// Connect to database
	cfg := &config.DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "codeatlas"),
		Password: getEnv("DB_PASSWORD", "codeatlas"),
		Database: getEnv("DB_NAME", "codeatlas"),
		SSLMode:  "disable",
	}

	db, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return
	}
	defer db.Close()

	// Run health check
	sm := models.NewSchemaManager(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sm.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestDatabaseStatistics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database statistics test in short mode")
	}

	// Connect to database
	cfg := &config.DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "codeatlas"),
		Password: getEnv("DB_PASSWORD", "codeatlas"),
		Database: getEnv("DB_NAME", "codeatlas"),
		SSLMode:  "disable",
	}

	db, err := models.NewDBWithConfig(cfg)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return
	}
	defer db.Close()

	// Get database statistics
	sm := models.NewSchemaManager(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := sm.GetDatabaseStats(ctx)
	if err != nil {
		t.Errorf("Failed to get database statistics: %v", err)
		return
	}

	// Verify statistics structure
	if stats == nil {
		t.Error("Expected statistics to be returned, got nil")
		return
	}

	// Statistics should be non-negative
	if stats.RepositoryCount < 0 {
		t.Errorf("Expected non-negative repository count, got %d", stats.RepositoryCount)
	}
	if stats.FileCount < 0 {
		t.Errorf("Expected non-negative file count, got %d", stats.FileCount)
	}
	if stats.SymbolCount < 0 {
		t.Errorf("Expected non-negative symbol count, got %d", stats.SymbolCount)
	}
	if stats.EdgeCount < 0 {
		t.Errorf("Expected non-negative edge count, got %d", stats.EdgeCount)
	}
}

func TestVerboseFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "no verbose flag",
			args:     []string{"cmd"},
			expected: false,
		},
		{
			name:     "short verbose flag",
			args:     []string{"cmd", "-v"},
			expected: true,
		},
		{
			name:     "long verbose flag",
			args:     []string{"cmd", "--verbose"},
			expected: true,
		},
		{
			name:     "verbose flag with other args",
			args:     []string{"cmd", "arg1", "-v", "arg2"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check for verbose flag
			verbose := false
			for _, arg := range tt.args {
				if arg == "-v" || arg == "--verbose" {
					verbose = true
					break
				}
			}

			if verbose != tt.expected {
				t.Errorf("Expected verbose=%v, got %v", tt.expected, verbose)
			}
		})
	}
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
