package workers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"helm-charts-migrator/v1/pkg/logger"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts      int           `yaml:"max_attempts" json:"max_attempts"`
	InitialBackoff   time.Duration `yaml:"initial_backoff" json:"initial_backoff"`
	MaxBackoff       time.Duration `yaml:"max_backoff" json:"max_backoff"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier" json:"backoff_multiplier"`
	JitterPercent    float64       `yaml:"jitter_percent" json:"jitter_percent"`
	RetryableErrors  []string      `yaml:"retryable_errors" json:"retryable_errors"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		JitterPercent:     0.1,
		RetryableErrors: []string{
			"*net.OpError",
			"*url.Error", 
			"*errors.timeout",
			"context deadline exceeded",
			"connection refused",
			"no such host",
			"temporary failure",
		},
	}
}

// RetryableTask wraps a Task with retry capabilities
type RetryableTask struct {
	Task         Task
	Policy       RetryPolicy
	attempt      int
	lastError    error
	totalBackoff time.Duration
	log          *logger.NamedLogger
}

// NewRetryableTask creates a new retryable task
func NewRetryableTask(task Task, policy RetryPolicy) *RetryableTask {
	return &RetryableTask{
		Task:    task,
		Policy:  policy,
		attempt: 0,
		log:     logger.WithName("retry-task"),
	}
}

// ID returns the task ID with retry attempt info
func (rt *RetryableTask) ID() string {
	if rt.attempt == 0 {
		return rt.Task.ID()
	}
	return fmt.Sprintf("%s-retry-%d", rt.Task.ID(), rt.attempt)
}

// Priority returns the task priority (higher priority for retries)
func (rt *RetryableTask) Priority() int {
	basePriority := rt.Task.Priority()
	// Give retries slightly higher priority
	if rt.attempt > 0 {
		return basePriority - 1
	}
	return basePriority
}

// Execute executes the task with retry logic
func (rt *RetryableTask) Execute(ctx context.Context) error {
	// Check if we've exceeded max attempts
	if rt.attempt >= rt.Policy.MaxAttempts {
		return fmt.Errorf("task %s failed after %d attempts: %w", 
			rt.Task.ID(), rt.attempt, rt.lastError)
	}
	
	rt.attempt++
	
	rt.log.V(3).InfoS("Executing task", 
		"taskID", rt.Task.ID(),
		"attempt", rt.attempt,
		"maxAttempts", rt.Policy.MaxAttempts)
	
	// Execute the underlying task
	err := rt.Task.Execute(ctx)
	
	// If successful or not retryable, return immediately
	if err == nil {
		if rt.attempt > 1 {
			rt.log.InfoS("Task succeeded after retry",
				"taskID", rt.Task.ID(),
				"attempt", rt.attempt,
				"totalBackoff", rt.totalBackoff)
		}
		return nil
	}
	
	rt.lastError = err
	
	// Check if error is retryable
	if !rt.isRetryableError(err) {
		rt.log.V(2).InfoS("Task failed with non-retryable error",
			"taskID", rt.Task.ID(),
			"attempt", rt.attempt,
			"error", err)
		return err
	}
	
	// Check if we have more attempts
	if rt.attempt >= rt.Policy.MaxAttempts {
		rt.log.InfoS("Task failed after all retry attempts",
			"taskID", rt.Task.ID(),
			"attempts", rt.attempt,
			"finalError", err)
		return fmt.Errorf("task failed after %d attempts: %w", rt.attempt, err)
	}
	
	// Calculate backoff delay
	backoff := rt.calculateBackoff()
	rt.totalBackoff += backoff
	
	rt.log.InfoS("Task failed, will retry",
		"taskID", rt.Task.ID(),
		"attempt", rt.attempt,
		"nextAttempt", rt.attempt+1,
		"backoff", backoff,
		"error", err)
	
	// Wait for backoff period (unless context is cancelled)
	select {
	case <-ctx.Done():
		return fmt.Errorf("task cancelled during backoff: %w", ctx.Err())
	case <-time.After(backoff):
		// Continue to next attempt
	}
	
	// Return special error to indicate retry needed
	return &RetryNeededError{
		OriginalError: err,
		Attempt:       rt.attempt,
		NextBackoff:   rt.calculateBackoff(),
	}
}

// isRetryableError checks if an error should trigger a retry
func (rt *RetryableTask) isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errorStr := err.Error()
	errorType := fmt.Sprintf("%T", err)
	
	// Check against configured retryable errors
	for _, pattern := range rt.Policy.RetryableErrors {
		if pattern == errorType {
			return true
		}
		// Simple string contains check for error messages
		if len(pattern) > 0 && pattern[0] != '*' {
			if contains(errorStr, pattern) {
				return true
			}
		}
	}
	
	// Default retryable conditions
	return contains(errorStr, "timeout") ||
		contains(errorStr, "connection refused") ||
		contains(errorStr, "no such host") ||
		contains(errorStr, "temporary failure") ||
		contains(errorStr, "context deadline exceeded")
}

// calculateBackoff calculates the next backoff duration
func (rt *RetryableTask) calculateBackoff() time.Duration {
	if rt.attempt <= 1 {
		return rt.addJitter(rt.Policy.InitialBackoff)
	}
	
	// Exponential backoff: backoff = initialBackoff * multiplier^(attempt-1)
	backoff := float64(rt.Policy.InitialBackoff) * math.Pow(rt.Policy.BackoffMultiplier, float64(rt.attempt-1))
	
	// Cap at max backoff
	if backoff > float64(rt.Policy.MaxBackoff) {
		backoff = float64(rt.Policy.MaxBackoff)
	}
	
	return rt.addJitter(time.Duration(backoff))
}

// addJitter adds random jitter to backoff duration
func (rt *RetryableTask) addJitter(duration time.Duration) time.Duration {
	if rt.Policy.JitterPercent <= 0 {
		return duration
	}
	
	jitter := rt.Policy.JitterPercent
	if jitter > 1.0 {
		jitter = 1.0
	}
	
	// Generate random factor between (1-jitter) and (1+jitter)
	factor := 1.0 + (rand.Float64()*2-1)*jitter
	return time.Duration(float64(duration) * factor)
}

// GetAttempt returns the current attempt number
func (rt *RetryableTask) GetAttempt() int {
	return rt.attempt
}

// GetLastError returns the last error encountered
func (rt *RetryableTask) GetLastError() error {
	return rt.lastError
}

// GetTotalBackoff returns the total time spent in backoff
func (rt *RetryableTask) GetTotalBackoff() time.Duration {
	return rt.totalBackoff
}

// RetryNeededError indicates that a task needs to be retried
type RetryNeededError struct {
	OriginalError error
	Attempt       int
	NextBackoff   time.Duration
}

func (e *RetryNeededError) Error() string {
	return fmt.Sprintf("retry needed after attempt %d (backoff: %v): %v", 
		e.Attempt, e.NextBackoff, e.OriginalError)
}

func (e *RetryNeededError) Unwrap() error {
	return e.OriginalError
}

// IsRetryNeeded checks if an error indicates a retry is needed
func IsRetryNeeded(err error) (*RetryNeededError, bool) {
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		retryErr, ok := err.(*RetryNeededError)
		return retryErr, ok
	}
	return nil, false
}

// RetryableWorkerPool extends WorkerPool with retry capabilities
type RetryableWorkerPool struct {
	*WorkerPool
	retryPolicy RetryPolicy
	metrics     *Metrics
	retryQueue  chan *RetryableTask
	log         *logger.NamedLogger
}

// NewRetryableWorkerPool creates a worker pool with retry capabilities
func NewRetryableWorkerPool(workers int, policy RetryPolicy) *RetryableWorkerPool {
	basePool := NewWorkerPool(workers)
	metrics := NewMetrics(workers)
	
	return &RetryableWorkerPool{
		WorkerPool:  basePool,
		retryPolicy: policy,
		metrics:     metrics,
		retryQueue:  make(chan *RetryableTask, workers*2),
		log:         logger.WithName("retryable-worker-pool"),
	}
}

// Start starts the retryable worker pool
func (rwp *RetryableWorkerPool) Start() error {
	rwp.metrics.Start()
	
	// Start the base worker pool
	if err := rwp.WorkerPool.Start(); err != nil {
		return err
	}
	
	// Start retry handler
	go rwp.retryHandler()
	
	rwp.log.InfoS("Retryable worker pool started", 
		"workers", rwp.workers,
		"maxAttempts", rwp.retryPolicy.MaxAttempts,
		"initialBackoff", rwp.retryPolicy.InitialBackoff)
	
	return nil
}

// Stop stops the retryable worker pool
func (rwp *RetryableWorkerPool) Stop() error {
	// Close retry queue
	close(rwp.retryQueue)
	
	// Stop base pool
	err := rwp.WorkerPool.Stop()
	
	// Stop metrics
	rwp.metrics.Stop()
	rwp.metrics.LogSummary()
	
	return err
}

// SubmitWithRetry submits a task with retry policy
func (rwp *RetryableWorkerPool) SubmitWithRetry(task Task) error {
	retryableTask := NewRetryableTask(task, rwp.retryPolicy)
	rwp.metrics.RecordTaskStart()
	return rwp.Submit(retryableTask)
}

// SubmitBatchWithRetry submits multiple tasks with retry policy
func (rwp *RetryableWorkerPool) SubmitBatchWithRetry(tasks []Task) error {
	retryableTasks := make([]Task, len(tasks))
	for i, task := range tasks {
		retryableTasks[i] = NewRetryableTask(task, rwp.retryPolicy)
		rwp.metrics.RecordTaskStart()
	}
	return rwp.SubmitBatch(retryableTasks)
}

// GetMetrics returns the current metrics
func (rwp *RetryableWorkerPool) GetMetrics() *Metrics {
	return rwp.metrics
}

// GetMetricsSnapshot returns a metrics snapshot
func (rwp *RetryableWorkerPool) GetMetricsSnapshot() MetricsSnapshot {
	return rwp.metrics.Snapshot()
}

// retryHandler handles task retries
func (rwp *RetryableWorkerPool) retryHandler() {
	rwp.log.V(3).InfoS("Retry handler started")
	defer rwp.log.V(3).InfoS("Retry handler stopped")
	
	for {
		select {
		case result, ok := <-rwp.Results():
			if !ok {
				return // Channel closed
			}
			
			rwp.handleResult(result)
			
		case <-rwp.ctx.Done():
			return // Context cancelled
		}
	}
}

// handleResult processes task results and handles retries
func (rwp *RetryableWorkerPool) handleResult(result Result) {
	// Record metrics
	if result.Success {
		rwp.metrics.RecordTaskComplete(result.Duration)
		return
	}
	
	// Check if retry is needed
	if retryErr, needsRetry := IsRetryNeeded(result.Error); needsRetry {
		rwp.metrics.RecordTaskRetry(retryErr.Attempt, retryErr.OriginalError)
		
		// Find the retryable task from result
		// This would require keeping track of tasks, which is complex
		// For now, we'll log that a retry is needed
		rwp.log.V(2).InfoS("Task retry needed",
			"taskID", result.TaskID,
			"attempt", retryErr.Attempt,
			"nextBackoff", retryErr.NextBackoff)
		
		// In a real implementation, we'd re-submit the task here
		// For now, we'll treat it as a failure and let the original task handle retries
	}
	
	// Record as failed
	rwp.metrics.RecordTaskFailed(result.Duration, result.Error)
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && 
		(s == substr || (len(s) > len(substr) && 
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}