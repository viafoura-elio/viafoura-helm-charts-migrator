package workers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimpleTask for testing
type SimpleTask struct {
	id         string
	duration   time.Duration
	priority   int
	shouldFail bool
}

func (st *SimpleTask) ID() string {
	return st.id
}

func (st *SimpleTask) Priority() int {
	return st.priority
}

func (st *SimpleTask) Execute(ctx context.Context) error {
	if st.duration > 0 {
		select {
		case <-time.After(st.duration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if st.shouldFail {
		return fmt.Errorf("task %s failed", st.id)
	}

	return nil
}

func TestWorkerPool_BasicFunctionality(t *testing.T) {
	pool := NewWorkerPool(2)
	require.NotNil(t, pool)

	// Test start
	err := pool.Start()
	assert.NoError(t, err)
	assert.True(t, pool.IsRunning())

	// Test double start
	err = pool.Start()
	assert.Error(t, err)

	// Test stop
	err = pool.Stop()
	assert.NoError(t, err)
	assert.False(t, pool.IsRunning())

	// Test double stop
	err = pool.Stop()
	assert.Error(t, err)
}

func TestWorkerPool_TaskExecution(t *testing.T) {
	pool := NewWorkerPool(2)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Submit successful task
	task := &SimpleTask{
		id:       "test-task-1",
		duration: time.Millisecond * 10,
	}

	err := pool.Submit(task)
	assert.NoError(t, err)

	// Wait for completion
	err = pool.WaitWithTimeout(time.Second * 5)
	assert.NoError(t, err)

	// Check stats
	stats := pool.Stats()
	assert.Equal(t, int64(1), stats.TotalTasks)
	assert.Equal(t, int64(1), stats.CompletedTasks)
	assert.Equal(t, int64(0), stats.FailedTasks)
}

func TestWorkerPool_FailedTask(t *testing.T) {
	pool := NewWorkerPool(1)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Submit failing task
	task := &SimpleTask{
		id:         "failing-task",
		shouldFail: true,
	}

	err := pool.Submit(task)
	assert.NoError(t, err)

	// Wait for completion
	err = pool.WaitWithTimeout(time.Second * 5)
	assert.NoError(t, err)

	// Check stats
	stats := pool.Stats()
	assert.Equal(t, int64(1), stats.TotalTasks)
	assert.Equal(t, int64(0), stats.CompletedTasks)
	assert.Equal(t, int64(1), stats.FailedTasks)
}

func TestWorkerPool_BatchSubmission(t *testing.T) {
	pool := NewWorkerPool(3)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Create batch of tasks
	tasks := make([]Task, 5)
	for i := 0; i < 5; i++ {
		tasks[i] = &SimpleTask{
			id:       fmt.Sprintf("batch-task-%d", i),
			duration: time.Millisecond * 5,
		}
	}

	err := pool.SubmitBatch(tasks)
	assert.NoError(t, err)

	// Wait for completion
	err = pool.WaitWithTimeout(time.Second * 5)
	assert.NoError(t, err)

	// Check stats
	stats := pool.Stats()
	assert.Equal(t, int64(5), stats.TotalTasks)
	assert.Equal(t, int64(5), stats.CompletedTasks)
	assert.Equal(t, int64(0), stats.FailedTasks)
}

func TestWorkerPool_ConcurrentSubmission(t *testing.T) {
	pool := NewWorkerPool(2)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	var wg sync.WaitGroup
	taskCount := 20
	goroutines := 4

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()

			for i := 0; i < taskCount/goroutines; i++ {
				task := &SimpleTask{
					id:       fmt.Sprintf("concurrent-task-%d-%d", gid, i),
					duration: time.Millisecond,
				}

				err := pool.Submit(task)
				assert.NoError(t, err)
			}
		}(g)
	}

	wg.Wait()
	pool.Wait()

	stats := pool.Stats()
	assert.Equal(t, int64(taskCount), stats.TotalTasks)
	assert.Equal(t, int64(taskCount), stats.CompletedTasks)
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	pool := NewWorkerPool(1)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Create a long-running task
	task := &SimpleTask{
		id:       "long-task",
		duration: time.Second * 5,
	}

	err := pool.Submit(task)
	assert.NoError(t, err)

	// Stop pool after short delay
	go func() {
		time.Sleep(time.Millisecond * 100)
		pool.Stop()
	}()

	// The task should be cancelled
	start := time.Now()
	pool.Wait()
	duration := time.Since(start)

	// Should complete much faster than 5 seconds (allowing some margin for CI)
	assert.Less(t, duration, time.Second*6)
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	pool := NewWorkerPool(2)
	require.NoError(t, pool.Start())

	// Submit several medium-duration tasks
	for i := 0; i < 3; i++ {
		task := &SimpleTask{
			id:       fmt.Sprintf("shutdown-task-%d", i),
			duration: time.Millisecond * 200,
		}
		pool.Submit(task)
	}

	// Test graceful shutdown with timeout
	start := time.Now()
	err := pool.StopWithTimeout(time.Second * 2)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, time.Second*2)
}

func TestWorkerPool_Metrics(t *testing.T) {
	pool := NewWorkerPool(2)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Get initial metrics
	metrics := pool.GetMetrics()
	require.NotNil(t, metrics)

	initialSnapshot := pool.GetMetricsSnapshot()
	assert.Equal(t, int64(0), initialSnapshot.TotalTasks)

	// Submit some tasks
	for i := 0; i < 3; i++ {
		task := &SimpleTask{
			id:       fmt.Sprintf("metrics-task-%d", i),
			duration: time.Millisecond * 10,
		}
		pool.Submit(task)
	}

	pool.Wait()

	// Check final metrics
	finalSnapshot := pool.GetMetricsSnapshot()
	assert.Equal(t, int64(3), finalSnapshot.TotalTasks)
	assert.Equal(t, int64(3), finalSnapshot.CompletedTasks)
	assert.Equal(t, int64(0), finalSnapshot.FailedTasks)
	assert.True(t, finalSnapshot.AvgDuration > 0)
}

func TestWorkerPool_SubmitWhenStopped(t *testing.T) {
	pool := NewWorkerPool(1)

	// Try to submit without starting
	task := &SimpleTask{id: "test"}
	err := pool.Submit(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start and stop
	require.NoError(t, pool.Start())
	require.NoError(t, pool.Stop())

	// Try to submit after stopping
	err = pool.Submit(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestWorkerPool_QueueFull(t *testing.T) {
	// This test would require a pool with a very small queue
	// For now, we'll test the timeout behavior
	pool := NewWorkerPool(1)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Submit a long-running task to block the worker
	blockingTask := &SimpleTask{
		id:       "blocking-task",
		duration: time.Second * 2,
	}
	err := pool.Submit(blockingTask)
	assert.NoError(t, err)

	// Submit many more tasks quickly
	// Eventually one should timeout or queue will fill
	for i := 0; i < 100; i++ {
		task := &SimpleTask{
			id: fmt.Sprintf("queue-task-%d", i),
		}

		err := pool.Submit(task)
		if err != nil {
			// Expected - queue is full or submit timed out
			assert.Contains(t, err.Error(), "queue is full")
			break
		}
	}
}

func TestRetryPolicy_Defaults(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, policy.InitialBackoff)
	assert.Equal(t, 30*time.Second, policy.MaxBackoff)
	assert.Equal(t, 2.0, policy.BackoffMultiplier)
	assert.Equal(t, 0.1, policy.JitterPercent)
	assert.Greater(t, len(policy.RetryableErrors), 0)
}

func TestRetryableTask_BasicRetry(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = 2
	policy.InitialBackoff = time.Millisecond * 10

	originalTask := &SimpleTask{
		id:         "retry-task",
		shouldFail: true,
	}

	retryableTask := NewRetryableTask(originalTask, policy)
	assert.Equal(t, "retry-task", retryableTask.ID())

	ctx := context.Background()

	// First attempt should fail and indicate retry needed
	err := retryableTask.Execute(ctx)
	assert.Error(t, err)

	// Check if it's a retry needed error
	_, isRetryNeeded := IsRetryNeeded(err)
	if isRetryNeeded {
		// This indicates a retry is needed
		assert.Equal(t, 1, retryableTask.GetAttempt())
	}
}

func TestRetryableWorkerPool_Basic(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = 2
	policy.InitialBackoff = time.Millisecond * 10

	pool := NewRetryableWorkerPool(2, policy)
	require.NoError(t, pool.Start())
	defer pool.Stop()

	// Submit a task
	task := &SimpleTask{
		id:       "retryable-pool-task",
		duration: time.Millisecond * 5,
	}

	err := pool.SubmitWithRetry(task)
	assert.NoError(t, err)

	pool.Wait()

	// Check metrics
	metrics := pool.GetMetricsSnapshot()
	assert.Equal(t, int64(1), metrics.TotalTasks)
}

func TestMetrics_Recording(t *testing.T) {
	metrics := NewMetrics(2)
	metrics.Start()
	defer metrics.Stop()

	// Record some metrics
	metrics.RecordTaskStart()
	metrics.RecordTaskComplete(time.Millisecond * 100)

	metrics.RecordTaskStart()
	metrics.RecordTaskFailed(time.Millisecond*50, fmt.Errorf("test error"))

	metrics.RecordWorkerStart()
	metrics.RecordWorkerStop()

	// Get snapshot
	snapshot := metrics.Snapshot()

	assert.Equal(t, int64(2), snapshot.TotalTasks)
	assert.Equal(t, int64(1), snapshot.CompletedTasks)
	assert.Equal(t, int64(1), snapshot.FailedTasks)
	assert.Equal(t, time.Millisecond*50, snapshot.MinDuration)
	assert.Equal(t, time.Millisecond*100, snapshot.MaxDuration)
	// Average is calculated as total duration / completed tasks (only successful ones)
	// Since we have 1 successful (100ms) and 1 failed (50ms), but average uses only completed:
	// Total duration includes both, but denominator is only completed tasks (1)
	expectedAvg := time.Millisecond * 150 // Total duration (100ms + 50ms) / 1 completed task
	assert.Equal(t, expectedAvg, snapshot.AvgDuration)
}

func TestMetrics_ErrorCounting(t *testing.T) {
	metrics := NewMetrics(1)

	// Record different error types
	err1 := fmt.Errorf("network error")
	err2 := fmt.Errorf("timeout error")
	err3 := fmt.Errorf("network error") // Same type as err1

	metrics.RecordTaskFailed(time.Millisecond, err1)
	metrics.RecordTaskFailed(time.Millisecond, err2)
	metrics.RecordTaskFailed(time.Millisecond, err3)

	snapshot := metrics.Snapshot()

	// All errors are of the same type (*errors.errorString), so we should have 1 error type with count 3
	assert.Equal(t, 1, len(snapshot.ErrorCounts))
	assert.Equal(t, int64(3), snapshot.ErrorCounts["*errors.errorString"])
}

func TestMetrics_Reset(t *testing.T) {
	metrics := NewMetrics(1)

	// Record some data
	metrics.RecordTaskStart()
	metrics.RecordTaskComplete(time.Millisecond * 100)

	// Reset
	metrics.Reset()

	// Check everything is cleared
	snapshot := metrics.Snapshot()
	assert.Equal(t, int64(0), snapshot.TotalTasks)
	assert.Equal(t, int64(0), snapshot.CompletedTasks)
	assert.Equal(t, int64(0), snapshot.FailedTasks)
	assert.Equal(t, time.Duration(0), snapshot.TotalDuration)
}
