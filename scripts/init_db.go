package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func main() {
	// Parse command line flags
	maxRetries := flag.Int("max-retries", 10, "Maximum number of connection retries")
	retryDelay := flag.Int("retry-delay", 2, "Delay between retries in seconds")
	createVectorIndex := flag.Bool("create-vector-index", false, "Create vector similarity index")
	vectorIndexLists := flag.Int("vector-index-lists", 100, "Number of lists for IVFFlat vector index")
	showStats := flag.Bool("stats", false, "Show database statistics")
	flag.Parse()

	log.Println("CodeAtlas Database Initialization Tool")
	log.Println("======================================")

	// Connect to database
	log.Println("Connecting to database...")
	db, err := models.WaitForDatabase(*maxRetries, time.Duration(*retryDelay)*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := models.NewSchemaManager(db)
	ctx := context.Background()

	// Initialize schema
	log.Println("Initializing database schema...")
	if err := sm.InitializeSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Get schema version
	version, err := sm.GetSchemaVersion(ctx)
	if err != nil {
		log.Printf("Warning: failed to get schema version: %v", err)
	} else {
		log.Printf("Schema version: %s", version)
	}

	// Create vector index if requested
	if *createVectorIndex {
		log.Println("Creating vector similarity index...")
		if err := sm.CreateVectorIndex(ctx, *vectorIndexLists); err != nil {
			log.Fatalf("Failed to create vector index: %v", err)
		}
	}

	// Run health check
	log.Println("Running health check...")
	if err := sm.HealthCheck(ctx); err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	log.Println("✓ Health check passed")

	// Show statistics if requested
	if *showStats {
		log.Println("Fetching database statistics...")
		stats, err := sm.GetDatabaseStats(ctx)
		if err != nil {
			log.Printf("Warning: failed to get stats: %v", err)
		} else {
			fmt.Println("\nDatabase Statistics:")
			fmt.Printf("  Repositories: %d\n", stats.RepositoryCount)
			fmt.Printf("  Files:        %d\n", stats.FileCount)
			fmt.Printf("  Symbols:      %d\n", stats.SymbolCount)
			fmt.Printf("  Edges:        %d\n", stats.EdgeCount)
			fmt.Printf("  Vectors:      %d\n", stats.VectorCount)
			fmt.Printf("  Database Size: %s\n", stats.DatabaseSize)
		}
	}

	fmt.Println("\n✓ Database initialization completed successfully")
}
