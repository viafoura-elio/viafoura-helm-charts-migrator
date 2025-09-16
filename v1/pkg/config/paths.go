package config

import (
	"path/filepath"
)

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
}

// NewPaths creates a new Paths instance with base paths
func NewPaths(sourcePath, targetPath, cacheDir string) *Paths {
	return &Paths{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		CacheDir:   cacheDir,
	}
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

// ServiceDir returns the base service directory: {target}/{service}
func (p *Paths) ServiceDir() string {
	return filepath.Join(p.TargetPath, p.serviceName)
}

// LegacyValuesPath returns: {target}/{service}/legacy-values.yaml
func (p *Paths) LegacyValuesPath() string {
	return filepath.Join(p.ServiceDir(), "legacy-values.yaml")
}

// ValuesPath returns: {target}/{service}/values.yaml
func (p *Paths) ValuesPath() string {
	return filepath.Join(p.ServiceDir(), "values.yaml")
}

// HelmValuesPath returns: {target}/{service}/helm-values.yaml
func (p *Paths) HelmValuesPath() string {
	return filepath.Join(p.ServiceDir(), "helm-values.yaml")
}

// ChartPath returns: {target}/{service}/Chart.yaml
func (p *Paths) ChartPath() string {
	return filepath.Join(p.ServiceDir(), "Chart.yaml")
}

// TemplatesDir returns: {target}/{service}/templates
func (p *Paths) TemplatesDir() string {
	return filepath.Join(p.ServiceDir(), "templates")
}

// EnvsDir returns: {target}/{service}/envs
func (p *Paths) EnvsDir() string {
	return filepath.Join(p.ServiceDir(), "envs")
}

// ClusterDir returns: {target}/{service}/envs/{cluster}
func (p *Paths) ClusterDir() string {
	return filepath.Join(p.EnvsDir(), p.clusterName)
}

// EnvironmentDir returns: {target}/{service}/envs/{cluster}/{environment}
func (p *Paths) EnvironmentDir() string {
	return filepath.Join(p.ClusterDir(), p.environment)
}

// NamespaceDir returns: {target}/{service}/envs/{cluster}/{environment}/{namespace}
func (p *Paths) NamespaceDir() string {
	return filepath.Join(p.EnvironmentDir(), p.namespace)
}

// EnvironmentValuesPath returns: {target}/{service}/envs/{cluster}/{environment}/{namespace}/values.yaml
func (p *Paths) EnvironmentValuesPath() string {
	return filepath.Join(p.NamespaceDir(), "values.yaml")
}

// CacheServiceDir returns: .cache/{cluster}/{namespace}/{service}
func (p *Paths) CacheServiceDir() string {
	return filepath.Join(p.CacheDir, p.clusterName, p.namespace, p.serviceName)
}

// CachedValuesPath returns: .cache/{cluster}/{namespace}/{service}/values.yaml
func (p *Paths) CachedValuesPath() string {
	return filepath.Join(p.CacheServiceDir(), "values.yaml")
}

// CachedManifestPath returns: .cache/{cluster}/{namespace}/{service}/manifest.yaml
func (p *Paths) CachedManifestPath() string {
	return filepath.Join(p.CacheServiceDir(), "manifest.yaml")
}

// TransformationReportPath returns: {target}/{service}/transformation-report.yaml
func (p *Paths) TransformationReportPath() string {
	return filepath.Join(p.ServiceDir(), "transformation-report.yaml")
}

// SourceServicePath returns the legacy source path for a service
func (p *Paths) SourceServicePath() string {
	if p.SourcePath == "" {
		return ""
	}
	return filepath.Join(p.SourcePath, p.serviceName)
}

// SourceValuesPath returns: {source}/{service}/values.yaml
func (p *Paths) SourceValuesPath() string {
	return filepath.Join(p.SourceServicePath(), "values.yaml")
}

// GetLegacyValuesFilename returns the configured legacy values filename
func (p *Paths) GetLegacyValuesFilename(migration Migration) string {
	if migration.LegacyValuesFilename != "" {
		return migration.LegacyValuesFilename
	}
	return "legacy-values.yaml"
}

// GetHelmValuesFilename returns the configured helm values filename
func (p *Paths) GetHelmValuesFilename(migration Migration) string {
	if migration.HelmValuesFilename != "" {
		return migration.HelmValuesFilename
	}
	return "values.yaml"
}
