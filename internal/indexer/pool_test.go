package indexer

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_BasicExecution(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 4)

	var counter int32
	numTasks := 100

	// Submit tasks
	for i := 0; i < numTasks; i++ {
		err := pool.Submit(func(ctx context.Context) error {
			atomic.AddInt32(&counter, 1)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}

	// Wait for completion
	if err := pool.Wait(); err != nil {
		t.Fatalf("Pool wait failed: %v", err)
	}

	if counter != int32(numTasks) {
		t.Errorf("Expected %d tasks executed, got %d", numTasks, counter)
	}
}

func TestWorkerPool_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 2)

	expectedError := errors.New("task error")

	// Submit tasks with errors
	for i := 0; i < 5; i++ {
		pool.Submit(func(ctx context.Context) error {
			return expectedError
		})
	}

	// Wait should return error
	err := pool.Wait()
	if err == nil {
		t.Fatal("Expected error from pool.Wait()")
	}

	// Check error count
	if pool.ErrorCount() != 5 {
		t.Errorf("Expected 5 errors, got %d", pool.ErrorCount())
	}
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewWorkerPool(ctx, 2)

	var counter int32

	// Submit long-running tasks
	for i := 0; i < 10; i++ {
		pool.Submit(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				atomic.AddInt32(&counter, 1)
				return nil
			}
		})
	}

	// Cancel context after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for pool to finish
	pool.Wait()

	// Not all tasks should have completed
	if counter >= 10 {
		t.Errorf("Expected fewer than 10 tasks completed due to cancellation, got %d", counter)
	}
}

func TestWorkerPool_SubmitBatch(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 4)

	var counter int32
	tasks := make([]Task, 50)
	for i := range tasks {
		tasks[i] = func(ctx context.Context) error {
			atomic.AddInt32(&counter, 1)
			return nil
		}
	}

	// Submit batch
	if err := pool.SubmitBatch(tasks); err != nil {
		t.Fatalf("Failed to submit batch: %v", err)
	}

	// Wait for completion
	if err := pool.Wait(); err != nil {
		t.Fatalf("Pool wait failed: %v", err)
	}

	if counter != int32(len(tasks)) {
		t.Errorf("Expected %d tasks executed, got %d", len(tasks), counter)
	}
}

func TestWorkerPool_ConcurrentExecution(t *testing.T) {
	ctx := context.Background()
	workerCount := 4
	pool := NewWorkerPool(ctx, workerCount)

	var maxConcurrent int32
	var currentConcurrent int32

	numTasks := 100

	for i := 0; i < numTasks; i++ {
		pool.Submit(func(ctx context.Context) error {
			current := atomic.AddInt32(&currentConcurrent, 1)
			
			// Track max concurrent
			for {
				max := atomic.LoadInt32(&maxConcurrent)
				if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&currentConcurrent, -1)
			return nil
		})
	}

	pool.Wait()

	// Max concurrent should not exceed worker count
	if maxConcurrent > int32(workerCount) {
		t.Errorf("Max concurrent %d exceeded worker count %d", maxConcurrent, workerCount)
	}

	// Should have had some concurrency
	if maxConcurrent < 2 {
		t.Errorf("Expected some concurrency, got max concurrent %d", maxConcurrent)
	}
}

func TestWorkerPool_Close(t *testing.T) {
	ctx := context.Background()
	pool := NewWorkerPool(ctx, 2)

	// Submit some tasks
	for i := 0; i < 5; i++ {
		pool.Submit(func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	// Close pool
	if err := pool.Close(); err != nil {
		t.Fatalf("Failed to close pool: %v", err)
	}

	// Submitting after close should fail
	err := pool.Submit(func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error when submitting to closed pool")
	}
}
