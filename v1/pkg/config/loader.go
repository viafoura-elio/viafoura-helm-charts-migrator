package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/yaml"
)

// ConfigLoader handles loading and merging configurations
type ConfigLoader interface {
	LoadFromFile(path string) (*Config, error)
	LoadFromDirectory(dir string) (*Config, error)
	LoadFromCluster(context string) (*Config, error)
	MergeConfigs(configs ...*Config) *Config
	LoadHierarchicalConfig(baseDir string) (*HierarchicalConfig, error)
}

// configLoader implements ConfigLoader
type configLoader struct {
	log *logger.NamedLogger
}

// NewConfigLoader creates a new ConfigLoader
func NewConfigLoader() ConfigLoader {
	return &configLoader{
		log: logger.WithName("config-loader"),
	}
}

// LoadFromFile loads configuration from a single file
func (c *configLoader) LoadFromFile(path string) (*Config, error) {
	c.log.V(2).InfoS("Loading config from file", "path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults
	c.applyDefaults(&config)

	c.log.InfoS("Loaded configuration",
		"clusters", len(config.Clusters),
		"services", len(config.Services))

	return &config, nil
}

// LoadFromDirectory loads all config files from a directory
func (c *configLoader) LoadFromDirectory(dir string) (*Config, error) {
	c.log.V(2).InfoS("Loading configs from directory", "dir", dir)

	// Start with empty config
	merged := &Config{
		Clusters: make(map[string]Cluster),
		Services: make(map[string]Service),
	}

	// Find all YAML files
	pattern := filepath.Join(dir, "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list config files: %w", err)
	}

	// Also check for .yml files
	pattern = filepath.Join(dir, "*.yml")
	ymlFiles, err := filepath.Glob(pattern)
	if err == nil {
		files = append(files, ymlFiles...)
	}

	// Load and merge each file
	for _, file := range files {
		c.log.V(3).InfoS("Loading config file", "file", file)

		cfg, err := c.LoadFromFile(file)
		if err != nil {
			c.log.Error(err, "Failed to load config file", "file", file)
			continue
		}

		merged = c.MergeConfigs(merged, cfg)
	}

	// Check subdirectories for specific configs
	subdirs := []string{"clusters", "services", "environments"}
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(dir, subdir)
		if info, err := os.Stat(subdirPath); err == nil && info.IsDir() {
			subdirConfig, err := c.loadSubdirectoryConfigs(subdirPath, subdir)
			if err != nil {
				c.log.Error(err, "Failed to load subdirectory configs", "subdir", subdirPath)
				continue
			}
			merged = c.MergeConfigs(merged, subdirConfig)
		}
	}

	c.log.InfoS("Loaded configuration from directory",
		"dir", dir,
		"clusters", len(merged.Clusters),
		"services", len(merged.Services))

	return merged, nil
}

// LoadFromCluster loads configuration from a Kubernetes cluster
func (c *configLoader) LoadFromCluster(context string) (*Config, error) {
	c.log.V(2).InfoS("Loading config from cluster", "context", context)

	// This would connect to the cluster and load ConfigMaps/Secrets
	// For now, return an error indicating it's not implemented
	return nil, fmt.Errorf("loading config from cluster not yet implemented")
}

// MergeConfigs merges multiple configurations together
func (c *configLoader) MergeConfigs(configs ...*Config) *Config {
	if len(configs) == 0 {
		return &Config{
			Clusters: make(map[string]Cluster),
			Services: make(map[string]Service),
		}
	}

	// Start with the first config
	merged := configs[0]
	if merged == nil {
		merged = &Config{
			Clusters: make(map[string]Cluster),
			Services: make(map[string]Service),
		}
	}

	// Ensure maps are initialized
	if merged.Clusters == nil {
		merged.Clusters = make(map[string]Cluster)
	}
	if merged.Services == nil {
		merged.Services = make(map[string]Service)
	}

	// Merge remaining configs
	for i := 1; i < len(configs); i++ {
		cfg := configs[i]
		if cfg == nil {
			continue
		}

		// Merge globals
		merged.Globals = c.mergeGlobals(merged.Globals, cfg.Globals)

		// Merge clusters
		for name, cluster := range cfg.Clusters {
			if existing, exists := merged.Clusters[name]; exists {
				merged.Clusters[name] = c.mergeCluster(existing, cluster)
			} else {
				merged.Clusters[name] = cluster
			}
		}

		// Merge services
		for name, service := range cfg.Services {
			if existing, exists := merged.Services[name]; exists {
				merged.Services[name] = c.mergeService(existing, service)
			} else {
				merged.Services[name] = service
			}
		}
	}

	c.log.V(3).InfoS("Merged configurations",
		"count", len(configs),
		"clusters", len(merged.Clusters),
		"services", len(merged.Services))

	return merged
}

// LoadHierarchicalConfig loads a hierarchical configuration structure
func (c *configLoader) LoadHierarchicalConfig(baseDir string) (*HierarchicalConfig, error) {
	c.log.InfoS("Loading hierarchical configuration", "baseDir", baseDir)

	hc := NewHierarchicalConfig()

	// Load from directory structure
	if err := hc.LoadFromDirectory(baseDir); err != nil {
		return nil, fmt.Errorf("failed to load hierarchical config: %w", err)
	}

	return hc, nil
}

// Helper functions

func (c *configLoader) applyDefaults(config *Config) {
	// Apply default global settings
	if config.Globals.Migration.HelmValuesFilename == "" {
		config.Globals.Migration.HelmValuesFilename = "values.yaml"
	}
	if config.Globals.Migration.LegacyValuesFilename == "" {
		config.Globals.Migration.LegacyValuesFilename = "legacy-values.yaml"
	}
	if config.Globals.Converter.MinUppercaseChars == 0 {
		config.Globals.Converter.MinUppercaseChars = 3
	}

	// Apply defaults to clusters
	for name, cluster := range config.Clusters {
		if cluster.DefaultNamespace == "" {
			cluster.DefaultNamespace = "default"
		}
		config.Clusters[name] = cluster
	}

	// Apply defaults to services
	for name, service := range config.Services {
		if service.Name == "" {
			service.Name = name
		}
		if service.Capitalized == "" {
			service.Capitalized = c.capitalize(name)
		}
		config.Services[name] = service
	}
}

func (c *configLoader) loadSubdirectoryConfigs(dir, configType string) (*Config, error) {
	merged := &Config{
		Clusters: make(map[string]Cluster),
		Services: make(map[string]Service),
	}

	// Find all YAML files in subdirectory
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	ymlFiles, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err == nil {
		files = append(files, ymlFiles...)
	}

	for _, file := range files {
		name := getNameFromFile(file)

		data, err := os.ReadFile(file)
		if err != nil {
			c.log.Error(err, "Failed to read file", "file", file)
			continue
		}

		switch configType {
		case "clusters":
			var cluster Cluster
			if err := yaml.Unmarshal(data, &cluster); err != nil {
				c.log.Error(err, "Failed to unmarshal cluster config", "file", file)
				continue
			}
			merged.Clusters[name] = cluster

		case "services":
			var service Service
			if err := yaml.Unmarshal(data, &service); err != nil {
				c.log.Error(err, "Failed to unmarshal service config", "file", file)
				continue
			}
			if service.Name == "" {
				service.Name = name
			}
			if service.Capitalized == "" {
				service.Capitalized = c.capitalize(name)
			}
			merged.Services[name] = service
		}
	}

	return merged, nil
}

func (c *configLoader) mergeGlobals(base, override Globals) Globals {
	// Start with base
	result := base

	// Override migration settings
	if override.Migration.HelmValuesFilename != "" {
		result.Migration.HelmValuesFilename = override.Migration.HelmValuesFilename
	}
	if override.Migration.LegacyValuesFilename != "" {
		result.Migration.LegacyValuesFilename = override.Migration.LegacyValuesFilename
	}
	if override.Migration.BaseValuesPath != "" {
		result.Migration.BaseValuesPath = override.Migration.BaseValuesPath
	}
	if override.Migration.EnvValuesPattern != "" {
		result.Migration.EnvValuesPattern = override.Migration.EnvValuesPattern
	}

	// Override converter settings
	if override.Converter.MinUppercaseChars > 0 {
		result.Converter.MinUppercaseChars = override.Converter.MinUppercaseChars
	}
	result.Converter.SkipJavaProperties = override.Converter.SkipJavaProperties || base.Converter.SkipJavaProperties
	result.Converter.SkipUppercaseKeys = override.Converter.SkipUppercaseKeys || base.Converter.SkipUppercaseKeys

	// Override performance settings
	if override.Performance.MaxConcurrentServices > 0 {
		result.Performance.MaxConcurrentServices = override.Performance.MaxConcurrentServices
	}

	return result
}

func (c *configLoader) mergeCluster(base, override Cluster) Cluster {
	result := base

	if override.Target != "" {
		result.Target = override.Target
	}
	if override.Source != "" {
		result.Source = override.Source
	}
	if override.AWSProfile != "" {
		result.AWSProfile = override.AWSProfile
	}
	if override.AWSRegion != "" {
		result.AWSRegion = override.AWSRegion
	}
	if override.DefaultNamespace != "" {
		result.DefaultNamespace = override.DefaultNamespace
	}

	// Merge environments
	if result.Environments == nil {
		result.Environments = make(map[string]Environment)
	}
	for name, env := range override.Environments {
		if existing, exists := result.Environments[name]; exists {
			result.Environments[name] = c.mergeEnvironment(existing, env)
		} else {
			result.Environments[name] = env
		}
	}

	// Override enabled flag explicitly
	result.Enabled = override.Enabled
	result.Default = override.Default

	return result
}

func (c *configLoader) mergeEnvironment(base, override Environment) Environment {
	result := base

	// Merge namespaces
	if result.Namespaces == nil {
		result.Namespaces = make(map[string]Namespace)
	}
	for name, ns := range override.Namespaces {
		result.Namespaces[name] = ns
	}

	result.Enabled = override.Enabled

	return result
}

func (c *configLoader) mergeService(base, override Service) Service {
	result := base

	if override.Name != "" {
		result.Name = override.Name
	}
	if override.Capitalized != "" {
		result.Capitalized = override.Capitalized
	}

	// Merge auto-inject rules
	if len(override.AutoInject) > 0 {
		result.AutoInject = override.AutoInject
	}

	// Merge secrets config
	if override.Secrets != nil {
		result.Secrets = override.Secrets
	}

	result.Enabled = override.Enabled

	return result
}

func (c *configLoader) capitalize(s string) string {
	if s == "" {
		return s
	}

	// Handle hyphenated names
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	return strings.Join(parts, "")
}
