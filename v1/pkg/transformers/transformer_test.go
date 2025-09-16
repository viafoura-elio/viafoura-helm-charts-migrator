package transformers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/elioetibr/yaml"

	"helm-charts-migrator/v1/pkg/config"
)

func TestLoadConfig(t *testing.T) {
	yamlData := []byte(`
globals:
  mappings:
    transform:
      enabled: true
      description: "Test transformer"
      rules:
        ingress_to_hosts:
          type: "ingress_to_hosts"
          source_path: "[Ii]ngress"
          target_path: "hosts.public.domains"
          description: "Extract hosts from ingress"
`)

	config, err := LoadConfig(yamlData)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.NotNil(t, config.Globals.Mappings)
	assert.NotNil(t, config.Globals.Mappings.Transform)
	assert.True(t, config.Globals.Mappings.Transform.Enabled)
	assert.Equal(t, "Test transformer", config.Globals.Mappings.Transform.Description)
	assert.Len(t, config.Globals.Mappings.Transform.Rules, 1)

	rule := config.Globals.Mappings.Transform.Rules["ingress_to_hosts"]
	assert.Equal(t, "ingress_to_hosts", rule.Type)
	assert.Equal(t, "[Ii]ngress", rule.SourcePath)
	assert.Equal(t, "hosts.public.domains", rule.TargetPath)
	assert.Equal(t, "Extract hosts from ingress", rule.Description)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "config cannot be nil",
		},
		{
			name:    "disabled transformer",
			config:  createTestConfig(false, map[string]config.TransformRule{}),
			wantErr: false,
		},
		{
			name: "valid config",
			config: createTestConfig(true, map[string]config.TransformRule{
				"test": {
					Type:       "ingress_to_hosts",
					SourcePath: "[Ii]ngress",
					TargetPath: "hosts.public.domains",
				},
			}),
			wantErr: false,
		},
		{
			name: "invalid regex",
			config: createTestConfig(true, map[string]config.TransformRule{
				"test": {
					Type:       "ingress_to_hosts",
					SourcePath: "[invalid",
					TargetPath: "target",
				},
			}),
			wantErr:     true,
			errContains: "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer, err := New(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, transformer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, transformer)
			}
		})
	}
}

func TestValidateHost(t *testing.T) {
	config := createTestConfig(true, map[string]config.TransformRule{})
	transformer, err := New(config)
	require.NoError(t, err)

	tests := []struct {
		input        string
		expectValid  bool
		expectedHost string
		reason       string
	}{
		{"auth.example.com", true, "auth.example.com", ""},
		{"api.test.co", true, "api.test.co", ""},
		{"https://auth.example.com", true, "auth.example.com", ""},
		{"http://api.test.org", true, "api.test.org", ""},
		{"auth.example.com:8080", true, "auth.example.com", ""},
		{"https://auth.example.com:443/path", true, "auth.example.com", ""},
		{"", false, "", "empty host"},
		{"   ", false, "   ", "empty host"},
		{"localhost", false, "localhost", "invalid hostname format"},
		{"invalid..host", false, "invalid..host", "invalid hostname format"},
		{"host_with_underscore.com", false, "host_with_underscore.com", "invalid hostname format"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := transformer.validateHost(tt.input)
			assert.Equal(t, tt.expectValid, result.IsValid, "Validation result mismatch")
			if tt.expectValid {
				assert.Equal(t, tt.expectedHost, result.Host, "Host mismatch")
			} else {
				assert.Contains(t, result.Reason, tt.reason, "Reason mismatch")
			}
		})
	}
}

func TestTransformYAML(t *testing.T) {
	config := createTestConfig(true, map[string]config.TransformRule{
		"ingress_to_hosts": {
			Type:       "ingress_to_hosts",
			SourcePath: "[Ii]ngress",
			TargetPath: "hosts.public.domains",
		},
	})

	transformer, err := New(config)
	require.NoError(t, err)

	tests := []struct {
		name           string
		input          string
		expectedHosts  []string
		expectedPaths  []string
		expectWarnings bool
	}{
		{
			name: "simple ingress with infoHost and orgHost",
			input: `
app:
  name: test
ingress:
  infoHost: auth.example.info
  orgHost: auth.example.org
  port: 80
other:
  setting: value
`,
			expectedHosts: []string{"auth.example.info", "auth.example.org"},
			expectedPaths: []string{"ingress"},
		},
		{
			name: "ingress with host field",
			input: `
ingress:
  host: api.test.com
  class: nginx
`,
			expectedHosts: []string{"api.test.com"},
			expectedPaths: []string{"ingress"},
		},
		{
			name: "ingress with hosts array",
			input: `
ingress:
  hosts:
    - api.prod.com
    - api.staging.com
`,
			expectedHosts: []string{"api.prod.com", "api.staging.com"},
			expectedPaths: []string{"ingress"},
		},
		{
			name: "ingress with URL cleanup",
			input: `
ingress:
  infoHost: https://auth.example.com:443/path
  orgHost: http://admin.example.org:8080
`,
			expectedHosts: []string{"auth.example.com", "admin.example.org"},
			expectedPaths: []string{"ingress"},
		},
		{
			name: "multiple ingress sections",
			input: `
publicIngress:
  host: public.example.com
privateIngress:
  infoHost: private.example.org
`,
			expectedHosts: []string{"public.example.com", "private.example.org"},
			expectedPaths: []string{"publicIngress", "privateIngress"},
		},
		{
			name: "no ingress sections",
			input: `
app:
  name: test
service:
  port: 8080
`,
			expectedHosts: []string{},
			expectedPaths: []string{},
		},
		{
			name: "ingress with invalid hosts",
			input: `
ingress:
  host: invalid..host
  infoHost: valid.example.com
  orgHost: ""
`,
			expectedHosts:  []string{"valid.example.com"},
			expectedPaths:  []string{"ingress"},
			expectWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, transformResult, err := transformer.TransformYAML([]byte(tt.input))
			require.NoError(t, err)
			require.NotNil(t, transformResult)

			// Parse the result to check the structure
			var resultData map[string]interface{}
			err = yaml.Unmarshal(result, &resultData)
			require.NoError(t, err)

			// Check extracted hosts
			if len(tt.expectedHosts) > 0 {
				hosts, exists := getNestedValue(resultData, "hosts.public.domains")
				assert.True(t, exists, "hosts.public.domains should exist")
				if exists {
					hostSlice, ok := hosts.([]interface{})
					require.True(t, ok, "hosts.public.domains should be a slice")

					actualHosts := make([]string, len(hostSlice))
					for i, h := range hostSlice {
						actualHosts[i] = h.(string)
					}

					// Check that all expected hosts are present (order may vary)
					for _, expectedHost := range tt.expectedHosts {
						assert.Contains(t, actualHosts, expectedHost)
					}
					assert.Len(t, actualHosts, len(tt.expectedHosts))
				}
			} else {
				_, exists := getNestedValue(resultData, "hosts.public.domains")
				assert.False(t, exists, "hosts.public.domains should not exist")
			}

			// Check transformation result
			assert.Len(t, transformResult.ModifiedPaths, len(tt.expectedPaths))
			for _, expectedPath := range tt.expectedPaths {
				assert.Contains(t, transformResult.ModifiedPaths, expectedPath)
			}

			if tt.expectWarnings {
				assert.NotEmpty(t, transformResult.Warnings, "Expected warnings but got none")
			}

			// Verify transformed paths are removed from original structure
			for _, path := range tt.expectedPaths {
				_, exists := getNestedValue(resultData, path)
				assert.False(t, exists, "Original ingress path %s should be removed", path)
			}
		})
	}
}

func TestTransformYAML_DisabledTransformer(t *testing.T) {
	config := createTestConfig(false, map[string]config.TransformRule{})
	transformer, err := New(config)
	require.NoError(t, err)

	input := []byte(`
ingress:
  host: test.example.com
`)

	result, transformResult, err := transformer.TransformYAML(input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
	assert.Empty(t, transformResult.ExtractedHosts)
	assert.Empty(t, transformResult.ModifiedPaths)
	assert.Empty(t, transformResult.Warnings)
}

func TestBuildPath(t *testing.T) {
	tests := []struct {
		parent   string
		key      string
		expected string
	}{
		{"", "key", "key"},
		{"parent", "key", "parent.key"},
		{"parent.child", "key", "parent.child.key"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := buildPath(tt.parent, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidHostname(t *testing.T) {
	tests := []struct {
		hostname string
		expected bool
	}{
		{"example.com", true},
		{"auth.example.com", true},
		{"api-v2.example.org", true},
		{"test123.example.net", true},
		{"a.b", true},
		{"", false},
		{"example", false},
		{".example.com", false},
		{"example.com.", false},
		{"example..com", false},
		{"example_com", false},
		{"-example.com", false},
		{"example.com-", false},
		{"very-very-very-very-very-very-very-very-long-hostname-that-exceeds-limits.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			result := isValidHostname(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformerMethods(t *testing.T) {
	rules := map[string]config.TransformRule{
		"test": {
			Type:       "ingress_to_hosts",
			SourcePath: "[Ii]ngress",
			TargetPath: "hosts.public.domains",
		},
	}
	config := createTestConfig(true, rules)
	config.Globals.Mappings.Transform.Description = "Test Description"

	transformer, err := New(config)
	require.NoError(t, err)

	assert.True(t, transformer.IsEnabled())
	assert.Equal(t, "Test Description", transformer.GetDescription())
	assert.Equal(t, rules, transformer.GetRules())
}

// Helper functions for tests
func createTestConfig(enabled bool, rules map[string]config.TransformRule) *config.Config {
	cfg := &config.Config{
		Globals: config.Globals{
			Mappings: &config.Mappings{
				Transform: &config.Transform{
					Enabled: enabled,
					Rules:   rules,
				},
			},
		},
	}
	return cfg
}

func getNestedValue(data map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			value, exists := current[part]
			return value, exists
		} else {
			next, exists := current[part]
			if !exists {
				return nil, false
			}
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil, false
			}
		}
	}
	return nil, false
}
