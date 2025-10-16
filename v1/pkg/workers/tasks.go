package workers

import (
	"context"
	"fmt"
	"path/filepath"

	"helm-charts-migrator/v1/pkg/adapters"
	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
)

// ServiceMigrationTask represents a task to migrate a single service
type ServiceMigrationTask struct {
	ServiceName     string
	ClusterName     string
	Config          *config.Config
	ChartCopier     adapters.ChartCopier
	ValuesExtractor adapters.ValuesExtractor
	FileManager     adapters.FileManager
	Pipeline        *adapters.TransformationPipeline
	DryRun          bool
	log             *logger.NamedLogger
}

// NewServiceMigrationTask creates a new service migration task
func NewServiceMigrationTask(
	serviceName, clusterName string,
	cfg *config.Config,
	chartCopier adapters.ChartCopier,
	extractor adapters.ValuesExtractor,
	fileManager adapters.FileManager,
	pipeline *adapters.TransformationPipeline,
	dryRun bool,
) *ServiceMigrationTask {
	return &ServiceMigrationTask{
		ServiceName:     serviceName,
		ClusterName:     clusterName,
		Config:          cfg,
		ChartCopier:     chartCopier,
		ValuesExtractor: extractor,
		FileManager:     fileManager,
		Pipeline:        pipeline,
		DryRun:          dryRun,
		log:             logger.WithName("service-migration-task"),
	}
}

func (t *ServiceMigrationTask) ID() string {
	return fmt.Sprintf("%s-%s", t.ServiceName, t.ClusterName)
}

func (t *ServiceMigrationTask) Priority() int {
	// Lower priority number = higher priority
	// Default cluster gets higher priority
	cluster := t.Config.GetCluster(t.ClusterName)
	if cluster != nil && cluster.Default {
		return 1
	}
	return 10
}

func (t *ServiceMigrationTask) Execute(ctx context.Context) error {
	t.log.InfoS("Starting service migration",
		"service", t.ServiceName,
		"cluster", t.ClusterName,
		"dryRun", t.DryRun)
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	if t.DryRun {
		t.log.InfoS("DRY RUN: Would migrate service",
			"service", t.ServiceName,
			"cluster", t.ClusterName)
		return nil
	}
	
	// Perform the actual migration
	// This is a simplified version - the real implementation would call
	// the actual migration logic from the Migrator
	
	service, exists := t.Config.Services[t.ServiceName]
	if !exists || !service.Enabled {
		return fmt.Errorf("service %s not found or not enabled", t.ServiceName)
	}
	
	paths := config.NewPaths("", "apps", ".cache").
		ForService(t.ServiceName).
		ForCluster(t.ClusterName)
	targetDir := paths.ServiceDir()
	
	// Copy base chart using the new method with full service config
	if err := t.ChartCopier.CopyBaseChartWithService(
		"migration/base-chart",
		paths.ServiceDir(),
		&service,
	); err != nil {
		return fmt.Errorf("failed to copy base chart: %w", err)
	}
	
	// Transform values
	if err := t.Pipeline.TransformService(t.ServiceName); err != nil {
		return fmt.Errorf("failed to transform service: %w", err)
	}
	
	t.log.InfoS("Service migration completed",
		"service", t.ServiceName,
		"cluster", t.ClusterName,
		"targetDir", targetDir)
	
	return nil
}

// ValuesExtractionTask represents a task to extract values from a release
type ValuesExtractionTask struct {
	ReleaseName string
	Namespace   string
	Cluster     string
	OutputPath  string
	Extractor   adapters.ValuesExtractor
	HelmService services.HelmService
	log         *logger.NamedLogger
}

// NewValuesExtractionTask creates a new values extraction task
func NewValuesExtractionTask(
	releaseName, namespace, cluster, outputPath string,
	extractor adapters.ValuesExtractor,
	helmService services.HelmService,
) *ValuesExtractionTask {
	return &ValuesExtractionTask{
		ReleaseName: releaseName,
		Namespace:   namespace,
		Cluster:     cluster,
		OutputPath:  outputPath,
		Extractor:   extractor,
		HelmService: helmService,
		log:         logger.WithName("values-extraction-task"),
	}
}

func (t *ValuesExtractionTask) ID() string {
	return fmt.Sprintf("extract-%s-%s-%s", t.Cluster, t.Namespace, t.ReleaseName)
}

func (t *ValuesExtractionTask) Priority() int {
	return 5 // Higher priority than migration
}

func (t *ValuesExtractionTask) Execute(ctx context.Context) error {
	t.log.V(3).InfoS("Extracting values",
		"release", t.ReleaseName,
		"namespace", t.Namespace,
		"cluster", t.Cluster)
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// This would use the actual helm service to get the release
	// and extract values - simplified for demonstration
	
	t.log.V(3).InfoS("Values extracted",
		"release", t.ReleaseName,
		"outputPath", t.OutputPath)
	
	return nil
}

// TransformationTask represents a task to transform values
type TransformationTask struct {
	ServiceName string
	FilePath    string
	Transform   services.TransformationService
	Config      services.TransformConfig
	log         *logger.NamedLogger
}

// NewTransformationTask creates a new transformation task
func NewTransformationTask(
	serviceName, filePath string,
	transform services.TransformationService,
	config services.TransformConfig,
) *TransformationTask {
	return &TransformationTask{
		ServiceName: serviceName,
		FilePath:    filePath,
		Transform:   transform,
		Config:      config,
		log:         logger.WithName("transformation-task"),
	}
}

func (t *TransformationTask) ID() string {
	return fmt.Sprintf("transform-%s-%s", t.ServiceName, filepath.Base(t.FilePath))
}

func (t *TransformationTask) Priority() int {
	return 20 // Lower priority, runs after extraction
}

func (t *TransformationTask) Execute(ctx context.Context) error {
	t.log.V(3).InfoS("Transforming values",
		"service", t.ServiceName,
		"file", t.FilePath)
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Read values
	values := make(map[string]interface{})
	// This would read the actual file
	
	// Apply transformation
	transformed, err := t.Transform.Transform(values, t.Config)
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}
	
	// Write back transformed values
	// This would write the actual file
	
	t.log.V(3).InfoS("Values transformed",
		"service", t.ServiceName,
		"file", t.FilePath,
		"transformed", transformed != nil)
	
	return nil
}

// SOPSEncryptionTask represents a task to encrypt files with SOPS
type SOPSEncryptionTask struct {
	FilePath    string
	AwsProfile  string
	SOPSService services.SOPSService
	log         *logger.NamedLogger
}

// NewSOPSEncryptionTask creates a new SOPS encryption task
func NewSOPSEncryptionTask(
	filePath, awsProfile string,
	sopsService services.SOPSService,
) *SOPSEncryptionTask {
	return &SOPSEncryptionTask{
		FilePath:    filePath,
		AwsProfile:  awsProfile,
		SOPSService: sopsService,
		log:         logger.WithName("sops-encryption-task"),
	}
}

func (t *SOPSEncryptionTask) ID() string {
	return fmt.Sprintf("sops-%s", filepath.Base(t.FilePath))
}

func (t *SOPSEncryptionTask) Priority() int {
	return 30 // Runs after transformation
}

func (t *SOPSEncryptionTask) Execute(ctx context.Context) error {
	t.log.V(3).InfoS("Encrypting file with SOPS",
		"file", t.FilePath,
		"profile", t.AwsProfile)
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Check if already encrypted
	if t.SOPSService.IsEncrypted(t.FilePath) {
		t.log.V(4).InfoS("File already encrypted, skipping", "file", t.FilePath)
		return nil
	}
	
	// Encrypt the file
	if err := t.SOPSService.Encrypt(t.FilePath); err != nil {
		return fmt.Errorf("failed to encrypt %s: %w", t.FilePath, err)
	}
	
	t.log.V(3).InfoS("File encrypted", "file", t.FilePath)
	return nil
}

// BatchTask represents a task that contains multiple sub-tasks
type BatchTask struct {
	Name     string
	SubTasks []Task
	pool     *WorkerPool
}

// NewBatchTask creates a new batch task
func NewBatchTask(name string, subTasks []Task, workers int) *BatchTask {
	return &BatchTask{
		Name:     name,
		SubTasks: subTasks,
		pool:     NewWorkerPool(workers),
	}
}

func (t *BatchTask) ID() string {
	return fmt.Sprintf("batch-%s", t.Name)
}

func (t *BatchTask) Priority() int {
	return 1 // High priority for batch tasks
}

func (t *BatchTask) Execute(ctx context.Context) error {
	log := logger.WithName("batch-task")
	log.InfoS("Starting batch task", "name", t.Name, "taskCount", len(t.SubTasks))
	
	// Start the pool
	if err := t.pool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}
	defer t.pool.Stop()
	
	// Submit all sub-tasks
	if err := t.pool.SubmitBatch(t.SubTasks); err != nil {
		return fmt.Errorf("failed to submit batch: %w", err)
	}
	
	// Collect results
	successCount := 0
	failCount := 0
	
	go func() {
		for result := range t.pool.Results() {
			if result.Success {
				successCount++
			} else {
				failCount++
				log.Error(result.Error, "Sub-task failed", "taskID", result.TaskID)
			}
		}
	}()
	
	// Wait for completion
	t.pool.Wait()
	
	log.InfoS("Batch task completed",
		"name", t.Name,
		"success", successCount,
		"failed", failCount)
	
	if failCount > 0 {
		return fmt.Errorf("batch task %s had %d failures", t.Name, failCount)
	}
	
	return nil
}