package yaml

import (
	"strings"
	"testing"

	"github.com/elioetibr/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var yamlData = `
Autoscaling:
  enabled: true
  maxReplicas: 250
  minReplicas: 50
ConfigMap:
  root.properties:
    3f4beddd-2061-49b0-ae80-6f1f2ed65b37: 6b7bae62-1bfe-452a-9217-b15afb37d16a
    auth.dataSource.password: s4HfQWmuJ8yvuGAwo9
    auth.dataSource.serverName: auth.cluster-ccfeqgys4gfe.us-east-1.rds.amazonaws.com
    auth.dataSource.user: auth
    com.viafoura.heimdall.access.refresh.local_client_uuid: c45638fe-4da8-4de6-ade9-d733625bd986
    com.viafoura.heimdall.jwt.secret_1: secret_value_1
    com.viafoura.heimdall.jwt.secret_2: secret_value_2
    dataSource.password: Z7QAXypiYdtBZcTWjsq
    dataSource.user: heimdall
  vfmetrics.properties:
    datadog.tags: environment:viafoura
Deployment:
  MaxSurge: 10%
  Replicas: 50
Env:
  JAVA_OPTIONS: -XX:+DisableExplicitGC -Xms600m -Xmx600m
Image:
  Tag: v10.16.1
Ingress:
  infoHost: auth.viafoura.io
  orgHost: auth.viafoura.co
Resources:
  limits:
    cpu: 1
    memory: 1Gi
  requests:
    cpu: 250m
    memory: 1Gi
`

func TestConvertKey(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		expected string
		reason   string
	}{
		// Rule 1: Java properties style (with dots) - keep as-is
		{
			name:     "Java property with dots",
			input:    "auth.dataSource.password",
			expected: "auth.dataSource.password",
			reason:   "Java properties style should be preserved",
		},
		{
			name:     "Complex Java property",
			input:    "com.viafoura.heimdall.jwt.secret_1",
			expected: "com.viafoura.heimdall.jwt.secret_1",
			reason:   "Java properties with dots should be preserved",
		},
		{
			name:     "Java property root.properties",
			input:    "root.properties",
			expected: "root.properties",
			reason:   "Properties file reference should be preserved",
		},

		// Rule 2: 3+ uppercase letters before separator - keep as-is
		{
			name:     "AWS prefix",
			input:    "AWS_REGION",
			expected: "AWS_REGION",
			reason:   "3+ uppercase letters (AWS) before separator",
		},
		{
			name:     "JAVA_OPTIONS",
			input:    "JAVA_OPTIONS",
			expected: "JAVA_OPTIONS",
			reason:   "4 uppercase letters (JAVA) before separator",
		},
		{
			name:     "URL pattern",
			input:    "URL_BASE",
			expected: "URL_BASE",
			reason:   "3 uppercase letters (URL) before separator",
		},

		// Rule 3: Already in camelCase - keep as-is
		{
			name:     "Already camelCase",
			input:    "maxReplicas",
			expected: "maxReplicas",
			reason:   "Already in camelCase",
		},
		{
			name:     "Simple camelCase",
			input:    "enabled",
			expected: "enabled",
			reason:   "Simple lowercase is valid camelCase",
		},
		{
			name:     "Complex camelCase",
			input:    "infoHost",
			expected: "infoHost",
			reason:   "Already in camelCase",
		},

		// Convert to camelCase
		{
			name:     "PascalCase to camelCase",
			input:    "Autoscaling",
			expected: "autoscaling",
			reason:   "PascalCase should convert to camelCase",
		},
		{
			name:     "Snake case to camelCase",
			input:    "Max_Surge",
			expected: "maxSurge",
			reason:   "Snake case should convert to camelCase",
		},
		{
			name:     "Mixed PascalCase",
			input:    "MaxSurge",
			expected: "maxSurge",
			reason:   "PascalCase should convert to camelCase",
		},
		{
			name:     "Single uppercase word",
			input:    "Replicas",
			expected: "replicas",
			reason:   "Single word PascalCase to camelCase",
		},
		{
			name:     "All caps word",
			input:    "TAG",
			expected: "tag",
			reason:   "All caps should convert to lowercase",
		},
		{
			name:     "ConfigMap key",
			input:    "ConfigMap",
			expected: "configMap",
			reason:   "PascalCase ConfigMap to camelCase",
		},
		{
			name:     "Resources key",
			input:    "Resources",
			expected: "resources",
			reason:   "PascalCase Resources to camelCase",
		},
		{
			name:     "Deployment key",
			input:    "Deployment",
			expected: "deployment",
			reason:   "PascalCase Deployment to camelCase",
		},
		{
			name:     "Image.Tag path",
			input:    "Image",
			expected: "image",
			reason:   "PascalCase Image to camelCase",
		},
		{
			name:     "Ingress key",
			input:    "Ingress",
			expected: "ingress",
			reason:   "PascalCase Ingress to camelCase",
		},
		{
			name:     "Env key",
			input:    "Env",
			expected: "env",
			reason:   "PascalCase Env to camelCase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertKey(tt.input)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

func TestConvertYAMLDocument(t *testing.T) {
	converter := NewConverter()

	inputYAML := yamlData

	// Convert the YAML
	output, err := converter.ConvertYAML([]byte(inputYAML))
	require.NoError(t, err)

	// Parse the output to verify structure
	var result map[string]interface{}
	err = yaml.Unmarshal(output, &result)
	require.NoError(t, err)

	// Test top-level keys are converted to camelCase
	assert.Contains(t, result, "autoscaling", "Autoscaling should be converted to autoscaling")
	assert.Contains(t, result, "configMap", "ConfigMap should be converted to configMap")
	assert.Contains(t, result, "deployment", "Deployment should be converted to deployment")
	assert.Contains(t, result, "env", "Env should be converted to env")
	assert.Contains(t, result, "image", "Image should be converted to image")
	assert.Contains(t, result, "ingress", "Ingress should be converted to ingress")
	assert.Contains(t, result, "resources", "Resources should be converted to resources")

	// Test nested keys in autoscaling
	autoscaling := result["autoscaling"].(map[string]interface{})
	assert.Contains(t, autoscaling, "enabled")
	assert.Contains(t, autoscaling, "maxReplicas")
	assert.Contains(t, autoscaling, "minReplicas")

	// Test ConfigMap with Java properties (should be preserved)
	configMap := result["configMap"].(map[string]interface{})
	assert.Contains(t, configMap, "root.properties", "root.properties should be preserved")

	rootProps := configMap["root.properties"].(map[string]interface{})
	assert.Contains(t, rootProps, "auth.dataSource.password", "Java property should be preserved")
	assert.Contains(t, rootProps, "auth.dataSource.serverName", "Java property should be preserved")
	assert.Contains(t, rootProps, "auth.dataSource.user", "Java property should be preserved")
	assert.Contains(t, rootProps, "com.viafoura.heimdall.jwt.secret_1", "Java property with underscore should be preserved")
	assert.Contains(t, rootProps, "dataSource.password", "Java property should be preserved")

	// Test Deployment nested keys
	deployment := result["deployment"].(map[string]interface{})
	assert.Contains(t, deployment, "maxSurge", "MaxSurge should be converted to maxSurge")
	assert.Contains(t, deployment, "replicas", "Replicas should be converted to replicas")

	// Test Env with JAVA_OPTIONS (should be preserved due to 4 uppercase letters)
	env := result["env"].(map[string]interface{})
	assert.Contains(t, env, "JAVA_OPTIONS", "JAVA_OPTIONS should be preserved")

	// Test Image nested keys
	image := result["image"].(map[string]interface{})
	assert.Contains(t, image, "tag", "Tag should be converted to tag")

	// Test Ingress nested keys (already camelCase)
	ingress := result["ingress"].(map[string]interface{})
	assert.Contains(t, ingress, "infoHost", "infoHost should remain as-is")
	assert.Contains(t, ingress, "orgHost", "orgHost should remain as-is")

	// Test Resources nested structure
	resources := result["resources"].(map[string]interface{})
	assert.Contains(t, resources, "limits")
	assert.Contains(t, resources, "requests")
}

func TestPreserveJavaProperties(t *testing.T) {
	converter := NewConverter()

	javaProps := []string{
		"auth.dataSource.password",
		"com.viafoura.heimdall.jwt.secret_1",
		"com.viafoura.heimdall.provider.loginradius.secret",
		"dataSource.user",
		"root.properties",
		"vfmetrics.properties",
		"datadog.tags",
	}

	for _, prop := range javaProps {
		result := converter.convertKey(prop)
		assert.Equal(t, prop, result, "Java property %s should be preserved", prop)
	}
}

func TestPreserveUppercaseKeys(t *testing.T) {
	converter := NewConverter()

	uppercaseKeys := []string{
		"JAVA_OPTIONS",
		"AWS_REGION",
		"URL_BASE",
		"API_KEY",
		"HTTP_PORT",
		"SQL_WRITE",
	}

	for _, key := range uppercaseKeys {
		result := converter.convertKey(key)
		assert.Equal(t, key, result, "Uppercase key %s should be preserved", key)
	}
}

func TestComplexNestedStructure(t *testing.T) {
	converter := NewConverter()

	input := map[string]interface{}{
		"TopLevel": map[string]interface{}{
			"NestedKey":         "value1",
			"another_key":       "value2",
			"java.property.key": "value3",
			"AWS_CONFIG":        "value4",
			"alreadyCamelCase":  "value5",
		},
		"ListExample": []interface{}{
			map[string]interface{}{
				"ItemKey":    "item1",
				"item_value": "item2",
			},
		},
	}

	result := converter.ConvertMap(input)

	// Check top level conversion
	assert.Contains(t, result, "topLevel")
	topLevel := result["topLevel"].(map[string]interface{})

	// Check nested conversions
	assert.Contains(t, topLevel, "nestedKey", "NestedKey should be converted to nestedKey")
	assert.Contains(t, topLevel, "anotherKey", "another_key should be converted to anotherKey")
	assert.Contains(t, topLevel, "java.property.key", "Java property should be preserved")
	assert.Contains(t, topLevel, "AWS_CONFIG", "AWS_CONFIG should be preserved")
	assert.Contains(t, topLevel, "alreadyCamelCase", "Already camelCase should be preserved")

	// Check list handling
	assert.Contains(t, result, "listExample")
	listExample := result["listExample"].([]interface{})
	assert.Len(t, listExample, 1)

	listItem := listExample[0].(map[string]interface{})
	assert.Contains(t, listItem, "itemKey", "ItemKey should be converted to itemKey")
	assert.Contains(t, listItem, "itemValue", "item_value should be converted to itemValue")
}

func TestEdgeCases(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single character", "a", "a"},
		{"Single uppercase", "A", "a"},
		{"Numbers in key", "key123", "key123"},
		{"UUID as key", "3f4beddd-2061-49b0-ae80-6f1f2ed65b37", "3f4beddd-2061-49b0-ae80-6f1f2ed65b37"},
		{"Mixed separators", "some-key_name", "someKeyName"},
		{"Multiple underscores", "key__with___many____underscores", "keyWithManyUnderscores"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRealWorldExample(t *testing.T) {
	// This test uses the exact YAML from the requirements
	converter := NewConverter()

	inputYAML := strings.TrimSpace(yamlData)

	output, err := converter.ConvertYAML([]byte(inputYAML))
	require.NoError(t, err)

	// Parse and verify
	var result map[string]interface{}
	err = yaml.Unmarshal(output, &result)
	require.NoError(t, err)

	// Verify all top-level keys are converted properly
	expectedTopLevel := map[string]bool{
		"autoscaling": true,
		"configMap":   true,
		"deployment":  true,
		"env":         true,
		"image":       true,
		"ingress":     true,
		"resources":   true,
	}

	for key, _ := range expectedTopLevel {
		assert.Contains(t, result, key, "Top-level key %s should exist", key)
	}

	// Verify Java properties are preserved in ConfigMap
	configMap := result["configMap"].(map[string]interface{})
	rootProps := configMap["root.properties"].(map[string]interface{})

	// All Java properties should be preserved
	javaProperties := []string{
		"auth.dataSource.password",
		"auth.dataSource.serverName",
		"auth.dataSource.user",
		"com.viafoura.heimdall.access.refresh.local_client_uuid",
		"com.viafoura.heimdall.jwt.secret_1",
		"com.viafoura.heimdall.jwt.secret_2",
		"dataSource.password",
		"dataSource.user",
	}

	for _, prop := range javaProperties {
		assert.Contains(t, rootProps, prop, "Java property %s should be preserved", prop)
	}

	// Verify JAVA_OPTIONS is preserved
	env := result["env"].(map[string]interface{})
	assert.Contains(t, env, "JAVA_OPTIONS", "JAVA_OPTIONS should be preserved")
}
