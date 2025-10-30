package indexer

import (
	"context"
	"fmt"
	"sync"
)

// WorkerPool manages a pool of workers for parallel processing
type WorkerPool struct {
	workerCount int
	taskQueue   chan Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	errors      []error
	errorsMu    sync.Mutex
	closed      bool
	closedMu    sync.RWMutex
}

// Task represents a unit of work to be processed
type Task func(context.Context) error

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(ctx context.Context, workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}

	poolCtx, cancel := context.WithCancel(ctx)

	pool := &WorkerPool{
		workerCount: workerCount,
		taskQueue:   make(chan Task, workerCount*2), // Buffer to reduce blocking
		ctx:         poolCtx,
		cancel:      cancel,
		errors:      make([]error, 0),
	}

	// Start workers
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	return pool
}

// worker processes tasks from the queue
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}

			// Execute task
			if err := task(wp.ctx); err != nil {
				wp.errorsMu.Lock()
				wp.errors = append(wp.errors, fmt.Errorf("worker %d: %w", id, err))
				wp.errorsMu.Unlock()
			}
		}
	}
}

// Submit adds a task to the worker pool
func (wp *WorkerPool) Submit(task Task) error {
	wp.closedMu.RLock()
	if wp.closed {
		wp.closedMu.RUnlock()
		return fmt.Errorf("worker pool is closed")
	}
	wp.closedMu.RUnlock()

	select {
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	case wp.taskQueue <- task:
		return nil
	}
}

// SubmitBatch adds multiple tasks to the worker pool
func (wp *WorkerPool) SubmitBatch(tasks []Task) error {
	for _, task := range tasks {
		if err := wp.Submit(task); err != nil {
			return err
		}
	}
	return nil
}

// Wait waits for all tasks to complete and closes the pool
func (wp *WorkerPool) Wait() error {
	wp.closedMu.Lock()
	if !wp.closed {
		wp.closed = true
		close(wp.taskQueue)
	}
	wp.closedMu.Unlock()

	wp.wg.Wait()

	wp.errorsMu.Lock()
	defer wp.errorsMu.Unlock()

	if len(wp.errors) > 0 {
		return fmt.Errorf("worker pool completed with %d errors: %v", len(wp.errors), wp.errors[0])
	}

	return nil
}

// Close cancels all workers and waits for them to finish
func (wp *WorkerPool) Close() error {
	wp.cancel()
	return wp.Wait()
}

// Errors returns all errors that occurred during task processing
func (wp *WorkerPool) Errors() []error {
	wp.errorsMu.Lock()
	defer wp.errorsMu.Unlock()

	errorsCopy := make([]error, len(wp.errors))
	copy(errorsCopy, wp.errors)
	return errorsCopy
}

// HasErrors returns true if any errors occurred
func (wp *WorkerPool) HasErrors() bool {
	wp.errorsMu.Lock()
	defer wp.errorsMu.Unlock()
	return len(wp.errors) > 0
}

// ErrorCount returns the number of errors that occurred
func (wp *WorkerPool) ErrorCount() int {
	wp.errorsMu.Lock()
	defer wp.errorsMu.Unlock()
	return len(wp.errors)
}
