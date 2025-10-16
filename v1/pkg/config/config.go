package config

import (
	"fmt"
	"os"

	"helm-charts-migrator/v1/pkg/logger"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// Config represents the main configuration structure
type Config struct {
	Accounts  map[string]Account `yaml:"accounts"`
	Services  map[string]Service `yaml:"services"`
	Globals   Globals            `yaml:"globals"`
	Cluster   string             `yaml:"cluster"`
	Namespace string             `yaml:"namespace"`

	// Paths provides centralized path management (not from YAML)
	Paths *Paths `yaml:"-"`
}

// Account represents an AWS account with its clusters
type Account struct {
	Clusters map[string]Cluster `yaml:"clusters"`
}

// Globals represents global configuration that applies to all services
type Globals struct {
	Pipeline    PipelineConfig            `yaml:"pipeline,omitempty"`
	Converter   ConverterConfig           `yaml:"converter,omitempty"`
	Performance PerformanceConfig         `yaml:"performance,omitempty"`
	SOPS        SOPSConfig                `yaml:"sops,omitempty"`
	AutoInject  map[string]AutoInjectFile `yaml:"autoInject,omitempty"`
	Mappings    *Mappings                 `yaml:"mappings,omitempty"`
	Secrets     *Secrets                  `yaml:"secrets,omitempty"`
	Migration   Migration                 `yaml:"migration"`
}

// PipelineConfig represents migration pipeline configuration
type PipelineConfig struct {
	Enabled bool           `yaml:"enabled"`
	Steps   []PipelineStep `yaml:"steps"`
}

// PipelineStep represents a single step in the migration pipeline
type PipelineStep struct {
	Name        string `yaml:"name"`
	Enabled     bool   `yaml:"enabled"`
	Description string `yaml:"description"`
}

// PerformanceConfig represents performance tuning configuration
type PerformanceConfig struct {
	MaxConcurrentServices int  `yaml:"maxConcurrentServices"`
	ShowProgress          bool `yaml:"showProgress"`
}

// SOPSConfig represents SOPS encryption/decryption configuration
type SOPSConfig struct {
	Enabled         bool   `yaml:"enabled"`
	AwsProfile      string `yaml:"awsProfile"`
	ParallelWorkers int    `yaml:"parallelWorkers"`
	ConfigFile      string `yaml:"configFile"`
	PathRegex       string `yaml:"pathRegex"`
	SkipUnchanged   bool   `yaml:"skipUnchanged"`
	Timeout         int    `yaml:"timeout"`
}

// ConverterConfig represents configuration for the camelCase converter
type ConverterConfig struct {
	SkipJavaProperties bool `yaml:"skipJavaProperties"`
	SkipUppercaseKeys  bool `yaml:"skipUppercaseKeys"`
	MinUppercaseChars  int  `yaml:"minUppercaseChars"`
}

// SetPaths initializes the Paths structure with the provided paths
func (c *Config) SetPaths(sourcePath, targetPath, cacheDir string) {
	c.Paths = NewPaths(sourcePath, targetPath, cacheDir)
}

// GetServicePaths returns paths configured for a specific service
func (c *Config) GetServicePaths(serviceName string) *Paths {
	if c.Paths == nil {
		return nil
	}
	return c.Paths.ForService(serviceName)
}

func LoadConfig(configPath string) (*Config, error) {
	log := logger.WithName("config")
	log.InfoS("Loading configuration", "path", configPath)

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Use gopkg.in/yaml.v3 directly to unmarshal
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Count total clusters across all accounts
	totalClusters := 0
	for _, account := range config.Accounts {
		totalClusters += len(account.Clusters)
	}

	log.InfoS("Configuration loaded successfully",
		"accounts", len(config.Accounts),
		"clusters", totalClusters,
		"services", len(config.Services),
		"cluster", config.Cluster,
		"namespace", config.Namespace)

	return &config, nil
}

func (c *Config) GetEnabledClusters() []string {
	var clusters []string
	for _, account := range c.Accounts {
		for name, cluster := range account.Clusters {
			if cluster.Enabled {
				clusters = append(clusters, name)
			}
		}
	}
	return clusters
}

func (c *Config) GetEnabledServices() []string {
	var services []string
	for name, service := range c.Services {
		if service.Enabled {
			services = append(services, name)
		}
	}
	return services
}

func (c *Config) GetEnabledNamespaces(clusterName string) []string {
	cluster := c.GetCluster(clusterName)
	if cluster == nil {
		return nil
	}

	var namespaces []string
	for _, ns := range cluster.Namespaces {
		if ns.Enabled {
			namespaces = append(namespaces, ns.Name)
		}
	}
	return namespaces
}

func (c *Config) GetDefaultCluster() (string, *Cluster) {
	for _, account := range c.Accounts {
		for name, cluster := range account.Clusters {
			if cluster.Default {
				return name, &cluster
			}
		}
	}
	return "", nil
}

// GetCluster retrieves a cluster by name from any account
func (c *Config) GetCluster(clusterName string) *Cluster {
	for _, account := range c.Accounts {
		if cluster, exists := account.Clusters[clusterName]; exists {
			return &cluster
		}
	}
	return nil
}

// GetEnabledNamespacesForCluster returns enabled namespaces for a cluster
func (c *Config) GetEnabledNamespacesForCluster(clusterName string) []Namespace {
	cluster := c.GetCluster(clusterName)
	if cluster == nil {
		return nil
	}

	var namespaces []Namespace
	for _, ns := range cluster.Namespaces {
		if ns.Enabled {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces
}

// GetMergedServiceConfig merges global configuration with service-specific configuration
func (c *Config) GetMergedServiceConfig(serviceName string) (*Service, []string) {
	service, exists := c.Services[serviceName]
	if !exists {
		return nil, []string{"Service not found"}
	}

	// Use ConfigurationMerger to handle the merging logic
	merger := NewConfigMerger()
	return merger.MergeServiceConfig(c.Globals, service)
}
