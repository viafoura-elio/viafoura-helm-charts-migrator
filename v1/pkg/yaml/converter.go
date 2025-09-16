package yaml

import (
	"strings"
	"unicode"

	"github.com/elioetibr/yaml"
)

// Converter handles YAML key case conversion
type Converter struct {
	// Configuration options
	preserveJavaProperties bool
	preserveUppercaseKeys  bool
}

// NewConverter creates a new YAML case converter
func NewConverter() *Converter {
	return &Converter{
		preserveJavaProperties: true,
		preserveUppercaseKeys:  true,
	}
}

// ConvertYAML converts all keys in a YAML document to camelCase
func (c *Converter) ConvertYAML(input []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(input, &data); err != nil {
		return nil, err
	}

	converted := c.convertValue(data)
	return yaml.Marshal(converted)
}

// ConvertMap converts all keys in a map to camelCase
func (c *Converter) ConvertMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		newKey := c.convertKey(key)
		result[newKey] = c.convertValue(value)
	}
	return result
}

// convertValue recursively converts values
func (c *Converter) convertValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[interface{}]interface{}:
		// Convert to map[string]interface{} first
		stringMap := make(map[string]interface{})
		for key, val := range v {
			stringMap[toString(key)] = val
		}
		return c.ConvertMap(stringMap)
	case map[string]interface{}:
		return c.ConvertMap(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = c.convertValue(item)
		}
		return result
	default:
		return value
	}
}

// convertKey converts a single key to camelCase based on rules
func (c *Converter) convertKey(key string) string {
	// Rule 1: If key contains dots (Java properties style), keep as-is
	if c.preserveJavaProperties && strings.Contains(key, ".") {
		return key
	}

	// Rule 2: If key has 3+ uppercase letters before first "_" or "." AND has a separator, keep as-is
	// This preserves things like AWS_PROFILE, JAVA_OPTIONS but not TAG
	if c.preserveUppercaseKeys && c.hasMultipleUppercasePrefix(key) && strings.ContainsAny(key, "_.") {
		return key
	}

	// Rule 3: Check if it's a UUID (preserve as-is)
	if c.isUUID(key) {
		return key
	}

	// Rule 4: If already in camelCase, keep as-is
	if c.isCamelCase(key) {
		return key
	}

	// Convert to camelCase
	return c.toCamelCase(key)
}

// isUUID checks if a string looks like a UUID
func (c *Converter) isUUID(s string) bool {
	// Basic UUID pattern check: 8-4-4-4-12 hex characters with dashes
	if len(s) != 36 {
		return false
	}

	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}

	// Check part lengths
	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		return false
	}

	// Check if all parts are hex
	for _, part := range parts {
		for _, r := range part {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}

	return true
}

// hasMultipleUppercasePrefix checks if key has 3+ uppercase letters before first separator
func (c *Converter) hasMultipleUppercasePrefix(key string) bool {
	// Find position of first separator
	sepIndex := strings.IndexAny(key, "_.")
	prefix := key
	if sepIndex > 0 {
		prefix = key[:sepIndex]
	}

	// Count uppercase letters in prefix
	uppercaseCount := 0
	for _, r := range prefix {
		if unicode.IsUpper(r) {
			uppercaseCount++
		}
	}

	return uppercaseCount >= 3
}

// isCamelCase checks if a string is already in camelCase
func (c *Converter) isCamelCase(s string) bool {
	// Empty string
	if len(s) == 0 {
		return true
	}

	// Single lowercase character is camelCase
	if len(s) == 1 {
		return unicode.IsLower(rune(s[0]))
	}

	// Check if it contains separators
	if strings.ContainsAny(s, "_-. ") {
		return false
	}

	// Check if first character is lowercase and has at least one uppercase
	if !unicode.IsLower(rune(s[0])) {
		return false
	}

	// Check if it has mixed case (indicating camelCase)
	hasUpper := false
	for _, r := range s[1:] {
		if unicode.IsUpper(r) {
			hasUpper = true
			break
		}
	}

	return hasUpper || !strings.ContainsAny(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

// toCamelCase converts a string to camelCase
func (c *Converter) toCamelCase(s string) string {
	// Handle special cases
	if s == "" {
		return ""
	}

	// Check if it's all uppercase without separators (like TAG, URL, API)
	if s == strings.ToUpper(s) && !strings.ContainsAny(s, "_-. ") {
		return strings.ToLower(s)
	}

	// Handle PascalCase (like ConfigMap, MaxSurge)
	if !strings.ContainsAny(s, "_- ") && len(s) > 0 {
		// Check if it starts with uppercase (PascalCase)
		if unicode.IsUpper(rune(s[0])) {
			// Split on uppercase letters to handle PascalCase
			var words []string
			var currentWord []rune

			for i, r := range s {
				if i > 0 && unicode.IsUpper(r) {
					// Check if next char is lowercase (new word boundary)
					if i < len(s)-1 {
						nextRune := rune(s[i+1])
						if unicode.IsLower(nextRune) && len(currentWord) > 0 {
							// New word starts
							words = append(words, string(currentWord))
							currentWord = []rune{r}
						} else {
							currentWord = append(currentWord, r)
						}
					} else {
						currentWord = append(currentWord, r)
					}
				} else {
					currentWord = append(currentWord, r)
				}
			}
			if len(currentWord) > 0 {
				words = append(words, string(currentWord))
			}

			// Build camelCase from words
			if len(words) > 0 {
				result := strings.ToLower(words[0])
				for i := 1; i < len(words); i++ {
					if words[i] != "" {
						result += strings.ToUpper(words[i][:1]) + strings.ToLower(words[i][1:])
					}
				}
				return result
			}
		}
	}

	// Replace separators with spaces for processing
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// Split into words
	words := strings.Fields(s)
	if len(words) == 0 {
		return strings.ToLower(s)
	}

	// Build camelCase string
	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		word := words[i]
		if word != "" {
			// Capitalize first letter, lowercase the rest
			result += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return result
}

// toString converts interface{} to string
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
