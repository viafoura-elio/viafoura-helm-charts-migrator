package cleaner

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// Cleaner handles removal of unwanted keys from YAML files
type Cleaner struct {
	config       *config.Config
	keyPatterns  []*regexp.Regexp
	pathPatterns []string
}

// CleanResult contains information about cleaned keys
type CleanResult struct {
	File        string   `yaml:"file"`
	RemovedKeys []string `yaml:"removed_keys"`
	KeyCount    int      `yaml:"key_count"`
}

// New creates a new Cleaner instance
func New(cfg *config.Config) (*Cleaner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	cleaner := &Cleaner{
		config:       cfg,
		keyPatterns:  make([]*regexp.Regexp, 0),
		pathPatterns: make([]string, 0),
	}

	if cfg.Globals.Mappings == nil || cfg.Globals.Mappings.Cleaner == nil || !cfg.Globals.Mappings.Cleaner.Enabled {
		return cleaner, nil
	}

	// Compile key patterns
	for _, pattern := range cfg.Globals.Mappings.Cleaner.KeyPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid key pattern '%s': %v", pattern, err)
		}
		cleaner.keyPatterns = append(cleaner.keyPatterns, regex)
	}

	// Store path patterns
	cleaner.pathPatterns = cfg.Globals.Mappings.Cleaner.PathPatterns

	return cleaner, nil
}

// CleanYAML removes unwanted keys from YAML content
func (c *Cleaner) CleanYAML(yamlData []byte, filePath string) ([]byte, *CleanResult, error) {
	result := &CleanResult{
		File:        filePath,
		RemovedKeys: []string{},
		KeyCount:    0,
	}

	if !c.IsEnabled() || !c.shouldProcessFile(filePath) {
		return yamlData, result, nil
	}

	var data interface{}
	if err := yaml.UnmarshalStrict(yamlData, &data); err != nil {
		return nil, result, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Clean the data - only remove root-level keys
	cleaned := c.cleanValue(data, "", 0, result)

	// Marshal back to YAML
	cleanedYAML, err := yaml.Marshal(cleaned)
	if err != nil {
		return nil, result, fmt.Errorf("failed to marshal cleaned YAML: %v", err)
	}

	result.KeyCount = len(result.RemovedKeys)
	return cleanedYAML, result, nil
}

// cleanValue recursively processes values, but only removes keys at root level
func (c *Cleaner) cleanValue(value interface{}, path string, depth int, result *CleanResult) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		cleaned := make(map[string]interface{})
		for key, val := range v {
			fullPath := buildPath(path, key)

			// Only check for removal at root level (depth 0)
			if depth == 0 && c.shouldRemoveKey(key) {
				result.RemovedKeys = append(result.RemovedKeys, fullPath)
				continue // Skip this key
			}

			// Process nested values without removing keys
			cleaned[key] = c.cleanValue(val, fullPath, depth+1, result)
		}
		return cleaned

	case []interface{}:
		cleaned := make([]interface{}, len(v))
		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", path, i)
			cleaned[i] = c.cleanValue(item, indexPath, depth+1, result)
		}
		return cleaned

	default:
		return value
	}
}

// shouldRemoveKey checks if a key matches any removal pattern
func (c *Cleaner) shouldRemoveKey(key string) bool {
	for _, pattern := range c.keyPatterns {
		if pattern.MatchString(key) {
			return true
		}
	}
	return false
}

// shouldProcessFile checks if a file path matches any path pattern
func (c *Cleaner) shouldProcessFile(filePath string) bool {
	if len(c.pathPatterns) == 0 {
		return true // Process all files if no patterns specified
	}

	for _, pattern := range c.pathPatterns {
		// Convert glob pattern to match
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
		// Also check if the pattern is contained in the path for ** patterns
		if strings.Contains(pattern, "**") {
			// Simple ** matching - could be enhanced
			simplifiedPattern := strings.ReplaceAll(pattern, "**", "*")
			if matched, _ := filepath.Match(simplifiedPattern, filePath); matched {
				return true
			}
			// Check if the path contains the pattern structure
			if matchesWildcardPattern(filePath, pattern) {
				return true
			}
		}
	}
	return false
}

// matchesWildcardPattern handles ** wildcard matching
func matchesWildcardPattern(path, pattern string) bool {
	// Simple implementation for ** matching
	// Convert pattern to regex
	pattern = strings.ReplaceAll(pattern, "**", ".*")
	pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
	pattern = "^" + pattern + "$"

	if regex, err := regexp.Compile(pattern); err == nil {
		return regex.MatchString(path)
	}
	return false
}

// buildPath constructs a dot-separated path
func buildPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// IsEnabled returns whether the cleaner is enabled
func (c *Cleaner) IsEnabled() bool {
	if c.config.Globals.Mappings == nil || c.config.Globals.Mappings.Cleaner == nil {
		return false
	}
	return c.config.Globals.Mappings.Cleaner.Enabled
}

// GetDescription returns the cleaner description
func (c *Cleaner) GetDescription() string {
	if c.config.Globals.Mappings == nil || c.config.Globals.Mappings.Cleaner == nil {
		return ""
	}
	return c.config.Globals.Mappings.Cleaner.Description
}

// GetKeyPatterns returns the configured key patterns
func (c *Cleaner) GetKeyPatterns() []string {
	if c.config.Globals.Mappings == nil || c.config.Globals.Mappings.Cleaner == nil {
		return nil
	}
	return c.config.Globals.Mappings.Cleaner.KeyPatterns
}

// GetPathPatterns returns the configured path patterns
func (c *Cleaner) GetPathPatterns() []string {
	if c.config.Globals.Mappings == nil || c.config.Globals.Mappings.Cleaner == nil {
		return nil
	}
	return c.config.Globals.Mappings.Cleaner.PathPatterns
}
