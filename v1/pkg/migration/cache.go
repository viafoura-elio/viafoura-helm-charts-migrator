package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/release"

	"helm-charts-migrator/v1/pkg/logger"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// ReleaseCache caches Helm releases per namespace to avoid redundant API calls
type ReleaseCache struct {
	cache         map[string][]*release.Release // key: "cluster:namespace"
	tempDir       string
	shouldCleanup bool // Whether to cleanup on exit
	log           *logger.NamedLogger
}

// NewReleaseCache creates a new release cache with a temporary directory
func NewReleaseCache(cacheDir string, cleanupOnExit bool) (*ReleaseCache, error) {
	var tempDir string
	var err error
	shouldCleanup := cleanupOnExit

	if cacheDir != "" {
		// Use specified cache directory (persistent cache)
		// Don't use PID subdirectory for persistent cache
		tempDir = cacheDir
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
		// For persistent cache, don't cleanup unless explicitly requested
		// cleanupOnExit is controlled by the --cleanup-cache flag
	} else {
		// Use system temp directory (always cleanup)
		tempDir, err = os.MkdirTemp("", "helm-migrator-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		// Always cleanup system temp directories
		shouldCleanup = true
	}

	return &ReleaseCache{
		cache:         make(map[string][]*release.Release),
		tempDir:       tempDir,
		shouldCleanup: shouldCleanup,
		log:           logger.WithName("cache"),
	}, nil
}

// GetReleases returns cached releases or nil if not cached
func (rc *ReleaseCache) GetReleases(cluster, namespace string) []*release.Release {
	key := fmt.Sprintf("%s:%s", cluster, namespace)
	return rc.cache[key]
}

// SetReleases caches releases for a cluster:namespace combination and saves values to disk
func (rc *ReleaseCache) SetReleases(cluster, namespace string, releases []*release.Release) {
	key := fmt.Sprintf("%s:%s", cluster, namespace)
	rc.cache[key] = releases
	rc.log.V(2).InfoS("Cached releases", "cluster", cluster, "namespace", namespace, "count", len(releases))

	// Save each release's values to disk for later use
	for _, rel := range releases {
		if rel == nil || rel.Config == nil {
			continue
		}

		// Create cache directory for this service
		cacheDir := filepath.Join(rc.tempDir, cluster, namespace, rel.Name)
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			rc.log.Error(err, "Failed to create cache directory", "path", cacheDir)
			continue
		}

		// Save values.yaml
		valuesPath := filepath.Join(cacheDir, "values.yaml")
		valuesYAML, err := yaml.Marshal(rel.Config)
		if err != nil {
			rc.log.Error(err, "Failed to marshal values", "service", rel.Name)
			continue
		}

		// Parse and format with the yaml package for consistent formatting
		nodeTree, err := yaml.UnmarshalYAML(valuesYAML)
		if err != nil {
			rc.log.Error(err, "Failed to parse values", "service", rel.Name)
			// Fall back to writing raw YAML
			if err := os.WriteFile(valuesPath, valuesYAML, 0644); err != nil {
				rc.log.Error(err, "Failed to write values", "service", rel.Name)
			}
			continue
		}

		formattedYAML, err := nodeTree.ToYAML()
		if err != nil {
			rc.log.Error(err, "Failed to format values", "service", rel.Name)
			// Fall back to writing raw YAML
			if err := os.WriteFile(valuesPath, valuesYAML, 0644); err != nil {
				rc.log.Error(err, "Failed to write values", "service", rel.Name)
			}
			continue
		}

		if err := os.WriteFile(valuesPath, formattedYAML, 0644); err != nil {
			rc.log.Error(err, "Failed to save values to cache", "path", valuesPath)
			continue
		}

		rc.log.V(3).InfoS("Cached values to disk", "service", rel.Name, "path", valuesPath)

		// Save pod manifest if available
		if rel.Manifest != "" {
			manifestPath := filepath.Join(cacheDir, "manifest.yaml")
			if err := os.WriteFile(manifestPath, []byte(rel.Manifest), 0644); err != nil {
				rc.log.Error(err, "Failed to save manifest to cache", "path", manifestPath)
			} else {
				rc.log.V(3).InfoS("Cached manifest to disk", "service", rel.Name, "path", manifestPath)
			}
		}
	}
}

// GetTempPath returns a temp path for storing resources
func (rc *ReleaseCache) GetTempPath(cluster, namespace, service, resourceType string) string {
	return filepath.Join(rc.tempDir, cluster, namespace, service, resourceType)
}

// Cleanup removes the temporary directory if shouldCleanup is true
func (rc *ReleaseCache) Cleanup() error {
	if !rc.shouldCleanup {
		rc.log.InfoS("Keeping cache directory", "path", rc.tempDir)
		return nil
	}

	if rc.tempDir != "" {
		rc.log.InfoS("Cleaning up cache directory", "path", rc.tempDir)
		return os.RemoveAll(rc.tempDir)
	}
	return nil
}
