package migration

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"helm.sh/helm/v3/pkg/release"

	"helm-charts-migrator/v1/pkg/adapters"
	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
)

// Migrator orchestrates the migration process using injected services
type Migrator struct {
	config      *config.Config
	kubernetes  services.KubernetesService
	helm        services.HelmService
	file        services.FileService
	transform   services.TransformationService
	cache       services.CacheService
	sops        services.SOPSService
	chartCopier adapters.ChartCopier
	extractor   adapters.ValuesExtractor
	fileManager adapters.FileManager
	pipeline    *adapters.TransformationPipeline
	log         *logger.NamedLogger
	dryRun      bool
	noSOPS      bool
}

// NewMigrator creates a new Migrator with all dependencies injected
func NewMigrator(
	cfg *config.Config,
	kubernetes services.KubernetesService,
	helm services.HelmService,
	file services.FileService,
	transform services.TransformationService,
	cache services.CacheService,
	sops services.SOPSService,
	dryRun bool,
	noSOPS bool,
) *Migrator {
	// Create adapter components
	chartCopier := adapters.NewChartCopier(cfg, file)
	extractor := adapters.NewValuesExtractor(cfg, transform)
	fileManager := adapters.NewFileManager(file)
	pipeline := adapters.NewTransformationPipeline(cfg, file, transform)

	return &Migrator{
		config:      cfg,
		kubernetes:  kubernetes,
		helm:        helm,
		file:        file,
		transform:   transform,
		cache:       cache,
		sops:        sops,
		chartCopier: chartCopier,
		extractor:   extractor,
		fileManager: fileManager,
		pipeline:    pipeline,
		log:         logger.WithName("migrator"),
		dryRun:      dryRun,
		noSOPS:      noSOPS,
	}
}

// MigrateServices migrates multiple services across clusters
func (m *Migrator) MigrateServices(ctx context.Context, services []string, clusters []ClusterInfo) error {
	if m.dryRun {
		m.log.InfoS("DRY RUN mode - no changes will be made")
	}

	// Check performance configuration
	maxWorkers := 1
	if m.config.Globals.Performance.MaxConcurrentServices > 0 {
		maxWorkers = m.config.Globals.Performance.MaxConcurrentServices
	}

	if maxWorkers > 1 {
		return m.processServicesParallel(ctx, services, clusters, maxWorkers)
	}

	// Sequential processing
	for _, serviceName := range services {
		if err := m.MigrateService(ctx, serviceName, clusters); err != nil {
			m.log.Error(err, "Failed to migrate service", "service", serviceName)
			return err
		}
	}

	return nil
}

// MigrateService migrates a single service across all clusters
func (m *Migrator) MigrateService(ctx context.Context, serviceName string, clusters []ClusterInfo) error {
	m.log.InfoS("Starting service migration", "service", serviceName, "clusters", len(clusters))
	startTime := time.Now()

	// Get service configuration
	serviceConfig := m.getServiceConfig(serviceName)
	if serviceConfig != nil && !serviceConfig.Enabled {
		m.log.InfoS("Service disabled in configuration, skipping", "service", serviceName)
		return nil
	}

	// Step 1: Copy base chart
	if err := m.copyBaseChart(serviceName, serviceConfig); err != nil {
		return fmt.Errorf("failed to copy base chart: %w", err)
	}

	// Step 2: Process Cluster (Helm Charts and Manifests)
	for _, cluster := range clusters {
		if err := m.processCluster(ctx, serviceName, cluster, serviceConfig); err != nil {
			m.log.Error(err, "Failed to process cluster", "cluster", cluster.Name)
			// Continue with other clusters
		}
	}

	// Step 3: Transform values files
	if err := m.pipeline.TransformService(serviceName); err != nil {
		m.log.Error(err, "Failed to transform service", "service", serviceName)
	}

	// Step 4: Encrypt secrets if not disabled
	if !m.noSOPS && !m.dryRun {
		if err := m.encryptServiceSecrets(serviceName); err != nil {
			m.log.Error(err, "Failed to encrypt secrets", "service", serviceName)
		}
	}

	duration := time.Since(startTime)
	m.log.InfoS("Service migration completed",
		"service", serviceName,
		"duration", duration.Round(time.Millisecond))

	return nil
}

// processCluster processes a single cluster for a service
func (m *Migrator) processCluster(ctx context.Context, serviceName string, cluster ClusterInfo, serviceConfig *config.Service) error {
	m.log.V(1).InfoS("Processing cluster", "service", serviceName, "cluster", cluster.Name)

	// Get releases from cluster
	releases, err := m.getReleases(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get releases: %w", err)
	}

	// Find service release
	serviceRelease := m.helm.GetReleaseByName(serviceName, releases)
	if serviceRelease == nil {
		m.log.InfoS("Service not found in cluster, skipping",
			"service", serviceName,
			"cluster", cluster.Name)
		return nil
	}

	// Extract and save values for each namespace
	for _, ns := range cluster.Namespaces {
		if err := m.processNamespace(ctx, serviceName, cluster, ns, serviceRelease); err != nil {
			m.log.Error(err, "Failed to process namespace",
				"namespace", ns.Name,
				"cluster", cluster.Name)
		}
	}

	return nil
}

// processNamespace processes a single namespace
func (m *Migrator) processNamespace(ctx context.Context, serviceName string, cluster ClusterInfo, ns NamespaceInfo, release *release.Release) error {
	// Build output path using centralized path management
	paths := config.NewPaths("", "apps", ".cache").
		ForService(serviceName).
		ForCluster(cluster.Name).
		ForEnvironment(ns.Environment, ns.Name)
	outputPath := paths.EnvironmentNamespaceDir()

	if m.dryRun {
		m.log.InfoS("DRY RUN: Would save values", "path", outputPath)
		return nil
	}

	// Extract values
	values, err := m.helm.ExtractValues(release)
	if err != nil {
		return fmt.Errorf("failed to extract values: %w", err)
	}

	// Transform values
	transformConfig := services.TransformConfig{
		ServiceName:   serviceName,
		ClusterName:   cluster.Name,
		Namespace:     ns.Name,
		GlobalConfig:  m.config.Globals,
		ServiceConfig: m.getServiceConfig(serviceName),
	}

	transformedValues, err := m.transform.Transform(values, transformConfig)
	if err != nil {
		return fmt.Errorf("failed to transform values: %w", err)
	}

	// Save values
	valuesPath := filepath.Join(outputPath, "values.yaml")
	if err := m.file.WriteYAML(valuesPath, transformedValues); err != nil {
		return fmt.Errorf("failed to save values: %w", err)
	}

	// Extract and save manifest if available
	if manifest, err := m.helm.ExtractManifest(release); err == nil && manifest != "" {
		manifestPath := filepath.Join(outputPath, "manifest.yaml")
		if err := m.file.WriteYAML(manifestPath, manifest); err != nil {
			m.log.Error(err, "Failed to save manifest", "path", manifestPath)
		}
	}

	m.log.V(2).InfoS("Processed namespace",
		"service", serviceName,
		"cluster", cluster.Name,
		"namespace", ns.Name,
		"path", outputPath)

	return nil
}

// getReleases gets releases from cache or cluster
func (m *Migrator) getReleases(ctx context.Context, cluster ClusterInfo) ([]*release.Release, error) {
	// Try cache first
	if cached := m.cache.GetReleases(cluster.Name, cluster.DefaultNamespace); cached != nil {
		m.log.V(2).InfoS("Using cached releases", "cluster", cluster.Name)
		return cached, nil
	}

	// Fetch from cluster
	m.log.V(1).InfoS("Fetching releases from cluster", "cluster", cluster.Name)
	releases, err := m.kubernetes.ListReleases(ctx, cluster.Context, cluster.DefaultNamespace)
	if err != nil {
		return nil, err
	}

	// Cache for future use
	if err := m.cache.SetReleases(cluster.Name, cluster.DefaultNamespace, releases); err != nil {
		m.log.Error(err, "Failed to cache releases", "cluster", cluster.Name)
	}

	return releases, nil
}

// copyBaseChart copies the base chart template for the service
func (m *Migrator) copyBaseChart(serviceName string, serviceConfig *config.Service) error {
	if m.dryRun {
		m.log.InfoS("DRY RUN: Would copy base chart", "service", serviceName)
		return nil
	}

	// If no service config provided, create a minimal one
	if serviceConfig == nil {
		caser := cases.Title(language.English)
		serviceConfig = &config.Service{
			Name:        serviceName,
			Capitalized: caser.String(serviceName),
		}
	} else {
		// Ensure Name is set
		if serviceConfig.Name == "" {
			serviceConfig.Name = serviceName
		}
		// Generate Capitalized if not provided
		if serviceConfig.Capitalized == "" {
			caser := cases.Title(language.English)
			serviceConfig.Capitalized = caser.String(serviceName)
		}
	}

	src := filepath.Join("migration", "base-chart")
	paths := config.NewPaths("", "apps", ".cache").ForService(serviceName)
	dst := paths.ServiceDir()

	return m.chartCopier.CopyBaseChartWithService(src, dst, serviceConfig)
}

// encryptServiceSecrets encrypts all secret files for a service
func (m *Migrator) encryptServiceSecrets(serviceName string) error {
	paths := config.NewPaths("", "apps", ".cache").ForService(serviceName)
	secretsDir := paths.EnvsDir()

	// Find all .dec.yaml files
	secretFiles, err := m.file.ListFiles(secretsDir, "secrets.dec.yaml")
	if err != nil {
		return fmt.Errorf("failed to list secret files: %w", err)
	}

	if len(secretFiles) == 0 {
		m.log.V(2).InfoS("No secret files to encrypt", "service", serviceName)
		return nil
	}

	// Encrypt in parallel
	workers := 5
	if m.config.Globals.SOPS.ParallelWorkers > 0 {
		workers = m.config.Globals.SOPS.ParallelWorkers
	}

	return m.sops.EncryptBatch(secretFiles, workers)
}

// processServicesParallel processes services in parallel
func (m *Migrator) processServicesParallel(ctx context.Context, services []string, clusters []ClusterInfo, maxWorkers int) error {
	var (
		wg             sync.WaitGroup
		sem            = make(chan struct{}, maxWorkers)
		completedCount int32
		failedCount    int32
	)

	totalServices := len(services)
	m.log.InfoS("Starting parallel service migration",
		"services", totalServices,
		"workers", maxWorkers)

	startTime := time.Now()

	for _, serviceName := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Process service
			if err := m.MigrateService(ctx, svc, clusters); err != nil {
				atomic.AddInt32(&failedCount, 1)
				m.log.Error(err, "Failed to migrate service", "service", svc)
			} else {
				completed := atomic.AddInt32(&completedCount, 1)
				m.log.InfoS("Service migration completed",
					"service", svc,
					"progress", fmt.Sprintf("%d/%d", completed, totalServices))
			}
		}(serviceName)
	}

	wg.Wait()

	duration := time.Since(startTime)
	m.log.InfoS("Parallel migration completed",
		"duration", duration.Round(time.Millisecond),
		"total", totalServices,
		"completed", atomic.LoadInt32(&completedCount),
		"failed", atomic.LoadInt32(&failedCount))

	if failedCount > 0 {
		return fmt.Errorf("%d services failed to migrate", failedCount)
	}

	return nil
}

// getServiceConfig returns the configuration for a service
func (m *Migrator) getServiceConfig(serviceName string) *config.Service {
	if m.config.Services == nil {
		return nil
	}

	if cfg, exists := m.config.Services[serviceName]; exists {
		return &cfg
	}

	return nil
}

// determineDefaultEnvironment determines the default environment for a cluster
// Note: With the new structure, environment is derived from namespace paths
func (m *Migrator) determineDefaultEnvironment(cluster config.Cluster) string {
	// Check if default_namespace exists and extract environment from its path
	if cluster.DefaultNamespace != "" {
		for nsName := range cluster.Namespaces {
			if nsName == cluster.DefaultNamespace {
				// Environment can be derived from namespace structure if needed
				// For now, return production as default
				return "production"
			}
		}
	}

	// Default to production
	return "production"
}

// Run implements the MigratorRunner interface
func (m *Migrator) Run(ctx context.Context) error {
	// Get enabled enabledServices from config
	enabledServices := m.getEnabledServices()
	if len(enabledServices) == 0 {
		return fmt.Errorf("no enabled enabledServices found in configuration")
	}

	// Get enabled clusters from config
	clusters, err := m.getEnabledClusters()
	if err != nil {
		return fmt.Errorf("failed to get clusters: %w", err)
	}

	m.log.InfoS("Starting migration",
		"enabledServices", len(enabledServices),
		"clusters", len(clusters))

	// Migrate all enabledServices
	if err := m.MigrateServices(ctx, enabledServices, clusters); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Cleanup cache if needed
	defer func() {
		if err := m.cache.Cleanup(); err != nil {
			m.log.Error(err, "Failed to cleanup cache")
		}
	}()

	return nil
}

// getEnabledServices returns list of enabled services from config
func (m *Migrator) getEnabledServices() []string {
	var enabledServices []string

	if m.config.Services != nil {
		for name, svc := range m.config.Services {
			if svc.Enabled {
				enabledServices = append(enabledServices, name)
			}
		}
	}

	return enabledServices
}

// getEnabledClusters returns list of enabled clusters from config
func (m *Migrator) getEnabledClusters() ([]ClusterInfo, error) {
	var clusters []ClusterInfo

	if m.config.Accounts == nil || len(m.config.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts configured")
	}

	for _, account := range m.config.Accounts {
		for name, cluster := range account.Clusters {
			if !cluster.Enabled {
				continue
			}

			info := ClusterInfo{
				Name:             name,
				Context:          cluster.Source, // Use Source to fetch legacy Helm releases
				DefaultNamespace: cluster.DefaultNamespace,
				IsDefault:        cluster.Default,
			}

			// Add enabled namespaces
			for nsName, ns := range cluster.Namespaces {
				if ns.Enabled {
					// Environment can be derived from paths or default to production
					info.Namespaces = append(info.Namespaces, NamespaceInfo{
						Name:        nsName,
						Environment: "production", // Default environment
					})
				}
			}

			m.log.V(2).InfoS("Cluster configured",
				"cluster", name,
				"defaultNamespace", info.DefaultNamespace,
				"namespaces", len(info.Namespaces))

			clusters = append(clusters, info)
		}
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no enabled clusters found")
	}

	return clusters, nil
}

// ClusterInfo holds cluster configuration
type ClusterInfo struct {
	Name               string
	Context            string
	DefaultNamespace   string
	DefaultEnvironment string
	Namespaces         []NamespaceInfo
	IsDefault          bool
}

// NamespaceInfo holds namespace configuration
type NamespaceInfo struct {
	Name        string
	Environment string
}
