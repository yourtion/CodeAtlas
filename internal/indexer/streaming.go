package indexer

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// StreamProcessor handles streaming processing of large data sets with memory management
type StreamProcessor struct {
	maxMemoryMB     int64
	maxGoroutines   int
	currentMemoryMB int64
	activeGoroutines int32
	mu              sync.Mutex
	semaphore       chan struct{}
}

// StreamConfig contains configuration for stream processing
type StreamConfig struct {
	// Maximum memory usage in MB before applying backpressure
	MaxMemoryMB int64

	// Maximum number of concurrent goroutines
	MaxGoroutines int

	// Batch size for processing
	BatchSize int
}

// DefaultStreamConfig returns default streaming configuration
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		MaxMemoryMB:   512, // 512 MB default limit
		MaxGoroutines: runtime.NumCPU() * 2,
		BatchSize:     100,
	}
}

// NewStreamProcessor creates a new stream processor with memory limits
func NewStreamProcessor(config *StreamConfig) *StreamProcessor {
	if config == nil {
		config = DefaultStreamConfig()
	}

	return &StreamProcessor{
		maxMemoryMB:   config.MaxMemoryMB,
		maxGoroutines: config.MaxGoroutines,
		semaphore:     make(chan struct{}, config.MaxGoroutines),
	}
}

// StreamFiles processes files in a streaming fashion with backpressure
func (sp *StreamProcessor) StreamFiles(
	ctx context.Context,
	files []schema.File,
	processor func(context.Context, schema.File) error,
) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
		}

		// Apply backpressure if memory usage is high
		if err := sp.waitForMemory(ctx); err != nil {
			wg.Wait()
			return err
		}

		// Acquire semaphore to limit concurrent goroutines
		sp.semaphore <- struct{}{}
		atomic.AddInt32(&sp.activeGoroutines, 1)

		wg.Add(1)
		go func(f schema.File) {
			defer func() {
				<-sp.semaphore
				atomic.AddInt32(&sp.activeGoroutines, -1)
				wg.Done()
			}()

			// Estimate memory usage for this file
			memUsage := sp.estimateFileMemory(f)
			atomic.AddInt64(&sp.currentMemoryMB, memUsage)
			defer atomic.AddInt64(&sp.currentMemoryMB, -memUsage)

			if err := processor(ctx, f); err != nil {
				select {
				case errChan <- err:
				default:
					// Error channel full, skip
				}
			}
		}(file)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("stream processing failed with %d errors: %v", len(errors), errors[0])
	}

	return nil
}

// StreamASTNodes processes AST nodes in a streaming fashion to avoid loading entire tree into memory
func (sp *StreamProcessor) StreamASTNodes(
	ctx context.Context,
	nodes []schema.ASTNode,
	batchSize int,
	processor func(context.Context, []schema.ASTNode) error,
) error {
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(nodes); i += batchSize {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Apply backpressure
		if err := sp.waitForMemory(ctx); err != nil {
			return err
		}

		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batch := nodes[i:end]

		// Estimate memory for batch
		memUsage := sp.estimateNodesMemory(batch)
		atomic.AddInt64(&sp.currentMemoryMB, memUsage)

		// Process batch
		err := processor(ctx, batch)

		// Release memory
		atomic.AddInt64(&sp.currentMemoryMB, -memUsage)

		if err != nil {
			return fmt.Errorf("failed to process AST nodes batch %d: %w", i/batchSize, err)
		}
	}

	return nil
}

// StreamSymbols processes symbols in batches with memory management
func (sp *StreamProcessor) StreamSymbols(
	ctx context.Context,
	symbols []schema.Symbol,
	batchSize int,
	processor func(context.Context, []schema.Symbol) error,
) error {
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(symbols); i += batchSize {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Apply backpressure
		if err := sp.waitForMemory(ctx); err != nil {
			return err
		}

		end := i + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]

		// Estimate memory for batch
		memUsage := sp.estimateSymbolsMemory(batch)
		atomic.AddInt64(&sp.currentMemoryMB, memUsage)

		// Process batch
		err := processor(ctx, batch)

		// Release memory
		atomic.AddInt64(&sp.currentMemoryMB, -memUsage)

		if err != nil {
			return fmt.Errorf("failed to process symbols batch %d: %w", i/batchSize, err)
		}
	}

	return nil
}

// waitForMemory 阻塞直到内存占用低于阈值，或 context 被取消。
//
// 修复说明：历史实现存在两个反模式——
//  1. 用 runtime.Gosched() 让出 CPU 仅一个时间片，高内存时空转烧 CPU（忙轮询）；
//     改为 time.Sleep 退避等待，让出调度资源。
//  2. 主动调用 runtime.GC() 试图降内存；但 currentMemoryMB 是由
//     acquire/release 维护的逻辑计数，并非堆内存，GC 无法释放它，
//     反而引入 STW 暂停。已删除。
func (sp *StreamProcessor) waitForMemory(ctx context.Context) error {
	const maxBackoff = 200 * time.Millisecond
	backoff := 10 * time.Millisecond

	for {
		if atomic.LoadInt64(&sp.currentMemoryMB) < sp.maxMemoryMB {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// 指数退避（上限 maxBackoff），避免长时间高内存时过度频繁检查
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// estimateFileMemory estimates memory usage for a file in MB
func (sp *StreamProcessor) estimateFileMemory(file schema.File) int64 {
	// Rough estimation:
	// - File metadata: ~1 KB
	// - Symbols: ~1 KB per symbol
	// - AST nodes: ~500 bytes per node
	symbolsMem := int64(len(file.Symbols)) * 1024
	nodesMem := int64(len(file.Nodes)) * 512
	totalBytes := 1024 + symbolsMem + nodesMem

	// Convert to MB (round up)
	mb := totalBytes / (1024 * 1024)
	if mb == 0 {
		mb = 1
	}
	return mb
}

// estimateNodesMemory estimates memory usage for AST nodes in MB
func (sp *StreamProcessor) estimateNodesMemory(nodes []schema.ASTNode) int64 {
	// Rough estimation: ~500 bytes per node
	totalBytes := int64(len(nodes)) * 512

	// Add text content size
	for _, node := range nodes {
		totalBytes += int64(len(node.Text))
	}

	// Convert to MB (round up)
	mb := totalBytes / (1024 * 1024)
	if mb == 0 {
		mb = 1
	}
	return mb
}

// estimateSymbolsMemory estimates memory usage for symbols in MB
func (sp *StreamProcessor) estimateSymbolsMemory(symbols []schema.Symbol) int64 {
	// Rough estimation: ~1 KB per symbol
	totalBytes := int64(len(symbols)) * 1024

	// Add docstring and signature sizes
	for _, symbol := range symbols {
		totalBytes += int64(len(symbol.Docstring))
		totalBytes += int64(len(symbol.Signature))
		totalBytes += int64(len(symbol.SemanticSummary))
	}

	// Convert to MB (round up)
	mb := totalBytes / (1024 * 1024)
	if mb == 0 {
		mb = 1
	}
	return mb
}

// GetMemoryStats returns current memory statistics
func (sp *StreamProcessor) GetMemoryStats() MemoryStats {
	return MemoryStats{
		CurrentMemoryMB:  atomic.LoadInt64(&sp.currentMemoryMB),
		MaxMemoryMB:      sp.maxMemoryMB,
		ActiveGoroutines: atomic.LoadInt32(&sp.activeGoroutines),
		MaxGoroutines:    int32(sp.maxGoroutines),
	}
}

// MemoryStats contains memory usage statistics
type MemoryStats struct {
	CurrentMemoryMB  int64
	MaxMemoryMB      int64
	ActiveGoroutines int32
	MaxGoroutines    int32
}

// MemoryPressure returns the current memory pressure as a percentage (0-100)
func (ms MemoryStats) MemoryPressure() float64 {
	if ms.MaxMemoryMB == 0 {
		return 0
	}
	return float64(ms.CurrentMemoryMB) / float64(ms.MaxMemoryMB) * 100
}

// GoroutinePressure returns the current goroutine pressure as a percentage (0-100)
func (ms MemoryStats) GoroutinePressure() float64 {
	if ms.MaxGoroutines == 0 {
		return 0
	}
	return float64(ms.ActiveGoroutines) / float64(ms.MaxGoroutines) * 100
}
