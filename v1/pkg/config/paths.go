package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathType represents the type of path being constructed
type PathType string

const (
	PathTypeTarget    PathType = "target"
	PathTypeBaseChart PathType = "basechart"
	PathTypeCache     PathType = "cache"
	PathTypeSource    PathType = "source"
)

// PathBuilder follows the Builder pattern for constructing paths
type PathBuilder interface {
	// ServiceDir Core directory builders
	ServiceDir() string
	EnvsDir() string
	TemplatesDir() string

	// EnvironmentDir Environment hierarchy builders (account/cluster/namespace pattern)
	EnvironmentDir() string           // envs/{environment}
	EnvironmentClustersDir() string   // envs/{environment}/clusters
	EnvironmentClusterDir() string    // envs/{environment}/clusters/{cluster}
	EnvironmentNamespacesDir() string // envs/{environment}/clusters/{cluster}/namespaces
	EnvironmentNamespaceDir() string  // envs/{environment}/clusters/{cluster}/namespaces/{namespace}

	// ValuesPath Values and config file paths
	ValuesPath() string
	EnvironmentValuesPath() string
	EnvironmentClusterValuesPath() string
	EnvironmentNamespaceValuesPath() string
	ChartPath() string

	// ManifestPath Specific file paths for common operations
	ManifestPath() string
	SecretsPath() string
	EnvironmentSecretsPath() string
	EnvironmentClusterSecretsPath() string
	EnvironmentNamespaceSecretsPath() string

	// ForType Factory method to change path type
	ForType(pathType PathType) PathBuilder
}

// Paths provides centralized path management for the migration process
type Paths struct {
	// Base paths
	SourcePath string // Legacy charts source path
	TargetPath string // Migration target path
	CacheDir   string // Cache directory for releases

	// Service-specific paths (built dynamically)
	serviceName string
	clusterName string
	environment string
	namespace   string

	// Configuration
	migration Migration
	pathType  PathType
}

// NewPaths creates a new Paths instance with base paths
func NewPaths(sourcePath, targetPath, cacheDir string) *Paths {
	return &Paths{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		CacheDir:   cacheDir,
		pathType:   PathTypeTarget, // Default to target paths
	}
}

// WithMigration sets the migration configuration for base-chart paths
func (p *Paths) WithMigration(migration Migration) *Paths {
	newPaths := *p
	newPaths.migration = migration
	return &newPaths
}

// ForService returns a new Paths instance configured for a specific service
func (p *Paths) ForService(serviceName string) *Paths {
	newPaths := *p
	newPaths.serviceName = serviceName
	return &newPaths
}

// ForCluster returns a new Paths instance configured for a specific cluster
func (p *Paths) ForCluster(clusterName string) *Paths {
	newPaths := *p
	newPaths.clusterName = clusterName
	return &newPaths
}

// ForEnvironment returns a new Paths instance configured for a specific environment and namespace
func (p *Paths) ForEnvironment(environment, namespace string) *Paths {
	newPaths := *p
	newPaths.environment = environment
	newPaths.namespace = namespace
	return &newPaths
}

// ForType changes the path type (target, basechart, cache, source)
func (p *Paths) ForType(pathType PathType) PathBuilder {
	newPaths := *p
	newPaths.pathType = pathType
	return &newPaths
}

// getBasePath returns the appropriate base path based on pathType
func (p *Paths) getBasePath() string {
	switch p.pathType {
	case PathTypeTarget:
		return p.TargetPath
	case PathTypeBaseChart:
		if p.migration.BaseChartPath != "" {
			return p.migration.BaseChartPath
		}
		return "migration/base-chart"
	case PathTypeCache:
		return p.CacheDir
	case PathTypeSource:
		return p.SourcePath
	default:
		return p.TargetPath
	}
}

// Core directory builders following DRY principle

// ServiceDir returns the base service directory: {basePath}/{service}
func (p *Paths) ServiceDir() string {
	basePath := p.getBasePath()
	if p.pathType == PathTypeCache {
		return filepath.Join(basePath, p.clusterName, p.namespace, p.serviceName)
	}
	return filepath.Join(basePath, p.serviceName)
}

// EnvsDir returns: {basePath}/envs or {basePath}/{service}/envs
func (p *Paths) EnvsDir() string {
	if p.pathType == PathTypeBaseChart {
		return filepath.Join(p.getBasePath(), "envs")
	}
	return filepath.Join(p.ServiceDir(), "envs")
}

// TemplatesDir returns: {basePath}/templates or {basePath}/{service}/templates
func (p *Paths) TemplatesDir() string {
	if p.pathType == PathTypeBaseChart {
		return filepath.Join(p.getBasePath(), "templates")
	}
	return filepath.Join(p.ServiceDir(), "templates")
}

// Environment hierarchy builders (account/cluster/namespace pattern)

// EnvironmentDir returns: {envsDir}/{environment}
func (p *Paths) EnvironmentDir() string {
	return filepath.Join(p.EnvsDir(), p.environment)
}

// EnvironmentClustersDir returns: {envsDir}/{environment}/clusters
func (p *Paths) EnvironmentClustersDir() string {
	return filepath.Join(p.EnvironmentDir(), "clusters")
}

// EnvironmentClusterDir returns: {envsDir}/{environment}/clusters/{cluster}
func (p *Paths) EnvironmentClusterDir() string {
	return filepath.Join(p.EnvironmentClustersDir(), p.clusterName)
}

// EnvironmentNamespacesDir returns: {envsDir}/{environment}/clusters/{cluster}/namespaces
func (p *Paths) EnvironmentNamespacesDir() string {
	return filepath.Join(p.EnvironmentClusterDir(), "namespaces")
}

// EnvironmentNamespaceDir returns: {envsDir}/{environment}/clusters/{cluster}/namespaces/{namespace}
func (p *Paths) EnvironmentNamespaceDir() string {
	return filepath.Join(p.EnvironmentNamespacesDir(), p.namespace)
}

// Values and config file paths

// ValuesPath returns: {serviceDir}/values.yaml
func (p *Paths) ValuesPath() string {
	return filepath.Join(p.ServiceDir(), "values.yaml")
}

// EnvironmentValuesPath returns: {envsDir}/{environment}/values.yaml
func (p *Paths) EnvironmentValuesPath() string {
	return filepath.Join(p.EnvironmentDir(), "values.yaml")
}

// EnvironmentClusterValuesPath returns: {envsDir}/{environment}/clusters/{cluster}/values.yaml
func (p *Paths) EnvironmentClusterValuesPath() string {
	return filepath.Join(p.EnvironmentClusterDir(), "values.yaml")
}

// EnvironmentNamespaceValuesPath returns: {envsDir}/{environment}/clusters/{cluster}/namespaces/{namespace}/values.yaml
func (p *Paths) EnvironmentNamespaceValuesPath() string {
	return filepath.Join(p.EnvironmentNamespaceDir(), "values.yaml")
}

// ChartPath returns: {serviceDir}/Chart.yaml or {basePath}/Chart.yaml
func (p *Paths) ChartPath() string {
	if p.pathType == PathTypeBaseChart {
		return filepath.Join(p.getBasePath(), "Chart.yaml")
	}
	return filepath.Join(p.ServiceDir(), "Chart.yaml")
}

// Specific file paths for common operations

// ManifestPath returns: {serviceDir}/manifest.yaml
func (p *Paths) ManifestPath() string {
	return filepath.Join(p.ServiceDir(), "manifest.yaml")
}

// SecretsPath returns: {serviceDir}/secrets.yaml
func (p *Paths) SecretsPath() string {
	return filepath.Join(p.ServiceDir(), "secrets.yaml")
}

// EnvironmentSecretsPath returns: {envsDir}/{environment}/secrets.yaml
func (p *Paths) EnvironmentSecretsPath() string {
	return filepath.Join(p.EnvironmentDir(), "secrets.yaml")
}

// EnvironmentClusterSecretsPath returns: {envsDir}/{environment}/clusters/{cluster}/secrets.yaml
func (p *Paths) EnvironmentClusterSecretsPath() string {
	return filepath.Join(p.EnvironmentClusterDir(), "secrets.yaml")
}

// EnvironmentNamespaceSecretsPath returns: {envsDir}/{environment}/clusters/{cluster}/namespaces/{namespace}/secrets.dec.yaml
func (p *Paths) EnvironmentNamespaceSecretsPath() string {
	return filepath.Join(p.EnvironmentNamespaceDir(), "secrets.dec.yaml")
}

// Exists checks if a file or directory exists
func (p *Paths) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates a directory if it doesn't exist
func (p *Paths) EnsureDir(path string) error {
	if p.Exists(path) {
		return nil
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

// ListFiles lists files in a directory matching a pattern
func (p *Paths) ListFiles(dir, pattern string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file matches pattern
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}

		if matched || strings.Contains(filepath.Base(path), pattern) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files in %s: %w", dir, err)
	}

	return files, nil
}

// Configuration helpers

// GetLegacyValuesFilename returns the configured legacy values filename
func (p *Paths) GetLegacyValuesFilename() string {
	if p.migration.LegacyValuesFilename != "" {
		return p.migration.LegacyValuesFilename
	}
	return "legacy-values.yaml"
}

// GetHelmValuesFilename returns the configured helm values filename
func (p *Paths) GetHelmValuesFilename() string {
	if p.migration.HelmValuesFilename != "" {
		return p.migration.HelmValuesFilename
	}
	return "values.yaml"
}
