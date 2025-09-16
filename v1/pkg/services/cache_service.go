package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"helm.sh/helm/v3/pkg/release"

	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/yaml"
)

// cacheService implements CacheService interface
type cacheService struct {
	cache         map[string][]*release.Release // key: "cluster:namespace"
	tempDir       string
	shouldCleanup bool
	mu            sync.RWMutex
	log           *logger.NamedLogger
}

// NewCacheService creates a new CacheService
func NewCacheService(cacheDir string, cleanupOnExit bool) (CacheService, error) {
	var tempDir string
	var err error
	shouldCleanup := cleanupOnExit

	if cacheDir != "" {
		// Use specified cache directory (persistent cache)
		tempDir = cacheDir
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	} else {
		// Use system temp directory (always cleanup)
		tempDir, err = os.MkdirTemp("", "helm-migrator-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		shouldCleanup = true
	}

	return &cacheService{
		cache:         make(map[string][]*release.Release),
		tempDir:       tempDir,
		shouldCleanup: shouldCleanup,
		log:           logger.WithName("cache-service"),
	}, nil
}

// GetReleases returns cached releases for a cluster:namespace
func (c *cacheService) GetReleases(cluster, namespace string) []*release.Release {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", cluster, namespace)
	return c.cache[key]
}

// SetReleases caches releases for a cluster:namespace and saves values to disk
func (c *cacheService) SetReleases(cluster, namespace string, releases []*release.Release) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s", cluster, namespace)
	c.cache[key] = releases
	c.log.V(2).InfoS("Cached releases", "cluster", cluster, "namespace", namespace, "count", len(releases))

	// Save each release's values to disk for later use
	for _, rel := range releases {
		if rel == nil || rel.Config == nil {
			continue
		}

		// Create cache directory for this service
		cacheDir := filepath.Join(c.tempDir, cluster, namespace, rel.Name)
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			c.log.Error(err, "Failed to create cache directory", "path", cacheDir)
			continue
		}

		// Save values.yaml using centralized yaml package
		valuesPath := filepath.Join(cacheDir, "values.yaml")

		// rel.Config is already a map[string]interface{} from Helm
		doc, err := yaml.FromMap(rel.Config)
		if err != nil {
			c.log.Error(err, "Failed to create values document", "service", rel.Name)
			continue
		}

		if err := doc.SaveFile(valuesPath, yaml.DefaultOptions()); err != nil {
			c.log.Error(err, "Failed to save values to cache", "path", valuesPath)
			continue
		}

		c.log.V(3).InfoS("Cached values to disk", "service", rel.Name, "path", valuesPath)

		// Save pod manifest if available
		if rel.Manifest != "" {
			manifestPath := filepath.Join(cacheDir, "manifest.yaml")
			if err := os.WriteFile(manifestPath, []byte(rel.Manifest), 0644); err != nil {
				c.log.Error(err, "Failed to save manifest to cache", "path", manifestPath)
			} else {
				c.log.V(3).InfoS("Cached manifest to disk", "service", rel.Name, "path", manifestPath)
			}
		}
	}

	return nil
}

// GetTempPath returns a temp path for storing resources
func (c *cacheService) GetTempPath(cluster, namespace, service, resourceType string) string {
	return filepath.Join(c.tempDir, cluster, namespace, service, resourceType)
}

// Clear clears the cache
func (c *cacheService) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear in-memory cache
	c.cache = make(map[string][]*release.Release)

	// Clear disk cache
	if c.tempDir != "" {
		entries, err := os.ReadDir(c.tempDir)
		if err != nil {
			return fmt.Errorf("failed to read cache directory: %w", err)
		}

		for _, entry := range entries {
			path := filepath.Join(c.tempDir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				c.log.Error(err, "Failed to remove cache entry", "path", path)
			}
		}
	}

	c.log.InfoS("Cache cleared")
	return nil
}

// Cleanup removes temporary files
func (c *cacheService) Cleanup() error {
	if !c.shouldCleanup {
		c.log.InfoS("Keeping cache directory", "path", c.tempDir)
		return nil
	}

	if c.tempDir != "" {
		c.log.InfoS("Cleaning up cache directory", "path", c.tempDir)
		return os.RemoveAll(c.tempDir)
	}

	return nil
}
