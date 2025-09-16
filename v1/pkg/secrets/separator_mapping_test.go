package secrets

import (
	"testing"

	"github.com/elioetibr/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm-charts-migrator/v1/pkg/config"
)

func TestSeparatorWithCustomKeyMapping(t *testing.T) {
	// Create config with custom key mapping
	cfg := &config.Config{
		Services: map[string]config.Service{
			"livecomments": {
				Enabled: true,
				Name:    "livecomments",
				Secrets: &config.Secrets{
					Merging: map[string]*config.MergeStrategy{
						"apps/{service}/values.yaml": {
							KeyMappings: map[string]string{
								"configMap.application.properties": "secrets.application.conf",
							},
						},
					},
				},
			},
		},
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Locations: &config.SecretLocations{
					BasePath:  "configMap",
					StorePath: "secrets",
					ScanMode:  config.ScanModeFiltered,
				},
				Patterns: []string{
					".*\\.password.*",
					".*\\.secret.*",
				},
			},
		},
	}

	// Create extractor and separator
	extractor, err := New(cfg)
	require.NoError(t, err)

	separator := NewSeparator(extractor)

	// Set target file to trigger merge strategy lookup
	separator.SetTargetFile("apps/livecomments/values.yaml")

	// Test YAML with secrets in configMap.application.properties
	testYAML := `
configMap:
  application.properties:
    database.url: "jdbc:mysql://localhost/db"
    database.user: "admin"
    database.password: "secret123"
    jwt.secret: "my-jwt-secret"
  root.properties:
    api.key: "not-secret"
image:
  tag: "v1.0.0"
service:
  port: 8080
`

	// Separate secrets
	result, separationResult, err := separator.SeparateSecrets([]byte(testYAML), "livecomments")
	require.NoError(t, err)
	assert.NotNil(t, separationResult)

	// Parse result
	var resultData map[string]interface{}
	err = yaml.Unmarshal(result, &resultData)
	require.NoError(t, err)

	// Check that secrets were moved to the custom mapped location
	secrets, ok := resultData["secrets"].(map[string]interface{})
	require.True(t, ok, "secrets section should exist")

	// Check that application.properties was mapped to application.conf
	appConf, ok := secrets["application.conf"].(map[string]interface{})
	require.True(t, ok, "secrets.application.conf should exist due to custom mapping")

	// Verify the secrets were moved there
	assert.Equal(t, "secret123", appConf["database.password"])
	assert.Equal(t, "my-jwt-secret", appConf["jwt.secret"])

	// Verify non-secrets remain in configMap
	configMap, ok := resultData["configMap"].(map[string]interface{})
	require.True(t, ok, "configMap section should still exist")

	appProps, ok := configMap["application.properties"].(map[string]interface{})
	require.True(t, ok, "configMap.application.properties should still exist")

	// Non-secret values should remain
	assert.Equal(t, "jdbc:mysql://localhost/db", appProps["database.url"])
	assert.Equal(t, "admin", appProps["database.user"])

	// Secrets should be removed from original location
	assert.Nil(t, appProps["database.password"])
	assert.Nil(t, appProps["jwt.secret"])

	// Check separation report
	assert.Equal(t, 2, separationResult.MovedCount, "Should have moved 2 secrets")
	assert.Len(t, separationResult.ExtractedSecrets, 2)

	// Verify the paths in the report
	for _, secret := range separationResult.ExtractedSecrets {
		// Original path includes the full key path
		assert.Contains(t, secret.OriginalPath, "configMap.application.properties")
		// New path should be under application.conf
		assert.Contains(t, secret.NewPath, "secrets.application.conf")
		assert.Contains(t, []string{"database.password", "jwt.secret"}, secret.Key)
	}
}

func TestSeparatorWithoutCustomMapping(t *testing.T) {
	// Create config WITHOUT custom key mapping
	cfg := &config.Config{
		Services: map[string]config.Service{
			"heimdall": {
				Enabled: true,
				Name:    "heimdall",
				Secrets: &config.Secrets{},
			},
		},
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Locations: &config.SecretLocations{
					BasePath:  "configMap",
					StorePath: "secrets",
					ScanMode:  config.ScanModeFiltered,
				},
				Patterns: []string{
					".*\\.password.*",
					".*\\.secret.*",
				},
			},
		},
	}

	// Create extractor and separator
	extractor, err := New(cfg)
	require.NoError(t, err)

	separator := NewSeparator(extractor)

	// Test same YAML
	testYAML := `
configMap:
  application.properties:
    database.url: "jdbc:mysql://localhost/db"
    database.user: "admin"
    database.password: "secret123"
    jwt.secret: "my-jwt-secret"
  root.properties:
    api.key: "not-secret"
image:
  tag: "v1.0.0"
`

	// Separate secrets
	result, separationResult, err := separator.SeparateSecrets([]byte(testYAML), "heimdall")
	require.NoError(t, err)
	assert.NotNil(t, separationResult)

	// Parse result
	var resultData map[string]interface{}
	err = yaml.Unmarshal(result, &resultData)
	require.NoError(t, err)

	// Check that secrets were moved to default location (not custom mapped)
	secrets, ok := resultData["secrets"].(map[string]interface{})
	require.True(t, ok, "secrets section should exist")

	// Without custom mapping, it should keep the original structure
	appProps, ok := secrets["application.properties"].(map[string]interface{})
	require.True(t, ok, "secrets.application.properties should exist (no custom mapping)")

	// Verify the secrets were moved there
	assert.Equal(t, "secret123", appProps["database.password"])
	assert.Equal(t, "my-jwt-secret", appProps["jwt.secret"])

	// Check separation report
	assert.Equal(t, 2, separationResult.MovedCount, "Should have moved 2 secrets")

	// Verify the paths in the report (without custom mapping)
	for _, secret := range separationResult.ExtractedSecrets {
		// Original path includes the full key path
		assert.Contains(t, secret.OriginalPath, "configMap.application.properties")
		// New path should be under application.properties (default mapping)
		assert.Contains(t, secret.NewPath, "secrets.application.properties")
		assert.Contains(t, []string{"database.password", "jwt.secret"}, secret.Key)
	}
}
