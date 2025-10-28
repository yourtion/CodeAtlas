package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/api"
	"github.com/yourtionguo/CodeAtlas/internal/config"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func main() {
	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Wait for database to be ready with retries
	log.Println("Connecting to database...")
	db, err := models.NewDBWithConfig(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize database schema
	log.Println("Initializing database schema...")
	sm := models.NewSchemaManager(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := sm.InitializeSchema(ctx); err != nil {
		log.Fatal("Failed to initialize database schema:", err)
	}

	// Run health check
	if err := sm.HealthCheck(ctx); err != nil {
		log.Fatal("Database health check failed:", err)
	}

	// Log database statistics
	stats, err := sm.GetDatabaseStats(ctx)
	if err != nil {
		log.Printf("Warning: failed to get database stats: %v", err)
	} else {
		log.Printf("Database ready - Repos: %d, Files: %d, Symbols: %d, Edges: %d",
			stats.RepositoryCount, stats.FileCount, stats.SymbolCount, stats.EdgeCount)
	}

	// Create server configuration from loaded config
	serverConfig := &api.ServerConfig{
		EnableAuth:  cfg.API.EnableAuth,
		AuthTokens:  cfg.API.AuthTokens,
		CORSOrigins: cfg.API.CORSOrigins,
	}
	log.Printf("Server configuration - Auth: %v, CORS Origins: %v", serverConfig.EnableAuth, serverConfig.CORSOrigins)

	// Create API server
	server := api.NewServer(db, serverConfig)

	// Setup router with middleware
	r := server.SetupRouter()

	// Start server
	address := cfg.API.Address()
	log.Printf("Starting CodeAtlas API server on %s", address)
	if err := r.Run(fmt.Sprintf(":%d", cfg.API.Port)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
