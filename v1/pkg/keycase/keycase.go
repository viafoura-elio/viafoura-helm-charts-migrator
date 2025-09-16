// Package keycase provides utilities for converting YAML keys to camelCase
package keycase

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/elioetibr/yaml"
)

// Converter handles key case conversion with configurable rules
type Converter struct {
	// SkipJavaProperties skips keys that look like Java properties (contain dots)
	SkipJavaProperties bool
	// SkipUppercaseKeys skips keys with at least N consecutive uppercase characters
	SkipUppercaseKeys bool
	// MinUppercaseChars minimum consecutive uppercase chars to skip conversion (default: 3)
	MinUppercaseChars int
	// PreserveSpecialKeys preserves specific keys from conversion
	PreserveSpecialKeys map[string]bool
	// CustomTransform allows a custom transformation function
	CustomTransform func(string) string
}

// NewConverter creates a new converter with default settings
func NewConverter() *Converter {
	return &Converter{
		SkipJavaProperties:  true,
		SkipUppercaseKeys:   true,
		MinUppercaseChars:   3,
		PreserveSpecialKeys: make(map[string]bool),
	}
}

// ConvertNode recursively converts all keys in a YAML node to camelCase
func (c *Converter) ConvertNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	result := copyNode(node)
	c.convertNodeInPlace(result)
	return result
}

// ConvertDocument converts all keys in a YAML document to camelCase
func (c *Converter) ConvertDocument(data []byte) ([]byte, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}

	converted := c.ConvertNode(&node)
	return yaml.Marshal(converted)
}

// convertNodeInPlace recursively converts keys in place
func (c *Converter) convertNodeInPlace(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			c.convertNodeInPlace(child)
		}
	case yaml.MappingNode:
		// Process key-value pairs
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}

			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Convert the key if it's a scalar
			if keyNode.Kind == yaml.ScalarNode {
				originalKey := keyNode.Value
				if c.shouldConvert(originalKey) {
					keyNode.Value = c.convertKey(originalKey)
				}
			}

			// Recursively process the value
			c.convertNodeInPlace(valueNode)
		}
	case yaml.SequenceNode:
		// Process array elements
		for _, child := range node.Content {
			c.convertNodeInPlace(child)
		}
	}
}

// shouldConvert determines if a key should be converted
func (c *Converter) shouldConvert(key string) bool {
	// Skip if in preserve list
	if c.PreserveSpecialKeys[key] {
		return false
	}

	// Skip Java/Spring properties style (contains dots)
	if c.SkipJavaProperties && strings.Contains(key, ".") {
		return false
	}

	// Skip keys that start with 3+ consecutive uppercase characters
	if c.SkipUppercaseKeys && startsWithConsecutiveUppercase(key, c.MinUppercaseChars) {
		return false
	}

	// Skip UUID-like keys (contain multiple dashes with hex-like segments)
	if looksLikeUUID(key) {
		return false
	}

	// Already in camelCase
	if isCamelCase(key) {
		return false
	}

	return true
}

// convertKey converts a key to camelCase
func (c *Converter) convertKey(key string) string {
	// Use custom transform if provided
	if c.CustomTransform != nil {
		return c.CustomTransform(key)
	}

	return toCamelCase(key)
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}

	// Split by common delimiters (underscore, dash, space)
	delimiters := regexp.MustCompile(`[-_\s]+`)
	parts := delimiters.Split(s, -1)

	// Filter empty parts
	var words []string
	for _, part := range parts {
		if part != "" {
			words = append(words, part)
		}
	}

	if len(words) == 0 {
		return s
	}

	// Handle PascalCase input by splitting on capital letters
	var allWords []string
	for _, word := range words {
		subWords := splitPascalCase(word)
		allWords = append(allWords, subWords...)
	}

	if len(allWords) == 0 {
		return s
	}

	// Build camelCase result
	var result strings.Builder
	for i, word := range allWords {
		if i == 0 {
			// First word is lowercase
			result.WriteString(strings.ToLower(word))
		} else {
			// Subsequent words are capitalized
			if len(word) > 0 {
				result.WriteString(strings.ToUpper(word[:1]))
				if len(word) > 1 {
					result.WriteString(strings.ToLower(word[1:]))
				}
			}
		}
	}

	return result.String()
}

// splitPascalCase splits a PascalCase or camelCase word into separate words
func splitPascalCase(s string) []string {
	if s == "" {
		return []string{}
	}

	var words []string
	var current strings.Builder
	runes := []rune(s)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Check if this is the start of a new word
			if i > 0 {
				// Look ahead to see if this is an acronym
				if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					// This uppercase is followed by lowercase, start new word
					if current.Len() > 0 {
						words = append(words, current.String())
						current.Reset()
					}
				} else if i+1 < len(runes) && unicode.IsUpper(runes[i+1]) {
					// Multiple uppercase letters (acronym)
					// Don't split yet
				} else if i > 0 && unicode.IsLower(runes[i-1]) {
					// Previous was lowercase, this is start of new word
					if current.Len() > 0 {
						words = append(words, current.String())
						current.Reset()
					}
				}
			}
			current.WriteRune(r)
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// isCamelCase checks if a string is already in camelCase
func isCamelCase(s string) bool {
	if s == "" {
		return true
	}

	// camelCase should:
	// 1. Start with lowercase
	// 2. Not contain underscores, dashes, or spaces
	// 3. May contain uppercase letters for word boundaries

	if len(s) > 0 && unicode.IsUpper(rune(s[0])) {
		return false // Starts with uppercase (PascalCase)
	}

	if strings.ContainsAny(s, "-_ ") {
		return false // Contains delimiters
	}

	// Single lowercase word or proper camelCase
	return true
}

// hasConsecutiveUppercase checks if string has N consecutive uppercase characters
func hasConsecutiveUppercase(s string, minCount int) bool {
	if minCount <= 0 {
		return false
	}

	count := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			count++
			if count >= minCount {
				return true
			}
		} else {
			count = 0
		}
	}

	return false
}

// startsWithConsecutiveUppercase checks if string starts with N consecutive uppercase characters
func startsWithConsecutiveUppercase(s string, minCount int) bool {
	if minCount <= 0 || len(s) < minCount {
		return false
	}

	count := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			count++
			if count >= minCount {
				return true
			}
		} else {
			// Stop checking once we hit a non-uppercase character
			break
		}
	}

	return false
}

// looksLikeUUID checks if a string looks like a UUID
func looksLikeUUID(s string) bool {
	// Basic UUID pattern: 8-4-4-4-12 hex characters
	// Example: 550e8400-e29b-41d4-a716-446655440000
	// Also check for variants without full structure

	// Must contain at least 2 dash-separated segments
	parts := strings.Split(s, "-")
	if len(parts) < 2 {
		return false
	}

	// Check if parts look like hex values
	hexCount := 0
	for _, part := range parts {
		if len(part) > 0 && isHexString(part) {
			hexCount++
		}
	}

	// If most parts are hex-like, it's probably a UUID
	return hexCount >= len(parts)-1 && hexCount >= 2
}

// isHexString checks if a string contains only hex characters
func isHexString(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// copyNode creates a deep copy of a yaml.Node
func copyNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	result := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}

	if node.Alias != nil {
		result.Alias = copyNode(node.Alias)
	}

	if len(node.Content) > 0 {
		result.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			result.Content[i] = copyNode(child)
		}
	}

	return result
}

// ConvertMap converts all keys in a map to camelCase
func (c *Converter) ConvertMap(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range input {
		newKey := key
		if c.shouldConvert(key) {
			newKey = c.convertKey(key)
		}

		// Recursively convert nested maps
		switch v := value.(type) {
		case map[string]interface{}:
			result[newKey] = c.ConvertMap(v)
		case []interface{}:
			result[newKey] = c.convertSlice(v)
		default:
			result[newKey] = value
		}
	}

	return result
}

// convertSlice recursively converts maps within a slice
func (c *Converter) convertSlice(input []interface{}) []interface{} {
	result := make([]interface{}, len(input))

	for i, item := range input {
		switch v := item.(type) {
		case map[string]interface{}:
			result[i] = c.ConvertMap(v)
		case []interface{}:
			result[i] = c.convertSlice(v)
		default:
			result[i] = item
		}
	}

	return result
}

// Stats holds statistics about the conversion
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
func (c *Converter) ConvertWithStats(node *yaml.Node) (*yaml.Node, *Stats) {
	stats := &Stats{}
	result := copyNode(node)
	c.convertNodeWithStats(result, stats)
	return result, stats
}

// convertNodeWithStats converts and collects statistics
func (c *Converter) convertNodeWithStats(node *yaml.Node, stats *Stats) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			c.convertNodeWithStats(child, stats)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}

			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.Kind == yaml.ScalarNode {
				stats.TotalKeys++
				originalKey := keyNode.Value

				if c.PreserveSpecialKeys[originalKey] {
					stats.SkippedKeys++
				} else if c.SkipJavaProperties && strings.Contains(originalKey, ".") {
					stats.JavaProperties++
					stats.SkippedKeys++
				} else if c.SkipUppercaseKeys && startsWithConsecutiveUppercase(originalKey, c.MinUppercaseChars) {
					stats.UppercaseKeys++
					stats.SkippedKeys++
				} else if looksLikeUUID(originalKey) {
					stats.UUIDKeys++
					stats.SkippedKeys++
				} else if isCamelCase(originalKey) {
					stats.AlreadyCamelCase++
				} else {
					keyNode.Value = c.convertKey(originalKey)
					stats.ConvertedKeys++
				}
			}

			c.convertNodeWithStats(valueNode, stats)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			c.convertNodeWithStats(child, stats)
		}
	case yaml.ScalarNode, yaml.AliasNode:
		// Scalar and alias nodes don't need conversion
		// They are leaf nodes that just contain values
	default:
		// Handle other node types gracefully (e.g., DocumentNode)
		for _, child := range node.Content {
			c.convertNodeWithStats(child, stats)
		}
	}
}
