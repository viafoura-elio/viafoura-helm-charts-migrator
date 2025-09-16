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
		assert.Equal(t, filepath.Join("/target", "heimdall", "legacy-values.yaml"), servicePaths.LegacyValuesPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "values.yaml"), servicePaths.ValuesPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "helm-values.yaml"), servicePaths.HelmValuesPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "Chart.yaml"), servicePaths.ChartPath())
		assert.Equal(t, filepath.Join("/target", "heimdall", "templates"), servicePaths.TemplatesDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs"), servicePaths.EnvsDir())
	})

	t.Run("Cluster paths", func(t *testing.T) {
		clusterPaths := paths.ForService("heimdall").ForCluster("prod01")

		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "prod01"), clusterPaths.ClusterDir())
	})

	t.Run("Environment paths", func(t *testing.T) {
		envPaths := paths.ForService("heimdall").ForCluster("prod01").ForEnvironment("production", "viafoura")

		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "prod01", "production"), envPaths.EnvironmentDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "prod01", "production", "viafoura"), envPaths.NamespaceDir())
		assert.Equal(t, filepath.Join("/target", "heimdall", "envs", "prod01", "production", "viafoura", "values.yaml"), envPaths.EnvironmentValuesPath())
	})

	t.Run("Cache paths", func(t *testing.T) {
		cachePaths := paths.ForService("heimdall").ForCluster("prod01").ForEnvironment("production", "viafoura")

		assert.Equal(t, filepath.Join("/cache", "prod01", "viafoura", "heimdall"), cachePaths.CacheServiceDir())
		assert.Equal(t, filepath.Join("/cache", "prod01", "viafoura", "heimdall", "values.yaml"), cachePaths.CachedValuesPath())
		assert.Equal(t, filepath.Join("/cache", "prod01", "viafoura", "heimdall", "manifest.yaml"), cachePaths.CachedManifestPath())
	})

	t.Run("Source paths", func(t *testing.T) {
		sourcePaths := paths.ForService("heimdall")

		assert.Equal(t, filepath.Join("/source", "heimdall"), sourcePaths.SourceServicePath())
		assert.Equal(t, filepath.Join("/source", "heimdall", "values.yaml"), sourcePaths.SourceValuesPath())
	})

	t.Run("Transformation report path", func(t *testing.T) {
		servicePaths := paths.ForService("heimdall")

		assert.Equal(t, filepath.Join("/target", "heimdall", "transformation-report.yaml"), servicePaths.TransformationReportPath())
	})
}

func TestPathsFilenames(t *testing.T) {
	paths := NewPaths("/source", "/target", "/cache")

	t.Run("Default filenames", func(t *testing.T) {
		migration := Migration{}

		assert.Equal(t, "legacy-values.yaml", paths.GetLegacyValuesFilename(migration))
		assert.Equal(t, "values.yaml", paths.GetHelmValuesFilename(migration))
	})

	t.Run("Custom filenames", func(t *testing.T) {
		migration := Migration{
			LegacyValuesFilename: "custom-legacy.yaml",
			HelmValuesFilename:   "custom-helm.yaml",
		}

		assert.Equal(t, "custom-legacy.yaml", paths.GetLegacyValuesFilename(migration))
		assert.Equal(t, "custom-helm.yaml", paths.GetHelmValuesFilename(migration))
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
