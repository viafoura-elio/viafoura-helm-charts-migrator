// Package keycase provides utilities for converting YAML keys to camelCase
package keycase

import (
	"regexp"
	"strings"
	"unicode"

	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// Converter handles key case conversion with configurable rules
type Converter struct {
	// SkipJavaProperties skips keys that look like Java properties (contain dots)
	SkipJavaProperties bool
	// SkipUppercaseKeys skips keys that are mostly uppercase
	SkipUppercaseKeys bool
	// MinUppercaseChars is the minimum number of consecutive uppercase chars to skip
	MinUppercaseChars int
	// PreserveSpecialKeys is a map of keys to always preserve
	PreserveSpecialKeys map[string]bool
}

// NewConverter creates a new converter with sensible defaults
func NewConverter() *Converter {
	return &Converter{
		SkipJavaProperties:  true,
		SkipUppercaseKeys:   true,
		MinUppercaseChars:   3,
		PreserveSpecialKeys: make(map[string]bool),
	}
}

// ConvertDocument converts all keys in a YAML document to camelCase
func (c *Converter) ConvertDocument(data []byte) ([]byte, error) {
	var doc interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	converted := c.ConvertValue(doc)
	return yaml.Marshal(converted)
}

// ConvertValue recursively converts all keys in a value to camelCase
func (c *Converter) ConvertValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return c.ConvertMap(v)
	case map[interface{}]interface{}:
		// Convert to map[string]interface{} first
		stringMap := make(map[string]interface{})
		for k, val := range v {
			if strKey, ok := k.(string); ok {
				stringMap[strKey] = val
			}
		}
		return c.ConvertMap(stringMap)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = c.ConvertValue(item)
		}
		return result
	default:
		return value
	}
}

// ConvertMap converts all keys in a map to camelCase
func (c *Converter) ConvertMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range m {
		// Convert the key if it should be converted
		newKey := key
		if c.shouldConvert(key) {
			newKey = c.convertKey(key)
		}

		// Recursively convert the value
		result[newKey] = c.ConvertValue(value)
	}

	return result
}

// shouldConvert determines if a key should be converted
func (c *Converter) shouldConvert(key string) bool {
	// Check if it's a special key to preserve
	if c.PreserveSpecialKeys[key] {
		return false
	}

	// Skip Java properties (keys with dots)
	if c.SkipJavaProperties && strings.Contains(key, ".") {
		return false
	}

	// Skip keys that start with uppercase letters (likely constants/special keys)
	if c.SkipUppercaseKeys && len(key) > 0 && unicode.IsUpper(rune(key[0])) {
		// Check if it has consecutive uppercase characters
		uppercaseCount := 0
		for _, ch := range key {
			if unicode.IsUpper(ch) {
				uppercaseCount++
				if uppercaseCount >= c.MinUppercaseChars {
					return false
				}
			} else {
				break
			}
		}
	}

	// Skip if already in camelCase
	if isCamelCase(key) {
		return false
	}

	// Skip keys that look like UUIDs or hex values
	if looksLikeUUID(key) || isHexValue(key) {
		return false
	}

	return true
}

// convertKey converts a key from snake_case or kebab-case to camelCase
func (c *Converter) convertKey(key string) string {
	// Handle empty string
	if key == "" {
		return key
	}

	// Replace underscores and hyphens with spaces for processing
	key = strings.ReplaceAll(key, "_", " ")
	key = strings.ReplaceAll(key, "-", " ")

	// Split into words
	words := strings.Fields(key)
	if len(words) == 0 {
		return key
	}

	// First word is lowercase
	result := strings.ToLower(words[0])

	// Subsequent words are title case
	for i := 1; i < len(words); i++ {
		if words[i] != "" {
			result += strings.Title(strings.ToLower(words[i]))
		}
	}

	return result
}

// isCamelCase checks if a string is already in camelCase
func isCamelCase(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Must start with lowercase
	if !unicode.IsLower(rune(s[0])) {
		return false
	}

	// Should not contain underscores or hyphens
	if strings.Contains(s, "_") || strings.Contains(s, "-") {
		return false
	}

	// Should have at least one uppercase letter (for multi-word camelCase)
	hasUppercase := false
	for _, ch := range s {
		if unicode.IsUpper(ch) {
			hasUppercase = true
			break
		}
	}

	// Single lowercase word is considered already in correct format
	return !hasUppercase || hasUppercase
}

// looksLikeUUID checks if a string looks like a UUID
func looksLikeUUID(s string) bool {
	// Simple UUID pattern check (with or without hyphens)
	uuidPattern := regexp.MustCompile(`^[a-fA-F0-9]{8}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{12}$`)
	return uuidPattern.MatchString(s)
}

// isHexValue checks if a string is a hex value
func isHexValue(s string) bool {
	if len(s) < 4 {
		return false
	}

	// Check if all characters are hex digits
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// Stats holds statistics about key conversion
type Stats struct {
	TotalKeys        int
	ConvertedKeys    int
	SkippedKeys      int
	JavaProperties   int
	UppercaseKeys    int
	UUIDKeys         int
	AlreadyCamelCase int
}

// ConvertWithStats converts and returns statistics
func (c *Converter) ConvertWithStats(value interface{}) (interface{}, *Stats) {
	stats := &Stats{}
	result := c.convertValueWithStats(value, stats)
	return result, stats
}

// convertValueWithStats converts and collects statistics
func (c *Converter) convertValueWithStats(value interface{}, stats *Stats) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return c.convertMapWithStats(v, stats)
	case map[interface{}]interface{}:
		// Convert to map[string]interface{} first
		stringMap := make(map[string]interface{})
		for k, val := range v {
			if strKey, ok := k.(string); ok {
				stringMap[strKey] = val
			}
		}
		return c.convertMapWithStats(stringMap, stats)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = c.convertValueWithStats(item, stats)
		}
		return result
	default:
		return value
	}
}

// convertMapWithStats converts map and collects statistics
func (c *Converter) convertMapWithStats(m map[string]interface{}, stats *Stats) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range m {
		stats.TotalKeys++

		newKey := key
		if c.PreserveSpecialKeys[key] {
			stats.SkippedKeys++
		} else if c.SkipJavaProperties && strings.Contains(key, ".") {
			stats.JavaProperties++
			stats.SkippedKeys++
		} else if c.SkipUppercaseKeys && hasConsecutiveUppercase(key, c.MinUppercaseChars) {
			stats.UppercaseKeys++
			stats.SkippedKeys++
		} else if looksLikeUUID(key) {
			stats.UUIDKeys++
			stats.SkippedKeys++
		} else if isCamelCase(key) {
			stats.AlreadyCamelCase++
		} else {
			newKey = c.convertKey(key)
			stats.ConvertedKeys++
		}

		// Recursively convert the value
		result[newKey] = c.convertValueWithStats(value, stats)
	}

	return result
}

// hasConsecutiveUppercase checks if a string has consecutive uppercase characters
func hasConsecutiveUppercase(s string, minCount int) bool {
	if len(s) == 0 {
		return false
	}

	uppercaseCount := 0
	for _, ch := range s {
		if unicode.IsUpper(ch) {
			uppercaseCount++
			if uppercaseCount >= minCount {
				return true
			}
		} else {
			uppercaseCount = 0
		}
	}
	return false
}

