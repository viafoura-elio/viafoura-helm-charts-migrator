package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"helm-charts-migrator/v1/pkg/config"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

func TestSeparateSecrets(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Services: map[string]config.Service{
			"heimdall": {
				Enabled: true,
				Name:    "heimdall",
				Secrets: &config.Secrets{
					Keys: []string{
						"3f4beddd-2061-49b0-ae80-6f1f2ed65b37",
						"936da557-6daa-4444-92cc-161fc290c603",
						"c23203d0-1b8e-4208-92dc-85dc79e6226b",
					},
					Patterns: []string{
						".*jwt.*secret.*",
						".*provider.*secret.*",
						".*apikey.*",
					},
				},
			},
		},
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Patterns: []string{
					"^.*\\.password$", // Matches fields ending with .password (e.g., database.password)
					".*secret.*",
					".*token.*",
				},
			},
		},
	}

	// Create extractor and separator
	extractor, err := New(cfg)
	require.NoError(t, err)

	separator := NewSeparator(extractor)

	// Test YAML with secrets mixed in configMap
	testYAML := `
configMap:
  root.properties:
    3f4beddd-2061-49b0-ae80-6f1f2ed65b37: "secret-value-1"
    936da557-6daa-4444-92cc-161fc290c603: "secret-value-2"
    c23203d0-1b8e-4208-92dc-85dc79e6226b: "secret-value-3"
    com.viafoura.heimdall.jwt.secret_1: "this is my secret"
    com.viafoura.heimdall.jwt.secret_2: "this is my other secret"
    com.viafoura.heimdall.provider.loginradius.secret: "provider-secret"
    com.viafoura.heimdall.thirdparty.apikey.loginradius: "api-key-value"
    com.viafoura.heimdall.cookie.access.domain: "vf-dev2.org"
    com.viafoura.heimdall.url.resetpassword: "//admin.vf-dev2.org/admin"
  application.properties:
    database.password: "db-secret"
    app.config.value: "not-a-secret"
image:
  tag: "PR-448-1"
secrets: {}
`

	// Separate secrets
	result, separationResult, err := separator.SeparateSecrets([]byte(testYAML), "heimdall")
	require.NoError(t, err)
	assert.NotNil(t, separationResult)

	// Parse result
	var resultData map[string]interface{}
	err = yaml.UnmarshalStrict(result, &resultData)
	require.NoError(t, err)

	// Check that secrets were moved
	assert.Greater(t, separationResult.MovedCount, 0)

	// Check that secrets section now contains the extracted secrets
	secrets, ok := resultData["secrets"].(map[string]interface{})
	assert.True(t, ok, "secrets section should exist")

	// Check that root.properties exists in secrets
	rootProps, ok := secrets["root.properties"].(map[string]interface{})
	assert.True(t, ok, "secrets.root.properties should exist")

	// Check specific secrets were moved
	assert.Contains(t, rootProps, "3f4beddd-2061-49b0-ae80-6f1f2ed65b37")
	assert.Contains(t, rootProps, "com.viafoura.heimdall.jwt.secret_1")
	assert.Contains(t, rootProps, "com.viafoura.heimdall.provider.loginradius.secret")

	// Check that application.properties password was moved
	if appProps, ok := secrets["application.properties"].(map[string]interface{}); ok {
		assert.Contains(t, appProps, "database.password")
	}

	// Check that non-secrets remain in configMap
	configMap, ok := resultData["configMap"].(map[string]interface{})
	assert.True(t, ok, "configMap should still exist")

	if cmRootProps, ok := configMap["root.properties"].(map[string]interface{}); ok {
		// Non-secrets should remain
		assert.Contains(t, cmRootProps, "com.viafoura.heimdall.cookie.access.domain")
		assert.Contains(t, cmRootProps, "com.viafoura.heimdall.url.resetpassword")

		// Secrets should be removed
		assert.NotContains(t, cmRootProps, "3f4beddd-2061-49b0-ae80-6f1f2ed65b37")
		assert.NotContains(t, cmRootProps, "com.viafoura.heimdall.jwt.secret_1")
	}
}

func TestSeparateSecretsWithExistingSecretsSection(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Patterns: []string{
					"^.*\\.password$", // Matches fields ending with .password
					".*secret.*",
				},
			},
		},
	}

	extractor, err := New(cfg)
	require.NoError(t, err)

	separator := NewSeparator(extractor)

	// Test YAML with existing secrets section
	testYAML := `
configMap:
  root.properties:
    app.secret.key: "secret-value"
    app.config.value: "not-a-secret"
secrets:
  root.properties:
    existing.secret: "already-here"
`

	// Separate secrets
	result, separationResult, err := separator.SeparateSecrets([]byte(testYAML), "test-service")
	require.NoError(t, err)
	assert.NotNil(t, separationResult)

	// Parse result
	var resultData map[string]interface{}
	err = yaml.UnmarshalStrict(result, &resultData)
	require.NoError(t, err)

	// Check that secrets section contains both old and new secrets
	secrets := resultData["secrets"].(map[string]interface{})
	rootProps := secrets["root.properties"].(map[string]interface{})

	// Should contain both existing and new secrets
	assert.Contains(t, rootProps, "existing.secret")
	assert.Contains(t, rootProps, "app.secret.key")

	// Check that secret was removed from configMap
	configMap := resultData["configMap"].(map[string]interface{})
	if cmRootProps, ok := configMap["root.properties"].(map[string]interface{}); ok {
		assert.NotContains(t, cmRootProps, "app.secret.key")
		assert.Contains(t, cmRootProps, "app.config.value")
	}
}
