package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Database DatabaseConfig
	API      APIConfig
	Indexer  IndexerConfig
	Embedder EmbedderConfig
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// APIConfig holds API server configuration
type APIConfig struct {
	Host        string
	Port        int
	EnableAuth  bool
	AuthTokens  []string
	CORSOrigins []string
	Timeout     time.Duration
}

// IndexerConfig holds indexer configuration
type IndexerConfig struct {
	BatchSize       int
	WorkerCount     int
	SkipVectors     bool
	Incremental     bool
	UseTransactions bool
	GraphName       string
	EmbeddingModel  string
}

// EmbedderConfig holds embedder configuration
type EmbedderConfig struct {
	Backend              string
	APIEndpoint          string
	APIKey               string
	Model                string
	Dimensions           int
	BatchSize            int
	MaxRequestsPerSecond int
	MaxRetries           int
	BaseRetryDelay       time.Duration
	MaxRetryDelay        time.Duration
	Timeout              time.Duration
}

// ToIndexerConfig converts config.EmbedderConfig to indexer.EmbedderConfig
// This is a helper to avoid import cycles and provide a clean conversion
func (e *EmbedderConfig) ToIndexerEmbedderConfig() map[string]interface{} {
	return map[string]interface{}{
		"Backend":              e.Backend,
		"APIEndpoint":          e.APIEndpoint,
		"APIKey":               e.APIKey,
		"Model":                e.Model,
		"Dimensions":           e.Dimensions,
		"BatchSize":            e.BatchSize,
		"MaxRequestsPerSecond": e.MaxRequestsPerSecond,
		"MaxRetries":           e.MaxRetries,
		"BaseRetryDelay":       e.BaseRetryDelay,
		"MaxRetryDelay":        e.MaxRetryDelay,
		"Timeout":              e.Timeout,
	}
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() (*Config, error) {
	config := &Config{
		Database: loadDatabaseConfig(),
		API:      loadAPIConfig(),
		Indexer:  loadIndexerConfig(),
		Embedder: loadEmbedderConfig(),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// loadDatabaseConfig loads database configuration from environment
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "codeatlas"),
		Password:        getEnv("DB_PASSWORD", "codeatlas"),
		Database:        getEnv("DB_NAME", "codeatlas"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
	}
}

// loadAPIConfig loads API server configuration from environment
func loadAPIConfig() APIConfig {
	return APIConfig{
		Host:        getEnv("API_HOST", "0.0.0.0"),
		Port:        getEnvInt("API_PORT", 8080),
		EnableAuth:  getEnvBool("ENABLE_AUTH", false),
		AuthTokens:  getEnvStringSlice("AUTH_TOKENS", []string{}),
		CORSOrigins: getEnvStringSlice("CORS_ORIGINS", []string{"*"}),
		Timeout:     getEnvDuration("API_TIMEOUT", 30*time.Second),
	}
}

// loadIndexerConfig loads indexer configuration from environment
func loadIndexerConfig() IndexerConfig {
	return IndexerConfig{
		BatchSize:       getEnvInt("INDEXER_BATCH_SIZE", 100),
		WorkerCount:     getEnvInt("INDEXER_WORKER_COUNT", 4),
		SkipVectors:     getEnvBool("INDEXER_SKIP_VECTORS", false),
		Incremental:     getEnvBool("INDEXER_INCREMENTAL", false),
		UseTransactions: getEnvBool("INDEXER_USE_TRANSACTIONS", true),
		GraphName:       getEnv("INDEXER_GRAPH_NAME", "code_graph"),
		EmbeddingModel:  getEnv("INDEXER_EMBEDDING_MODEL", ""),
	}
}

// loadEmbedderConfig loads embedder configuration from environment
func loadEmbedderConfig() EmbedderConfig {
	return EmbedderConfig{
		Backend:              getEnv("EMBEDDING_BACKEND", "openai"),
		APIEndpoint:          getEnv("EMBEDDING_API_ENDPOINT", "http://localhost:1234/v1/embeddings"),
		APIKey:               getEnv("EMBEDDING_API_KEY", ""),
		Model:                getEnv("EMBEDDING_MODEL", "text-embedding-qwen3-embedding-0.6b"),
		Dimensions:           getEnvInt("EMBEDDING_DIMENSIONS", 768),
		BatchSize:            getEnvInt("EMBEDDING_BATCH_SIZE", 50),
		MaxRequestsPerSecond: getEnvInt("EMBEDDING_MAX_REQUESTS_PER_SECOND", 10),
		MaxRetries:           getEnvInt("EMBEDDING_MAX_RETRIES", 3),
		BaseRetryDelay:       getEnvDuration("EMBEDDING_BASE_RETRY_DELAY", 100*time.Millisecond),
		MaxRetryDelay:        getEnvDuration("EMBEDDING_MAX_RETRY_DELAY", 5*time.Second),
		Timeout:              getEnvDuration("EMBEDDING_TIMEOUT", 30*time.Second),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate database config
	if c.Database.Host == "" {
		return fmt.Errorf("database host cannot be empty")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user cannot be empty")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("database max open connections must be at least 1")
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database max idle connections cannot be negative")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("database max idle connections cannot exceed max open connections")
	}

	// Validate API config
	if c.API.Port <= 0 || c.API.Port > 65535 {
		return fmt.Errorf("API port must be between 1 and 65535")
	}
	if c.API.EnableAuth && len(c.API.AuthTokens) == 0 {
		return fmt.Errorf("authentication is enabled but no auth tokens are configured")
	}

	// Validate indexer config
	if c.Indexer.BatchSize < 1 {
		return fmt.Errorf("indexer batch size must be at least 1")
	}
	if c.Indexer.WorkerCount < 1 {
		return fmt.Errorf("indexer worker count must be at least 1")
	}
	if c.Indexer.GraphName == "" {
		return fmt.Errorf("indexer graph name cannot be empty")
	}

	// Validate embedder config
	if !c.Indexer.SkipVectors {
		if c.Embedder.Backend != "openai" && c.Embedder.Backend != "local" {
			return fmt.Errorf("embedder backend must be 'openai' or 'local'")
		}
		if c.Embedder.APIEndpoint == "" {
			return fmt.Errorf("embedder API endpoint cannot be empty")
		}
		if c.Embedder.Model == "" {
			return fmt.Errorf("embedder model cannot be empty")
		}
		if c.Embedder.Dimensions < 1 {
			return fmt.Errorf("embedder dimensions must be at least 1")
		}
		if c.Embedder.BatchSize < 1 {
			return fmt.Errorf("embedder batch size must be at least 1")
		}
		if c.Embedder.MaxRequestsPerSecond < 1 {
			return fmt.Errorf("embedder max requests per second must be at least 1")
		}
		if c.Embedder.MaxRetries < 0 {
			return fmt.Errorf("embedder max retries cannot be negative")
		}
	}

	return nil
}

// ConnectionString returns the PostgreSQL connection string
func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// Address returns the API server address
func (c *APIConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
