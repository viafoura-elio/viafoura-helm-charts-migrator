package workers

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"helm-charts-migrator/v1/pkg/logger"
)

// Task represents a unit of work to be processed
type Task interface {
	// ID returns a unique identifier for the task
	ID() string

	// Execute performs the task
	Execute(ctx context.Context) error

	// Priority returns the task priority (lower numbers = higher priority)
	Priority() int
}

// Result represents the result of a task execution
type Result struct {
	TaskID   string
	Success  bool
	Error    error
	Duration time.Duration
	Data     interface{}
}

// WorkerPool manages a pool of workers for parallel task execution
type WorkerPool struct {
	workers         int
	taskQueue       chan Task
	resultQueue     chan Result
	errorQueue      chan error
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
	running         atomic.Bool
	shutdownStarted atomic.Bool
	tasksTotal      atomic.Int64
	tasksComplete   atomic.Int64
	tasksFailed     atomic.Int64
	metrics         *Metrics
	signalChan      chan os.Signal
	shutdownTimeout time.Duration
	log             *logger.NamedLogger
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workers:         workers,
		taskQueue:       make(chan Task, workers*2),
		resultQueue:     make(chan Result, workers*2),
		errorQueue:      make(chan error, workers),
		ctx:             ctx,
		cancel:          cancel,
		metrics:         NewMetrics(workers),
		signalChan:      make(chan os.Signal, 1),
		shutdownTimeout: 30 * time.Second,
		log:             logger.WithName("worker-pool"),
	}

	// Set up signal handling for graceful shutdown
	signal.Notify(pool.signalChan, syscall.SIGINT, syscall.SIGTERM)
	go pool.handleSignals()

	return pool
}

// Start starts the worker pool
func (p *WorkerPool) Start() error {
	if p.running.Load() {
		return fmt.Errorf("worker pool is already running")
	}

	p.running.Store(true)
	p.metrics.Start()

	// Start workers
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	p.log.InfoS("Worker pool started", "workers", p.workers)
	return nil
}

// Stop stops the worker pool gracefully
func (p *WorkerPool) Stop() error {
	return p.StopWithTimeout(p.shutdownTimeout)
}

// StopWithTimeout stops the worker pool with a custom timeout
func (p *WorkerPool) StopWithTimeout(timeout time.Duration) error {
	if !p.running.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	p.log.InfoS("Starting graceful shutdown", "timeout", timeout)
	p.shutdownStarted.Store(true)

	// Create a context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), timeout)
	defer shutdownCancel()

	// Signal shutdown
	p.running.Store(false)

	// Close task queue to signal workers to stop accepting new tasks
	close(p.taskQueue)

	// Wait for workers to finish current tasks
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.InfoS("All workers stopped gracefully")
	case <-shutdownCtx.Done():
		p.log.InfoS("Shutdown timeout reached, forcing stop")
		// Cancel the main context to force workers to stop
		p.cancel()
		// Wait a bit more for forced shutdown
		select {
		case <-done:
			p.log.InfoS("Workers stopped after forced shutdown")
		case <-time.After(5 * time.Second):
			p.log.InfoS("Some workers may still be running after forced shutdown")
		}
	}

	// Close result channels
	close(p.resultQueue)
	close(p.errorQueue)

	// Stop signal handling
	signal.Stop(p.signalChan)
	close(p.signalChan)

	// Stop metrics and log summary
	p.metrics.Stop()
	p.metrics.LogSummary()

	p.log.InfoS("Worker pool stopped",
		"total", p.tasksTotal.Load(),
		"completed", p.tasksComplete.Load(),
		"failed", p.tasksFailed.Load())

	return nil
}

// Submit submits a task to the worker pool
func (p *WorkerPool) Submit(task Task) error {
	if !p.running.Load() || p.shutdownStarted.Load() {
		return fmt.Errorf("worker pool is not running or shutting down")
	}

	select {
	case p.taskQueue <- task:
		p.tasksTotal.Add(1)
		p.metrics.RecordTaskStart()
		p.metrics.RecordQueueDepth(int32(len(p.taskQueue)))
		p.log.V(4).InfoS("Task submitted", "taskID", task.ID())
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		// Queue is full, try with timeout
		select {
		case p.taskQueue <- task:
			p.tasksTotal.Add(1)
			p.metrics.RecordTaskStart()
			p.metrics.RecordQueueDepth(int32(len(p.taskQueue)))
			p.log.V(4).InfoS("Task submitted after queue wait", "taskID", task.ID())
			return nil
		case <-time.After(5 * time.Second):
			return fmt.Errorf("task queue is full, failed to submit task %s", task.ID())
		case <-p.ctx.Done():
			return fmt.Errorf("worker pool is shutting down")
		}
	}
}

// SubmitBatch submits multiple tasks to the worker pool
func (p *WorkerPool) SubmitBatch(tasks []Task) error {
	if !p.running.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	for _, task := range tasks {
		if err := p.Submit(task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.ID(), err)
		}
	}

	p.log.V(3).InfoS("Batch submitted", "count", len(tasks))
	return nil
}

// Results returns the result channel
func (p *WorkerPool) Results() <-chan Result {
	return p.resultQueue
}

// Errors returns the error channel
func (p *WorkerPool) Errors() <-chan error {
	return p.errorQueue
}

// Wait waits for all submitted tasks to complete
func (p *WorkerPool) Wait() {
	// Wait for task queue to be empty
	for len(p.taskQueue) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all tasks to complete
	for p.tasksTotal.Load() > (p.tasksComplete.Load() + p.tasksFailed.Load()) {
		time.Sleep(10 * time.Millisecond)
	}
}

// WaitWithTimeout waits for all tasks to complete with a timeout
func (p *WorkerPool) WaitWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})

	go func() {
		p.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for tasks to complete")
	}
}

// Stats returns current pool statistics
func (p *WorkerPool) Stats() PoolStats {
	return PoolStats{
		Workers:        p.workers,
		Running:        p.running.Load(),
		TotalTasks:     p.tasksTotal.Load(),
		CompletedTasks: p.tasksComplete.Load(),
		FailedTasks:    p.tasksFailed.Load(),
		PendingTasks:   len(p.taskQueue),
	}
}

// worker is the worker goroutine
func (p *WorkerPool) worker(id int) {
	defer func() {
		p.metrics.RecordWorkerStop()
		p.wg.Done()

		// Recover from panics
		if r := recover(); r != nil {
			p.log.InfoS("Worker recovered from panic",
				"workerID", id,
				"panic", r)
		}
	}()

	p.log.V(4).InfoS("Worker started", "workerID", id)
	p.metrics.RecordWorkerStart()

	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				// Channel closed, shutdown
				p.log.V(4).InfoS("Worker received shutdown signal", "workerID", id)
				return
			}

			if task == nil {
				continue
			}

			p.processTask(id, task)

		case <-p.ctx.Done():
			// Context cancelled, shutdown immediately
			p.log.V(4).InfoS("Worker context cancelled", "workerID", id)
			return
		}
	}
}

// processTask processes a single task
func (p *WorkerPool) processTask(workerID int, task Task) {
	start := time.Now()
	taskID := task.ID()

	defer func() {
		// Handle panics in task execution
		if r := recover(); r != nil {
			duration := time.Since(start)
			err := fmt.Errorf("task %s panicked: %v", taskID, r)

			p.log.InfoS("Task panicked",
				"workerID", workerID,
				"taskID", taskID,
				"panic", r,
				"duration", duration)

			result := Result{
				TaskID:   taskID,
				Success:  false,
				Error:    err,
				Duration: duration,
			}

			// Try to send result
			select {
			case p.resultQueue <- result:
				// Result sent
			default:
				// Queue full
			}

			p.tasksFailed.Add(1)
			p.metrics.RecordTaskFailed(duration, err)
		}
	}()

	p.log.V(4).InfoS("Processing task",
		"workerID", workerID,
		"taskID", taskID,
		"priority", task.Priority())

	// Execute task with context
	err := task.Execute(p.ctx)
	duration := time.Since(start)

	result := Result{
		TaskID:   taskID,
		Success:  err == nil,
		Error:    err,
		Duration: duration,
	}

	// Send result (non-blocking)
	select {
	case p.resultQueue <- result:
		// Result sent successfully
	case <-p.ctx.Done():
		// Context cancelled
		return
	default:
		// Result queue full, log warning
		p.log.V(2).InfoS("Result queue full, dropping result",
			"taskID", taskID,
			"workerID", workerID)
	}

	// Update statistics and metrics
	if err != nil {
		p.tasksFailed.Add(1)
		p.metrics.RecordTaskFailed(duration, err)

		// Send error to error channel (non-blocking)
		select {
		case p.errorQueue <- fmt.Errorf("task %s failed: %w", taskID, err):
			// Error sent
		default:
			// Error channel full, just log it
			p.log.Error(err, "Task failed",
				"taskID", taskID,
				"workerID", workerID,
				"duration", duration)
		}
	} else {
		p.tasksComplete.Add(1)
		p.metrics.RecordTaskComplete(duration)
		p.log.V(4).InfoS("Task completed",
			"workerID", workerID,
			"taskID", taskID,
			"duration", duration)
	}

	// Update queue depth
	p.metrics.RecordQueueDepth(int32(len(p.taskQueue)))
}

// PoolStats contains worker pool statistics
type PoolStats struct {
	Workers        int
	Running        bool
	TotalTasks     int64
	CompletedTasks int64
	FailedTasks    int64
	PendingTasks   int
}

// ProcessWithPool processes tasks using a worker pool
func ProcessWithPool(ctx context.Context, tasks []Task, workers int) ([]Result, []error) {
	pool := NewWorkerPool(workers)

	// Start the pool
	if err := pool.Start(); err != nil {
		return nil, []error{err}
	}
	defer pool.Stop()

	// Submit all tasks
	if err := pool.SubmitBatch(tasks); err != nil {
		return nil, []error{err}
	}

	// Collect results
	var results []Result
	var errors []error

	// Start result collectors
	var wg sync.WaitGroup
	wg.Add(2)

	// Collect results
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			results = append(results, result)
		}
	}()

	// Collect errors
	go func() {
		defer wg.Done()
		for err := range pool.Errors() {
			errors = append(errors, err)
		}
	}()

	// Wait for all tasks to complete
	pool.Wait()

	// Stop the pool to close channels
	pool.Stop()

	// Wait for collectors to finish
	wg.Wait()

	return results, errors
}

// handleSignals handles OS signals for graceful shutdown
func (p *WorkerPool) handleSignals() {
	for sig := range p.signalChan {
		p.log.InfoS("Received signal, initiating graceful shutdown", "signal", sig)

		// Start graceful shutdown
		go func() {
			if err := p.Stop(); err != nil {
				p.log.Error(err, "Error during graceful shutdown")
			}
		}()

		// Only handle the first signal
		break
	}
}

// GetMetrics returns the metrics collector
func (p *WorkerPool) GetMetrics() *Metrics {
	return p.metrics
}

// GetMetricsSnapshot returns a current metrics snapshot
func (p *WorkerPool) GetMetricsSnapshot() MetricsSnapshot {
	return p.metrics.Snapshot()
}

// SetShutdownTimeout sets the shutdown timeout
func (p *WorkerPool) SetShutdownTimeout(timeout time.Duration) {
	p.shutdownTimeout = timeout
}

// IsRunning returns true if the pool is currently running
func (p *WorkerPool) IsRunning() bool {
	return p.running.Load()
}

// IsShuttingDown returns true if the pool is currently shutting down
func (p *WorkerPool) IsShuttingDown() bool {
	return p.shutdownStarted.Load()
}
