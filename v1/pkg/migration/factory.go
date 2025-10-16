package migration

import (
	"context"
	"fmt"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
)

// MigratorOptions contains configuration options for migration
type MigratorOptions struct {
	ConfigPath   string
	SourcePath   string
	TargetPath   string
	BasePath     string
	CacheDir     string
	CleanupCache bool
	RefreshCache bool
	DryRun       bool
	// Override options from CLI
	Cluster    string
	Namespaces []string
	Services   []string
	AwsProfile string
	NoSOPS     bool // Skip SOPS encryption when true
}

// MigratorFactory creates migrators with proper dependencies - Factory Pattern
type MigratorFactory struct {
	config *config.Config
	log    *logger.NamedLogger
}

// NewMigratorFactory creates a new migrator factory
func NewMigratorFactory(cfg *config.Config) *MigratorFactory {
	return &MigratorFactory{
		config: cfg,
		log:    logger.WithName("migration-factory"),
	}
}

// CreateMigrator creates a migrator based on the provided options
func (f *MigratorFactory) CreateMigrator(opts MigratorOptions) (*Migrator, error) {
	// Create services with dependency injection
	kubernetes := services.NewKubernetesService()
	helm := services.NewHelmService()
	file := services.NewFileService()
	transform := services.NewTransformationService(f.config)
	
	// Create cache service
	cache, err := services.NewCacheService(opts.CacheDir, opts.CleanupCache)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache service: %w", err)
	}
	
	// Create SOPS service
	sopsConfig := &f.config.Globals.SOPS
	// Set defaults if needed
	if sopsConfig.ParallelWorkers == 0 {
		sopsConfig.ParallelWorkers = 5
	}
	if sopsConfig.Timeout == 0 {
		sopsConfig.Timeout = 30
	}
	sops := services.NewSOPSService(sopsConfig)
	
	// Create migrator with all dependencies
	migrator := NewMigrator(
		f.config,
		kubernetes,
		helm,
		file,
		transform,
		cache,
		sops,
		opts.DryRun,
		opts.NoSOPS,
	)
	
	f.log.V(2).InfoS("Created migrator with dependency injection", 
		"dryRun", opts.DryRun,
		"noSOPS", opts.NoSOPS,
		"cacheDir", opts.CacheDir)
	
	return migrator, nil
}

// MigratorRunner interface for running migrations - Interface Segregation Principle
type MigratorRunner interface {
	Run(ctx context.Context) error
}

// RunMigrationWithFactory is the entry point using the factory pattern
func RunMigrationWithFactory(opts MigratorOptions) error {
	log := logger.WithName("migration")
	log.Info("Starting migration process")

	// Set default config path if not provided
	if opts.ConfigPath == "" {
		opts.ConfigPath = "config.yaml"
	}

	// Load configuration
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize paths in config
	cfg.SetPaths(opts.SourcePath, opts.TargetPath, opts.CacheDir)

	// Create factory and migrator
	factory := NewMigratorFactory(cfg)
	migrator, err := factory.CreateMigrator(opts)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Override with CLI options if provided
	if len(opts.Services) > 0 {
		// Filter to only specified services
		for name := range cfg.Services {
			found := false
			for _, svc := range opts.Services {
				if name == svc {
					found = true
					break
				}
			}
			if !found {
				// Disable services not in the list
				s := cfg.Services[name]
				s.Enabled = false
				cfg.Services[name] = s
			}
		}
	}

	if opts.Cluster != "" {
		// Disable all clusters except the specified one
		for accountName, account := range cfg.Accounts {
			for clusterName := range account.Clusters {
				if clusterName != opts.Cluster {
					cluster := account.Clusters[clusterName]
					cluster.Enabled = false
					account.Clusters[clusterName] = cluster
				}
			}
			cfg.Accounts[accountName] = account
		}
	}

	// Run migration
	ctx := context.Background()
	if err := migrator.Run(ctx); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Info("Migration completed successfully")
	return nil
}
