package normalizers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "github.com/elioetibr/golang-yaml-advanced"

	"helm-charts-migrator/v1/pkg/config"
)

func TestLoadConfig(t *testing.T) {
	yamlData := []byte(`
globals:
  mappings:
    normalizer:
      enabled: true
      description: "Test normalizer"
      patterns:
        '[Cc]ontainer.[Pp]ort': 'service.targetPort'
        '[Rr]eplicas': 'replicaCount'
`)

	config, err := LoadConfig(yamlData)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.NotNil(t, config.Globals.Mappings)
	assert.NotNil(t, config.Globals.Mappings.Normalizer)
	assert.True(t, config.Globals.Mappings.Normalizer.Enabled)
	assert.Equal(t, "Test normalizer", config.Globals.Mappings.Normalizer.Description)
	assert.Len(t, config.Globals.Mappings.Normalizer.Patterns, 2)
	assert.Equal(t, "service.targetPort", config.Globals.Mappings.Normalizer.Patterns["[Cc]ontainer.[Pp]ort"])
	assert.Equal(t, "replicaCount", config.Globals.Mappings.Normalizer.Patterns["[Rr]eplicas"])
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
			name: "disabled normalizer",
			config: &config.Config{
				Globals: config.Globals{
					Mappings: &config.Mappings{
						Normalizer: &config.Normalizer{
							Enabled: false,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config",
			config: &config.Config{
				Globals: config.Globals{
					Mappings: &config.Mappings{
						Normalizer: &config.Normalizer{
							Enabled: true,
							Patterns: map[string]string{
								"[Cc]ontainer.[Pp]ort": "service.targetPort",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid regex",
			config: &config.Config{
				Globals: config.Globals{
					Mappings: &config.Mappings{
						Normalizer: &config.Normalizer{
							Enabled: true,
							Patterns: map[string]string{
								"[invalid": "target",
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizer, err := New(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, normalizer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, normalizer)
			}
		})
	}
}

func TestNormalizeYAML(t *testing.T) {
	config := &config.Config{
		Globals: config.Globals{
			Mappings: &config.Mappings{
				Normalizer: &config.Normalizer{},
			},
		},
	}
	config.Globals.Mappings.Normalizer.Enabled = true
	config.Globals.Mappings.Normalizer.Patterns = map[string]string{
		"[Cc]ontainer.[Pp]ort":            "service.targetPort",
		"[Aa]utoscaling.[Tt]arget.[Cc]pu": "autoscaling.targetCPUUtilizationPercentage",
		"[Rr]eplicas":                     "replicaCount",
	}

	normalizer, err := New(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple container port transformation",
			input: `
container:
  port: 8080
`,
			expected: `service:
    targetPort: 8080
`,
		},
		{
			name: "replicas transformation",
			input: `
replicas: 3
`,
			expected: `replicaCount: 3
`,
		},
		{
			name: "multiple transformations",
			input: `
container:
  port: 8080
replicas: 3
autoscaling:
  target:
    cpu: 80
`,
			expected: `service:
    targetPort: 8080
replicaCount: 3
autoscaling:
    targetCPUUtilizationPercentage: 80
`,
		},
		{
			name: "nested structure with no transformations",
			input: `
app:
  name: test
  version: 1
`,
			expected: `app:
    name: test
    version: 1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizer.NormalizeYAML([]byte(tt.input))
			require.NoError(t, err)

			// Compare the normalized structure by unmarshaling both
			var expected, actual interface{}
			require.NoError(t, yaml.Unmarshal([]byte(tt.expected), &expected))
			require.NoError(t, yaml.Unmarshal(result, &actual))

			assert.Equal(t, expected, actual)
		})
	}
}

func TestNormalizeYAML_DisabledNormalizer(t *testing.T) {
	config := &config.Config{
		Globals: config.Globals{
			Mappings: &config.Mappings{
				Normalizer: &config.Normalizer{},
			},
		},
	}
	config.Globals.Mappings.Normalizer.Enabled = false

	normalizer, err := New(config)
	require.NoError(t, err)

	input := []byte(`
container:
  port: 8080
`)

	result, err := normalizer.NormalizeYAML(input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
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

func TestExtractKeyFromPath(t *testing.T) {
	tests := []struct {
		fullPath    string
		currentPath string
		expected    string
	}{
		{"service.targetPort", "", "service.targetPort"},
		{"service.targetPort", "service", "targetPort"},
		{"autoscaling.targetCPUUtilizationPercentage", "autoscaling", "targetCPUUtilizationPercentage"},
		{"replicaCount", "", "replicaCount"},
		{"deep.nested.path.key", "deep.nested", "path.key"},
		{"unrelated.path", "different.path", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.fullPath+"_"+tt.currentPath, func(t *testing.T) {
			result := extractKeyFromPath(tt.fullPath, tt.currentPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizerMethods(t *testing.T) {
	config := &config.Config{
		Globals: config.Globals{
			Mappings: &config.Mappings{
				Normalizer: &config.Normalizer{},
			},
		},
	}
	config.Globals.Mappings.Normalizer.Enabled = true
	config.Globals.Mappings.Normalizer.Description = "Test Description"
	config.Globals.Mappings.Normalizer.Patterns = map[string]string{
		"pattern1": "target1",
		"pattern2": "target2",
	}

	normalizer, err := New(config)
	require.NoError(t, err)

	assert.True(t, normalizer.IsEnabled())
	assert.Equal(t, "Test Description", normalizer.GetDescription())
	assert.Equal(t, config.Globals.Mappings.Normalizer.Patterns, normalizer.GetPatterns())
}
