package migration

import (
	"context"
	"fmt"
	"path/filepath"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
	"helm-charts-migrator/v1/pkg/transformers"
	"helm-charts-migrator/v1/pkg/workers"
)

// HierarchicalMigratorFactory creates migrators with hierarchical configuration support
type HierarchicalMigratorFactory struct {
	baseConfig *config.Config
	hierarchy  *config.HierarchicalConfig
	loader     config.ConfigLoader
	log        *logger.NamedLogger
}

// NewHierarchicalMigratorFactory creates a new factory with hierarchical config support
func NewHierarchicalMigratorFactory(baseConfig *config.Config) *HierarchicalMigratorFactory {
	factory := &HierarchicalMigratorFactory{
		baseConfig: baseConfig,
		hierarchy:  config.NewHierarchicalConfig(),
		loader:     config.NewConfigLoader(),
		log:        logger.WithName("hierarchical-factory"),
	}
	
	// Initialize hierarchy from base config
	factory.initializeHierarchy()
	
	return factory
}

// initializeHierarchy sets up the configuration hierarchy
func (f *HierarchicalMigratorFactory) initializeHierarchy() {
	// Set defaults
	f.hierarchy.SetDefaults(map[string]interface{}{
		"globals": map[string]interface{}{
			"converter": map[string]interface{}{
				"minUppercaseChars":  3,
				"skipJavaProperties": false,
				"skipUppercaseKeys":  false,
			},
			"performance": map[string]interface{}{
				"maxConcurrentServices": 5,
				"showProgress":         true,
			},
		},
	})
	
	// Convert globals to map and set
	if f.baseConfig != nil {
		globalsMap := f.configToMap(f.baseConfig.Globals)
		f.hierarchy.SetGlobals(globalsMap)
		
		// Set cluster configs
		for name, cluster := range f.baseConfig.Clusters {
			clusterMap := f.clusterToMap(cluster)
			f.hierarchy.SetClusterConfig(name, clusterMap)
		}
		
		// Set service configs
		for name, service := range f.baseConfig.Services {
			serviceMap := f.serviceToMap(service)
			f.hierarchy.SetServiceConfig(name, serviceMap)
		}
	}
	
	f.log.InfoS("Initialized configuration hierarchy",
		"clusters", len(f.baseConfig.Clusters),
		"services", len(f.baseConfig.Services))
}

// CreateMigratorForService creates a migrator with effective config for a specific service
func (f *HierarchicalMigratorFactory) CreateMigratorForService(
	serviceName string,
	cluster string,
	env string,
	namespace string,
	opts MigratorOptions,
) (*Migrator, error) {
	// Get effective configuration
	effectiveLayer := f.hierarchy.GetEffectiveConfig(cluster, env, namespace, serviceName)
	if effectiveLayer == nil {
		return nil, fmt.Errorf("failed to get effective config for service %s", serviceName)
	}
	
	// Convert layer to Config struct
	effectiveConfig, err := f.layerToConfig(effectiveLayer)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config layer: %w", err)
	}
	
	// Log effective configuration
	f.log.V(2).InfoS("Using effective configuration",
		"service", serviceName,
		"cluster", cluster,
		"environment", env,
		"namespace", namespace)
	
	// Create services with effective config
	kubernetes := services.NewKubernetesService()
	helm := services.NewHelmService()
	file := services.NewFileService()
	
	// Create transformation service with effective config
	transform := services.NewTransformationService(effectiveConfig)
	
	// Create transformer factory with effective config
	transformerFactory := transformers.NewTransformerFactory(effectiveConfig)
	
	// Create cache service
	cache, err := services.NewCacheService(opts.CacheDir, opts.CleanupCache)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache service: %w", err)
	}
	
	// Create SOPS service with effective config
	sopsConfig := &effectiveConfig.Globals.SOPS
	if sopsConfig.ParallelWorkers == 0 {
		sopsConfig.ParallelWorkers = 5
	}
	if sopsConfig.Timeout == 0 {
		sopsConfig.Timeout = 30
	}
	sops := services.NewSOPSService(sopsConfig)
	
	// Create migrator with effective configuration
	migrator := NewMigrator(
		effectiveConfig,
		kubernetes,
		helm,
		file,
		transform,
		cache,
		sops,
		opts.DryRun,
		opts.NoSOPS,
	)
	
	// Attach transformer registry if needed
	if transformerFactory != nil {
		migrator.SetTransformerRegistry(transformerFactory.GetRegistry())
	}
	
	f.log.InfoS("Created service-specific migrator",
		"service", serviceName,
		"dryRun", opts.DryRun)
	
	return migrator, nil
}

// CreateParallelMigrator creates a migrator that processes services in parallel
func (f *HierarchicalMigratorFactory) CreateParallelMigrator(
	services []string,
	cluster string,
	opts MigratorOptions,
) (*ParallelMigrator, error) {
	// Get cluster config
	clusterLayer := f.hierarchy.GetEffectiveConfig(cluster, "", "", "")
	if clusterLayer == nil {
		return nil, fmt.Errorf("cluster %s not found", cluster)
	}
	
	// Get max concurrent services from config
	maxWorkers := 5
	if clusterLayer.Values != nil {
		if globals, ok := clusterLayer.Values["globals"].(map[string]interface{}); ok {
			if perf, ok := globals["performance"].(map[string]interface{}); ok {
				if max, ok := perf["maxConcurrentServices"].(int); ok && max > 0 {
					maxWorkers = max
				}
			}
		}
	}
	
	// Create worker pool
	pool := workers.NewWorkerPool(maxWorkers)
	
	// Create parallel migrator
	pm := &ParallelMigrator{
		factory:  f,
		services: services,
		cluster:  cluster,
		opts:     opts,
		pool:     pool,
		log:      logger.WithName("parallel-migrator"),
	}
	
	f.log.InfoS("Created parallel migrator",
		"services", len(services),
		"workers", maxWorkers,
		"cluster", cluster)
	
	return pm, nil
}

// LoadHierarchicalConfig loads configuration from a directory structure
func (f *HierarchicalMigratorFactory) LoadHierarchicalConfig(configDir string) error {
	// Load defaults
	defaultsPath := filepath.Join(configDir, "defaults.yaml")
	if defaults, err := f.loader.LoadFromFile(defaultsPath); err == nil {
		f.hierarchy.SetDefaults(f.configToMap(defaults.Globals))
		f.log.V(2).InfoS("Loaded defaults", "path", defaultsPath)
	}
	
	// Load globals
	globalsPath := filepath.Join(configDir, "globals.yaml")
	if globals, err := f.loader.LoadFromFile(globalsPath); err == nil {
		f.hierarchy.SetGlobals(f.configToMap(globals.Globals))
		f.log.V(2).InfoS("Loaded globals", "path", globalsPath)
	}
	
	// Load cluster configs
	clustersDir := filepath.Join(configDir, "clusters")
	if clusters, err := f.loader.LoadFromDirectory(clustersDir); err == nil {
		for name, cluster := range clusters.Clusters {
			f.hierarchy.SetClusterConfig(name, f.clusterToMap(cluster))
			f.log.V(2).InfoS("Loaded cluster config", "cluster", name)
		}
	}
	
	// Load service configs
	servicesDir := filepath.Join(configDir, "services")
	if services, err := f.loader.LoadFromDirectory(servicesDir); err == nil {
		for name, service := range services.Services {
			f.hierarchy.SetServiceConfig(name, f.serviceToMap(service))
			f.log.V(2).InfoS("Loaded service config", "service", name)
		}
	}
	
	f.log.InfoS("Loaded hierarchical configuration", "configDir", configDir)
	return nil
}

// Helper methods for conversion

func (f *HierarchicalMigratorFactory) configToMap(globals config.Globals) map[string]interface{} {
	// Convert using YAML marshaling for simplicity
	// In production, use proper struct-to-map conversion
	return map[string]interface{}{
		"converter": map[string]interface{}{
			"minUppercaseChars":  globals.Converter.MinUppercaseChars,
			"skipJavaProperties": globals.Converter.SkipJavaProperties,
			"skipUppercaseKeys":  globals.Converter.SkipUppercaseKeys,
		},
		"performance": map[string]interface{}{
			"maxConcurrentServices": globals.Performance.MaxConcurrentServices,
			"showProgress":         globals.Performance.ShowProgress,
		},
		"sops": map[string]interface{}{
			"enabled":         globals.SOPS.Enabled,
			"awsProfile":      globals.SOPS.AwsProfile,
			"parallelWorkers": globals.SOPS.ParallelWorkers,
			"timeout":         globals.SOPS.Timeout,
		},
	}
}

func (f *HierarchicalMigratorFactory) clusterToMap(cluster config.Cluster) map[string]interface{} {
	return map[string]interface{}{
		"source":           cluster.Source,
		"target":           cluster.Target,
		"default":          cluster.Default,
		"enabled":          cluster.Enabled,
		"defaultNamespace": cluster.DefaultNamespace,
		"awsProfile":       cluster.AWSProfile,
		"awsRegion":        cluster.AWSRegion,
	}
}

func (f *HierarchicalMigratorFactory) serviceToMap(service config.Service) map[string]interface{} {
	return map[string]interface{}{
		"enabled":     service.Enabled,
		"capitalized": service.Capitalized,
		"autoInject":  service.AutoInject,
		"secrets":     service.Secrets,
	}
}

func (f *HierarchicalMigratorFactory) layerToConfig(layer *config.ConfigLayer) (*config.Config, error) {
	// Convert layer values to Config struct
	// This is a simplified version - in production, use proper conversion
	cfg := &config.Config{
		Clusters: make(map[string]config.Cluster),
		Services: make(map[string]config.Service),
	}
	
	// Extract values from layer
	if layer.Values != nil {
		// Set cluster if present
		if cluster, ok := layer.Values["cluster"].(string); ok {
			cfg.Cluster = cluster
		}
		
		// Set namespace if present
		if namespace, ok := layer.Values["namespace"].(string); ok {
			cfg.Namespace = namespace
		}
		
		// Set globals if present
		if globals, ok := layer.Values["globals"].(map[string]interface{}); ok {
			cfg.Globals = f.mapToGlobals(globals)
		}
		
		// Set clusters if present
		if clusters, ok := layer.Values["clusters"].(map[string]interface{}); ok {
			for name, clusterData := range clusters {
				if clusterMap, ok := clusterData.(map[string]interface{}); ok {
					cfg.Clusters[name] = f.mapToCluster(clusterMap)
				}
			}
		}
		
		// Set services if present
		if services, ok := layer.Values["services"].(map[string]interface{}); ok {
			for name, serviceData := range services {
				if serviceMap, ok := serviceData.(map[string]interface{}); ok {
					cfg.Services[name] = f.mapToService(serviceMap)
				}
			}
		}
	}
	
	return cfg, nil
}

func (f *HierarchicalMigratorFactory) mapToGlobals(m map[string]interface{}) config.Globals {
	globals := config.Globals{}
	
	if converter, ok := m["converter"].(map[string]interface{}); ok {
		if val, ok := converter["minUppercaseChars"].(int); ok {
			globals.Converter.MinUppercaseChars = val
		}
		if val, ok := converter["skipJavaProperties"].(bool); ok {
			globals.Converter.SkipJavaProperties = val
		}
		if val, ok := converter["skipUppercaseKeys"].(bool); ok {
			globals.Converter.SkipUppercaseKeys = val
		}
	}
	
	if performance, ok := m["performance"].(map[string]interface{}); ok {
		if val, ok := performance["maxConcurrentServices"].(int); ok {
			globals.Performance.MaxConcurrentServices = val
		}
		if val, ok := performance["showProgress"].(bool); ok {
			globals.Performance.ShowProgress = val
		}
	}
	
	if sops, ok := m["sops"].(map[string]interface{}); ok {
		if val, ok := sops["enabled"].(bool); ok {
			globals.SOPS.Enabled = val
		}
		if val, ok := sops["awsProfile"].(string); ok {
			globals.SOPS.AwsProfile = val
		}
		if val, ok := sops["parallelWorkers"].(int); ok {
			globals.SOPS.ParallelWorkers = val
		}
		if val, ok := sops["timeout"].(int); ok {
			globals.SOPS.Timeout = val
		}
	}
	
	return globals
}

func (f *HierarchicalMigratorFactory) mapToCluster(m map[string]interface{}) config.Cluster {
	cluster := config.Cluster{}
	
	if val, ok := m["source"].(string); ok {
		cluster.Source = val
	}
	if val, ok := m["target"].(string); ok {
		cluster.Target = val
	}
	if val, ok := m["default"].(bool); ok {
		cluster.Default = val
	}
	if val, ok := m["enabled"].(bool); ok {
		cluster.Enabled = val
	}
	if val, ok := m["defaultNamespace"].(string); ok {
		cluster.DefaultNamespace = val
	}
	if val, ok := m["awsProfile"].(string); ok {
		cluster.AWSProfile = val
	}
	if val, ok := m["awsRegion"].(string); ok {
		cluster.AWSRegion = val
	}
	
	return cluster
}

func (f *HierarchicalMigratorFactory) mapToService(m map[string]interface{}) config.Service {
	service := config.Service{}
	
	if val, ok := m["enabled"].(bool); ok {
		service.Enabled = val
	}
	if val, ok := m["capitalized"].(string); ok {
		service.Capitalized = val
	}
	
	// Handle AutoInject and Secrets if needed
	
	return service
}

// ParallelMigrator handles parallel service migrations
type ParallelMigrator struct {
	factory  *HierarchicalMigratorFactory
	services []string
	cluster  string
	opts     MigratorOptions
	pool     *workers.WorkerPool
	log      *logger.NamedLogger
}

// Run executes parallel migrations
func (pm *ParallelMigrator) Run(ctx context.Context) error {
	// Start the worker pool
	if err := pm.pool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}
	defer pm.pool.Stop()
	
	// Create tasks for each service
	var tasks []workers.Task
	for _, serviceName := range pm.services {
		task := workers.NewServiceMigrationTask(
			serviceName,
			pm.cluster,
			pm.factory.baseConfig,
			nil, // These will be created per-task
			nil,
			nil,
			pm.opts.DryRun,
		)
		tasks = append(tasks, task)
	}
	
	// Submit tasks
	if err := pm.pool.SubmitBatch(tasks); err != nil {
		return fmt.Errorf("failed to submit migration tasks: %w", err)
	}
	
	// Wait for completion
	pm.pool.Wait()
	
	// Check results
	stats := pm.pool.Stats()
	pm.log.InfoS("Parallel migration completed",
		"total", stats.TotalTasks,
		"completed", stats.CompletedTasks,
		"failed", stats.FailedTasks)
	
	if stats.FailedTasks > 0 {
		return fmt.Errorf("%d services failed to migrate", stats.FailedTasks)
	}
	
	return nil
}

// SetTransformerRegistry sets a custom transformer registry on the migrator
func (m *Migrator) SetTransformerRegistry(registry *transformers.TransformerRegistry) {
	// This would be implemented in the actual Migrator struct
	// For now, just log that we would set it
	log := logger.WithName("migrator")
	log.V(2).InfoS("Would set transformer registry", "transformerCount", registry.Count())
}