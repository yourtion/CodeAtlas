package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/config"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func main() {
	// Parse command line flags
	vectorDim := flag.Int("dimension", 0, "New vector dimension (required)")
	force := flag.Bool("force", false, "Force change by truncating vectors table")
	dryRun := flag.Bool("dry-run", false, "Show what would be done without making changes")
	flag.Parse()

	// Get dimension from flag or environment
	if *vectorDim <= 0 {
		if envDim := os.Getenv("EMBEDDING_DIMENSIONS"); envDim != "" {
			if dim, err := strconv.Atoi(envDim); err == nil && dim > 0 {
				*vectorDim = dim
			}
		}
	}

	if *vectorDim <= 0 {
		fmt.Fprintln(os.Stderr, "Error: vector dimension must be specified")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  alter_vector_dimension -dimension <dim> [-force] [-dry-run]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  alter_vector_dimension -dimension 1536")
		fmt.Fprintln(os.Stderr, "  alter_vector_dimension -dimension 768 -force")
		fmt.Fprintln(os.Stderr, "  EMBEDDING_DIMENSIONS=1536 alter_vector_dimension")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Common dimensions:")
		fmt.Fprintln(os.Stderr, "  768  - nomic-embed-text")
		fmt.Fprintln(os.Stderr, "  1024 - text-embedding-qwen3-embedding-0.6b")
		fmt.Fprintln(os.Stderr, "  1536 - text-embedding-3-small (OpenAI)")
		fmt.Fprintln(os.Stderr, "  3072 - text-embedding-3-large (OpenAI)")
		os.Exit(1)
	}

	log.Printf("Vector Dimension Alteration Tool")
	log.Printf("================================")
	log.Printf("Target dimension: %d", *vectorDim)
	if *force {
		log.Printf("Mode: FORCE (will truncate vectors table)")
	}
	if *dryRun {
		log.Printf("Mode: DRY RUN (no changes will be made)")
	}
	log.Println()

	// Connect to database
	cfg := &config.DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "codeatlas"),
		Password:        getEnv("DB_PASSWORD", "codeatlas"),
		Database:        getEnv("DB_NAME", "codeatlas"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	log.Println("Connecting to database...")
	db, err := models.NewDBWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check current dimension
	log.Println("Checking current vector dimension...")
	var currentDim int
	var vectorCount int
	
	err = db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(
				(SELECT COUNT(*) FROM vectors LIMIT 1),
				0
			) as count
	`).Scan(&vectorCount)
	
	if err != nil {
		log.Printf("Warning: Could not check vector count: %v", err)
	} else {
		log.Printf("Current vectors in table: %d", vectorCount)
	}

	// Try to get current dimension from a sample vector
	if vectorCount > 0 {
		err = db.QueryRowContext(ctx, `
			SELECT array_length(embedding::real[], 1) 
			FROM vectors 
			LIMIT 1
		`).Scan(&currentDim)
		
		if err != nil {
			log.Printf("Warning: Could not determine current dimension: %v", err)
		} else {
			log.Printf("Current vector dimension: %d", currentDim)
			if currentDim == *vectorDim {
				log.Printf("✓ Dimension is already %d, no change needed", *vectorDim)
				return
			}
		}
	}

	if *dryRun {
		log.Println()
		log.Println("DRY RUN - Would execute:")
		if *force && vectorCount > 0 {
			log.Println("  1. TRUNCATE TABLE vectors;")
		}
		log.Printf("  2. ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(%d);", *vectorDim)
		log.Println()
		log.Println("No changes made (dry-run mode)")
		return
	}

	// Perform the alteration
	log.Println()
	if *force && vectorCount > 0 {
		log.Println("⚠️  Truncating vectors table (force mode)...")
		_, err = db.ExecContext(ctx, "TRUNCATE TABLE vectors")
		if err != nil {
			log.Fatalf("Failed to truncate vectors table: %v", err)
		}
		log.Println("✓ Vectors table truncated")
	}

	log.Printf("Altering vector dimension to %d...", *vectorDim)
	alterSQL := fmt.Sprintf("ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(%d)", *vectorDim)
	_, err = db.ExecContext(ctx, alterSQL)
	if err != nil {
		log.Printf("✗ Failed to alter vector dimension: %v", err)
		log.Println()
		log.Println("Possible solutions:")
		log.Println("  1. Use -force flag to truncate existing vectors")
		log.Println("  2. Manually truncate: TRUNCATE TABLE vectors;")
		log.Println("  3. Drop and recreate the table")
		os.Exit(1)
	}

	log.Printf("✓ Vector dimension changed to %d", *vectorDim)
	
	// Verify the change
	log.Println("Verifying change...")
	var newDim string
	err = db.QueryRowContext(ctx, `
		SELECT 
			format_type(atttypid, atttypmod) as type
		FROM pg_attribute
		WHERE attrelid = 'vectors'::regclass 
		AND attname = 'embedding'
	`).Scan(&newDim)
	
	if err != nil {
		log.Printf("Warning: Could not verify dimension: %v", err)
	} else {
		log.Printf("✓ Verified: embedding column type is now %s", newDim)
	}

	log.Println()
	log.Println("✓ Vector dimension alteration completed successfully")
	log.Println()
	log.Println("Next steps:")
	log.Printf("  1. Update EMBEDDING_DIMENSIONS=%d in your .env file", *vectorDim)
	log.Println("  2. Re-index your repositories with the new embedding model")
}

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
	var intValue int
	fmt.Sscanf(value, "%d", &intValue)
	return intValue
}
