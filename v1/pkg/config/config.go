package config

import (
	"fmt"

	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/yaml"
)

// Config represents the main configuration structure
type Config struct {
	Clusters  map[string]Cluster `yaml:"clusters"`
	Services  map[string]Service `yaml:"services"`
	Globals   Globals            `yaml:"globals"`
	Cluster   string             `yaml:"cluster"`
	Namespace string             `yaml:"namespace"`

	// Paths provides centralized path management (not from YAML)
	Paths *Paths `yaml:"-"`
}

// Globals represents global configuration that applies to all services
type Globals struct {
	Converter   ConverterConfig           `yaml:"converter,omitempty"`
	Performance PerformanceConfig         `yaml:"performance,omitempty"`
	SOPS        SOPSConfig                `yaml:"sops,omitempty"`
	AutoInject  map[string]AutoInjectFile `yaml:"autoInject,omitempty"`
	Mappings    *Mappings                 `yaml:"mappings,omitempty"`
	Secrets     *Secrets                  `yaml:"secrets,omitempty"`
	Migration   Migration                 `yaml:"migration"`
}

// PerformanceConfig represents performance tuning configuration
type PerformanceConfig struct {
	MaxConcurrentServices int  `yaml:"maxConcurrentServices"`
	ShowProgress         bool `yaml:"showProgress"`
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

	// Use yaml to load the config file with proper formatting
	doc, err := yaml.LoadFile(configPath, yaml.DefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Convert to map and then to Config struct
	configMap, err := doc.ToMap()
	if err != nil {
		return nil, fmt.Errorf("failed to convert config to map: %w", err)
	}

	// Marshal and unmarshal to convert map to struct
	// This is a workaround since yaml doesn't directly decode to struct
	yamlBytes, err := yaml.MarshalWithOptions(configMap, yaml.DefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config map: %w", err)
	}

	var config Config
	if err := yaml.UnmarshalStrict(yamlBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	log.InfoS("Configuration loaded successfully",
		"clusters", len(config.Clusters),
		"services", len(config.Services),
		"cluster", config.Cluster,
		"namespace", config.Namespace)

	return &config, nil
}

func (c *Config) GetEnabledClusters() []string {
	var clusters []string
	for name, cluster := range c.Clusters {
		if cluster.Enabled {
			clusters = append(clusters, name)
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
	cluster, exists := c.Clusters[clusterName]
	if !exists {
		return nil
	}

	var namespaces []string
	for _, env := range cluster.Environments {
		for _, ns := range env.Namespaces {
			if ns.Enabled {
				namespaces = append(namespaces, ns.Name)
			}
		}
	}
	return namespaces
}

func (c *Config) GetDefaultCluster() (string, *Cluster) {
	for name, cluster := range c.Clusters {
		if cluster.Default {
			return name, &cluster
		}
	}
	return "", nil
}

// GetEnabledEnvironmentsWithNamespaces returns a map of environments with their enabled namespaces
func (c *Config) GetEnabledEnvironmentsWithNamespaces(clusterName string) map[string][]Namespace {
	cluster, exists := c.Clusters[clusterName]
	if !exists {
		return nil
	}

	result := make(map[string][]Namespace)
	for envName, env := range cluster.Environments {
		// Skip disabled environments
		if !env.Enabled {
			continue
		}

		var enabledNamespaces []Namespace
		for _, ns := range env.Namespaces {
			if ns.Enabled {
				enabledNamespaces = append(enabledNamespaces, ns)
			}
		}
		if len(enabledNamespaces) > 0 {
			result[envName] = enabledNamespaces
		}
	}
	return result
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
