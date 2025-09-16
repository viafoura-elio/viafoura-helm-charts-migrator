package services

import (
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/keycase"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/secrets"
	"helm-charts-migrator/v1/pkg/yaml"
)

// transformationService implements TransformationService interface
type transformationService struct {
	log             *logger.NamedLogger
	config          *config.Config
	secretExtractor *secrets.SecretExtractor
}

// NewTransformationService creates a new TransformationService with dependency injection
func NewTransformationService(cfg *config.Config) TransformationService {
	// Create secret extractor from config
	extractor, err := secrets.NewFromMainConfig(cfg)
	if err != nil {
		// Log error but continue with nil extractor
		// This allows the service to work even if secret extraction is misconfigured
		logger.WithName("transformation-service").Error(err, "Failed to create secret extractor")
	}

	return &transformationService{
		log:             logger.WithName("transformation-service"),
		config:          cfg,
		secretExtractor: extractor,
	}
}

// Transform applies transformations to values based on config
func (t *transformationService) Transform(values map[string]interface{}, cfg TransformConfig) (map[string]interface{}, error) {
	if values == nil {
		return make(map[string]interface{}), nil
	}

	// For now, just return a deep copy
	// TODO: Implement normalizations and transformations when ready
	result := yaml.DeepCopyMap(values)

	return result, nil
}

// NormalizeKeys normalizes keys in a values map
func (t *transformationService) NormalizeKeys(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return make(map[string]interface{})
	}

	// For now, just return a deep copy
	// TODO: Implement normalizations when ready
	result := yaml.DeepCopyMap(values)

	return result
}

// ExtractSecrets extracts secrets from values using the injected secret extractor
// This follows the Single Responsibility Principle by delegating to the specialized extractor
func (t *transformationService) ExtractSecrets(values map[string]interface{}) (secrets, cleaned map[string]interface{}) {
	if values == nil {
		return make(map[string]interface{}), make(map[string]interface{})
	}

	// Deep copy the values to avoid modifying the original
	cleanedValues := yaml.DeepCopyMap(values)
	extractedSecrets := make(map[string]interface{})

	// If no extractor is available, fall back to pattern-based extraction
	if t.secretExtractor == nil {
		// Get patterns from configuration - no fallback
		var patterns []string
		if t.config != nil && t.config.Globals.Secrets != nil {
			patterns = t.config.Globals.Secrets.Patterns
		}

		// Only extract secrets if patterns are configured
		if len(patterns) > 0 {
			t.extractSecretsWithPatterns(cleanedValues, extractedSecrets, "", patterns)
		}

		return extractedSecrets, cleanedValues
	}

	// Use the dedicated secret extractor for more sophisticated detection
	yamlData, err := yaml.Marshal(values)
	if err != nil {
		t.log.Error(err, "Failed to marshal values for secret extraction")
		return extractedSecrets, cleanedValues
	}

	// Extract secrets using the dedicated extractor
	// Empty serviceName means use global patterns only
	result, err := t.secretExtractor.ExtractSecrets(yamlData, "")
	if err != nil {
		t.log.Error(err, "Failed to extract secrets")
		return extractedSecrets, cleanedValues
	}

	// Process extraction results
	for _, match := range result.Secrets {
		// Remove secret from cleaned values and add to secrets
		t.removeFromMap(cleanedValues, match.Path)
		t.setInMap(extractedSecrets, match.Path, match.Value)
		t.log.V(3).InfoS("Extracted secret",
			"path", match.Path,
			"classification", match.Classification,
			"confidence", match.Confidence)
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

// extractSecretsWithPatterns is the pattern-based extraction method
// Used as fallback when the specialized secretExtractor is not available
func (t *transformationService) extractSecretsWithPatterns(cleaned, secrets map[string]interface{}, path string, patterns []string) {
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
			t.extractSecretsWithPatterns(nestedMap, nestedSecrets, currentPath, patterns)

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

// removeFromMap removes a value from a map using a dot-separated path
func (t *transformationService) removeFromMap(m map[string]interface{}, path string) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return
	}

	// Navigate to the parent map
	current := m
	for i := 0; i < len(parts)-1; i++ {
		if next, ok := current[parts[i]].(map[string]interface{}); ok {
			current = next
		} else {
			return // Path doesn't exist
		}
	}

	// Remove the final key
	if len(parts) > 0 {
		delete(current, parts[len(parts)-1])
	}
}

// setInMap sets a value in a map using a dot-separated path
func (t *transformationService) setInMap(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return
	}

	// Navigate and create nested maps as needed
	current := m
	for i := 0; i < len(parts)-1; i++ {
		if next, ok := current[parts[i]].(map[string]interface{}); ok {
			current = next
		} else {
			// Create nested map
			next = make(map[string]interface{})
			current[parts[i]] = next
			current = next
		}
	}

	// Set the final value
	if len(parts) > 0 {
		current[parts[len(parts)-1]] = value
	}
}

// matchesPattern checks if a path matches a pattern string
func (t *transformationService) matchesPattern(path, pattern string) (bool, error) {
	// Simple string matching as fallback
	// The specialized extractor uses compiled regex patterns for better performance
	return strings.Contains(path, pattern), nil
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
