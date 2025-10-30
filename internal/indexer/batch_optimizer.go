package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchOptimizer optimizes batch processing with adaptive batch sizing
type BatchOptimizer struct {
	minBatchSize     int
	maxBatchSize     int
	currentBatchSize int
	targetLatency    time.Duration
	recentLatencies  []time.Duration
	mu               sync.Mutex
}

// BatchOptimizerConfig contains configuration for batch optimization
type BatchOptimizerConfig struct {
	MinBatchSize  int
	MaxBatchSize  int
	TargetLatency time.Duration
}

// DefaultBatchOptimizerConfig returns default batch optimizer configuration
func DefaultBatchOptimizerConfig() *BatchOptimizerConfig {
	return &BatchOptimizerConfig{
		MinBatchSize:  10,
		MaxBatchSize:  1000,
		TargetLatency: 500 * time.Millisecond,
	}
}

// NewBatchOptimizer creates a new batch optimizer
func NewBatchOptimizer(config *BatchOptimizerConfig) *BatchOptimizer {
	if config == nil {
		config = DefaultBatchOptimizerConfig()
	}

	return &BatchOptimizer{
		minBatchSize:     config.MinBatchSize,
		maxBatchSize:     config.MaxBatchSize,
		currentBatchSize: config.MinBatchSize,
		targetLatency:    config.TargetLatency,
		recentLatencies:  make([]time.Duration, 0, 10),
	}
}

// GetBatchSize returns the current optimal batch size
func (bo *BatchOptimizer) GetBatchSize() int {
	bo.mu.Lock()
	defer bo.mu.Unlock()
	return bo.currentBatchSize
}

// RecordLatency records the latency of a batch operation and adjusts batch size
func (bo *BatchOptimizer) RecordLatency(latency time.Duration) {
	bo.mu.Lock()
	defer bo.mu.Unlock()

	// Add to recent latencies
	bo.recentLatencies = append(bo.recentLatencies, latency)
	if len(bo.recentLatencies) > 10 {
		bo.recentLatencies = bo.recentLatencies[1:]
	}

	// Adjust batch size based on latency
	if latency > bo.targetLatency*2 {
		// Latency too high, reduce batch size
		bo.currentBatchSize = max(bo.minBatchSize, bo.currentBatchSize*8/10)
	} else if latency < bo.targetLatency/2 {
		// Latency low, increase batch size
		bo.currentBatchSize = min(bo.maxBatchSize, bo.currentBatchSize*12/10)
	}
}

// GetAverageLatency returns the average latency of recent batches
func (bo *BatchOptimizer) GetAverageLatency() time.Duration {
	bo.mu.Lock()
	defer bo.mu.Unlock()

	if len(bo.recentLatencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range bo.recentLatencies {
		total += latency
	}
	return total / time.Duration(len(bo.recentLatencies))
}

// OptimizedBatchProcessor processes items in optimized batches
type OptimizedBatchProcessor struct {
	optimizer *BatchOptimizer
	processor func(context.Context, []interface{}) error
}

// NewOptimizedBatchProcessor creates a new optimized batch processor
func NewOptimizedBatchProcessor(
	config *BatchOptimizerConfig,
	processor func(context.Context, []interface{}) error,
) *OptimizedBatchProcessor {
	return &OptimizedBatchProcessor{
		optimizer: NewBatchOptimizer(config),
		processor: processor,
	}
}

// Process processes items in optimized batches
func (obp *OptimizedBatchProcessor) Process(ctx context.Context, items []interface{}) error {
	if len(items) == 0 {
		return nil
	}

	for i := 0; i < len(items); {
		batchSize := obp.optimizer.GetBatchSize()
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		startTime := time.Now()

		err := obp.processor(ctx, batch)
		latency := time.Since(startTime)

		obp.optimizer.RecordLatency(latency)

		if err != nil {
			return fmt.Errorf("failed to process batch at index %d: %w", i, err)
		}

		i = end
	}

	return nil
}

// GetStats returns statistics about the batch processor
func (obp *OptimizedBatchProcessor) GetStats() BatchStats {
	return BatchStats{
		CurrentBatchSize: obp.optimizer.GetBatchSize(),
		AverageLatency:   obp.optimizer.GetAverageLatency(),
	}
}

// BatchStats contains statistics about batch processing
type BatchStats struct {
	CurrentBatchSize int
	AverageLatency   time.Duration
}

// PreparedStatementCache caches prepared statements for reuse
type PreparedStatementCache struct {
	cache map[string]*CachedStatement
	mu    sync.RWMutex
}

// CachedStatement represents a cached prepared statement
type CachedStatement struct {
	Query     string
	UseCount  int64
	CreatedAt time.Time
	LastUsed  time.Time
}

// NewPreparedStatementCache creates a new prepared statement cache
func NewPreparedStatementCache() *PreparedStatementCache {
	return &PreparedStatementCache{
		cache: make(map[string]*CachedStatement),
	}
}

// Get retrieves a cached statement
func (psc *PreparedStatementCache) Get(key string) (*CachedStatement, bool) {
	psc.mu.RLock()
	defer psc.mu.RUnlock()

	stmt, ok := psc.cache[key]
	if ok {
		stmt.UseCount++
		stmt.LastUsed = time.Now()
	}
	return stmt, ok
}

// Set stores a statement in the cache
func (psc *PreparedStatementCache) Set(key string, query string) {
	psc.mu.Lock()
	defer psc.mu.Unlock()

	psc.cache[key] = &CachedStatement{
		Query:     query,
		UseCount:  1,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
}

// GetStats returns cache statistics
func (psc *PreparedStatementCache) GetStats() map[string]interface{} {
	psc.mu.RLock()
	defer psc.mu.RUnlock()

	totalUses := int64(0)
	for _, stmt := range psc.cache {
		totalUses += stmt.UseCount
	}

	return map[string]interface{}{
		"cache_size":  len(psc.cache),
		"total_uses":  totalUses,
		"avg_uses":    float64(totalUses) / float64(max(1, len(psc.cache))),
	}
}

// Clear removes all cached statements
func (psc *PreparedStatementCache) Clear() {
	psc.mu.Lock()
	defer psc.mu.Unlock()
	psc.cache = make(map[string]*CachedStatement)
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
