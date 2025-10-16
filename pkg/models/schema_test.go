package models

import (
	"context"
	"testing"
	"time"
)

func TestSchemaManager_InitializeSchema(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	err = sm.InitializeSchema(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
}

func TestSchemaManager_EnsureExtensions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	err = sm.ensureExtensions(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure extensions: %v", err)
	}

	// Verify pgvector extension
	var vectorExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&vectorExists)
	if err != nil {
		t.Fatalf("Failed to check vector extension: %v", err)
	}
	if !vectorExists {
		t.Error("pgvector extension not found")
	}

	// Verify AGE extension
	var ageExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'age')").Scan(&ageExists)
	if err != nil {
		t.Fatalf("Failed to check age extension: %v", err)
	}
	if !ageExists {
		t.Error("AGE extension not found")
	}
}

func TestSchemaManager_EnsureAGEGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	// Ensure extensions first
	if err := sm.ensureExtensions(ctx); err != nil {
		t.Fatalf("Failed to ensure extensions: %v", err)
	}

	// Ensure graph
	err = sm.ensureAGEGraph(ctx)
	if err != nil {
		t.Fatalf("Failed to ensure AGE graph: %v", err)
	}

	// Verify graph exists
	var graphExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM ag_catalog.ag_graph WHERE name = 'code_graph')").Scan(&graphExists)
	if err != nil {
		t.Fatalf("Failed to check graph existence: %v", err)
	}
	if !graphExists {
		t.Error("code_graph not found")
	}
}

func TestSchemaManager_VerifyCoreTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	err = sm.verifyCoreTables(ctx)
	if err != nil {
		t.Fatalf("Failed to verify core tables: %v", err)
	}

	// Verify specific tables
	requiredTables := []string{
		"repositories",
		"files",
		"symbols",
		"ast_nodes",
		"edges",
		"vectors",
		"docstrings",
		"summaries",
	}

	for _, table := range requiredTables {
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`
		err := db.QueryRowContext(ctx, query, table).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Required table %s does not exist", table)
		}
	}
}

func TestSchemaManager_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	// Initialize schema first
	if err := sm.InitializeSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Run health check
	err = sm.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

func TestSchemaManager_GetDatabaseStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	stats, err := sm.GetDatabaseStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get database stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	// Stats should be non-negative
	if stats.RepositoryCount < 0 {
		t.Error("Repository count should be non-negative")
	}
	if stats.FileCount < 0 {
		t.Error("File count should be non-negative")
	}
	if stats.SymbolCount < 0 {
		t.Error("Symbol count should be non-negative")
	}
	if stats.EdgeCount < 0 {
		t.Error("Edge count should be non-negative")
	}
	if stats.VectorCount < 0 {
		t.Error("Vector count should be non-negative")
	}
	if stats.DatabaseSize == "" {
		t.Error("Database size should not be empty")
	}

	t.Logf("Database stats: %+v", stats)
}

func TestSchemaManager_GetSchemaVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	version, err := sm.GetSchemaVersion(ctx)
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}

	if version == "" {
		t.Error("Schema version should not be empty")
	}

	t.Logf("Schema version: %s", version)
}

func TestWaitForDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := WaitForDatabase(5, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to wait for database: %v", err)
	}
	defer db.Close()

	// Verify connection works
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

func TestSchemaManager_CreateVectorIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := NewDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	sm := NewSchemaManager(db)
	ctx := context.Background()

	// Initialize schema first
	if err := sm.InitializeSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create vector index
	err = sm.CreateVectorIndex(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to create vector index: %v", err)
	}

	// Verify index exists
	var indexExists bool
	query := `SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname = 'idx_vectors_embedding')`
	err = db.QueryRowContext(ctx, query).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check index existence: %v", err)
	}
	if !indexExists {
		t.Error("Vector index not found")
	}
}
