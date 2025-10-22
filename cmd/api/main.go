package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/api"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func main() {
	// Wait for database to be ready with retries
	log.Println("Connecting to database...")
	db, err := models.WaitForDatabase(30, 2*time.Second)
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

	// Load server configuration from environment
	config := loadServerConfig()
	log.Printf("Server configuration - Auth: %v, CORS Origins: %v", config.EnableAuth, config.CORSOrigins)

	// Create API server
	server := api.NewServer(db, config)

	// Setup router with middleware
	r := server.SetupRouter()

	// Get port from environment or use default
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Starting CodeAtlas API server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// loadServerConfig loads server configuration from environment variables
func loadServerConfig() *api.ServerConfig {
	config := &api.ServerConfig{
		EnableAuth:  false,
		AuthTokens:  []string{},
		CORSOrigins: []string{"*"},
	}

	// Check if authentication is enabled
	if os.Getenv("ENABLE_AUTH") == "true" {
		config.EnableAuth = true

		// Load auth tokens from environment
		tokensEnv := os.Getenv("AUTH_TOKENS")
		if tokensEnv != "" {
			config.AuthTokens = strings.Split(tokensEnv, ",")
			for i := range config.AuthTokens {
				config.AuthTokens[i] = strings.TrimSpace(config.AuthTokens[i])
			}
		}
	}

	// Load CORS origins from environment
	originsEnv := os.Getenv("CORS_ORIGINS")
	if originsEnv != "" {
		config.CORSOrigins = strings.Split(originsEnv, ",")
		for i := range config.CORSOrigins {
			config.CORSOrigins[i] = strings.TrimSpace(config.CORSOrigins[i])
		}
	}

	return config
}
