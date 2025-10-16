package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
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

	// Create API server
	server := api.NewServer(db)

	// Create Gin router
	r := gin.Default()

	// Register routes
	server.RegisterRoutes(r)

	// Start server
	log.Println("Starting CodeAtlas API server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}