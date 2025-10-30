package indexer

import (
	"testing"
	"time"
)

func TestBatchOptimizer_AdaptiveSizing(t *testing.T) {
	config := &BatchOptimizerConfig{
		MinBatchSize:  10,
		MaxBatchSize:  1000,
		TargetLatency: 500 * time.Millisecond,
	}

	optimizer := NewBatchOptimizer(config)

	// Initial batch size should be minimum
	if size := optimizer.GetBatchSize(); size != config.MinBatchSize {
		t.Errorf("Expected initial batch size %d, got %d", config.MinBatchSize, size)
	}

	// Record low latency - should increase batch size
	optimizer.RecordLatency(100 * time.Millisecond)
	newSize := optimizer.GetBatchSize()
	if newSize <= config.MinBatchSize {
		t.Errorf("Expected batch size to increase, got %d", newSize)
	}

	// Record high latency - should decrease batch size
	optimizer.RecordLatency(2 * time.Second)
	reducedSize := optimizer.GetBatchSize()
	if reducedSize >= newSize {
		t.Errorf("Expected batch size to decrease from %d, got %d", newSize, reducedSize)
	}
}

func TestBatchOptimizer_BoundaryConditions(t *testing.T) {
	config := &BatchOptimizerConfig{
		MinBatchSize:  10,
		MaxBatchSize:  100,
		TargetLatency: 500 * time.Millisecond,
	}

	optimizer := NewBatchOptimizer(config)

	// Record very low latencies - should not exceed max
	for i := 0; i < 20; i++ {
		optimizer.RecordLatency(10 * time.Millisecond)
	}

	if size := optimizer.GetBatchSize(); size > config.MaxBatchSize {
		t.Errorf("Batch size %d exceeded maximum %d", size, config.MaxBatchSize)
	}

	// Record very high latencies - should not go below min
	for i := 0; i < 20; i++ {
		optimizer.RecordLatency(10 * time.Second)
	}

	if size := optimizer.GetBatchSize(); size < config.MinBatchSize {
		t.Errorf("Batch size %d below minimum %d", size, config.MinBatchSize)
	}
}

func TestBatchOptimizer_AverageLatency(t *testing.T) {
	optimizer := NewBatchOptimizer(DefaultBatchOptimizerConfig())

	latencies := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	for _, latency := range latencies {
		optimizer.RecordLatency(latency)
	}

	avgLatency := optimizer.GetAverageLatency()
	expectedAvg := 200 * time.Millisecond

	if avgLatency != expectedAvg {
		t.Errorf("Expected average latency %v, got %v", expectedAvg, avgLatency)
	}
}

func TestPreparedStatementCache(t *testing.T) {
	cache := NewPreparedStatementCache()

	// Test Set and Get
	key := "insert_symbol"
	query := "INSERT INTO symbols ..."
	cache.Set(key, query)

	stmt, ok := cache.Get(key)
	if !ok {
		t.Fatal("Expected to find cached statement")
	}

	if stmt.Query != query {
		t.Errorf("Expected query %s, got %s", query, stmt.Query)
	}

	if stmt.UseCount != 2 { // 1 from Set, 1 from Get
		t.Errorf("Expected use count 2, got %d", stmt.UseCount)
	}

	// Test Get non-existent
	_, ok = cache.Get("non_existent")
	if ok {
		t.Error("Expected not to find non-existent statement")
	}

	// Test Clear
	cache.Clear()
	_, ok = cache.Get(key)
	if ok {
		t.Error("Expected cache to be empty after Clear")
	}
}

func TestPreparedStatementCache_Stats(t *testing.T) {
	cache := NewPreparedStatementCache()

	// Add multiple statements
	cache.Set("stmt1", "SELECT ...")
	cache.Set("stmt2", "INSERT ...")
	cache.Set("stmt3", "UPDATE ...")

	// Use them
	cache.Get("stmt1")
	cache.Get("stmt1")
	cache.Get("stmt2")

	stats := cache.GetStats()

	if cacheSize := stats["cache_size"].(int); cacheSize != 3 {
		t.Errorf("Expected cache size 3, got %d", cacheSize)
	}

	totalUses := stats["total_uses"].(int64)
	if totalUses < 6 { // At least 3 from Set + 3 from Get
		t.Errorf("Expected at least 6 total uses, got %d", totalUses)
	}
}
