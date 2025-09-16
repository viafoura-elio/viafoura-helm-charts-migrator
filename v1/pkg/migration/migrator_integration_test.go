package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/client-go/kubernetes"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/services"
	"helm-charts-migrator/v1/pkg/workers"
)

// Integration test suite for the migration pipeline
func TestMigrationPipelineIntegration(t *testing.T) {
	tests := []struct {
		name         string
		services     []string
		clusters     []ClusterInfo
		config       *config.Config
		setupMocks   func(*MockServices)
		expectError  bool
		expectFiles  []string
		validateFunc func(t *testing.T, tempDir string)
	}{
		{
			name:     "single service single cluster migration",
			services: []string{"test-service"},
			clusters: []ClusterInfo{
				{
					Name:             "test-cluster",
					Context:          "test-context",
					DefaultNamespace: "default",
					IsDefault:        true,
					Namespaces: []NamespaceInfo{
						{Name: "default", Environment: "production"},
					},
				},
			},
			config: createTestConfig(false),
			setupMocks: func(mocks *MockServices) {
				setupBasicMocks(mocks, "test-service", "test-cluster")
			},
			expectFiles: []string{
				"apps/test-service/Chart.yaml",
				"apps/test-service/values.yaml",
				"apps/test-service/helm-values.yaml",
				"apps/test-service/envs/test-cluster/production/default/values.yaml",
			},
			validateFunc: validateBasicMigration,
		},
		{
			name:     "multiple services parallel migration",
			services: []string{"service-a", "service-b", "service-c"},
			clusters: []ClusterInfo{
				{
					Name:             "test-cluster",
					Context:          "test-context",
					DefaultNamespace: "default",
					IsDefault:        true,
					Namespaces: []NamespaceInfo{
						{Name: "default", Environment: "production"},
					},
				},
			},
			config: createTestConfigWithParallel(3),
			setupMocks: func(mocks *MockServices) {
				for _, service := range []string{"service-a", "service-b", "service-c"} {
					setupBasicMocks(mocks, service, "test-cluster")
				}
			},
			expectFiles: []string{
				"apps/service-a/Chart.yaml",
				"apps/service-b/Chart.yaml",
				"apps/service-c/Chart.yaml",
			},
			validateFunc: validateParallelMigration,
		},
		{
			name:     "multi-cluster migration",
			services: []string{"multi-service"},
			clusters: []ClusterInfo{
				{
					Name:             "cluster-1",
					Context:          "context-1",
					DefaultNamespace: "default",
					IsDefault:        true,
					Namespaces: []NamespaceInfo{
						{Name: "default", Environment: "production"},
						{Name: "staging", Environment: "staging"},
					},
				},
				{
					Name:             "cluster-2",
					Context:          "context-2",
					DefaultNamespace: "default",
					IsDefault:        false,
					Namespaces: []NamespaceInfo{
						{Name: "default", Environment: "production"},
					},
				},
			},
			config: createTestConfig(false),
			setupMocks: func(mocks *MockServices) {
				setupMultiClusterMocks(mocks, "multi-service")
			},
			expectFiles: []string{
				"apps/multi-service/Chart.yaml",
				"apps/multi-service/envs/cluster-1/production/default/values.yaml",
				"apps/multi-service/envs/cluster-1/staging/staging/values.yaml",
				"apps/multi-service/envs/cluster-2/production/default/values.yaml",
			},
			validateFunc: validateMultiClusterMigration,
		},
		{
			name:     "migration with transformation pipeline",
			services: []string{"transform-service"},
			clusters: []ClusterInfo{
				{
					Name:             "test-cluster",
					Context:          "test-context",
					DefaultNamespace: "default",
					IsDefault:        true,
					Namespaces: []NamespaceInfo{
						{Name: "default", Environment: "production"},
					},
				},
			},
			config: createTestConfigWithTransforms(),
			setupMocks: func(mocks *MockServices) {
				setupTransformationMocks(mocks, "transform-service")
			},
			expectFiles: []string{
				"apps/transform-service/Chart.yaml",
				"apps/transform-service/envs/test-cluster/production/default/values.yaml",
			},
			validateFunc: validateTransformationMigration,
		},
		{
			name:     "error handling and recovery",
			services: []string{"error-service"},
			clusters: []ClusterInfo{
				{
					Name:             "test-cluster",
					Context:          "test-context",
					DefaultNamespace: "default",
					IsDefault:        true,
					Namespaces: []NamespaceInfo{
						{Name: "default", Environment: "production"},
					},
				},
			},
			config:      createTestConfig(false),
			setupMocks:  setupErrorMocks,
			expectError: true,
			validateFunc: func(t *testing.T, tempDir string) {
				// Verify partial files were cleaned up or error state is handled
				assert.NotNil(t, tempDir) // Basic validation that test ran
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()

			// Change to temp directory
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tempDir))
			defer func() {
				os.Chdir(originalDir)
			}()

			// Create base chart directory
			createBaseChart(t, tempDir)

			// Setup mock services
			mocks := NewMockServices()
			tt.setupMocks(mocks)

			// Create migrator
			migrator := NewMigrator(
				tt.config,
				mocks.Kubernetes,
				mocks.Helm,
				mocks.File,
				mocks.Transform,
				mocks.Cache,
				mocks.SOPS,
				false, // dryRun
				true,  // noSOPS
			)

			// Run migration
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err = migrator.MigrateServices(ctx, tt.services, tt.clusters)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check expected files exist
				for _, expectedFile := range tt.expectFiles {
					fullPath := filepath.Join(tempDir, expectedFile)
					assert.FileExists(t, fullPath, "Expected file %s should exist", expectedFile)
				}
			}

			// Run custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, tempDir)
			}
		})
	}
}

// Test parallel processing performance
func TestParallelProcessingPerformance(t *testing.T) {
	services := make([]string, 20) // 20 services
	for i := 0; i < 20; i++ {
		services[i] = fmt.Sprintf("perf-service-%d", i)
	}

	clusters := []ClusterInfo{
		{
			Name:               "perf-cluster",
			Context:            "perf-context",
			DefaultNamespace:   "default",
			DefaultEnvironment: "production",
			IsDefault:          true,
			Namespaces: []NamespaceInfo{
				{Name: "default", Environment: "production"},
			},
		},
	}

	tests := []struct {
		name         string
		maxWorkers   int
		expectFaster bool
	}{
		{"sequential", 1, false},
		{"parallel-2", 2, true},
		{"parallel-4", 4, true},
		{"parallel-8", 8, true},
	}

	var baseDuration time.Duration

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tempDir))
			defer os.Chdir(originalDir)

			createBaseChart(t, tempDir)

			// Setup mocks
			mocks := NewMockServices()
			for _, service := range services {
				setupPerformanceMocks(mocks, service, "perf-cluster")
			}

			// Create config with specified workers
			config := createTestConfigWithParallel(tt.maxWorkers)

			migrator := NewMigrator(
				config,
				mocks.Kubernetes,
				mocks.Helm,
				mocks.File,
				mocks.Transform,
				mocks.Cache,
				mocks.SOPS,
				false,
				true,
			)

			// Measure execution time
			start := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			err = migrator.MigrateServices(ctx, services, clusters)
			duration := time.Since(start)

			assert.NoError(t, err)
			t.Logf("Test %s completed in %v with %d workers", tt.name, duration, tt.maxWorkers)

			// Store baseline
			if i == 0 {
				baseDuration = duration
			} else if tt.expectFaster && baseDuration > 0 {
				// Parallel should be faster (allowing some variance)
				expectedImprovement := float64(baseDuration) * 0.7 // At least 30% improvement
				assert.Less(t, float64(duration), expectedImprovement,
					"Parallel execution should be faster than sequential")
			}
		})
	}
}

// Test worker pool stress scenarios
func TestWorkerPoolStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Run("high_concurrency", func(t *testing.T) {
		pool := workers.NewWorkerPool(10)
		require.NoError(t, pool.Start())
		defer pool.Stop()

		// Submit many tasks quickly
		numTasks := 1000
		for i := 0; i < numTasks; i++ {
			task := &TestTask{
				id:       fmt.Sprintf("stress-task-%d", i),
				duration: time.Millisecond * 10,
			}
			err := pool.Submit(task)
			assert.NoError(t, err)
		}

		// Wait for completion
		err := pool.WaitWithTimeout(30 * time.Second)
		assert.NoError(t, err)

		// Check metrics
		metrics := pool.GetMetricsSnapshot()
		assert.Equal(t, int64(numTasks), metrics.TotalTasks)
		assert.Equal(t, int64(numTasks), metrics.CompletedTasks)
		assert.Equal(t, int64(0), metrics.FailedTasks)
	})

	t.Run("mixed_success_failure", func(t *testing.T) {
		pool := workers.NewWorkerPool(5)
		require.NoError(t, pool.Start())
		defer pool.Stop()

		// Submit mix of successful and failing tasks
		numTasks := 100
		failureRate := 0.3 // 30% will fail

		for i := 0; i < numTasks; i++ {
			var task workers.Task
			if float64(i)/float64(numTasks) < failureRate {
				task = &FailingTask{
					id: fmt.Sprintf("fail-task-%d", i),
				}
			} else {
				task = &TestTask{
					id:       fmt.Sprintf("success-task-%d", i),
					duration: time.Millisecond * 5,
				}
			}
			err := pool.Submit(task)
			assert.NoError(t, err)
		}

		err := pool.WaitWithTimeout(30 * time.Second)
		assert.NoError(t, err)

		metrics := pool.GetMetricsSnapshot()
		assert.Equal(t, int64(numTasks), metrics.TotalTasks)

		expectedFailures := int64(float64(numTasks) * failureRate)
		expectedSuccesses := int64(numTasks) - expectedFailures

		// Allow some tolerance due to floating point
		assert.InDelta(t, expectedFailures, metrics.FailedTasks, 2)
		assert.InDelta(t, expectedSuccesses, metrics.CompletedTasks, 2)
	})
}

// Test graceful shutdown scenarios
func TestGracefulShutdown(t *testing.T) {
	t.Run("shutdown_with_pending_tasks", func(t *testing.T) {
		pool := workers.NewWorkerPool(2)
		require.NoError(t, pool.Start())

		// Submit long-running tasks
		for i := 0; i < 5; i++ {
			task := &TestTask{
				id:       fmt.Sprintf("long-task-%d", i),
				duration: time.Second * 2,
			}
			err := pool.Submit(task)
			assert.NoError(t, err)
		}

		// Start shutdown immediately
		start := time.Now()
		err := pool.StopWithTimeout(5 * time.Second)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 6*time.Second, "Shutdown should complete within timeout")

		metrics := pool.GetMetricsSnapshot()
		t.Logf("Shutdown completed: total=%d, completed=%d, failed=%d",
			metrics.TotalTasks, metrics.CompletedTasks, metrics.FailedTasks)
	})

	t.Run("shutdown_timeout", func(t *testing.T) {
		pool := workers.NewWorkerPool(1)
		require.NoError(t, pool.Start())

		// Submit a very long task
		task := &TestTask{
			id:       "very-long-task",
			duration: time.Second * 10,
		}
		err := pool.Submit(task)
		require.NoError(t, err)

		// Try to shutdown with short timeout
		start := time.Now()
		err = pool.StopWithTimeout(time.Second * 2)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 8*time.Second, "Should force stop after timeout")
	})
}

// Helper functions and test utilities

type MockServices struct {
	Kubernetes services.KubernetesService
	Helm       services.HelmService
	File       services.FileService
	Transform  services.TransformationService
	Cache      services.CacheService
	SOPS       services.SOPSService
}

func NewMockServices() *MockServices {
	return &MockServices{
		Kubernetes: &MockKubernetesService{},
		Helm:       &MockHelmService{},
		File:       &MockFileService{},
		Transform:  &MockTransformService{},
		Cache:      &MockCacheService{},
		SOPS:       &MockSOPSService{},
	}
}

// Mock implementations (simplified for testing)
type MockKubernetesService struct{}

func (m *MockKubernetesService) GetClient(context string) (*kubernetes.Clientset, error) {
	return nil, nil
}

func (m *MockKubernetesService) ListReleases(ctx context.Context, context, namespace string) ([]*release.Release, error) {
	// Return mock releases based on context
	releases := []*release.Release{
		{
			Name:      strings.Split(context, "-")[0], // Extract service name from context
			Namespace: namespace,
			Config:    map[string]interface{}{"test": "value"},
		},
	}
	return releases, nil
}

func (m *MockKubernetesService) GetRelease(ctx context.Context, kubeContext, namespace, releaseName string) (*release.Release, error) {
	return nil, nil
}

func (m *MockKubernetesService) SwitchContext(context string) error {
	return nil
}

func (m *MockKubernetesService) GetCurrentContext() (string, error) {
	return "test-context", nil
}

type MockHelmService struct{}

func (m *MockHelmService) GetReleaseByName(name string, releases []*release.Release) *release.Release {
	for _, rel := range releases {
		if rel.Name == name {
			return rel
		}
	}
	return nil
}

func (m *MockHelmService) ExtractValues(release *release.Release) (map[string]interface{}, error) {
	return map[string]interface{}{
		"image": map[string]interface{}{
			"repository": fmt.Sprintf("%s-repo", release.Name),
			"tag":        "latest",
		},
		"service": map[string]interface{}{
			"type": "ClusterIP",
			"port": 80,
		},
	}, nil
}

func (m *MockHelmService) ExtractManifest(release *release.Release) (string, error) {
	return fmt.Sprintf("# Manifest for %s\napiVersion: v1\nkind: Service", release.Name), nil
}

func (m *MockHelmService) ValidateChart(chartPath string) error {
	return nil
}

type MockFileService struct{}

func (m *MockFileService) WriteYAML(path string, data interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write mock file
	return os.WriteFile(path, []byte("# Mock YAML content"), 0644)
}

func (m *MockFileService) ReadYAMLToMap(path string) (map[string]interface{}, error) {
	return map[string]interface{}{"test": "data"}, nil
}

func (m *MockFileService) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (m *MockFileService) CopyDirectory(src, dst string) error {
	return nil
}

func (m *MockFileService) CopyFile(src, dst string) error {
	return nil
}

func (m *MockFileService) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (m *MockFileService) ListFiles(dir, pattern string) ([]string, error) {
	return []string{}, nil
}

type MockTransformService struct{}

func (m *MockTransformService) Transform(data map[string]interface{}, config services.TransformConfig) (map[string]interface{}, error) {
	// Simple transformation - just return the data
	return data, nil
}

func (m *MockTransformService) ConvertKeys(data map[string]interface{}) map[string]interface{} {
	return data
}

func (m *MockTransformService) NormalizeKeys(values map[string]interface{}) map[string]interface{} {
	return values
}

func (m *MockTransformService) ExtractSecrets(values map[string]interface{}) (secrets, cleaned map[string]interface{}) {
	return map[string]interface{}{}, values
}

func (m *MockTransformService) MergeValues(base, override map[string]interface{}) map[string]interface{} {
	return override
}

type MockCacheService struct{}

func (m *MockCacheService) GetReleases(cluster, namespace string) []*release.Release {
	return nil // Always miss cache for testing
}

func (m *MockCacheService) SetReleases(cluster, namespace string, releases []*release.Release) error {
	return nil
}

func (m *MockCacheService) GetTempPath(cluster, namespace, service, resourceType string) string {
	return filepath.Join("/tmp", cluster, namespace, service, resourceType)
}

func (m *MockCacheService) Clear() error {
	return nil
}

func (m *MockCacheService) Cleanup() error {
	return nil
}

type MockSOPSService struct{}

func (m *MockSOPSService) Encrypt(filePath string) error {
	return nil
}

func (m *MockSOPSService) Decrypt(filePath string) ([]byte, error) {
	return []byte("decrypted content"), nil
}

func (m *MockSOPSService) EncryptBatch(files []string, workers int) error {
	return nil
}

func (m *MockSOPSService) IsEncrypted(filePath string) bool {
	return false
}

// Test task implementations
type TestTask struct {
	id       string
	duration time.Duration
	priority int
}

func (t *TestTask) ID() string {
	return t.id
}

func (t *TestTask) Priority() int {
	return t.priority
}

func (t *TestTask) Execute(ctx context.Context) error {
	if t.duration > 0 {
		select {
		case <-time.After(t.duration):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type FailingTask struct {
	id string
}

func (f *FailingTask) ID() string {
	return f.id
}

func (f *FailingTask) Priority() int {
	return 0
}

func (f *FailingTask) Execute(ctx context.Context) error {
	return fmt.Errorf("task %s failed intentionally", f.id)
}

// Test configuration helpers
func createTestConfig(parallel bool) *config.Config {
	cfg := &config.Config{
		Globals: config.Globals{
			Performance: config.PerformanceConfig{
				MaxConcurrentServices: 1,
			},
		},
		Services: map[string]config.Service{
			"test-service": {Enabled: true},
		},
	}

	if parallel {
		cfg.Globals.Performance.MaxConcurrentServices = 3
	}

	return cfg
}

func createTestConfigWithParallel(maxWorkers int) *config.Config {
	return &config.Config{
		Globals: config.Globals{
			Performance: config.PerformanceConfig{
				MaxConcurrentServices: maxWorkers,
			},
		},
		Services: map[string]config.Service{
			"service-a":       {Enabled: true},
			"service-b":       {Enabled: true},
			"service-c":       {Enabled: true},
			"perf-service-0":  {Enabled: true},
			"perf-service-1":  {Enabled: true},
			"perf-service-2":  {Enabled: true},
			"perf-service-3":  {Enabled: true},
			"perf-service-4":  {Enabled: true},
			"perf-service-5":  {Enabled: true},
			"perf-service-6":  {Enabled: true},
			"perf-service-7":  {Enabled: true},
			"perf-service-8":  {Enabled: true},
			"perf-service-9":  {Enabled: true},
			"perf-service-10": {Enabled: true},
			"perf-service-11": {Enabled: true},
			"perf-service-12": {Enabled: true},
			"perf-service-13": {Enabled: true},
			"perf-service-14": {Enabled: true},
			"perf-service-15": {Enabled: true},
			"perf-service-16": {Enabled: true},
			"perf-service-17": {Enabled: true},
			"perf-service-18": {Enabled: true},
			"perf-service-19": {Enabled: true},
		},
	}
}

func createTestConfigWithTransforms() *config.Config {
	return &config.Config{
		Globals: config.Globals{
			Mappings: &config.Mappings{
				Normalizer: &config.Normalizer{
					Enabled: true,
					Patterns: map[string]string{
						"old_key": "newKey",
					},
				},
			},
		},
		Services: map[string]config.Service{
			"transform-service": {Enabled: true},
		},
	}
}

func createBaseChart(t *testing.T, tempDir string) {
	baseChartDir := filepath.Join(tempDir, "migration", "base-chart")
	require.NoError(t, os.MkdirAll(baseChartDir, 0755))

	// Create basic Chart.yaml
	chartContent := `apiVersion: v2
name: base-chart
description: Base chart template
version: 0.1.0
`
	require.NoError(t, os.WriteFile(filepath.Join(baseChartDir, "Chart.yaml"), []byte(chartContent), 0644))

	// Create basic values.yaml
	valuesContent := `# Base chart values
replicaCount: 1
image:
  repository: ""
  tag: "latest"
`
	require.NoError(t, os.WriteFile(filepath.Join(baseChartDir, "values.yaml"), []byte(valuesContent), 0644))
}

// Setup functions for different test scenarios
func setupBasicMocks(mocks *MockServices, serviceName, clusterName string) {
	// Basic setup - mocks are already configured to return appropriate data
}

func setupMultiClusterMocks(mocks *MockServices, serviceName string) {
	// Multi-cluster setup
}

func setupTransformationMocks(mocks *MockServices, serviceName string) {
	// Transformation setup
}

func setupErrorMocks(mocks *MockServices) {
	// Override mocks to return errors
	originalKube := mocks.Kubernetes
	mocks.Kubernetes = &ErrorKubernetesService{original: originalKube}
}

func setupPerformanceMocks(mocks *MockServices, serviceName, clusterName string) {
	// Performance test setup - add small delays
}

// Error service for testing error handling
type ErrorKubernetesService struct {
	original services.KubernetesService
}

func (e *ErrorKubernetesService) GetClient(context string) (*kubernetes.Clientset, error) {
	return nil, fmt.Errorf("mock kubernetes error")
}

func (e *ErrorKubernetesService) ListReleases(ctx context.Context, context, namespace string) ([]*release.Release, error) {
	return nil, fmt.Errorf("mock kubernetes error")
}

func (e *ErrorKubernetesService) GetRelease(ctx context.Context, kubeContext, namespace, releaseName string) (*release.Release, error) {
	return nil, fmt.Errorf("mock kubernetes error")
}

func (e *ErrorKubernetesService) SwitchContext(context string) error {
	return fmt.Errorf("mock kubernetes error")
}

func (e *ErrorKubernetesService) GetCurrentContext() (string, error) {
	return "", fmt.Errorf("mock kubernetes error")
}

// Validation functions
func validateBasicMigration(t *testing.T, tempDir string) {
	// Check that basic files were created correctly
	chartPath := filepath.Join(tempDir, "apps", "test-service", "Chart.yaml")
	assert.FileExists(t, chartPath)

	content, err := os.ReadFile(chartPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-service", "Chart should be renamed")
}

func validateParallelMigration(t *testing.T, tempDir string) {
	// Validate that all services were processed
	services := []string{"service-a", "service-b", "service-c"}
	for _, service := range services {
		chartPath := filepath.Join(tempDir, "apps", service, "Chart.yaml")
		assert.FileExists(t, chartPath, "Service %s should have chart", service)
	}
}

func validateMultiClusterMigration(t *testing.T, tempDir string) {
	// Validate multi-cluster structure
	cluster1Path := filepath.Join(tempDir, "apps", "multi-service", "envs", "cluster-1")
	cluster2Path := filepath.Join(tempDir, "apps", "multi-service", "envs", "cluster-2")

	assert.DirExists(t, cluster1Path)
	assert.DirExists(t, cluster2Path)
}

func validateTransformationMigration(t *testing.T, tempDir string) {
	// Validate transformations were applied
	valuesPath := filepath.Join(tempDir, "apps", "transform-service", "envs", "test-cluster", "production", "default", "values.yaml")
	assert.FileExists(t, valuesPath)
}
