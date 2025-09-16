package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/yaml"
)

// ConfigLayer represents a single layer in the configuration hierarchy
type ConfigLayer struct {
	Name   string
	Values map[string]interface{}
}

// Clone creates a deep copy of the config layer
func (c *ConfigLayer) Clone() *ConfigLayer {
	if c == nil {
		return &ConfigLayer{
			Values: make(map[string]interface{}),
		}
	}

	cloned := &ConfigLayer{
		Name:   c.Name,
		Values: deepCopyMap(c.Values),
	}
	return cloned
}

// Merge merges another config layer into this one
func (c *ConfigLayer) Merge(other *ConfigLayer) {
	if other == nil || other.Values == nil {
		return
	}

	if c.Values == nil {
		c.Values = make(map[string]interface{})
	}

	mergeMapRecursive(c.Values, other.Values)
}

// HierarchicalConfig manages the configuration hierarchy
type HierarchicalConfig struct {
	defaults   *ConfigLayer
	globals    *ConfigLayer
	clusters   map[string]*ConfigLayer
	envs       map[string]*ConfigLayer
	namespaces map[string]*ConfigLayer
	services   map[string]*ConfigLayer
	log        *logger.NamedLogger
}

// NewHierarchicalConfig creates a new hierarchical configuration
func NewHierarchicalConfig() *HierarchicalConfig {
	return &HierarchicalConfig{
		defaults:   &ConfigLayer{Name: "defaults", Values: make(map[string]interface{})},
		globals:    &ConfigLayer{Name: "globals", Values: make(map[string]interface{})},
		clusters:   make(map[string]*ConfigLayer),
		envs:       make(map[string]*ConfigLayer),
		namespaces: make(map[string]*ConfigLayer),
		services:   make(map[string]*ConfigLayer),
		log:        logger.WithName("hierarchical-config"),
	}
}

// SetDefaults sets the default configuration layer
func (h *HierarchicalConfig) SetDefaults(values map[string]interface{}) {
	h.defaults = &ConfigLayer{
		Name:   "defaults",
		Values: values,
	}
}

// SetGlobals sets the global configuration layer
func (h *HierarchicalConfig) SetGlobals(values map[string]interface{}) {
	h.globals = &ConfigLayer{
		Name:   "globals",
		Values: values,
	}
}

// SetClusterConfig sets configuration for a specific cluster
func (h *HierarchicalConfig) SetClusterConfig(cluster string, values map[string]interface{}) {
	h.clusters[cluster] = &ConfigLayer{
		Name:   fmt.Sprintf("cluster:%s", cluster),
		Values: values,
	}
}

// SetEnvironmentConfig sets configuration for a specific environment
func (h *HierarchicalConfig) SetEnvironmentConfig(cluster, env string, values map[string]interface{}) {
	key := h.buildEnvKey(cluster, env)
	h.envs[key] = &ConfigLayer{
		Name:   fmt.Sprintf("env:%s/%s", cluster, env),
		Values: values,
	}
}

// SetNamespaceConfig sets configuration for a specific namespace
func (h *HierarchicalConfig) SetNamespaceConfig(cluster, env, namespace string, values map[string]interface{}) {
	key := h.buildNamespaceKey(cluster, env, namespace)
	h.namespaces[key] = &ConfigLayer{
		Name:   fmt.Sprintf("namespace:%s/%s/%s", cluster, env, namespace),
		Values: values,
	}
}

// SetServiceConfig sets configuration for a specific service
func (h *HierarchicalConfig) SetServiceConfig(service string, values map[string]interface{}) {
	h.services[service] = &ConfigLayer{
		Name:   fmt.Sprintf("service:%s", service),
		Values: values,
	}
}

// GetEffectiveConfig returns the effective configuration after applying the override chain
func (h *HierarchicalConfig) GetEffectiveConfig(cluster, env, namespace, service string) *ConfigLayer {
	// Start with defaults
	effective := h.defaults.Clone()

	// Apply override chain
	layers := []struct {
		name  string
		layer *ConfigLayer
	}{
		{"globals", h.globals},
		{"cluster", h.clusters[cluster]},
		{"environment", h.envs[h.buildEnvKey(cluster, env)]},
		{"namespace", h.namespaces[h.buildNamespaceKey(cluster, env, namespace)]},
		{"service", h.services[service]},
	}

	for _, l := range layers {
		if l.layer != nil {
			h.log.V(3).InfoS("Applying configuration layer",
				"layer", l.name,
				"cluster", cluster,
				"env", env,
				"namespace", namespace,
				"service", service)
			effective.Merge(l.layer)
		}
	}

	h.log.V(2).InfoS("Generated effective configuration",
		"cluster", cluster,
		"env", env,
		"namespace", namespace,
		"service", service)

	return effective
}

// LoadFromFile loads a configuration layer from a YAML file
func (h *HierarchicalConfig) LoadFromFile(path string) (*ConfigLayer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from %s: %w", path, err)
	}

	return &ConfigLayer{
		Name:   filepath.Base(path),
		Values: values,
	}, nil
}

// LoadFromDirectory loads all configuration files from a directory hierarchy
func (h *HierarchicalConfig) LoadFromDirectory(baseDir string) error {
	// Load defaults.yaml if exists
	defaultsPath := filepath.Join(baseDir, "defaults.yaml")
	if layer, err := h.LoadFromFile(defaultsPath); err == nil {
		h.SetDefaults(layer.Values)
		h.log.InfoS("Loaded defaults", "path", defaultsPath)
	}

	// Load globals.yaml if exists
	globalsPath := filepath.Join(baseDir, "globals.yaml")
	if layer, err := h.LoadFromFile(globalsPath); err == nil {
		h.SetGlobals(layer.Values)
		h.log.InfoS("Loaded globals", "path", globalsPath)
	}

	// Load cluster configs
	clustersDir := filepath.Join(baseDir, "clusters")
	if clusters, err := filepath.Glob(filepath.Join(clustersDir, "*.yaml")); err == nil {
		for _, clusterFile := range clusters {
			clusterName := getNameFromFile(clusterFile)
			if layer, err := h.LoadFromFile(clusterFile); err == nil {
				h.SetClusterConfig(clusterName, layer.Values)
				h.log.InfoS("Loaded cluster config", "cluster", clusterName, "path", clusterFile)
			}
		}
	}

	// Load service configs
	servicesDir := filepath.Join(baseDir, "services")
	if services, err := filepath.Glob(filepath.Join(servicesDir, "*.yaml")); err == nil {
		for _, serviceFile := range services {
			serviceName := getNameFromFile(serviceFile)
			if layer, err := h.LoadFromFile(serviceFile); err == nil {
				h.SetServiceConfig(serviceName, layer.Values)
				h.log.InfoS("Loaded service config", "service", serviceName, "path", serviceFile)
			}
		}
	}

	return nil
}

// GetConfigDiff returns the differences between two configuration layers
func (h *HierarchicalConfig) GetConfigDiff(base, override *ConfigLayer) map[string]interface{} {
	diff := make(map[string]interface{})

	if base == nil || base.Values == nil {
		return override.Values
	}

	if override == nil || override.Values == nil {
		return make(map[string]interface{})
	}

	findDifferences(base.Values, override.Values, diff, "")

	return diff
}

// Helper functions

func (h *HierarchicalConfig) buildEnvKey(cluster, env string) string {
	return fmt.Sprintf("%s:%s", cluster, env)
}

func (h *HierarchicalConfig) buildNamespaceKey(cluster, env, namespace string) string {
	return fmt.Sprintf("%s:%s:%s", cluster, env, namespace)
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopyMap(val)
		case []interface{}:
			result[k] = deepCopySlice(val)
		default:
			result[k] = v
		}
	}
	return result
}

func deepCopySlice(s []interface{}) []interface{} {
	if s == nil {
		return nil
	}

	result := make([]interface{}, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = deepCopyMap(val)
		case []interface{}:
			result[i] = deepCopySlice(val)
		default:
			result[i] = v
		}
	}
	return result
}

func mergeMapRecursive(dest, src map[string]interface{}) {
	for key, srcValue := range src {
		if destValue, exists := dest[key]; exists {
			// Both are maps - merge recursively
			destMap, destIsMap := destValue.(map[string]interface{})
			srcMap, srcIsMap := srcValue.(map[string]interface{})

			if destIsMap && srcIsMap {
				mergeMapRecursive(destMap, srcMap)
			} else {
				// Different types or non-maps - override
				dest[key] = srcValue
			}
		} else {
			// Key doesn't exist in dest - add it
			dest[key] = srcValue
		}
	}
}

func findDifferences(base, override map[string]interface{}, diff map[string]interface{}, path string) {
	for key, overrideValue := range override {
		fullPath := key
		if path != "" {
			fullPath = path + "." + key
		}

		if baseValue, exists := base[key]; exists {
			// Key exists in both - check if different
			if !reflect.DeepEqual(baseValue, overrideValue) {
				// Values are different
				baseMap, baseIsMap := baseValue.(map[string]interface{})
				overrideMap, overrideIsMap := overrideValue.(map[string]interface{})

				if baseIsMap && overrideIsMap {
					// Both are maps - recurse
					nestedDiff := make(map[string]interface{})
					findDifferences(baseMap, overrideMap, nestedDiff, fullPath)
					if len(nestedDiff) > 0 {
						diff[key] = nestedDiff
					}
				} else {
					// Different types or values
					diff[key] = map[string]interface{}{
						"old": baseValue,
						"new": overrideValue,
					}
				}
			}
		} else {
			// Key only in override - it's new
			diff[key] = overrideValue
		}
	}
}

func getNameFromFile(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}
