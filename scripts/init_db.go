// CodeAtlas 数据库初始化工具。
//
// 连接数据库并执行 goose 迁移（真源：pkg/models/migrations/*.sql），
// 随后进行健康检查并可选地打印统计信息。
//
// 用法：
//
//	go run scripts/init_db.go [-stats]
//
// 历史版本曾提供 -create-vector-index 标志手动创建 IVFFlat 索引；
// 向量索引现已由迁移 20260101000002_vector_hnsw.sql 以 HNSW 方式创建，
// 该标志已移除。
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

	// Initialize schema (executes goose migrations)
	log.Println("Initializing database schema (running goose migrations)...")
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
