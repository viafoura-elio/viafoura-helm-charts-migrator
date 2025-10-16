package workers

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkTask represents a benchmarking task
type BenchmarkTask struct {
	id           string
	workDuration time.Duration
	priority     int
	shouldFail   bool
}

func (bt *BenchmarkTask) ID() string {
	return bt.id
}

func (bt *BenchmarkTask) Priority() int {
	return bt.priority
}

func (bt *BenchmarkTask) Execute(ctx context.Context) error {
	// Simulate work
	if bt.workDuration > 0 {
		select {
		case <-time.After(bt.workDuration):
			// Work completed
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	// Simulate CPU work
	sum := 0
	for i := 0; i < 1000; i++ {
		sum += i * i
	}
	
	if bt.shouldFail {
		return fmt.Errorf("benchmark task %s failed intentionally", bt.id)
	}
	
	return nil
}

// Benchmark worker pool with different worker counts
func BenchmarkWorkerPool_ThroughputByWorkerCount(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16, 32}
	taskCount := 1000
	
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers_%d", workers), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				
				pool := NewWorkerPool(workers)
				if err := pool.Start(); err != nil {
					b.Fatal(err)
				}
				
				// Create tasks
				tasks := make([]Task, taskCount)
				for j := 0; j < taskCount; j++ {
					tasks[j] = &BenchmarkTask{
						id:           fmt.Sprintf("task-%d", j),
						workDuration: time.Microsecond * 100,
						priority:     rand.Intn(10),
					}
				}
				
				b.StartTimer()
				
				// Submit tasks
				for _, task := range tasks {
					if err := pool.Submit(task); err != nil {
						b.Fatal(err)
					}
				}
				
				// Wait for completion
				pool.Wait()
				
				b.StopTimer()
				pool.Stop()
			}
		})
	}
}

// Benchmark task submission speed
func BenchmarkWorkerPool_TaskSubmission(b *testing.B) {
	pool := NewWorkerPool(runtime.NumCPU())
	if err := pool.Start(); err != nil {
		b.Fatal(err)
	}
	defer pool.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		task := &BenchmarkTask{
			id:           fmt.Sprintf("submit-task-%d", i),
			workDuration: time.Microsecond * 10,
		}
		
		if err := pool.Submit(task); err != nil {
			b.Fatal(err)
		}
	}
	
	pool.Wait()
}

// Benchmark batch submission
func BenchmarkWorkerPool_BatchSubmission(b *testing.B) {
	batchSizes := []int{10, 50, 100, 500, 1000}
	
	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("batch_%d", batchSize), func(b *testing.B) {
			pool := NewWorkerPool(runtime.NumCPU())
			if err := pool.Start(); err != nil {
				b.Fatal(err)
			}
			defer pool.Stop()
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				
				// Create batch
				tasks := make([]Task, batchSize)
				for j := 0; j < batchSize; j++ {
					tasks[j] = &BenchmarkTask{
						id:           fmt.Sprintf("batch-task-%d-%d", i, j),
						workDuration: time.Microsecond * 50,
					}
				}
				
				b.StartTimer()
				
				if err := pool.SubmitBatch(tasks); err != nil {
					b.Fatal(err)
				}
				
				pool.Wait()
			}
		})
	}
}

// Benchmark mixed workload (different task durations)
func BenchmarkWorkerPool_MixedWorkload(b *testing.B) {
	pool := NewWorkerPool(runtime.NumCPU())
	if err := pool.Start(); err != nil {
		b.Fatal(err)
	}
	defer pool.Stop()
	
	workloads := []struct {
		name      string
		durations []time.Duration
	}{
		{
			name:      "uniform_fast",
			durations: []time.Duration{time.Microsecond * 100},
		},
		{
			name:      "uniform_medium",
			durations: []time.Duration{time.Millisecond * 1},
		},
		{
			name: "mixed",
			durations: []time.Duration{
				time.Microsecond * 50,
				time.Microsecond * 100,
				time.Microsecond * 500,
				time.Millisecond * 1,
				time.Millisecond * 5,
			},
		},
	}
	
	for _, workload := range workloads {
		b.Run(workload.name, func(b *testing.B) {
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				// Pick random duration from workload
				duration := workload.durations[rand.Intn(len(workload.durations))]
				
				task := &BenchmarkTask{
					id:           fmt.Sprintf("mixed-task-%d", i),
					workDuration: duration,
				}
				
				if err := pool.Submit(task); err != nil {
					b.Fatal(err)
				}
			}
			
			pool.Wait()
		})
	}
}

// Benchmark error handling overhead
func BenchmarkWorkerPool_ErrorHandling(b *testing.B) {
	errorRates := []float64{0.0, 0.1, 0.25, 0.5}
	
	for _, errorRate := range errorRates {
		b.Run(fmt.Sprintf("error_rate_%.0f_percent", errorRate*100), func(b *testing.B) {
			pool := NewWorkerPool(runtime.NumCPU())
			if err := pool.Start(); err != nil {
				b.Fatal(err)
			}
			defer pool.Stop()
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				shouldFail := rand.Float64() < errorRate
				
				task := &BenchmarkTask{
					id:           fmt.Sprintf("error-task-%d", i),
					workDuration: time.Microsecond * 100,
					shouldFail:   shouldFail,
				}
				
				if err := pool.Submit(task); err != nil {
					b.Fatal(err)
				}
			}
			
			pool.Wait()
		})
	}
}

// Benchmark context cancellation
func BenchmarkWorkerPool_ContextCancellation(b *testing.B) {
	pool := NewWorkerPool(runtime.NumCPU())
	if err := pool.Start(); err != nil {
		b.Fatal(err)
	}
	defer pool.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		
		// Create context that will be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		
		task := &CancellableTask{
			id:  fmt.Sprintf("cancel-task-%d", i),
			ctx: ctx,
		}
		
		b.StartTimer()
		
		if err := pool.Submit(task); err != nil {
			b.Fatal(err)
		}
		
		// Cancel immediately
		cancel()
		
		b.StopTimer()
	}
	
	pool.Wait()
}

// Benchmark retryable worker pool
func BenchmarkRetryableWorkerPool_Performance(b *testing.B) {
	policy := DefaultRetryPolicy()
	pool := NewRetryableWorkerPool(runtime.NumCPU(), policy)
	if err := pool.Start(); err != nil {
		b.Fatal(err)
	}
	defer pool.Stop()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		task := &BenchmarkTask{
			id:           fmt.Sprintf("retry-task-%d", i),
			workDuration: time.Microsecond * 100,
			shouldFail:   rand.Float64() < 0.1, // 10% failure rate
		}
		
		if err := pool.SubmitWithRetry(task); err != nil {
			b.Fatal(err)
		}
	}
	
	pool.Wait()
}

// Benchmark metrics collection overhead
func BenchmarkWorkerPool_MetricsOverhead(b *testing.B) {
	scenarios := []struct {
		name           string
		enableMetrics  bool
	}{
		{"without_metrics", false},
		{"with_metrics", true},
	}
	
	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			pool := NewWorkerPool(runtime.NumCPU())
			if !scenario.enableMetrics {
				// Disable metrics by setting to nil (this would require pool modification)
				// For now, we'll just run the benchmark as-is
			}
			
			if err := pool.Start(); err != nil {
				b.Fatal(err)
			}
			defer pool.Stop()
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				task := &BenchmarkTask{
					id:           fmt.Sprintf("metrics-task-%d", i),
					workDuration: time.Microsecond * 50,
				}
				
				if err := pool.Submit(task); err != nil {
					b.Fatal(err)
				}
			}
			
			pool.Wait()
		})
	}
}

// Benchmark queue saturation
func BenchmarkWorkerPool_QueueSaturation(b *testing.B) {
	// Test with different queue sizes relative to worker count
	workerCount := 4
	queueMultipliers := []int{1, 2, 4, 8, 16}
	
	for _, multiplier := range queueMultipliers {
		b.Run(fmt.Sprintf("queue_size_%dx", multiplier), func(b *testing.B) {
			// Note: This would require modifying NewWorkerPool to accept queue size
			// For now, we use the default queue size
			pool := NewWorkerPool(workerCount)
			if err := pool.Start(); err != nil {
				b.Fatal(err)
			}
			defer pool.Stop()
			
			b.ResetTimer()
			
			// Submit many tasks to saturate queue
			for i := 0; i < b.N; i++ {
				task := &BenchmarkTask{
					id:           fmt.Sprintf("saturate-task-%d", i),
					workDuration: time.Millisecond * 10, // Longer tasks to fill queue
				}
				
				if err := pool.Submit(task); err != nil {
					// Queue full, wait a bit
					time.Sleep(time.Microsecond * 100)
					if err := pool.Submit(task); err != nil {
						b.Fatal(err)
					}
				}
			}
			
			pool.Wait()
		})
	}
}

// Benchmark concurrent access patterns
func BenchmarkWorkerPool_ConcurrentSubmission(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8, 16}
	
	for _, goroutines := range goroutineCounts {
		b.Run(fmt.Sprintf("goroutines_%d", goroutines), func(b *testing.B) {
			pool := NewWorkerPool(runtime.NumCPU())
			if err := pool.Start(); err != nil {
				b.Fatal(err)
			}
			defer pool.Stop()
			
			b.ResetTimer()
			
			var wg sync.WaitGroup
			tasksPerGoroutine := b.N / goroutines
			
			for g := 0; g < goroutines; g++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()
					
					for i := 0; i < tasksPerGoroutine; i++ {
						task := &BenchmarkTask{
							id:           fmt.Sprintf("concurrent-task-%d-%d", goroutineID, i),
							workDuration: time.Microsecond * 100,
						}
						
						if err := pool.Submit(task); err != nil {
							b.Error(err)
							return
						}
					}
				}(g)
			}
			
			wg.Wait()
			pool.Wait()
		})
	}
}

// Benchmark memory allocation patterns
func BenchmarkWorkerPool_MemoryAllocations(b *testing.B) {
	pool := NewWorkerPool(runtime.NumCPU())
	if err := pool.Start(); err != nil {
		b.Fatal(err)
	}
	defer pool.Stop()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		task := &BenchmarkTask{
			id:           fmt.Sprintf("alloc-task-%d", i),
			workDuration: time.Microsecond * 10,
		}
		
		if err := pool.Submit(task); err != nil {
			b.Fatal(err)
		}
	}
	
	pool.Wait()
}

// Benchmark graceful shutdown performance
func BenchmarkWorkerPool_GracefulShutdown(b *testing.B) {
	timeouts := []time.Duration{
		time.Second * 1,
		time.Second * 5,
		time.Second * 10,
	}
	
	for _, timeout := range timeouts {
		b.Run(fmt.Sprintf("timeout_%s", timeout), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				
				pool := NewWorkerPool(runtime.NumCPU())
				if err := pool.Start(); err != nil {
					b.Fatal(err)
				}
				
				// Submit some long-running tasks
				for j := 0; j < 10; j++ {
					task := &BenchmarkTask{
						id:           fmt.Sprintf("shutdown-task-%d-%d", i, j),
						workDuration: time.Millisecond * 500,
					}
					pool.Submit(task)
				}
				
				b.StartTimer()
				
				// Measure shutdown time
				if err := pool.StopWithTimeout(timeout); err != nil {
					b.Fatal(err)
				}
				
				b.StopTimer()
			}
		})
	}
}

// CancellableTask for context cancellation benchmarks
type CancellableTask struct {
	id  string
	ctx context.Context
}

func (ct *CancellableTask) ID() string {
	return ct.id
}

func (ct *CancellableTask) Priority() int {
	return 0
}

func (ct *CancellableTask) Execute(ctx context.Context) error {
	// Use the task's context if available, otherwise use the provided context
	taskCtx := ct.ctx
	if taskCtx == nil {
		taskCtx = ctx
	}
	
	// Simulate work that respects cancellation
	select {
	case <-time.After(time.Millisecond * 100):
		return nil
	case <-taskCtx.Done():
		return taskCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Example benchmark results analysis
func Example() {
	// This function demonstrates how to analyze benchmark results
	// Run: go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof
	
	fmt.Println("Benchmark Analysis Tips:")
	fmt.Println("1. Look for ns/op (nanoseconds per operation) - lower is better")
	fmt.Println("2. Check B/op (bytes allocated per operation) - lower is better") 
	fmt.Println("3. Monitor allocs/op (allocations per operation) - lower is better")
	fmt.Println("4. Compare scaling across different worker counts")
	fmt.Println("5. Identify optimal worker count for your workload")
	fmt.Println("6. Use pprof for detailed profiling: go tool pprof cpu.prof")
	
	// Output:
	// Benchmark Analysis Tips:
	// 1. Look for ns/op (nanoseconds per operation) - lower is better
	// 2. Check B/op (bytes allocated per operation) - lower is better
	// 3. Monitor allocs/op (allocations per operation) - lower is better
	// 4. Compare scaling across different worker counts
	// 5. Identify optimal worker count for your workload
	// 6. Use pprof for detailed profiling: go tool pprof cpu.prof
}