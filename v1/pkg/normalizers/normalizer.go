package normalizers

import (
	"fmt"
	"regexp"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/yaml"
)

// Normalizer handles key path transformations based on regex patterns
type Normalizer struct {
	config   *config.Config
	patterns []patternMapping
}

type patternMapping struct {
	regex  *regexp.Regexp
	target string
	source string
}

// New creates a new Normalizer instance with the provided configuration
func New(cfg *config.Config) (*Normalizer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	normalizer := &Normalizer{
		config:   cfg,
		patterns: make([]patternMapping, 0),
	}

	if cfg.Globals.Mappings == nil || cfg.Globals.Mappings.Normalizer == nil || !cfg.Globals.Mappings.Normalizer.Enabled {
		return normalizer, nil
	}

	// Compile regex patterns
	for pattern, target := range cfg.Globals.Mappings.Normalizer.Patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %v", pattern, err)
		}

		normalizer.patterns = append(normalizer.patterns, patternMapping{
			regex:  regex,
			target: target,
			source: pattern,
		})
	}

	return normalizer, nil
}

// LoadConfig loads the normalizer configuration from a YAML byte slice
func LoadConfig(yamlData []byte) (*config.Config, error) {
	var cfg config.Config
	if err := yaml.UnmarshalStrict(yamlData, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return &cfg, nil
}

// NormalizeYAML applies key path transformations to YAML content
func (n *Normalizer) NormalizeYAML(yamlData []byte) ([]byte, error) {
	if n.config.Globals.Mappings == nil || n.config.Globals.Mappings.Normalizer == nil || !n.config.Globals.Mappings.Normalizer.Enabled || len(n.patterns) == 0 {
		return yamlData, nil
	}

	var data interface{}
	if err := yaml.UnmarshalStrict(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Collect all transformations first
	transformations := make(map[string]transformationInfo)
	n.collectTransformations(data, "", transformations)

	// Apply transformations to create new structure
	normalized := n.applyTransformations(data, transformations)

	result, err := yaml.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized YAML: %v", err)
	}

	return result, nil
}

type transformationInfo struct {
	sourcePath string
	targetPath string
	value      interface{}
}

// collectTransformations recursively finds all paths that need transformation
func (n *Normalizer) collectTransformations(value interface{}, currentPath string, transformations map[string]transformationInfo) {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			fullPath := buildPath(currentPath, key)
			if targetPath := n.findTransformation(fullPath); targetPath != "" {
				transformations[fullPath] = transformationInfo{
					sourcePath: fullPath,
					targetPath: targetPath,
					value:      val,
				}
			}
			n.collectTransformations(val, fullPath, transformations)
		}
	case []interface{}:
		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", currentPath, i)
			n.collectTransformations(item, indexPath, transformations)
		}
	}
}

// applyTransformations creates a new structure with transformations applied
func (n *Normalizer) applyTransformations(data interface{}, transformations map[string]transformationInfo) interface{} {
	result := make(map[string]interface{})

	// First, copy all non-transformed data
	n.copyNonTransformed(data, "", result, transformations)

	// Then, apply all transformations
	for _, transform := range transformations {
		n.setNestedValue(result, transform.targetPath, transform.value, "")
	}

	return result
}

// copyNonTransformed copies values that don't have transformations
func (n *Normalizer) copyNonTransformed(value interface{}, currentPath string, result map[string]interface{}, transformations map[string]transformationInfo) {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			fullPath := buildPath(currentPath, key)

			// Skip if this path has a transformation
			if _, hasTransform := transformations[fullPath]; hasTransform {
				continue
			}

			// Check if any child has transformation
			hasChildTransform := false
			for transformPath := range transformations {
				if strings.HasPrefix(transformPath, fullPath+".") {
					hasChildTransform = true
					break
				}
			}

			if !hasChildTransform {
				// No transformations in this subtree, copy as-is
				if currentPath == "" {
					result[key] = val
				} else {
					n.setNestedValue(result, fullPath, val, "")
				}
			} else {
				// This subtree has transformations, process recursively
				n.copyNonTransformed(val, fullPath, result, transformations)
			}
		}
	case []interface{}:
		// For arrays, we need to copy them unless the parent path is being transformed
		if _, hasTransform := transformations[currentPath]; !hasTransform {
			processedArray := make([]interface{}, len(v))
			for i, item := range v {
				indexPath := fmt.Sprintf("%s[%d]", currentPath, i)
				processedArray[i] = n.processArrayItem(item, indexPath, transformations)
			}
			n.setNestedValue(result, currentPath, processedArray, "")
		}
	}
}

// processArrayItem processes individual array items
func (n *Normalizer) processArrayItem(item interface{}, currentPath string, transformations map[string]transformationInfo) interface{} {
	switch v := item.(type) {
	case map[string]interface{}:
		itemResult := make(map[string]interface{})
		n.copyNonTransformed(v, currentPath, itemResult, transformations)
		return itemResult
	default:
		return item
	}
}

// findTransformation checks if a path matches any transformation pattern
func (n *Normalizer) findTransformation(path string) string {
	for _, pattern := range n.patterns {
		if pattern.regex.MatchString(path) {
			return pattern.target
		}
	}
	return ""
}

// buildPath constructs a dot-separated path
func buildPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// setNestedValue creates nested map structure and sets the value at the specified path
func (n *Normalizer) setNestedValue(result map[string]interface{}, targetPath string, value interface{}, currentPath string) {
	// Use the full target path - transformations should create the complete structure
	parts := strings.Split(targetPath, ".")
	current := result

	// Navigate/create the nested structure
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - set the value
			current[part] = value
		} else {
			// Intermediate part - ensure map exists
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			if nestedMap, ok := current[part].(map[string]interface{}); ok {
				current = nestedMap
			}
		}
	}
}

// extractKeyFromPath extracts the key part from a transformation path
func extractKeyFromPath(fullPath, currentPath string) string {
	if currentPath == "" {
		return fullPath
	}

	// Remove the current path prefix and return the remaining part
	if strings.HasPrefix(fullPath, currentPath+".") {
		return strings.TrimPrefix(fullPath, currentPath+".")
	}

	// If the transformation path doesn't match current path structure,
	// return the last part of the path
	parts := strings.Split(fullPath, ".")
	return parts[len(parts)-1]
}

// IsEnabled returns whether the normalizer is enabled in configuration
func (n *Normalizer) IsEnabled() bool {
	if n.config.Globals.Mappings == nil || n.config.Globals.Mappings.Normalizer == nil {
		return false
	}
	return n.config.Globals.Mappings.Normalizer.Enabled
}

// GetDescription returns the normalizer description from configuration
func (n *Normalizer) GetDescription() string {
	if n.config.Globals.Mappings == nil || n.config.Globals.Mappings.Normalizer == nil {
		return ""
	}
	return n.config.Globals.Mappings.Normalizer.Description
}

// GetPatterns returns the configured transformation patterns
func (n *Normalizer) GetPatterns() map[string]string {
	if n.config.Globals.Mappings == nil || n.config.Globals.Mappings.Normalizer == nil {
		return nil
	}
	return n.config.Globals.Mappings.Normalizer.Patterns
}
