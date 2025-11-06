package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/api"
	"github.com/yourtionguo/CodeAtlas/internal/config"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func main() {
	// Check for verbose flag
	verbose := false
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
			break
		}
	}

	// Create logger
	logger := utils.NewLogger(verbose)

	// Load configuration
	logger.Info("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	logger.InfoWithFields("Configuration loaded",
		utils.Field{Key: "api_port", Value: cfg.API.Port},
		utils.Field{Key: "api_host", Value: cfg.API.Host},
		utils.Field{Key: "db_host", Value: cfg.Database.Host},
		utils.Field{Key: "db_port", Value: cfg.Database.Port},
		utils.Field{Key: "db_name", Value: cfg.Database.Database},
	)

	// Wait for database to be ready with retries
	logger.Info("Connecting to database...")
	db, err := models.NewDBWithConfig(&cfg.Database)
	if err != nil {
		logger.ErrorWithFields("Failed to connect to database", err,
			utils.Field{Key: "db_host", Value: cfg.Database.Host},
			utils.Field{Key: "db_port", Value: cfg.Database.Port},
			utils.Field{Key: "db_name", Value: cfg.Database.Database},
		)
		os.Exit(1)
	}
	defer db.Close()
	logger.InfoWithFields("Database connection established",
		utils.Field{Key: "db_host", Value: cfg.Database.Host},
		utils.Field{Key: "db_name", Value: cfg.Database.Database},
	)

	// Initialize database schema
	logger.Info("Initializing database schema...")
	sm := models.NewSchemaManager(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := sm.InitializeSchema(ctx); err != nil {
		logger.ErrorWithFields("Failed to initialize database schema", err)
		os.Exit(1)
	}
	logger.Info("Database schema initialized successfully")

	// Run health check
	logger.Debug("Running database health check...")
	if err := sm.HealthCheck(ctx); err != nil {
		logger.ErrorWithFields("Database health check failed", err)
		os.Exit(1)
	}
	logger.Info("Database health check passed")

	// Log database statistics
	stats, err := sm.GetDatabaseStats(ctx)
	if err != nil {
		logger.WarnWithFields("Failed to get database stats",
			utils.Field{Key: "error", Value: err.Error()},
		)
	} else {
		logger.InfoWithFields("Database statistics",
			utils.Field{Key: "repositories", Value: stats.RepositoryCount},
			utils.Field{Key: "files", Value: stats.FileCount},
			utils.Field{Key: "symbols", Value: stats.SymbolCount},
			utils.Field{Key: "edges", Value: stats.EdgeCount},
		)
	}

	// Convert config.EmbedderConfig to indexer.EmbedderConfig
	embedderConfig := &indexer.EmbedderConfig{
		Backend:              cfg.Embedder.Backend,
		APIEndpoint:          cfg.Embedder.APIEndpoint,
		APIKey:               cfg.Embedder.APIKey,
		Model:                cfg.Embedder.Model,
		Dimensions:           cfg.Embedder.Dimensions,
		BatchSize:            cfg.Embedder.BatchSize,
		MaxRequestsPerSecond: cfg.Embedder.MaxRequestsPerSecond,
		MaxRetries:           cfg.Embedder.MaxRetries,
		BaseRetryDelay:       cfg.Embedder.BaseRetryDelay,
		MaxRetryDelay:        cfg.Embedder.MaxRetryDelay,
		Timeout:              cfg.Embedder.Timeout,
	}

	// Create server configuration from loaded config
	serverConfig := &api.ServerConfig{
		EnableAuth:     cfg.API.EnableAuth,
		AuthTokens:     cfg.API.AuthTokens,
		CORSOrigins:    cfg.API.CORSOrigins,
		EmbedderConfig: embedderConfig,
	}
	logger.InfoWithFields("Server configuration",
		utils.Field{Key: "auth_enabled", Value: serverConfig.EnableAuth},
		utils.Field{Key: "cors_origins", Value: serverConfig.CORSOrigins},
		utils.Field{Key: "auth_tokens_count", Value: len(serverConfig.AuthTokens)},
		utils.Field{Key: "embedder_backend", Value: embedderConfig.Backend},
		utils.Field{Key: "embedder_model", Value: embedderConfig.Model},
	)

	// Create API server
	server := api.NewServer(db, serverConfig)

	// Setup router with middleware
	r := server.SetupRouter()

	// Start server
	address := cfg.API.Address()
	logger.InfoWithFields("Starting CodeAtlas API server",
		utils.Field{Key: "address", Value: address},
		utils.Field{Key: "port", Value: cfg.API.Port},
		utils.Field{Key: "host", Value: cfg.API.Host},
	)

	if err := r.Run(fmt.Sprintf(":%d", cfg.API.Port)); err != nil {
		logger.ErrorWithFields("Failed to start server", err,
			utils.Field{Key: "address", Value: address},
		)
		os.Exit(1)
	}
}
