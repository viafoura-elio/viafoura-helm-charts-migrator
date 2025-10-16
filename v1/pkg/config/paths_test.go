package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaths(t *testing.T) {
	paths := NewPaths("/source", "/target", "/cache")

	t.Run("Basic paths", func(t *testing.T) {
		assert.Equal(t, "/source", paths.SourcePath)
		assert.Equal(t, "/target", paths.TargetPath)
		assert.Equal(t, "/cache", paths.CacheDir)
	})

	t.Run("Service paths", func(t *testing.T) {
		servicePaths := paths.ForService("heimdall")

		assert.Equal(t, filepath.Join("/target", "heimdall"), servicePaths.ServiceDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "values.yaml"), servicePaths.ValuesPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "Chart.yaml"), servicePaths.ChartPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "templates"), servicePaths.TemplatesDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs"), servicePaths.EnvsDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "manifest.yaml"), servicePaths.ManifestPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "secrets.yaml"), servicePaths.SecretsPath())
	})

	t.Run("Cluster paths", func(t *testing.T) {
		clusterPaths := paths.ForService("heimdall").ForCluster("prod01")

		// Old pattern: envs/{cluster} - now using new hierarchical pattern
		// Need environment context for new pattern
		clusterWithEnv := clusterPaths.ForEnvironment("production", "viafoura")
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "production", "clusters", "prod01"), clusterWithEnv.EnvironmentClusterDir())
	})

	t.Run("Environment paths", func(t *testing.T) {
		envPaths := paths.ForService("heimdall").ForCluster("prod01").ForEnvironment("production", "viafoura")

		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "production"), envPaths.EnvironmentDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "production", "clusters", "prod01", "namespaces", "viafoura"), envPaths.EnvironmentNamespaceDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "production", "clusters", "prod01", "namespaces", "viafoura", "values.yaml"), envPaths.EnvironmentNamespaceValuesPath())
	})

	t.Run("Cache paths", func(t *testing.T) {
		cachePaths := paths.ForService("heimdall").ForCluster("prod01").ForEnvironment("production", "viafoura").ForType(PathTypeCache)

		assert.Equal(t, filepath.Join("/cache", "prod01", "viafoura", "heimdall"), cachePaths.ServiceDir())
		assert.Equal(t, filepath.Join("/cache", "prod01", "viafoura", "heimdall", "values.yaml"), cachePaths.ValuesPath())
		assert.Equal(t, filepath.Join("/cache", "prod01", "viafoura", "heimdall", "manifest.yaml"), cachePaths.ManifestPath())
	})

	t.Run("Source paths", func(t *testing.T) {
		sourcePaths := paths.ForService("heimdall").ForType(PathTypeSource)

		assert.Equal(t, filepath.Join("/source", "heimdall"), sourcePaths.ServiceDir())
		assert.Equal(t, filepath.Join("/source", "heimdall", "values.yaml"), sourcePaths.ValuesPath())
	})

	t.Run("Environment secrets paths", func(t *testing.T) {
		envPaths := paths.ForService("heimdall").ForCluster("prod01").ForEnvironment("production", "viafoura")

		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "production", "secrets.yaml"), envPaths.EnvironmentSecretsPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "production", "clusters", "prod01", "secrets.yaml"), envPaths.EnvironmentClusterSecretsPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "production", "clusters", "prod01", "namespaces", "viafoura", "secrets.yaml"), envPaths.EnvironmentNamespaceSecretsPath())
	})
}

func TestPathsFilenames(t *testing.T) {
	paths := NewPaths("/source", "/target", "/cache")

	t.Run("Default filenames", func(t *testing.T) {
		migration := Migration{}
		pathsWithMigration := paths.WithMigration(migration)

		assert.Equal(t, "legacy-values.yaml", pathsWithMigration.GetLegacyValuesFilename())
		assert.Equal(t, "values.yaml", pathsWithMigration.GetHelmValuesFilename())
	})

	t.Run("Custom filenames", func(t *testing.T) {
		migration := Migration{
			LegacyValuesFilename: "custom-legacy.yaml",
			HelmValuesFilename:   "custom-helm.yaml",
		}
		pathsWithMigration := paths.WithMigration(migration)

		assert.Equal(t, "custom-legacy.yaml", pathsWithMigration.GetLegacyValuesFilename())
		assert.Equal(t, "custom-helm.yaml", pathsWithMigration.GetHelmValuesFilename())
	})
}

func TestConfigSetPaths(t *testing.T) {
	config := &Config{}

	// Initially paths should be nil
	assert.Nil(t, config.Paths)

	// Set paths
	config.SetPaths("/source", "/target", "/cache")

	// Verify paths are set
	assert.NotNil(t, config.Paths)
	assert.Equal(t, "/source", config.Paths.SourcePath)
	assert.Equal(t, "/target", config.Paths.TargetPath)
	assert.Equal(t, "/cache", config.Paths.CacheDir)

	// Test GetServicePaths
	servicePaths := config.GetServicePaths("heimdall")
	assert.NotNil(t, servicePaths)
	assert.Equal(t, filepath.Join("/target", "heimdall"), servicePaths.ServiceDir())
}
