package services

import (
	"regexp"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/keycase"
	"helm-charts-migrator/v1/pkg/logger"
)

// transformationService implements TransformationService interface
type transformationService struct {
	log    *logger.NamedLogger
	config *config.Config
}

// NewTransformationService creates a new TransformationService
func NewTransformationService(cfg *config.Config) TransformationService {
	return &transformationService{
		log:    logger.WithName("transformation-service"),
		config: cfg,
	}
}

// Transform applies transformations to values based on config
func (t *transformationService) Transform(values map[string]interface{}, cfg TransformConfig) (map[string]interface{}, error) {
	if values == nil {
		return make(map[string]interface{}), nil
	}

	// For now, just return a deep copy
	// TODO: Implement normalizations and transformations when ready
	// Use merge with empty base to create a deep copy
	result := t.deepMerge(make(map[string]interface{}), values)

	return result, nil
}

// NormalizeKeys normalizes keys in a values map
func (t *transformationService) NormalizeKeys(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return make(map[string]interface{})
	}

	// For now, just return a deep copy
	// TODO: Implement normalizations when ready
	// Use merge with empty base to create a deep copy
	result := t.deepMerge(make(map[string]interface{}), values)

	return result
}

// ExtractSecrets extracts secrets from values
func (t *transformationService) ExtractSecrets(values map[string]interface{}) (secrets, cleaned map[string]interface{}) {
	if values == nil {
		return make(map[string]interface{}), make(map[string]interface{})
	}

	// Deep copy the values to avoid modifying the original
	// Deep copy the values to avoid modifying the original
	cleanedValues := t.deepMerge(make(map[string]interface{}), values)
	extractedSecrets := make(map[string]interface{})

	// Get patterns from configuration - no fallback
	var patterns []string
	if t.config != nil && t.config.Globals.Secrets != nil {
		patterns = t.config.Globals.Secrets.Patterns
	}

	// Only extract secrets if patterns are configured
	if len(patterns) > 0 {
		t.extractSecretsRecursive(cleanedValues, extractedSecrets, "", patterns)
	}

	return extractedSecrets, cleanedValues
}

// ConvertKeys converts keys based on configured rules (e.g., camelCase)
func (t *transformationService) ConvertKeys(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return make(map[string]interface{})
	}

	// Create converter with configuration
	converter := keycase.NewConverter()

	// Apply configuration from globals
	cfg := t.config.Globals.Converter
	converter.SkipJavaProperties = cfg.SkipJavaProperties
	converter.SkipUppercaseKeys = cfg.SkipUppercaseKeys
	converter.MinUppercaseChars = cfg.MinUppercaseChars

	// Convert the map
	return converter.ConvertMap(values)
}

// MergeValues merges multiple value sources
func (t *transformationService) MergeValues(base, override map[string]interface{}) map[string]interface{} {
	if base == nil {
		base = make(map[string]interface{})
	}
	if override == nil {
		return base
	}

	// Deep merge the values
	return t.deepMerge(base, override)
}

// extractSecretsRecursive recursively extracts secrets from nested structures
func (t *transformationService) extractSecretsRecursive(cleaned, secrets map[string]interface{}, path string, patterns []string) {
	for key, value := range cleaned {
		currentPath := key
		if path != "" {
			currentPath = path + "." + key
		}

		// Check if this key matches any of the configured regex patterns
		isSecret := false
		for _, pattern := range patterns {
			matched, err := t.matchesPattern(currentPath, pattern)
			if err != nil {
				t.log.V(4).InfoS("Pattern matching error", "pattern", pattern, "error", err)
				continue
			}
			if matched {
				isSecret = true
				break
			}
		}

		if isSecret {
			// Move to secrets
			secrets[key] = value
			delete(cleaned, key)
			t.log.V(3).InfoS("Extracted secret", "path", currentPath)
			continue
		}

		// Recursively process nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			nestedSecrets := make(map[string]interface{})
			t.extractSecretsRecursive(nestedMap, nestedSecrets, currentPath, patterns)

			if len(nestedSecrets) > 0 {
				// Add nested secrets
				secrets[key] = nestedSecrets
			}

			// Remove empty maps from cleaned values
			if len(nestedMap) == 0 {
				delete(cleaned, key)
			}
		}
	}
}

// matchesPattern checks if a path matches a regex pattern
func (t *transformationService) matchesPattern(path, pattern string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(path), nil
}

// deepMerge performs a deep merge of two maps
func (t *transformationService) deepMerge(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy base values
	for k, v := range base {
		result[k] = v
	}

	// Merge override values
	for k, v := range override {
		if existing, exists := result[k]; exists {
			// Both are maps, merge recursively
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if overrideMap, ok := v.(map[string]interface{}); ok {
					result[k] = t.deepMerge(existingMap, overrideMap)
					continue
				}
			}
		}
		// Override the value
		result[k] = v
	}

	return result
}
