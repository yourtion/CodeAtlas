package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment
	originalEnv := saveEnv()
	defer restoreEnv(originalEnv)

	// Test with default values
	t.Run("defaults", func(t *testing.T) {
		clearEnv()
		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() failed: %v", err)
		}

		// Check database defaults
		if config.Database.Host != "localhost" {
			t.Errorf("expected database host 'localhost', got '%s'", config.Database.Host)
		}
		if config.Database.Port != 5432 {
			t.Errorf("expected database port 5432, got %d", config.Database.Port)
		}
		if config.Database.User != "codeatlas" {
			t.Errorf("expected database user 'codeatlas', got '%s'", config.Database.User)
		}

		// Check API defaults
		if config.API.Port != 8080 {
			t.Errorf("expected API port 8080, got %d", config.API.Port)
		}
		if config.API.EnableAuth {
			t.Error("expected API auth to be disabled by default")
		}

		// Check indexer defaults
		if config.Indexer.BatchSize != 100 {
			t.Errorf("expected indexer batch size 100, got %d", config.Indexer.BatchSize)
		}
		if config.Indexer.WorkerCount != 4 {
			t.Errorf("expected indexer worker count 4, got %d", config.Indexer.WorkerCount)
		}
		if config.Indexer.GraphName != "code_graph" {
			t.Errorf("expected graph name 'code_graph', got '%s'", config.Indexer.GraphName)
		}

		// Check embedder defaults
		if config.Embedder.Backend != "openai" {
			t.Errorf("expected embedder backend 'openai', got '%s'", config.Embedder.Backend)
		}
		if config.Embedder.Dimensions != 768 {
			t.Errorf("expected embedder dimensions 768, got %d", config.Embedder.Dimensions)
		}
	})

	// Test with custom environment variables
	t.Run("custom_values", func(t *testing.T) {
		clearEnv()
		os.Setenv("DB_HOST", "db.example.com")
		os.Setenv("DB_PORT", "5433")
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("DB_NAME", "testdb")
		os.Setenv("API_PORT", "9090")
		os.Setenv("ENABLE_AUTH", "true")
		os.Setenv("AUTH_TOKENS", "token1,token2,token3")
		os.Setenv("INDEXER_BATCH_SIZE", "200")
		os.Setenv("INDEXER_WORKER_COUNT", "8")
		os.Setenv("EMBEDDING_DIMENSIONS", "1536")

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() failed: %v", err)
		}

		if config.Database.Host != "db.example.com" {
			t.Errorf("expected database host 'db.example.com', got '%s'", config.Database.Host)
		}
		if config.Database.Port != 5433 {
			t.Errorf("expected database port 5433, got %d", config.Database.Port)
		}
		if config.API.Port != 9090 {
			t.Errorf("expected API port 9090, got %d", config.API.Port)
		}
		if !config.API.EnableAuth {
			t.Error("expected API auth to be enabled")
		}
		if len(config.API.AuthTokens) != 3 {
			t.Errorf("expected 3 auth tokens, got %d", len(config.API.AuthTokens))
		}
		if config.Indexer.BatchSize != 200 {
			t.Errorf("expected indexer batch size 200, got %d", config.Indexer.BatchSize)
		}
		if config.Embedder.Dimensions != 1536 {
			t.Errorf("expected embedder dimensions 1536, got %d", config.Embedder.Dimensions)
		}
	})
}

func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid_config",
			config: DatabaseConfig{
				Host:         "localhost",
				Port:         5432,
				User:         "user",
				Database:     "db",
				MaxOpenConns: 10,
				MaxIdleConns: 5,
			},
			wantErr: false,
		},
		{
			name: "empty_host",
			config: DatabaseConfig{
				Host:     "",
				Port:     5432,
				User:     "user",
				Database: "db",
			},
			wantErr: true,
		},
		{
			name: "invalid_port",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     0,
				User:     "user",
				Database: "db",
			},
			wantErr: true,
		},
		{
			name: "empty_user",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "",
				Database: "db",
			},
			wantErr: true,
		},
		{
			name: "idle_exceeds_max",
			config: DatabaseConfig{
				Host:         "localhost",
				Port:         5432,
				User:         "user",
				Database:     "db",
				MaxOpenConns: 5,
				MaxIdleConns: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Database: tt.config,
				API: APIConfig{
					Port: 8080,
				},
				Indexer: IndexerConfig{
					BatchSize:   100,
					WorkerCount: 4,
					GraphName:   "test_graph",
				},
				Embedder: EmbedderConfig{
					Backend:              "openai",
					APIEndpoint:          "http://localhost:1234",
					Model:                "test-model",
					Dimensions:           768,
					BatchSize:            50,
					MaxRequestsPerSecond: 10,
				},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  APIConfig
		wantErr bool
	}{
		{
			name: "valid_config",
			config: APIConfig{
				Port:       8080,
				EnableAuth: false,
			},
			wantErr: false,
		},
		{
			name: "valid_with_auth",
			config: APIConfig{
				Port:       8080,
				EnableAuth: true,
				AuthTokens: []string{"token1"},
			},
			wantErr: false,
		},
		{
			name: "invalid_port",
			config: APIConfig{
				Port: 0,
			},
			wantErr: true,
		},
		{
			name: "auth_without_tokens",
			config: APIConfig{
				Port:       8080,
				EnableAuth: true,
				AuthTokens: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Database: DatabaseConfig{
					Host:         "localhost",
					Port:         5432,
					User:         "user",
					Database:     "db",
					MaxOpenConns: 10,
				},
				API: tt.config,
				Indexer: IndexerConfig{
					BatchSize:   100,
					WorkerCount: 4,
					GraphName:   "test_graph",
				},
				Embedder: EmbedderConfig{
					Backend:              "openai",
					APIEndpoint:          "http://localhost:1234",
					Model:                "test-model",
					Dimensions:           768,
					BatchSize:            50,
					MaxRequestsPerSecond: 10,
				},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIndexerConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  IndexerConfig
		wantErr bool
	}{
		{
			name: "valid_config",
			config: IndexerConfig{
				BatchSize:   100,
				WorkerCount: 4,
				GraphName:   "test_graph",
			},
			wantErr: false,
		},
		{
			name: "invalid_batch_size",
			config: IndexerConfig{
				BatchSize:   0,
				WorkerCount: 4,
				GraphName:   "test_graph",
			},
			wantErr: true,
		},
		{
			name: "invalid_worker_count",
			config: IndexerConfig{
				BatchSize:   100,
				WorkerCount: 0,
				GraphName:   "test_graph",
			},
			wantErr: true,
		},
		{
			name: "empty_graph_name",
			config: IndexerConfig{
				BatchSize:   100,
				WorkerCount: 4,
				GraphName:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Database: DatabaseConfig{
					Host:         "localhost",
					Port:         5432,
					User:         "user",
					Database:     "db",
					MaxOpenConns: 10,
				},
				API: APIConfig{
					Port: 8080,
				},
				Indexer: tt.config,
				Embedder: EmbedderConfig{
					Backend:              "openai",
					APIEndpoint:          "http://localhost:1234",
					Model:                "test-model",
					Dimensions:           768,
					BatchSize:            50,
					MaxRequestsPerSecond: 10,
				},
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbedderConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      EmbedderConfig
		skipVectors bool
		wantErr     bool
	}{
		{
			name: "valid_config",
			config: EmbedderConfig{
				Backend:              "openai",
				APIEndpoint:          "http://localhost:1234",
				Model:                "test-model",
				Dimensions:           768,
				BatchSize:            50,
				MaxRequestsPerSecond: 10,
			},
			skipVectors: false,
			wantErr:     false,
		},
		{
			name: "skip_vectors",
			config: EmbedderConfig{
				Backend: "invalid",
			},
			skipVectors: true,
			wantErr:     false,
		},
		{
			name: "invalid_backend",
			config: EmbedderConfig{
				Backend:              "invalid",
				APIEndpoint:          "http://localhost:1234",
				Model:                "test-model",
				Dimensions:           768,
				BatchSize:            50,
				MaxRequestsPerSecond: 10,
			},
			skipVectors: false,
			wantErr:     true,
		},
		{
			name: "empty_endpoint",
			config: EmbedderConfig{
				Backend:              "openai",
				APIEndpoint:          "",
				Model:                "test-model",
				Dimensions:           768,
				BatchSize:            50,
				MaxRequestsPerSecond: 10,
			},
			skipVectors: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Database: DatabaseConfig{
					Host:         "localhost",
					Port:         5432,
					User:         "user",
					Database:     "db",
					MaxOpenConns: 10,
				},
				API: APIConfig{
					Port: 8080,
				},
				Indexer: IndexerConfig{
					BatchSize:   100,
					WorkerCount: 4,
					GraphName:   "test_graph",
					SkipVectors: tt.skipVectors,
				},
				Embedder: tt.config,
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConnectionString(t *testing.T) {
	config := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	actual := config.ConnectionString()

	if actual != expected {
		t.Errorf("ConnectionString() = %s, want %s", actual, expected)
	}
}

func TestAPIAddress(t *testing.T) {
	config := APIConfig{
		Host: "0.0.0.0",
		Port: 8080,
	}

	expected := "0.0.0.0:8080"
	actual := config.Address()

	if actual != expected {
		t.Errorf("Address() = %s, want %s", actual, expected)
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// Save original environment
	originalEnv := saveEnv()
	defer restoreEnv(originalEnv)

	t.Run("getEnv", func(t *testing.T) {
		clearEnv()
		os.Setenv("TEST_STRING", "value")
		if got := getEnv("TEST_STRING", "default"); got != "value" {
			t.Errorf("getEnv() = %s, want 'value'", got)
		}
		if got := getEnv("MISSING", "default"); got != "default" {
			t.Errorf("getEnv() = %s, want 'default'", got)
		}
	})

	t.Run("getEnvInt", func(t *testing.T) {
		clearEnv()
		os.Setenv("TEST_INT", "42")
		if got := getEnvInt("TEST_INT", 0); got != 42 {
			t.Errorf("getEnvInt() = %d, want 42", got)
		}
		if got := getEnvInt("MISSING", 10); got != 10 {
			t.Errorf("getEnvInt() = %d, want 10", got)
		}
		os.Setenv("TEST_INT", "invalid")
		if got := getEnvInt("TEST_INT", 10); got != 10 {
			t.Errorf("getEnvInt() with invalid value = %d, want 10", got)
		}
	})

	t.Run("getEnvBool", func(t *testing.T) {
		clearEnv()
		os.Setenv("TEST_BOOL", "true")
		if got := getEnvBool("TEST_BOOL", false); !got {
			t.Error("getEnvBool() = false, want true")
		}
		if got := getEnvBool("MISSING", true); !got {
			t.Error("getEnvBool() = false, want true")
		}
		os.Setenv("TEST_BOOL", "invalid")
		if got := getEnvBool("TEST_BOOL", true); !got {
			t.Error("getEnvBool() with invalid value = false, want true")
		}
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		clearEnv()
		os.Setenv("TEST_DURATION", "5s")
		if got := getEnvDuration("TEST_DURATION", 0); got != 5*time.Second {
			t.Errorf("getEnvDuration() = %v, want 5s", got)
		}
		if got := getEnvDuration("MISSING", 10*time.Second); got != 10*time.Second {
			t.Errorf("getEnvDuration() = %v, want 10s", got)
		}
		os.Setenv("TEST_DURATION", "invalid")
		if got := getEnvDuration("TEST_DURATION", 10*time.Second); got != 10*time.Second {
			t.Errorf("getEnvDuration() with invalid value = %v, want 10s", got)
		}
	})

	t.Run("getEnvStringSlice", func(t *testing.T) {
		clearEnv()
		os.Setenv("TEST_SLICE", "a,b,c")
		got := getEnvStringSlice("TEST_SLICE", []string{})
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("getEnvStringSlice() = %v, want [a b c]", got)
		}

		os.Setenv("TEST_SLICE", " a , b , c ")
		got = getEnvStringSlice("TEST_SLICE", []string{})
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("getEnvStringSlice() with spaces = %v, want [a b c]", got)
		}

		got = getEnvStringSlice("MISSING", []string{"default"})
		if len(got) != 1 || got[0] != "default" {
			t.Errorf("getEnvStringSlice() = %v, want [default]", got)
		}
	})
}

// Helper functions for test environment management

func saveEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

func restoreEnv(env map[string]string) {
	os.Clearenv()
	for k, v := range env {
		os.Setenv(k, v)
	}
}

func clearEnv() {
	// Clear only test-related environment variables
	testVars := []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_IDLE_TIME",
		"API_HOST", "API_PORT", "ENABLE_AUTH", "AUTH_TOKENS", "CORS_ORIGINS", "API_TIMEOUT",
		"INDEXER_BATCH_SIZE", "INDEXER_WORKER_COUNT", "INDEXER_SKIP_VECTORS",
		"INDEXER_INCREMENTAL", "INDEXER_USE_TRANSACTIONS", "INDEXER_GRAPH_NAME", "INDEXER_EMBEDDING_MODEL",
		"EMBEDDING_BACKEND", "EMBEDDING_API_ENDPOINT", "EMBEDDING_API_KEY", "EMBEDDING_MODEL",
		"EMBEDDING_DIMENSIONS", "EMBEDDING_BATCH_SIZE", "EMBEDDING_MAX_REQUESTS_PER_SECOND",
		"EMBEDDING_MAX_RETRIES", "EMBEDDING_BASE_RETRY_DELAY", "EMBEDDING_MAX_RETRY_DELAY", "EMBEDDING_TIMEOUT",
		"TEST_STRING", "TEST_INT", "TEST_BOOL", "TEST_DURATION", "TEST_SLICE",
	}
	for _, v := range testVars {
		os.Unsetenv(v)
	}
}
