package transformers

import (
	"fmt"
	"strings"

	"helm-charts-migrator/v1/pkg/keycase"
	"helm-charts-migrator/v1/pkg/logger"
)

// CamelCaseTransformer converts map keys to camelCase
type CamelCaseTransformer struct {
	converter *keycase.Converter
	log       *logger.NamedLogger
}

// NewCamelCaseTransformer creates a new camelCase transformer
func NewCamelCaseTransformer(skipJavaProperties, skipUppercase bool, minUppercase int) *CamelCaseTransformer {
	converter := keycase.NewConverter()
	converter.SkipJavaProperties = skipJavaProperties
	converter.SkipUppercaseKeys = skipUppercase
	converter.MinUppercaseChars = minUppercase
	
	return &CamelCaseTransformer{
		converter: converter,
		log:       logger.WithName("camelcase-transformer"),
	}
}

func (t *CamelCaseTransformer) Name() string {
	return "camelCase"
}

func (t *CamelCaseTransformer) Description() string {
	return "Converts map keys from snake_case to camelCase"
}

func (t *CamelCaseTransformer) Priority() int {
	return 10 // Early in the chain
}

func (t *CamelCaseTransformer) Validate(data interface{}) error {
	if _, ok := data.(map[string]interface{}); !ok {
		return fmt.Errorf("camelCase transformer requires map[string]interface{}, got %T", data)
	}
	return nil
}

func (t *CamelCaseTransformer) Transform(data interface{}) (interface{}, error) {
	input, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid input type: %T", data)
	}
	
	result := t.converter.ConvertMap(input)
	t.log.V(3).InfoS("Converted keys to camelCase", "keyCount", len(result))
	
	return result, nil
}

// KeyNormalizerTransformer normalizes key names based on patterns
type KeyNormalizerTransformer struct {
	patterns map[string]string // oldPattern -> newPattern
	log      *logger.NamedLogger
}

// NewKeyNormalizerTransformer creates a new key normalizer
func NewKeyNormalizerTransformer(patterns map[string]string) *KeyNormalizerTransformer {
	return &KeyNormalizerTransformer{
		patterns: patterns,
		log:      logger.WithName("key-normalizer"),
	}
}

func (t *KeyNormalizerTransformer) Name() string {
	return "keyNormalizer"
}

func (t *KeyNormalizerTransformer) Description() string {
	return "Normalizes key names based on configured patterns"
}

func (t *KeyNormalizerTransformer) Priority() int {
	return 20 // After camelCase conversion
}

func (t *KeyNormalizerTransformer) Validate(data interface{}) error {
	if _, ok := data.(map[string]interface{}); !ok {
		return fmt.Errorf("key normalizer requires map[string]interface{}, got %T", data)
	}
	return nil
}

func (t *KeyNormalizerTransformer) Transform(data interface{}) (interface{}, error) {
	input, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid input type: %T", data)
	}
	
	result := t.normalizeKeys(input)
	t.log.V(3).InfoS("Normalized keys", "patternCount", len(t.patterns))
	
	return result, nil
}

func (t *KeyNormalizerTransformer) normalizeKeys(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		// Check if key matches any pattern
		newKey := key
		for oldPattern, newPattern := range t.patterns {
			if strings.Contains(key, oldPattern) {
				newKey = strings.ReplaceAll(key, oldPattern, newPattern)
				t.log.V(4).InfoS("Normalized key", "old", key, "new", newKey)
				break
			}
		}
		
		// Recursively process nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			result[newKey] = t.normalizeKeys(nestedMap)
		} else {
			result[newKey] = value
		}
	}
	
	return result
}

// ValueDefaultsTransformer adds default values for missing keys
type ValueDefaultsTransformer struct {
	defaults map[string]interface{}
	log      *logger.NamedLogger
}

// NewValueDefaultsTransformer creates a new defaults transformer
func NewValueDefaultsTransformer(defaults map[string]interface{}) *ValueDefaultsTransformer {
	return &ValueDefaultsTransformer{
		defaults: defaults,
		log:      logger.WithName("value-defaults"),
	}
}

func (t *ValueDefaultsTransformer) Name() string {
	return "valueDefaults"
}

func (t *ValueDefaultsTransformer) Description() string {
	return "Adds default values for missing keys"
}

func (t *ValueDefaultsTransformer) Priority() int {
	return 30 // After normalization
}

func (t *ValueDefaultsTransformer) Validate(data interface{}) error {
	if _, ok := data.(map[string]interface{}); !ok {
		return fmt.Errorf("value defaults transformer requires map[string]interface{}, got %T", data)
	}
	return nil
}

func (t *ValueDefaultsTransformer) Transform(data interface{}) (interface{}, error) {
	input, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid input type: %T", data)
	}
	
	result := t.applyDefaults(input, t.defaults)
	t.log.V(3).InfoS("Applied default values", "defaultCount", len(t.defaults))
	
	return result, nil
}

func (t *ValueDefaultsTransformer) applyDefaults(data, defaults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy existing data
	for k, v := range data {
		result[k] = v
	}
	
	// Apply defaults for missing keys
	for key, defaultValue := range defaults {
		if _, exists := result[key]; !exists {
			result[key] = defaultValue
			t.log.V(4).InfoS("Added default value", "key", key, "value", defaultValue)
		} else if nestedMap, ok := result[key].(map[string]interface{}); ok {
			// Recursively apply defaults to nested maps
			if nestedDefaults, ok := defaultValue.(map[string]interface{}); ok {
				result[key] = t.applyDefaults(nestedMap, nestedDefaults)
			}
		}
	}
	
	return result
}

// PathRemapTransformer remaps values from one path to another
type PathRemapTransformer struct {
	mappings map[string]string // sourcePath -> targetPath
	log      *logger.NamedLogger
}

// NewPathRemapTransformer creates a new path remap transformer
func NewPathRemapTransformer(mappings map[string]string) *PathRemapTransformer {
	return &PathRemapTransformer{
		mappings: mappings,
		log:      logger.WithName("path-remap"),
	}
}

func (t *PathRemapTransformer) Name() string {
	return "pathRemap"
}

func (t *PathRemapTransformer) Description() string {
	return "Remaps values from one path to another in the structure"
}

func (t *PathRemapTransformer) Priority() int {
	return 40 // After defaults
}

func (t *PathRemapTransformer) Validate(data interface{}) error {
	if _, ok := data.(map[string]interface{}); !ok {
		return fmt.Errorf("path remap transformer requires map[string]interface{}, got %T", data)
	}
	return nil
}

func (t *PathRemapTransformer) Transform(data interface{}) (interface{}, error) {
	input, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid input type: %T", data)
	}
	
	result := deepCopyMap(input)
	
	// Apply each mapping
	for sourcePath, targetPath := range t.mappings {
		value := t.getValueAtPath(result, sourcePath)
		if value != nil {
			t.setValueAtPath(result, targetPath, value)
			t.deleteValueAtPath(result, sourcePath)
			t.log.V(4).InfoS("Remapped path", "from", sourcePath, "to", targetPath)
		}
	}
	
	t.log.V(3).InfoS("Applied path remappings", "mappingCount", len(t.mappings))
	return result, nil
}

func (t *PathRemapTransformer) getValueAtPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}
		
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	
	return nil
}

func (t *PathRemapTransformer) setValueAtPath(data map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}
		
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			// Can't traverse further
			return
		}
	}
}

func (t *PathRemapTransformer) deleteValueAtPath(data map[string]interface{}, path string) {
	parts := strings.Split(path, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
			return
		}
		
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return
		}
	}
}

// Helper function to deep copy a map
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
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

// Helper function to deep copy a slice
func deepCopySlice(s []interface{}) []interface{} {
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