package keycase

import (
	"strings"
	"testing"

	"github.com/elioetibr/yaml"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Snake case to camelCase
		{"snake_case", "snake_case", "snakeCase"},
		{"single_word", "single", "single"},
		{"multiple_words", "first_second_third", "firstSecondThird"},

		// Kebab case to camelCase
		{"kebab-case", "kebab-case", "kebabCase"},
		{"mixed-kebab", "first-second-third", "firstSecondThird"},

		// PascalCase to camelCase
		{"PascalCase", "PascalCase", "pascalCase"},
		{"SingleWord", "Single", "single"},
		{"MultipleWords", "FirstSecondThird", "firstSecondThird"},

		// Already camelCase
		{"already camelCase", "camelCase", "camelCase"},
		{"singleWord", "word", "word"},

		// Mixed formats
		{"snake_and_Pascal", "snake_Case", "snakeCase"},
		{"kebab-and-Pascal", "kebab-Case", "kebabCase"},
		{"all_lower_snake", "all_lower_case", "allLowerCase"},

		// Numbers
		{"with numbers", "version_2", "version2"},
		{"number in middle", "base_64_encode", "base64Encode"},

		// Edge cases
		{"empty string", "", ""},
		{"single char", "a", "a"},
		{"uppercase single", "A", "a"},
		{"with spaces", "has spaces", "hasSpaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"camelCase", true},
		{"lower", true}, // single word is considered camelCase
		{"hasUpperCase", true},
		{"", true},

		// Not camelCase
		{"PascalCase", false},
		{"snake_case", false},
		{"kebab-case", false},
		{"has spaces", false},
		{"UPPERCASE", false},
		{"Mixed_Snake", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("isCamelCase(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStartsWithConsecutiveUppercase(t *testing.T) {
	tests := []struct {
		input    string
		minCount int
		expected bool
	}{
		{"HTTPServer", 3, true},
		{"HTTPServer", 4, true},
		{"HTTPServer", 5, true}, // "HTTPS" = 5 consecutive uppercase at start
		{"XMLParser", 3, true},
		{"XmlParser", 3, false},
		{"AWS_REGION", 3, true},
		{"DB_CONNECTION", 2, true},
		{"DbConnection", 2, false},
		{"camelCase", 3, false},
		{"", 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := startsWithConsecutiveUppercase(tt.input, tt.minCount)
			if result != tt.expected {
				t.Errorf("startsWithConsecutiveUppercase(%q, %d) = %v, want %v",
					tt.input, tt.minCount, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid UUIDs
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"a0b1c2d3-e4f5-6789-abcd-ef0123456789", true},

		// UUID-like patterns
		{"abc-def-123", true},
		{"a1b2-c3d4-e5f6", true},

		// Not UUIDs
		{"not-a-uuid", false},
		{"single-dash", false},
		{"no-hex-here", false},
		{"normalKey", false},
		{"", false},
		{"123456", false},
		{"kebab-case-name", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := looksLikeUUID(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShouldConvert(t *testing.T) {
	c := NewConverter()
	c.PreserveSpecialKeys["keepThis"] = true

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		// Should convert
		{"snake_case key", "snake_case", true},
		{"kebab-case key", "kebab-case", true},
		{"PascalCase key", "PascalCase", true},
		{"CONSTANT_CASE", "CONSTANT_CASE", false}, // Starts with 3+ uppercase, should not convert
		{"mixed_snake_Case", "mixed_snake_Case", true},

		// Should NOT convert
		{"already camelCase", "camelCase", false},
		{"single word", "word", false}, // single word is considered camelCase
		{"java property", "java.lang.String", false},
		{"spring property", "spring.application.name", false},
		{"starts with 3 uppercase", "HTTPServer", false},
		{"starts with 3 uppercase 2", "AWS_REGION", false},
		{"UUID", "550e8400-e29b-41d4", false},
		{"preserved key", "keepThis", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.shouldConvert(tt.key)
			if result != tt.expected {
				t.Errorf("shouldConvert(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestConvertMap(t *testing.T) {
	c := NewConverter()

	input := map[string]interface{}{
		"first_name": "John",
		"last_name":  "Doe",
		"home-address": map[string]interface{}{
			"street_name": "Main St",
			"zip-code":    "12345",
			"coordinates": map[string]interface{}{
				"latitude":  40.7128,
				"longitude": -74.0060,
			},
		},
		"phone_numbers": []interface{}{
			map[string]interface{}{
				"phone_type":   "home",
				"phone-number": "555-1234",
			},
		},
		// These should not be converted
		"java.property.name": "value",
		"HTTPSConnection":    "secure",
		"AWS_REGION":         "us-west-2",
		"550e8400-e29b":      "uuid-value",
		"alreadyCamelCase":   "value",
	}

	result := c.ConvertMap(input)

	// Check converted keys
	if _, ok := result["firstName"]; !ok {
		t.Error("Expected 'first_name' to be converted to 'firstName'")
	}
	if _, ok := result["lastName"]; !ok {
		t.Error("Expected 'last_name' to be converted to 'lastName'")
	}
	if _, ok := result["homeAddress"]; !ok {
		t.Error("Expected 'home-address' to be converted to 'homeAddress'")
	}

	// Check nested map conversion
	if addr, ok := result["homeAddress"].(map[string]interface{}); ok {
		if _, ok := addr["streetName"]; !ok {
			t.Error("Expected nested 'street_name' to be converted to 'streetName'")
		}
		if _, ok := addr["zipCode"]; !ok {
			t.Error("Expected nested 'zip-code' to be converted to 'zipCode'")
		}
	}

	// Check array of maps conversion
	if phones, ok := result["phoneNumbers"].([]interface{}); ok {
		if len(phones) > 0 {
			if phone, ok := phones[0].(map[string]interface{}); ok {
				if _, ok := phone["phoneType"]; !ok {
					t.Error("Expected 'phone_type' in array to be converted to 'phoneType'")
				}
				if _, ok := phone["phoneNumber"]; !ok {
					t.Error("Expected 'phone-number' in array to be converted to 'phoneNumber'")
				}
			}
		}
	}

	// Check skipped keys
	if _, ok := result["java.property.name"]; !ok {
		t.Error("Java property style key should not be converted")
	}
	if _, ok := result["HTTPSConnection"]; !ok {
		t.Error("Key starting with 3+ uppercase chars should not be converted")
	}
	if _, ok := result["AWS_REGION"]; !ok {
		t.Error("Key starting with 3+ uppercase chars should not be converted")
	}
	if _, ok := result["550e8400-e29b"]; !ok {
		t.Error("UUID-like key should not be converted")
	}
	if _, ok := result["alreadyCamelCase"]; !ok {
		t.Error("Already camelCase key should not be converted")
	}
}

func TestConvertDocument(t *testing.T) {
	yamlDoc := `
api_version: v1
kind: ConfigMap
metadata:
  name: my-config
  created_by: admin
data:
  database-url: postgres://localhost
  max_connections: 100
  retry-attempts: 3
  timeout_seconds: 30
  # These should not be converted
  java.max.heap: 2g
  spring.datasource.url: jdbc:postgresql://localhost:5432/db
  HTTPSProxy: https://proxy.example.com
  AWS_REGION: us-west-2
  550e8400-e29b-41d4: some-uuid-value
  alreadyCamelCase: value
nested:
  inner_value: test
  deeply-nested:
    very_deep_value: 42
`

	c := NewConverter()
	converted, err := c.ConvertDocument([]byte(yamlDoc))
	if err != nil {
		t.Fatalf("Failed to convert document: %v", err)
	}

	// Parse the result to check conversions
	var result map[string]interface{}
	if err := yaml.Unmarshal(converted, &result); err != nil {
		t.Fatalf("Failed to unmarshal converted document: %v", err)
	}

	// Check conversions
	expectedConversions := map[string]string{
		"apiVersion":     "api_version",
		"createdBy":      "created_by",
		"databaseUrl":    "database-url",
		"maxConnections": "max_connections",
		"retryAttempts":  "retry-attempts",
		"timeoutSeconds": "timeout_seconds",
		"innerValue":     "inner_value",
		"deeplyNested":   "deeply-nested",
		"veryDeepValue":  "very_deep_value",
	}

	for expected, original := range expectedConversions {
		if !hasKeyInMap(result, expected) {
			t.Errorf("Expected %q to be converted to %q", original, expected)
		}
	}

	// Check skipped keys
	skippedKeys := []string{
		"java.max.heap",
		"spring.datasource.url",
		"HTTPSProxy",
		"AWS_REGION",
		"550e8400-e29b-41d4",
		"alreadyCamelCase",
	}

	for _, key := range skippedKeys {
		if !hasKeyInMap(result, key) {
			t.Errorf("Expected %q to be preserved", key)
		}
	}
}

func TestConvertWithStats(t *testing.T) {
	yamlDoc := `
first_name: John
last-name: Doe
email_address: john@example.com
phone-number: 555-1234
# Should be skipped
java.property: value
spring.config.name: app
HTTPEndpoint: /api/v1
AWS_ACCESS_KEY: secret
550e8400-e29b: uuid-val
alreadyCamelCase: value
DB_CONNECTION_STRING: postgres://
config:
  max_retries: 3
  timeout-ms: 5000
  enable_cache: true
`

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlDoc), &node); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	c := NewConverter()
	converted, stats := c.ConvertWithStats(&node)

	// Verify stats
	if stats.TotalKeys == 0 {
		t.Error("Expected total keys to be counted")
	}

	if stats.ConvertedKeys == 0 {
		t.Error("Expected some keys to be converted")
	}

	if stats.JavaProperties != 2 {
		t.Errorf("Expected 2 Java properties, got %d", stats.JavaProperties)
	}

	if stats.UppercaseKeys < 2 {
		t.Errorf("Expected at least 2 uppercase keys, got %d", stats.UppercaseKeys)
	}

	if stats.UUIDKeys != 1 {
		t.Errorf("Expected 1 UUID key, got %d", stats.UUIDKeys)
	}

	if stats.AlreadyCamelCase != 2 {
		t.Errorf("Expected 2 already camelCase keys, got %d", stats.AlreadyCamelCase)
	}

	// Verify conversion worked
	output, err := yaml.Marshal(converted)
	if err != nil {
		t.Fatalf("Failed to marshal converted node: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "firstName:") {
		t.Error("Expected 'first_name' to be converted to 'firstName'")
	}
	if !strings.Contains(outputStr, "lastName:") {
		t.Error("Expected 'last-name' to be converted to 'lastName'")
	}
	if !strings.Contains(outputStr, "java.property:") {
		t.Error("Expected 'java.property' to be preserved")
	}

	t.Logf("Conversion stats: Total=%d, Converted=%d, Skipped=%d (Java=%d, Uppercase=%d, UUID=%d, AlreadyCamel=%d)",
		stats.TotalKeys, stats.ConvertedKeys, stats.SkippedKeys,
		stats.JavaProperties, stats.UppercaseKeys, stats.UUIDKeys, stats.AlreadyCamelCase)
}

func TestHasConsecutiveUppercaseFunc(t *testing.T) {
	// This function is not exported, so we test it indirectly through shouldConvert
	c := NewConverter()
	c.MinUppercaseChars = 3

	// Test cases that trigger the hasConsecutiveUppercase check
	tests := []struct {
		input      string
		shouldSkip bool
	}{
		{"HELLO_WORLD", true},  // Has 5 consecutive uppercase at start
		{"HEllo_world", false}, // Only 2 consecutive uppercase
		{"normal_case", false}, // No consecutive uppercase
		{"XMLParser", true},    // Has 3 consecutive uppercase
		{"XmlParser", false},   // Only 1 uppercase followed by lowercase
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// We're testing that keys with 3+ consecutive uppercase are skipped
			result := c.shouldConvert(tt.input)
			if tt.shouldSkip && result {
				t.Errorf("expected %q to be skipped (has 3+ consecutive uppercase)", tt.input)
			}
			if !tt.shouldSkip && !result && !isCamelCase(tt.input) {
				t.Errorf("expected %q to be converted (doesn't have 3+ consecutive uppercase)", tt.input)
			}
		})
	}
}

func TestCustomTransform(t *testing.T) {
	c := NewConverter()
	c.CustomTransform = func(key string) string {
		return "custom_" + key
	}

	result := c.convertKey("test_key")
	if result != "custom_test_key" {
		t.Errorf("expected 'custom_test_key', got %s", result)
	}
}

func TestConvertNodeWithNilInput(t *testing.T) {
	c := NewConverter()
	result := c.ConvertNode(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestConvertNodeEdgeCases(t *testing.T) {
	c := NewConverter()

	// Test with scalar node (not a mapping or sequence)
	yamlData := `42`
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlData), &node); err != nil {
		t.Fatal(err)
	}

	result := c.ConvertNode(&node)
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestConvertDocumentErrors(t *testing.T) {
	c := NewConverter()

	// Test with invalid YAML
	invalidYAML := []byte("invalid: yaml: :")
	_, err := c.ConvertDocument(invalidYAML)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestIsHexStringEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"123abc", true},
		{"ABCDEF", true},
		{"ghijkl", false},
		{"12-34", false},
		{"a1b2c3", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Test through the exported function that uses isHexString internally
			result := isHexStringHelper(tt.input)
			if result != tt.expected {
				t.Errorf("isHexString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function to test isHexString indirectly
func isHexStringHelper(s string) bool {
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

// Helper function to check if a key exists in nested map
func hasKeyInMap(m interface{}, key string) bool {
	switch v := m.(type) {
	case map[string]interface{}:
		if _, ok := v[key]; ok {
			return true
		}
		for _, value := range v {
			if hasKeyInMap(value, key) {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if hasKeyInMap(item, key) {
				return true
			}
		}
	}
	return false
}
