package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLayer_Clone(t *testing.T) {
	tests := []struct {
		name     string
		layer    *ConfigLayer
		expected *ConfigLayer
	}{
		{
			name:  "nil layer",
			layer: nil,
			expected: &ConfigLayer{
				Values: make(map[string]interface{}),
			},
		},
		{
			name: "simple layer",
			layer: &ConfigLayer{
				Name: "test",
				Values: map[string]interface{}{
					"key1": "value1",
					"key2": 42,
				},
			},
			expected: &ConfigLayer{
				Name: "test",
				Values: map[string]interface{}{
					"key1": "value1",
					"key2": 42,
				},
			},
		},
		{
			name: "nested layer",
			layer: &ConfigLayer{
				Name: "nested",
				Values: map[string]interface{}{
					"level1": map[string]interface{}{
						"level2": map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
			expected: &ConfigLayer{
				Name: "nested",
				Values: map[string]interface{}{
					"level1": map[string]interface{}{
						"level2": map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := tt.layer.Clone()
			assert.Equal(t, tt.expected, cloned)
			
			// Verify deep copy by modifying original
			if tt.layer != nil && tt.layer.Values != nil {
				if nested, ok := tt.layer.Values["level1"].(map[string]interface{}); ok {
					nested["newKey"] = "newValue"
					// Cloned should not have the new key
					clonedNested := cloned.Values["level1"].(map[string]interface{})
					assert.NotContains(t, clonedNested, "newKey")
				}
			}
		})
	}
}

func TestConfigLayer_Merge(t *testing.T) {
	tests := []struct {
		name     string
		base     *ConfigLayer
		override *ConfigLayer
		expected map[string]interface{}
	}{
		{
			name: "merge into empty",
			base: &ConfigLayer{
				Name:   "base",
				Values: nil,
			},
			override: &ConfigLayer{
				Name: "override",
				Values: map[string]interface{}{
					"key": "value",
				},
			},
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "merge with override",
			base: &ConfigLayer{
				Name: "base",
				Values: map[string]interface{}{
					"key1": "value1",
					"key2": "old",
				},
			},
			override: &ConfigLayer{
				Name: "override",
				Values: map[string]interface{}{
					"key2": "new",
					"key3": "value3",
				},
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "new",
				"key3": "value3",
			},
		},
		{
			name: "deep merge",
			base: &ConfigLayer{
				Name: "base",
				Values: map[string]interface{}{
					"nested": map[string]interface{}{
						"key1": "value1",
						"key2": "old",
					},
				},
			},
			override: &ConfigLayer{
				Name: "override",
				Values: map[string]interface{}{
					"nested": map[string]interface{}{
						"key2": "new",
						"key3": "value3",
					},
				},
			},
			expected: map[string]interface{}{
				"nested": map[string]interface{}{
					"key1": "value1",
					"key2": "new",
					"key3": "value3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.override)
			assert.Equal(t, tt.expected, tt.base.Values)
		})
	}
}

func TestHierarchicalConfig_GetEffectiveConfig(t *testing.T) {
	// Use the controlled test hierarchy
	hc := CreateTestHierarchy()

	// Test effective config
	tests := []struct {
		name      string
		cluster   string
		service   string
		validate  func(t *testing.T, cfg *ConfigLayer)
	}{
		{
			name:    "cluster and service override",
			cluster: "prod01",
			service: "heimdall",
			validate: func(t *testing.T, cfg *ConfigLayer) {
				require.NotNil(t, cfg)
				require.NotNil(t, cfg.Values)
				
				// Check globals
				globals := cfg.Values["globals"].(map[string]interface{})
				converter := globals["converter"].(map[string]interface{})
				assert.Equal(t, 5, converter["minUppercaseChars"])
				assert.True(t, converter["skipJavaProperties"].(bool))
				
				performance := globals["performance"].(map[string]interface{})
				assert.Equal(t, 20, performance["maxConcurrentServices"])
				
				assert.Equal(t, "prod01", cfg.Values["cluster"])
				
				// Check services
				services := cfg.Values["services"].(map[string]interface{})
				heimdall := services["heimdall"].(map[string]interface{})
				assert.True(t, heimdall["enabled"].(bool))
				assert.Equal(t, "Heimdall", heimdall["capitalized"])
			},
		},
		{
			name:    "dev cluster without override",
			cluster: "dev01",
			service: "auth",
			validate: func(t *testing.T, cfg *ConfigLayer) {
				require.NotNil(t, cfg)
				require.NotNil(t, cfg.Values)
				
				// Should use global values since dev01 doesn't override performance
				globals := cfg.Values["globals"].(map[string]interface{})
				converter := globals["converter"].(map[string]interface{})
				assert.Equal(t, 5, converter["minUppercaseChars"])
				assert.True(t, converter["skipJavaProperties"].(bool))
				
				performance := globals["performance"].(map[string]interface{})
				assert.Equal(t, 10, performance["maxConcurrentServices"])
				
				// Check cluster values
				assert.Equal(t, "dev01", cfg.Values["cluster"])
				assert.Equal(t, "kops-dev", cfg.Values["source"])
				assert.Equal(t, "eks-dev", cfg.Values["target"])
			},
		},
		{
			name:    "service-specific overrides",
			cluster: "prod01",
			service: "heimdall",
			validate: func(t *testing.T, cfg *ConfigLayer) {
				require.NotNil(t, cfg)
				require.NotNil(t, cfg.Values)
				
				// Check service values
				services := cfg.Values["services"].(map[string]interface{})
				heimdall := services["heimdall"].(map[string]interface{})
				assert.True(t, heimdall["enabled"].(bool))
				assert.Equal(t, "Heimdall", heimdall["capitalized"])
				
				// Service may have its own converter override
				if heimdallConverter, ok := heimdall["converter"].(map[string]interface{}); ok {
					if minChars, ok := heimdallConverter["minUppercaseChars"].(int); ok {
						assert.Equal(t, 10, minChars)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := hc.GetEffectiveConfig(tt.cluster, "", "", tt.service)
			tt.validate(t, cfg)
		})
	}
}

func TestDeepCopyMap(t *testing.T) {
	original := map[string]interface{}{
		"key1": "value1",
		"nested": map[string]interface{}{
			"key2": "value2",
			"deeper": map[string]interface{}{
				"key3": "value3",
			},
		},
		"array": []interface{}{"item1", "item2"},
	}

	copied := deepCopyMap(original)

	// Verify structure is the same
	assert.Equal(t, original, copied)

	// Modify original and verify copy is unchanged
	original["key1"] = "modified"
	assert.Equal(t, "value1", copied["key1"])

	nested := original["nested"].(map[string]interface{})
	nested["key2"] = "modified"
	copiedNested := copied["nested"].(map[string]interface{})
	assert.Equal(t, "value2", copiedNested["key2"])

	arr := original["array"].([]interface{})
	arr[0] = "modified"
	copiedArr := copied["array"].([]interface{})
	assert.Equal(t, "item1", copiedArr[0])
}