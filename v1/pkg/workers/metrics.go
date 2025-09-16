package workers

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"helm-charts-migrator/v1/pkg/logger"
)

// Metrics tracks worker pool performance metrics
type Metrics struct {
	// Task metrics
	totalTasks     atomic.Int64
	completedTasks atomic.Int64
	failedTasks    atomic.Int64
	retryTasks     atomic.Int64

	// Timing metrics
	totalDuration atomic.Int64 // nanoseconds
	minDuration   atomic.Int64 // nanoseconds
	maxDuration   atomic.Int64 // nanoseconds

	// Worker metrics
	activeWorkers atomic.Int32
	peakWorkers   atomic.Int32
	totalWorkers  int32

	// Queue metrics
	queueDepth     atomic.Int32
	peakQueueDepth atomic.Int32

	// Error tracking
	errorCounts   map[string]*atomic.Int64
	errorCountsMu sync.RWMutex

	// Pool lifecycle
	startTime time.Time
	endTime   time.Time
	isRunning atomic.Bool

	log *logger.NamedLogger
}

// NewMetrics creates a new metrics collector
func NewMetrics(workers int) *Metrics {
	return &Metrics{
		totalWorkers: int32(workers),
		errorCounts:  make(map[string]*atomic.Int64),
		log:          logger.WithName("worker-metrics"),
	}
}

// MetricsSnapshot represents a point-in-time metrics snapshot
type MetricsSnapshot struct {
	// Task statistics
	TotalTasks     int64   `json:"total_tasks"`
	CompletedTasks int64   `json:"completed_tasks"`
	FailedTasks    int64   `json:"failed_tasks"`
	RetryTasks     int64   `json:"retry_tasks"`
	SuccessRate    float64 `json:"success_rate"`

	// Performance statistics
	AvgDuration   time.Duration `json:"avg_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
	TotalDuration time.Duration `json:"total_duration"`

	// Throughput statistics
	TasksPerSecond float64 `json:"tasks_per_second"`
	TasksPerMinute float64 `json:"tasks_per_minute"`

	// Worker statistics
	ActiveWorkers     int32   `json:"active_workers"`
	PeakWorkers       int32   `json:"peak_workers"`
	TotalWorkers      int32   `json:"total_workers"`
	WorkerUtilization float64 `json:"worker_utilization"`

	// Queue statistics
	QueueDepth     int32 `json:"queue_depth"`
	PeakQueueDepth int32 `json:"peak_queue_depth"`

	// Error statistics
	ErrorCounts map[string]int64 `json:"error_counts"`
	TopErrors   []ErrorCount     `json:"top_errors"`

	// Time statistics
	Uptime    time.Duration `json:"uptime"`
	StartTime time.Time     `json:"start_time"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	IsRunning bool          `json:"is_running"`
}

// ErrorCount represents an error type and its count
type ErrorCount struct {
	ErrorType string `json:"error_type"`
	Count     int64  `json:"count"`
}

// Start marks the beginning of metrics collection
func (m *Metrics) Start() {
	m.startTime = time.Now()
	m.isRunning.Store(true)
	m.log.V(3).InfoS("Metrics collection started")
}

// Stop marks the end of metrics collection
func (m *Metrics) Stop() {
	m.endTime = time.Now()
	m.isRunning.Store(false)
	m.log.V(3).InfoS("Metrics collection stopped", "duration", m.endTime.Sub(m.startTime))
}

// RecordTaskStart records when a task starts
func (m *Metrics) RecordTaskStart() {
	m.totalTasks.Add(1)
	m.log.V(5).InfoS("Task started", "total", m.totalTasks.Load())
}

// RecordTaskComplete records when a task completes successfully
func (m *Metrics) RecordTaskComplete(duration time.Duration) {
	m.completedTasks.Add(1)
	m.recordDuration(duration)

	m.log.V(5).InfoS("Task completed",
		"duration", duration,
		"completed", m.completedTasks.Load())
}

// RecordTaskFailed records when a task fails
func (m *Metrics) RecordTaskFailed(duration time.Duration, err error) {
	m.failedTasks.Add(1)
	m.recordDuration(duration)

	// Record error type
	if err != nil {
		errorType := fmt.Sprintf("%T", err)
		m.recordErrorType(errorType)
	}

	m.log.V(5).InfoS("Task failed",
		"duration", duration,
		"failed", m.failedTasks.Load(),
		"error", err)
}

// RecordTaskRetry records when a task is retried
func (m *Metrics) RecordTaskRetry(attempt int, err error) {
	m.retryTasks.Add(1)

	if err != nil {
		errorType := fmt.Sprintf("%T", err)
		m.recordErrorType(errorType)
	}

	m.log.V(4).InfoS("Task retry",
		"attempt", attempt,
		"retries", m.retryTasks.Load(),
		"error", err)
}

// RecordWorkerStart records when a worker starts processing
func (m *Metrics) RecordWorkerStart() {
	active := m.activeWorkers.Add(1)

	// Update peak workers
	for {
		current := m.peakWorkers.Load()
		if active <= current || m.peakWorkers.CompareAndSwap(current, active) {
			break
		}
	}

	m.log.V(5).InfoS("Worker started", "active", active, "peak", m.peakWorkers.Load())
}

// RecordWorkerStop records when a worker stops processing
func (m *Metrics) RecordWorkerStop() {
	active := m.activeWorkers.Add(-1)
	m.log.V(5).InfoS("Worker stopped", "active", active)
}

// RecordQueueDepth records current queue depth
func (m *Metrics) RecordQueueDepth(depth int32) {
	m.queueDepth.Store(depth)

	// Update peak queue depth
	for {
		current := m.peakQueueDepth.Load()
		if depth <= current || m.peakQueueDepth.CompareAndSwap(current, depth) {
			break
		}
	}
}

// recordDuration records task duration and updates min/max
func (m *Metrics) recordDuration(duration time.Duration) {
	nanos := duration.Nanoseconds()
	m.totalDuration.Add(nanos)

	// Update min duration
	for {
		current := m.minDuration.Load()
		if current != 0 && nanos >= current {
			break
		}
		if m.minDuration.CompareAndSwap(current, nanos) {
			break
		}
	}

	// Update max duration
	for {
		current := m.maxDuration.Load()
		if nanos <= current || m.maxDuration.CompareAndSwap(current, nanos) {
			break
		}
	}
}

// recordErrorType records an error type occurrence
func (m *Metrics) recordErrorType(errorType string) {
	m.errorCountsMu.RLock()
	counter, exists := m.errorCounts[errorType]
	m.errorCountsMu.RUnlock()

	if !exists {
		m.errorCountsMu.Lock()
		// Double-check after acquiring write lock
		if counter, exists = m.errorCounts[errorType]; !exists {
			counter = &atomic.Int64{}
			m.errorCounts[errorType] = counter
		}
		m.errorCountsMu.Unlock()
	}

	counter.Add(1)
}

// Snapshot returns a current metrics snapshot
func (m *Metrics) Snapshot() MetricsSnapshot {
	now := time.Now()

	// Basic task metrics
	total := m.totalTasks.Load()
	completed := m.completedTasks.Load()
	failed := m.failedTasks.Load()
	retries := m.retryTasks.Load()

	// Calculate success rate
	var successRate float64
	if total > 0 {
		successRate = float64(completed) / float64(total) * 100
	}

	// Duration metrics
	totalDurationNanos := m.totalDuration.Load()
	minDurationNanos := m.minDuration.Load()
	maxDurationNanos := m.maxDuration.Load()

	var avgDuration time.Duration
	if completed > 0 {
		avgDuration = time.Duration(totalDurationNanos / completed)
	}

	// Calculate throughput
	var tasksPerSecond, tasksPerMinute float64
	uptime := now.Sub(m.startTime)
	if uptime > 0 {
		seconds := uptime.Seconds()
		tasksPerSecond = float64(completed) / seconds
		tasksPerMinute = tasksPerSecond * 60
	}

	// Worker utilization
	var workerUtilization float64
	if m.totalWorkers > 0 {
		workerUtilization = float64(m.activeWorkers.Load()) / float64(m.totalWorkers) * 100
	}

	// Collect error counts
	errorCounts := make(map[string]int64)
	var topErrors []ErrorCount

	m.errorCountsMu.RLock()
	for errorType, counter := range m.errorCounts {
		count := counter.Load()
		errorCounts[errorType] = count
		topErrors = append(topErrors, ErrorCount{
			ErrorType: errorType,
			Count:     count,
		})
	}
	m.errorCountsMu.RUnlock()

	// Sort top errors by count (simple bubble sort for small lists)
	for i := 0; i < len(topErrors)-1; i++ {
		for j := 0; j < len(topErrors)-i-1; j++ {
			if topErrors[j].Count < topErrors[j+1].Count {
				topErrors[j], topErrors[j+1] = topErrors[j+1], topErrors[j]
			}
		}
	}

	// Limit to top 10 errors
	if len(topErrors) > 10 {
		topErrors = topErrors[:10]
	}

	snapshot := MetricsSnapshot{
		TotalTasks:        total,
		CompletedTasks:    completed,
		FailedTasks:       failed,
		RetryTasks:        retries,
		SuccessRate:       successRate,
		AvgDuration:       avgDuration,
		MinDuration:       time.Duration(minDurationNanos),
		MaxDuration:       time.Duration(maxDurationNanos),
		TotalDuration:     time.Duration(totalDurationNanos),
		TasksPerSecond:    tasksPerSecond,
		TasksPerMinute:    tasksPerMinute,
		ActiveWorkers:     m.activeWorkers.Load(),
		PeakWorkers:       m.peakWorkers.Load(),
		TotalWorkers:      m.totalWorkers,
		WorkerUtilization: workerUtilization,
		QueueDepth:        m.queueDepth.Load(),
		PeakQueueDepth:    m.peakQueueDepth.Load(),
		ErrorCounts:       errorCounts,
		TopErrors:         topErrors,
		Uptime:            uptime,
		StartTime:         m.startTime,
		IsRunning:         m.isRunning.Load(),
	}

	if !m.isRunning.Load() {
		snapshot.EndTime = &m.endTime
	}

	return snapshot
}

// LogSummary logs a summary of the metrics
func (m *Metrics) LogSummary() {
	snapshot := m.Snapshot()

	m.log.InfoS("Worker pool metrics summary",
		"total_tasks", snapshot.TotalTasks,
		"completed_tasks", snapshot.CompletedTasks,
		"failed_tasks", snapshot.FailedTasks,
		"retry_tasks", snapshot.RetryTasks,
		"success_rate", fmt.Sprintf("%.2f%%", snapshot.SuccessRate),
		"avg_duration", snapshot.AvgDuration,
		"tasks_per_second", fmt.Sprintf("%.2f", snapshot.TasksPerSecond),
		"peak_workers", snapshot.PeakWorkers,
		"peak_queue_depth", snapshot.PeakQueueDepth,
		"uptime", snapshot.Uptime)

	// Log top errors if any
	if len(snapshot.TopErrors) > 0 {
		m.log.InfoS("Top errors encountered", "count", len(snapshot.TopErrors))
		for i, errCount := range snapshot.TopErrors {
			if i >= 5 { // Log only top 5
				break
			}
			m.log.InfoS("Error type", "rank", i+1, "type", errCount.ErrorType, "count", errCount.Count)
		}
	}
}

// MonitoringContext wraps context with metrics collection
type MonitoringContext struct {
	context.Context
	metrics *Metrics
	taskID  string
	start   time.Time
}

// WithMetrics wraps a context with metrics collection
func WithMetrics(ctx context.Context, metrics *Metrics, taskID string) *MonitoringContext {
	return &MonitoringContext{
		Context: ctx,
		metrics: metrics,
		taskID:  taskID,
		start:   time.Now(),
	}
}

// RecordSuccess records successful task completion
func (mc *MonitoringContext) RecordSuccess() {
	duration := time.Since(mc.start)
	mc.metrics.RecordTaskComplete(duration)
}

// RecordFailure records task failure
func (mc *MonitoringContext) RecordFailure(err error) {
	duration := time.Since(mc.start)
	mc.metrics.RecordTaskFailed(duration, err)
}

// RecordRetry records task retry
func (mc *MonitoringContext) RecordRetry(attempt int, err error) {
	mc.metrics.RecordTaskRetry(attempt, err)
}

// Reset resets all metrics (useful for testing)
func (m *Metrics) Reset() {
	m.totalTasks.Store(0)
	m.completedTasks.Store(0)
	m.failedTasks.Store(0)
	m.retryTasks.Store(0)
	m.totalDuration.Store(0)
	m.minDuration.Store(0)
	m.maxDuration.Store(0)
	m.activeWorkers.Store(0)
	m.peakWorkers.Store(0)
	m.queueDepth.Store(0)
	m.peakQueueDepth.Store(0)
	m.isRunning.Store(false)

	m.errorCountsMu.Lock()
	m.errorCounts = make(map[string]*atomic.Int64)
	m.errorCountsMu.Unlock()

	m.startTime = time.Time{}
	m.endTime = time.Time{}
}
