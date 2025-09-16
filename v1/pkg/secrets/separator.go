package secrets

import (
	"fmt"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/yaml"
)

// Separator handles the extraction and separation of secrets from values
type Separator struct {
	extractor   *SecretExtractor
	config      *config.Config
	serviceName string // Current service being processed
	targetFile  string // Target file pattern being processed (e.g., "apps/{service}/values.yaml")
}

// SeparationResult contains the results of secret separation
type SeparationResult struct {
	ModifiedData     interface{}       `yaml:"-"` // The modified YAML data
	ExtractedSecrets []ExtractedSecret `yaml:"extracted_secrets"`
	MovedCount       int               `yaml:"moved_count"`
	Warnings         []string          `yaml:"warnings"`
}

// ExtractedSecret represents a secret that was extracted and moved
type ExtractedSecret struct {
	OriginalPath string `yaml:"original_path"`
	NewPath      string `yaml:"new_path"`
	Key          string `yaml:"key"`
	Value        string `yaml:"value,omitempty"`
	MaskedValue  string `yaml:"masked_value"`
}

// NewSeparator creates a new secret separator
func NewSeparator(extractor *SecretExtractor) *Separator {
	return &Separator{
		extractor: extractor,
		config:    extractor.config,
	}
}

// SetTargetFile sets the target file pattern for merge strategy lookups
func (s *Separator) SetTargetFile(targetFile string) {
	s.targetFile = targetFile
}

// SeparateSecrets extracts secrets from YAML data and moves them to a secrets section
func (s *Separator) SeparateSecrets(yamlData []byte, serviceName string) ([]byte, *SeparationResult, error) {
	// Store the service name for configuration lookups
	s.serviceName = serviceName

	result := &SeparationResult{
		ExtractedSecrets: make([]ExtractedSecret, 0),
		Warnings:         make([]string, 0),
	}

	// Parse YAML data
	var data map[string]interface{}
	if err := yaml.UnmarshalStrict(yamlData, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// First, detect all secrets using the extractor
	extraction, err := s.extractor.ExtractSecrets(yamlData, serviceName)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to extract secrets: %v", err))
		return yamlData, result, nil
	}

	// Get the store path from config
	storePath := s.getStorePath()

	// Build secrets map, starting with existing secrets if any
	secrets := make(map[string]interface{})

	// First, copy existing secrets to preserve them
	if existing, exists := data[storePath].(map[string]interface{}); exists {
		// Deep copy existing secrets
		for k, v := range existing {
			secrets[k] = v
		}
	}

	// Process each detected secret
	for _, secret := range extraction.Secrets {
		// Extract and move the secret
		if moved := s.moveSecretToMap(data, secrets, secret, result); moved {
			result.MovedCount++
		}
	}

	// Only update secrets section if we have secrets (new or existing)
	if len(secrets) > 0 {
		data[storePath] = secrets
	}

	result.ModifiedData = data

	// Marshal the modified data
	modifiedYAML, err := yaml.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal modified YAML: %v", err)
	}

	return modifiedYAML, result, nil
}

// moveSecretToMap moves a secret from data to the secrets map, preserving structure
func (s *Separator) moveSecretToMap(data map[string]interface{}, secrets map[string]interface{}, secret SecretMatch, result *SeparationResult) bool {
	// Find and extract the value by walking the data structure
	value, parent, key := s.findValue(data, secret.Path, "")
	if value == nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Could not find value at path %s", secret.Path))
		return false
	}

	// Determine where to place in secrets based on the original path
	targetPath := s.placeInSecrets(secrets, secret.Path, key, value)

	// Remove from original location
	if parent != nil {
		delete(parent, key)
	}

	// Record the extraction
	result.ExtractedSecrets = append(result.ExtractedSecrets, ExtractedSecret{
		OriginalPath: secret.Path,
		NewPath:      targetPath,
		Key:          secret.Key,
		MaskedValue:  secret.MaskedValue,
	})

	return true
}

// findValue recursively searches for a value at the given path
func (s *Separator) findValue(data interface{}, targetPath string, currentPath string) (value interface{}, parent map[string]interface{}, key string) {
	switch d := data.(type) {
	case map[string]interface{}:
		for k, v := range d {
			newPath := currentPath
			if newPath == "" {
				newPath = k
			} else {
				newPath = currentPath + "." + k
			}

			// Check if we found the target
			if newPath == targetPath {
				return v, d, k
			}

			// Recurse into nested structures
			if val, par, foundKey := s.findValue(v, targetPath, newPath); val != nil {
				return val, par, foundKey
			}
		}
	}
	return nil, nil, ""
}

// placeInSecrets determines where to place the secret and adds it
func (s *Separator) placeInSecrets(secrets map[string]interface{}, originalPath string, key string, value interface{}) string {
	// Get the store path from config (default to "secrets" if not set)
	storePath := s.getStorePath()

	// Try to find custom mapping for various levels of the path
	// Start with the container level and work up
	// e.g., for "configMap.application.properties.database.password"
	// try: "configMap.application.properties", "configMap.application", "configMap"
	customPath := ""
	pathParts := strings.Split(originalPath, ".")

	// Try progressively shorter paths to find a mapping
	for i := len(pathParts) - 1; i >= 2; i-- {
		testPath := strings.Join(pathParts[:i], ".")
		if mapped := s.getCustomMapping(testPath); mapped != "" {
			customPath = mapped
			break
		}
	}

	// If still no mapping, try the full original path
	if customPath == "" {
		customPath = s.getCustomMapping(originalPath)
	}

	if customPath != "" {
		// Parse the custom path to extract the structure
		parts := strings.Split(customPath, ".")
		if len(parts) > 1 && parts[0] == "secrets" {
			// Remove "secrets." prefix as it will be handled by the storePath
			customPath = strings.Join(parts[1:], ".")
		}

		// Now we need to place the value under the custom path with the original key
		// For example, if customPath is "application.conf" and key is "database.password"
		// we want secrets["application.conf"]["database.password"] = value

		// Ensure the container exists
		if _, exists := secrets[customPath]; !exists {
			secrets[customPath] = make(map[string]interface{})
		}

		// Add the value under the original key
		if container, ok := secrets[customPath].(map[string]interface{}); ok {
			container[key] = value
		} else {
			// If the container is not a map, create a new one
			secrets[customPath] = map[string]interface{}{
				key: value,
			}
		}

		return storePath + "." + customPath + "." + key
	}

	// Get the base path from config (default to "configMap" if not set)
	basePath := s.getBasePath()

	// If the path starts with the configured base path, mirror the structure under store path
	if basePath != "" && strings.HasPrefix(originalPath, basePath+".") {
		// Remove base path prefix
		subPath := strings.TrimPrefix(originalPath, basePath+".")

		// Check if this is a properties file structure
		if strings.Contains(subPath, ".properties.") {
			// Find where .properties ends
			propsIdx := strings.Index(subPath, ".properties.")
			if propsIdx > 0 {
				// Extract the properties file name (e.g., "root.properties")
				propsEnd := propsIdx + len(".properties")
				propsFile := subPath[:propsEnd]

				// Get or create the properties map
				if existing, exists := secrets[propsFile]; !exists {
					secrets[propsFile] = make(map[string]interface{})
				} else if _, isMap := existing.(map[string]interface{}); !isMap {
					// If it exists but is not a map, replace it
					secrets[propsFile] = make(map[string]interface{})
				}
				propsMap := secrets[propsFile].(map[string]interface{})

				// Add the key (everything after .properties.)
				finalKey := subPath[propsEnd+1:] // +1 for the dot
				propsMap[finalKey] = value

				return storePath + "." + propsFile + "." + finalKey
			}
		}

		// For simple keys directly under base path
		secrets[key] = value
		return storePath + "." + key
	}

	// For other paths, just use the key
	secrets[key] = value
	return storePath + "." + key
}

// getStorePath returns the configured store path or default
func (s *Separator) getStorePath() string {
	// First check for service-specific configuration
	if s.serviceName != "" && s.config != nil && s.config.Services != nil {
		if svc, exists := s.config.Services[s.serviceName]; exists && svc.Secrets != nil && svc.Secrets.Locations != nil {
			if svc.Secrets.Locations.StorePath != "" {
				return svc.Secrets.Locations.StorePath
			}
		}
	}

	// Fall back to global configuration
	if s.config != nil && s.config.Globals.Secrets.Locations != nil && s.config.Globals.Secrets.Locations.StorePath != "" {
		return s.config.Globals.Secrets.Locations.StorePath
	}

	// Default value
	return "secrets"
}

// getBasePath returns the configured base path or default
func (s *Separator) getBasePath() string {
	// First check for service-specific configuration
	if s.serviceName != "" && s.config != nil && s.config.Services != nil {
		if svc, exists := s.config.Services[s.serviceName]; exists && svc.Secrets != nil && svc.Secrets.Locations != nil {
			if svc.Secrets.Locations.BasePath != "" {
				return svc.Secrets.Locations.BasePath
			}
		}
	}

	// Fall back to global configuration
	if s.config != nil && s.config.Globals.Secrets.Locations != nil && s.config.Globals.Secrets.Locations.BasePath != "" {
		return s.config.Globals.Secrets.Locations.BasePath
	}

	// Default value
	return "configMap"
}

// getCustomMapping returns the custom mapping for a given path if configured
func (s *Separator) getCustomMapping(originalPath string) string {
	// If we have a target file, check for merge strategy mappings
	if s.targetFile != "" {
		// Check service-specific merge strategy
		if s.serviceName != "" && s.config != nil && s.config.Services != nil {
			if svc, exists := s.config.Services[s.serviceName]; exists && svc.Secrets != nil && svc.Secrets.Merging != nil {
				// Replace {service} placeholder in target file pattern
				resolvedTarget := strings.ReplaceAll(s.targetFile, "{service}", s.serviceName)

				// Debug logging (disabled)
				// fmt.Printf("DEBUG getCustomMapping: serviceName=%s, targetFile=%s, resolvedTarget=%s, originalPath=%s\n",
				//     s.serviceName, s.targetFile, resolvedTarget, originalPath)
				// for k, v := range svc.Secrets.Merging {
				// 	fmt.Printf("DEBUG getCustomMapping: Merging[%s]=%+v\n", k, v)
				// 	if v != nil && v.KeyMappings != nil {
				// 		for mapKey, mapVal := range v.KeyMappings {
				// 			fmt.Printf("  KeyMapping: %s -> %s\n", mapKey, mapVal)
				// 		}
				// 	}
				// }

				// Try exact match first
				if strategy, found := svc.Secrets.Merging[resolvedTarget]; found && strategy.KeyMappings != nil {
					if mappedPath, found := strategy.KeyMappings[originalPath]; found {
						return mappedPath
					}
				}

				// Also try the pattern as-is (with {service} placeholder)
				if strategy, found := svc.Secrets.Merging[s.targetFile]; found && strategy.KeyMappings != nil {
					if mappedPath, found := strategy.KeyMappings[originalPath]; found {
						return mappedPath
					}
				}

				// Try pattern with {service} replaced
				pattern := "apps/{service}/values.yaml"
				if strategy, found := svc.Secrets.Merging[pattern]; found && strategy.KeyMappings != nil {
					if mappedPath, found := strategy.KeyMappings[originalPath]; found {
						return mappedPath
					}
				}
			}
		}

		// Check global merge strategy
		if s.config != nil && s.config.Globals.Secrets.Merging != nil {
			// Replace {service} placeholder in target file pattern
			resolvedTarget := s.targetFile
			if s.serviceName != "" {
				resolvedTarget = strings.ReplaceAll(resolvedTarget, "{service}", s.serviceName)
			}

			// Check if we have a merge strategy for this target file
			if strategy, found := s.config.Globals.Secrets.Merging[resolvedTarget]; found && strategy.KeyMappings != nil {
				if mappedPath, found := strategy.KeyMappings[originalPath]; found {
					return mappedPath
				}
			}
		}
	}

	// Fall back to deprecated direct key mappings for backward compatibility
	// Check for service-specific key mappings (deprecated)
	if s.serviceName != "" && s.config != nil && s.config.Services != nil {
		if svc, exists := s.config.Services[s.serviceName]; exists && svc.Secrets != nil && svc.Secrets.KeyMappings != nil {
			if mappedPath, found := svc.Secrets.KeyMappings[originalPath]; found {
				return mappedPath
			}
		}
	}

	// Check global key mappings (deprecated)
	if s.config != nil && s.config.Globals.Secrets.KeyMappings != nil {
		if mappedPath, found := s.config.Globals.Secrets.KeyMappings[originalPath]; found {
			return mappedPath
		}
	}

	return ""
}
