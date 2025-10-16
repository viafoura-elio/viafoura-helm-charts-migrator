package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"helm-charts-migrator/v1/pkg/config"
)

func TestLoadConfig(t *testing.T) {
	yamlData := []byte(`
globals:
  secrets:
    patterns:
      - ".*\\.password.*"
      - ".*\\.secret.*"
    uuids:
      - pattern: ".*client.*uuid.*"
        sensitive: true
        description: "Client UUIDs"
    values:
      - pattern: "^[A-Za-z0-9+/]{40,}={0,2}$"
        sensitive: true
        description: "Base64 encoded values"
services:
  test-service:
    enabled: true
    name: test-service
    secrets:
      keys:
        - "auth.client_secret"
        - "database.password"
      patterns:
        - ".*client_secret.*"
      description: "Test service secrets"
`)

	cfg, err := LoadConfig(yamlData)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check global patterns
	assert.Len(t, cfg.Globals.Secrets.Patterns, 2)
	assert.Contains(t, cfg.Globals.Secrets.Patterns, ".*\\.password.*")

	// Check UUID patterns
	assert.Len(t, cfg.Globals.Secrets.UUIDs, 1)
	assert.True(t, cfg.Globals.Secrets.UUIDs[0].Sensitive)

	// Check value patterns
	assert.Len(t, cfg.Globals.Secrets.Values, 1)
	assert.True(t, cfg.Globals.Secrets.Values[0].Sensitive)

	// Check service config
	assert.True(t, cfg.Services["test-service"].Enabled)
	assert.Len(t, cfg.Services["test-service"].Secrets.Keys, 2)
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
			name:    "valid config",
			config:  createTestConfig(),
			wantErr: false,
		},
		{
			name: "invalid global pattern",
			config: &config.Config{
				Globals: config.Globals{
					Secrets: &config.Secrets{
						Patterns: []string{"[invalid"},
					},
				},
				Services: make(map[string]config.Service),
			},
			wantErr:     true,
			errContains: "invalid global pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := New(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, extractor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, extractor)
			}
		})
	}
}

func TestExtractSecrets_GlobalPatterns(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
app:
  name: test
auth:
  password: "secret123"
  api_key: "ak_1234567890abcdef"
database:
  user: testuser
  password: "db_secret"
jwt:
  secret: "jwt_super_secret"
normal:
  setting: "not_a_secret"
`)

	result, err := extractor.ExtractSecrets(yamlData, "unknown-service")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find the password and secret fields
	assert.GreaterOrEqual(t, len(result.Secrets), 3)

	// Check that passwords are detected
	var passwordSecrets []SecretMatch
	for _, secret := range result.Secrets {
		if secret.Classification == ClassificationPassword {
			passwordSecrets = append(passwordSecrets, secret)
		}
	}
	assert.GreaterOrEqual(t, len(passwordSecrets), 2)

	// Check summary
	assert.Equal(t, len(result.Secrets), result.Summary.TotalSecrets)
	assert.GreaterOrEqual(t, result.Summary.ByClassification[ClassificationPassword], 2)
}

func TestExtractSecrets_ServiceSpecific(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
auth:
  client_secret: "client_abc123"
  other_secret: "other_value"
database:
  password: "db_pass"
jwt:
  signing:
    key: "jwt_key_123"
configMap:
  root.properties:
    loginradius.secret: "lr_secret"
`)

	// Test with heimdall service (has specific config)
	result, err := extractor.ExtractSecrets(yamlData, "heimdall")
	require.NoError(t, err)

	// Should find service-specific secrets
	var serviceSpecificSecrets []SecretMatch
	for _, secret := range result.Secrets {
		if secret.ServiceSpecific {
			serviceSpecificSecrets = append(serviceSpecificSecrets, secret)
		}
	}

	assert.GreaterOrEqual(t, len(serviceSpecificSecrets), 1)
	assert.GreaterOrEqual(t, result.Summary.ServiceSpecific, 1)
}

func TestExtractSecrets_UUIDDetection(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
client:
  uuid: "3f4beddd-2061-49b0-ae80-6f1f2ed65b37"
  name: "test-client"
generic:
  id: "936da557-6daa-4444-92cc-161fc290c603"
  setting: "not_uuid_related"
`)

	result, err := extractor.ExtractSecrets(yamlData, "heimdall")
	require.NoError(t, err)

	// Should detect UUIDs
	var uuidSecrets []SecretMatch
	for _, secret := range result.Secrets {
		if secret.Classification == ClassificationUUID {
			uuidSecrets = append(uuidSecrets, secret)
		}
	}

	assert.GreaterOrEqual(t, len(uuidSecrets), 1)
}

func TestExtractSecrets_ValuePatterns(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
tokens:
  base64_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
  long_base64: "VGhpcyBpcyBhIGxvbmcgYmFzZTY0IGVuY29kZWQgc3RyaW5nIHRoYXQgc2hvdWxkIGJlIGRldGVjdGVkIGFzIGEgc2VjcmV0"
  hex_key: "deadbeef123456789abcdef0123456789abcdef0123456789abcdef"
  short_value: "abc123"
normal:
  config: "regular_value"
`)

	result, err := extractor.ExtractSecrets(yamlData, "test-service")
	require.NoError(t, err)

	// Should detect value-based secrets
	var base64Secrets []SecretMatch
	var jwtSecrets []SecretMatch
	var hexSecrets []SecretMatch

	for _, secret := range result.Secrets {
		switch secret.Classification {
		case ClassificationBase64:
			base64Secrets = append(base64Secrets, secret)
		case ClassificationJWT:
			jwtSecrets = append(jwtSecrets, secret)
		case ClassificationHex:
			hexSecrets = append(hexSecrets, secret)
		}
	}

	assert.GreaterOrEqual(t, len(base64Secrets), 1)
	assert.GreaterOrEqual(t, len(jwtSecrets), 1)
	assert.GreaterOrEqual(t, len(hexSecrets), 1)
}

func TestExtractSecrets_ExactKeyMatch(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
# Exact key match from heimdall service config
configMap:
  root.properties:
    3f4beddd-2061-49b0-ae80-6f1f2ed65b37: "client_secret_value"
    682843b1-d3e0-460e-ab90-6556bc31470f: "another_secret"
    regular_config: "not_a_secret"
`)

	result, err := extractor.ExtractSecrets(yamlData, "heimdall")
	require.NoError(t, err)

	// Should find exact key matches
	var exactKeyMatches []SecretMatch
	for _, secret := range result.Secrets {
		for _, reason := range secret.MatchedBy {
			if reason.Type == MatchTypeExactKey {
				exactKeyMatches = append(exactKeyMatches, secret)
				break
			}
		}
	}

	assert.GreaterOrEqual(t, len(exactKeyMatches), 2)

	// All exact key matches should be high confidence and service-specific
	for _, match := range exactKeyMatches {
		assert.Equal(t, ConfidenceHigh, match.Confidence)
		assert.True(t, match.ServiceSpecific)
	}
}

func TestClassifySecret(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		key           string
		value         string
		expectedClass Classification
	}{
		{"auth.password", "secret123", ClassificationPassword},
		{"api_key", "ak_123456", ClassificationAPIKey},
		{"jwt.secret", "jwt_token", ClassificationJWT},
		{"oauth.token", "token_value", ClassificationToken},
		{"app.secret", "secret_value", ClassificationSecret},
		{"client.uuid", "3f4beddd-2061-49b0-ae80-6f1f2ed65b37", ClassificationUUID},
		{"encoded", "VGhpcyBpcyBhIGxvbmcgYmFzZTY0IGVuY29kZWQgc3RyaW5n", ClassificationBase64},
		{"hex_key", "deadbeef123456789abcdef0123456789abcdef0", ClassificationHex},
		{"unknown_key", "unknown_value", ClassificationUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			match := SecretMatch{
				Key:   tt.key,
				Value: tt.value,
			}

			extractor.classifySecret(&match)
			assert.Equal(t, tt.expectedClass, match.Classification)
		})
	}
}

func TestMaskValue(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected string
	}{
		{"short", "*****"},
		{"password123", "pass***d123"},
		{"very_long_secret_value_here", "very*******************here"},
		{"", ""},
		{"a", "*"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractor.maskValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetServiceNames(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	services := extractor.GetServiceNames()
	assert.Contains(t, services, "heimdall")
	assert.Contains(t, services, "livecomments")
	assert.Contains(t, services, "realtime-event-feed")
	assert.NotContains(t, services, "auth-service") // disabled
}

func TestHasServiceConfig(t *testing.T) {
	cfg := createTestConfig()
	extractor, err := New(cfg)
	require.NoError(t, err)

	assert.True(t, extractor.HasServiceConfig("heimdall"))
	assert.False(t, extractor.HasServiceConfig("livecomments")) // no secrets config
	assert.False(t, extractor.HasServiceConfig("auth-service")) // disabled
	assert.False(t, extractor.HasServiceConfig("non-existent"))
}

func TestServiceConfigurationMerging(t *testing.T) {
	cfg := createTestConfigWithServiceExtensions()
	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
service:
  name: test-service-extended
auth:
  password: "global_password_match"
  special_client_id: "c47b7e3d-8b5a-4c2d-9f1e-6a8b3d2e7f90"
secrets:
  custom_jwt: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature"
  service_hex: "deadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678"
  normal_setting: "not_a_secret"
`)

	result, err := extractor.ExtractSecrets(yamlData, "test-service-extended")
	require.NoError(t, err)

	// Should find secrets from both global and service-specific patterns
	var globalSecrets []SecretMatch
	var serviceSpecificSecrets []SecretMatch

	for _, secret := range result.Secrets {
		if secret.ServiceSpecific {
			serviceSpecificSecrets = append(serviceSpecificSecrets, secret)
		} else {
			globalSecrets = append(globalSecrets, secret)
		}
	}

	// Should have both global and service-specific matches
	assert.GreaterOrEqual(t, len(globalSecrets), 1, "Should have global pattern matches")
	assert.GreaterOrEqual(t, len(serviceSpecificSecrets), 1, "Should have service-specific pattern matches")

	// Check that service-specific patterns have higher confidence
	for _, secret := range serviceSpecificSecrets {
		assert.Equal(t, ConfidenceHigh, secret.Confidence, "Service-specific secrets should have high confidence")
	}
}

// Helper function to create test configuration
func createTestConfig() *config.Config {
	cfg := &config.Config{
		Services: make(map[string]config.Service),
		Globals: config.Globals{
			Secrets: &config.Secrets{},
		},
	}

	cfg.Globals.Secrets.Patterns = []string{
		".*\\.password.*",
		".*\\.secret.*",
		".*jwt\\.secret.*",
		".*\\.key$",
		".*\\.token.*",
		".*client_secret.*",
		".*api_key.*",
	}

	// UUID patterns
	cfg.Globals.Secrets.UUIDs = []config.UUIDPattern{
		{
			Pattern:     ".*client.*uuid.*",
			Sensitive:   true,
			Description: "Client UUIDs are sensitive",
		},
		{
			Pattern:     "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}",
			Sensitive:   true,
			Description: "Generic UUID pattern",
		},
	}

	// Value patterns
	cfg.Globals.Secrets.Values = []config.ValuePattern{
		{
			Pattern:     "^[A-Za-z0-9+/]{40,}={0,2}$",
			Sensitive:   true,
			Description: "Base64 encoded values",
		},
		{
			Pattern:     "^eyJ[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+$",
			Sensitive:   true,
			Description: "JWT tokens",
		},
		{
			Pattern:     "^[A-Fa-f0-9]{32,}$",
			Sensitive:   true,
			Description: "Hex encoded secrets",
		},
	}

	// Services
	cfg.Services = map[string]config.Service{
		"heimdall": {
			Enabled: true,
			Name:    "heimdall",
			Secrets: &config.Secrets{
				Keys: []string{
					"3f4beddd-2061-49b0-ae80-6f1f2ed65b37",
					"682843b1-d3e0-460e-ab90-6556bc31470f",
					"936da557-6daa-4444-92cc-161fc290c603",
				},
				Patterns: []string{
					".*loginradius.*secret.*",
					".*oauth.*secret.*",
					".*thirdparty.*apikey.*",
				},
				Description: "Heimdall authentication secrets",
			},
		},
		"livecomments": {
			Enabled: true,
			Name:    "livecomments",
			// No secrets config
		},
		"realtime-event-feed": {
			Enabled: true,
			Name:    "realtime-event-feed",
			// No secrets config
		},
		"auth-service": {
			Enabled: false,
			Name:    "auth-service",
			Secrets: &config.Secrets{
				Keys:        []string{"auth.client_secret"},
				Patterns:    []string{".*client_secret.*"},
				Description: "Auth service secrets",
			},
		},
	}

	return cfg
}

// Helper function to create test configuration with service-specific extensions
func createTestConfigWithServiceExtensions() *config.Config {
	cfg := createTestConfig()

	// Add a test service with extended configuration
	cfg.Services["test-service-extended"] = config.Service{
		Enabled: true,
		Name:    "test-service-extended",
		Secrets: &config.Secrets{
			Keys: []string{
				"special_client_id",
			},
			Patterns: []string{
				".*extended.*secret.*",
			},
			// Service-specific UUID patterns that extend global ones
			UUIDs: []config.UUIDPattern{
				{
					Pattern:     ".*special_client_id.*",
					Sensitive:   true,
					Description: "Special client IDs are highly sensitive",
				},
			},
			// Service-specific value patterns that extend global ones
			Values: []config.ValuePattern{
				{
					Pattern:     "^eyJ[A-Za-z0-9-_]+\\.eyJ[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+$",
					Sensitive:   true,
					Description: "Custom JWT format for this service",
				},
				{
					Pattern:     "^[A-Fa-f0-9]{64,}$",
					Sensitive:   true,
					Description: "Extended hex secrets (64+ chars)",
				},
			},
			Description: "Extended test service with merged configuration",
		},
	}

	return cfg
}

func TestTargetedSecretLocations(t *testing.T) {
	// Test targeted scan mode - only scan specified locations
	cfg := &config.Config{
		Services: make(map[string]config.Service),
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Patterns: []string{".*\\.password.*", ".*\\.secret.*", ".*\\.api_key.*"},
				Locations: &config.SecretLocations{
					ScanMode:        config.ScanModeTargeted,
					AdditionalPaths: []string{"auth.password"},
					PathPatterns:    []string{"database\\..*"},
				},
			},
		},
	}

	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
auth:
  password: "should_be_found"
  username: "should_not_be_scanned"
secrets:
  api_key: "should_be_found"
  other_secret: "should_not_be_scanned"
database:
  password: "should_be_found_by_pattern"
  host: "should_not_be_scanned"
config:
  password: "should_not_be_found"
`)

	result, err := extractor.ExtractSecrets(yamlData, "test")
	require.NoError(t, err)

	// Should find only the targeted locations
	expectedPaths := []string{
		"auth.password",
		"secrets.api_key",
		"database.password",
	}

	assert.Len(t, result.Secrets, 3)
	for i, secret := range result.Secrets {
		assert.Contains(t, expectedPaths, secret.Path, "Secret %d path should be in expected paths", i)
	}
}

func TestFilteredSecretLocations(t *testing.T) {
	// Test filtered scan mode - scan all but respect include/exclude
	cfg := &config.Config{
		Services: make(map[string]config.Service),
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Patterns: []string{".*\\.password.*", ".*\\.secret.*", ".*\\.api_secret.*"},
				Locations: &config.SecretLocations{
					ScanMode: config.ScanModeFiltered,
					Include:  []string{"auth.*", "secrets.*"},
					Exclude:  []string{"secrets.excluded"},
				},
			},
		},
	}

	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
auth:
  password: "should_be_found"
  secret: "should_be_found"
secrets:
  api_secret: "should_be_found"
  excluded: "should_be_excluded"
config:
  password: "should_be_excluded_by_include"
database:
  secret: "should_be_excluded_by_include"
`)

	result, err := extractor.ExtractSecrets(yamlData, "test")
	require.NoError(t, err)

	// Should find only included paths minus excluded ones
	expectedPaths := []string{
		"auth.password",
		"auth.secret",
		"secrets.api_secret",
	}

	assert.Len(t, result.Secrets, 3)
	for i, secret := range result.Secrets {
		assert.Contains(t, expectedPaths, secret.Path, "Secret %d path should be in expected paths", i)
	}
}

func TestServiceSpecificLocations(t *testing.T) {
	// Test service-specific locations override global locations
	cfg := &config.Config{
		Services: make(map[string]config.Service),
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Patterns: []string{".*\\.password.*", ".*\\.secret.*"},
				Locations: &config.SecretLocations{
					ScanMode:        config.ScanModeAll,
					AdditionalPaths: []string{"global.password"},
				},
			},
		},
	}

	// Add a test service with its own location config
	cfg.Services["test-service"] = config.Service{
		Enabled: true,
		Name:    "test-service",
		Secrets: &config.Secrets{
			Description: "Service with custom locations",
			Locations: &config.SecretLocations{
				ScanMode:        config.ScanModeTargeted,
				AdditionalPaths: []string{"service.password", "service.secret"},
			},
		},
	}

	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
global:
  password: "should_not_be_found_for_special_service"
service:
  password: "should_be_found"
  secret: "should_be_found"
other:
  password: "should_not_be_found"
`)

	// Test with special service - should use service locations
	result, err := extractor.ExtractSecrets(yamlData, "special-service")
	require.NoError(t, err)

	expectedPaths := []string{"service.password", "service.secret"}
	assert.Len(t, result.Secrets, 2)
	for i, secret := range result.Secrets {
		assert.Contains(t, expectedPaths, secret.Path, "Secret %d path should be in expected paths", i)
	}

	// Test with regular service - should use global locations
	result, err = extractor.ExtractSecrets(yamlData, "regular-service")
	require.NoError(t, err)

	assert.Len(t, result.Secrets, 1)
	assert.Equal(t, "global.password", result.Secrets[0].Path)
}

func TestLocationPatterns(t *testing.T) {
	// Test path patterns in locations
	cfg := &config.Config{
		Services: make(map[string]config.Service),
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Patterns: []string{".*\\.password.*", ".*\\.secret.*", ".*\\.api_key.*", ".*\\.api_secret.*"},
				Locations: &config.SecretLocations{
					ScanMode:     config.ScanModeTargeted,
					PathPatterns: []string{"auth\\..*", "secrets\\.api_.*", "db\\.[^.]*\\.password"},
				},
			},
		},
	}

	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
auth:
  password: "should_match_auth_pattern"
  secret: "should_match_auth_pattern"
secrets:
  api_key: "should_match_api_pattern"
  api_secret: "should_match_api_pattern"
  other_secret: "should_not_match"
db:
  primary:
    password: "should_match_db_pattern"
  secondary:
    password: "should_match_db_pattern"
  deep:
    nested:
      password: "should_not_match_deep_nesting"
config:
  password: "should_not_match"
`)

	result, err := extractor.ExtractSecrets(yamlData, "test")
	require.NoError(t, err)

	// Should find paths matching the patterns
	expectedPaths := []string{
		"auth.password",
		"auth.secret",
		"secrets.api_key",
		"secrets.api_secret",
		"db.primary.password",
		"db.secondary.password",
	}

	assert.Len(t, result.Secrets, 6)
	for i, secret := range result.Secrets {
		assert.Contains(t, expectedPaths, secret.Path, "Secret %d path should match patterns", i)
	}
}

func TestScanModeAll(t *testing.T) {
	// Test that ScanModeAll ignores location specifications
	cfg := &config.Config{
		Services: make(map[string]config.Service),
		Globals: config.Globals{
			Secrets: &config.Secrets{
				Patterns: []string{".*\\.password.*"},
				Locations: &config.SecretLocations{
					ScanMode:        config.ScanModeAll,
					AdditionalPaths: []string{"config.password", "secrets.api_key"},
					Exclude:         []string{"config.password"},
				},
			},
		},
	}

	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlData := []byte(`
auth:
  password: "should_be_found"
config:
  password: "should_also_be_found"
database:
  password: "should_also_be_found"
`)

	result, err := extractor.ExtractSecrets(yamlData, "test")
	require.NoError(t, err)

	// Should find all matching patterns regardless of location settings
	assert.Len(t, result.Secrets, 3)

	foundPaths := make([]string, len(result.Secrets))
	for i, secret := range result.Secrets {
		foundPaths[i] = secret.Path
	}

	assert.Contains(t, foundPaths, "auth.password")
	assert.Contains(t, foundPaths, "config.password")
	assert.Contains(t, foundPaths, "database.password")
}

func TestSecretsLocationPatterns(t *testing.T) {
	// Test path patterns in locations
	yamlData := []byte(`
# Global patterns that apply to all services
globals:
  # Auto Inject Key Values Pairs
  autoInject:
    "values.yaml":
      keys:
        - key: 'secrets."root.properties"."9487e74c-2d27-4085-b637-30a82239b0b2"'
          value: misconfigured
          condition: disabled # ifExists, ifNotExists, always, disabled
          description: "Set Default Secret Value for 9487e74c-2d27-4085-b637-30a82239b0b2"
    "envs/dev01/*/values.yaml":
      keys:
        - key: 'configMap."root.properties"."auth.dataSource.user"'
          value: "{environment}-auth"
          condition: disabled # ifExists, ifNotExists, always, disabled
          description: "Set environment-specific auth datasource user"

  # Hierarchical mapping for accurate secrets extraction
  mappings:
    locations:
      scan_mode: filtered
      include: [ "configMap" ]

    normalizer:
      enabled: true
      description: "Global Normalizations Configuration"
      patterns:
        '[Aa]utoscaling.[Tt]arget.[Cc]pu': 'autoscaling.targetCPUUtilizationPercentage'
        '[Aa]utoscaling.[Tt]arget.[Mm]emory': 'autoscaling.targetMemoryUtilizationPercentage'
        '[Cc]ontainer.[Pp]ort': 'service.targetPort'
        '[Ee]nv': 'envVars'
        '[Rr]eplicas': 'replicaCount'
    transform:
      enabled: true
      description: "Global Transformations Configuration"
      rules:
        ingress_to_hosts:
          type: "ingress_to_hosts"
          source_path: "[Ii]ngress"
          target_path: "hosts.public.domains"
          description: "Extract valid hosts from ingress configurations and collect them into hosts.public.domains list"
    extract:
      enabled: true
      description: "Global Extractions Configuration"
      patterns: {}
    cleaner:
      enabled: true
      description: "Global Extractions Configuration"
      patterns:
        - '[Cc]anary'

  # Hierarchical secrets mapping for accurate secrets extraction
  # Global patterns that apply to all services
  secrets:
    # Location configuration for targeted secret scanning
    locations:
      # Base path where secrets are typically located (defaults to "secrets")
      base_path: "configMap" # Required

      # The new destination of extracted secrets
      store_path: "secrets" # Required

      # Additional specific paths where secrets might be found
      additional_paths: [] # Optional
        # - "auth"
        # - "database"
        # - "configMap.data"
        # - "app.secrets"

      # Path patterns for flexible secret detection
      path_patterns: [] # Optional
        # - ".*\\.auth\\..*"
        # - ".*\\.credentials\\..*"
        # - ".*\\.env\\..*"

      # Scan mode: filtered focuses on secrets section + additional paths
      scan_mode: filtered

    # Common secret patterns across all services
    patterns:
      - ".*\\.password.*"
      - ".*\\.secret.*"
      - ".*jwt\\.secret.*"
      - ".*\\.key$"
      - ".*\\.token.*"
      # Removed overly broad auth pattern - replaced with specific ones below
      - ".*client_secret.*"
      - ".*api_key.*"
      - ".*private_key.*"
      - ".*signing_key.*"
      - ".*encryption_key.*"
      # More specific auth patterns to avoid false positives
      - ".*\\.auth\\.key.*"
      - ".*\\.auth\\.token.*"
      - ".*\\.auth\\.secret.*"

    # Global UUID patterns for client IDs and API keys
    uuids:
      - pattern: ".*client.*uuid.*"
        sensitive: true
        description: "Client UUIDs are typically sensitive identifiers"
      - pattern: "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
        sensitive: true
        description: "Generic UUID pattern - evaluate based on context"

    # Common sensitive values regardless of key name
    values:
      # Base64 encoded secrets (more restrictive to avoid file paths)
      - pattern: "^[A-Za-z0-9+/]{40,}={0,2}$"
        sensitive: true
        description: "Base64 encoded values are likely secrets (40+ chars, no slashes at start)"
      # JWT tokens (start with eyJ)
      - pattern: "^eyJ[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+$"
        sensitive: true
        description: "JWT-like tokens"
      - pattern: "^[A-Fa-f0-9]{32,}$"  # Hex encoded secrets
        sensitive: true
        description: "Long hex strings are likely secrets"

  # Migration Configuration
  migration:
    baseValuesPath: "**/values.yaml"
    envValuesPattern: "**/envs/{cluster}/{namespace}/values.yaml"
    helmValuesFilename: "values.yaml"
    legacyValuesFilename: "legacy-values.yaml"

# Services to Migrate
services:
  heimdall:
    enabled: true
    name: heimdall
    capitalized: Heimdall
    autoInject:
      "values.yaml":
        keys:
          - key: 'secrets."root.properties"."9487e74c-2d27-4085-b637-30a82239b0b2"'
            value: misconfigured
            condition: ifExists # ifExists, ifNotExists, always, disabled
            description: "Set Default Secret Value for 9487e74c-2d27-4085-b637-30a82239b0b2"
      "envs/dev01/*/values.yaml":
        keys:
          - key: 'configMap."root.properties"."auth.dataSource.user"'
            value: "{environment}-auth"
            condition: ifExists # ifExists, ifNotExists, always, disabled
            description: "Set environment-specific auth datasource user"
    mappings: {}
    migration: {}
    secrets:
      # These specific UUIDs are client secrets for heimdall
      keys:
        - "3f4beddd-2061-49b0-ae80-6f1f2ed65b37"
        - "682843b1-d3e0-460e-ab90-6556bc31470f"
        - "936da557-6daa-4444-92cc-161fc290c603"
        - "9487e74c-2d27-4085-b637-30a82239b0b2"
        - "c23203d0-1b8e-4208-92dc-85dc79e6226b"
      patterns:
        - ".*access.refresh.local_client_uuid.*"
        - ".*loginradius.*secret.*"
        - ".*oauth.*secret.*"
        - ".*provider.*secret.*"
        - ".*thirdparty.*apikey.*"
        - ".*thirdparty.parameter.loginradius.*"
      description: "Heimdall authentication service secrets"
`)

	cfg, err := LoadConfig(yamlData)
	require.NoError(t, err)
	cfg.Services = make(map[string]config.Service)
	extractor, err := New(cfg)
	require.NoError(t, err)

	yamlValuesData := []byte(`
configMap:
  root.properties:
    3f4beddd-2061-49b0-ae80-6f1f2ed65b37: 5e137203-053f-445e-8a6e-4defcf6e4618
    936da557-6daa-4444-92cc-161fc290c603: 4453aab9-5c5d-45c7-b852-fe229f0c51fc
    c23203d0-1b8e-4208-92dc-85dc79e6226b: 4adfcb2f-f541-453e-89a3-1ec92a2d8743
    com.viafoura.heimdall.auth.mfa.url: http://auth2.vf-dev2.org/login
    com.viafoura.heimdall.cookie.access.domain: vf-dev2.org
    com.viafoura.heimdall.cookie.refresh.domain: auth.vf-dev2.org
    com.viafoura.heimdall.jwt.secret_1: this is my secret
    com.viafoura.heimdall.jwt.secret_2: this is my other secret
    com.viafoura.heimdall.provider.loginradius.secret: 4adfcb2f-f541-453e-89a3-1ec92a2d8743
    com.viafoura.heimdall.thirdparty.apikey.loginradius: c23203d0-1b8e-4208-92dc-85dc79e6226b
    com.viafoura.heimdall.thirdparty.callbackparameter.loginradius: '&same_window=1&callback=https%3A%2F%2Fauth.vf-dev2.info%2Fthirdpartycallback%3Fcode%3D'
    com.viafoura.heimdall.thirdparty.parameter.loginradius: ?apikey=c23203d0-1b8e-4208-92dc-85dc79e6226b
    com.viafoura.heimdall.thirdparty.url.loginradius: https://viafoura-dev.hub.loginradius.com/RequestHandlor.aspx
    com.viafoura.heimdall.url.resetpassword: //admin.vf-dev2.org/admin/0/0/login/forgot_password
    com.viafoura.heimdall.v2api.host: https://api.vf-dev2.org
    com.viafoura.heimdall.viafoura_core.base_url: https://api.vf-dev2.org/v2
  vfmetrics.properties:
    datadog.tags: environment:vf-dev2
image:
  tag: PR-448-1-4ce270b1
`)

	result, err := extractor.ExtractSecrets(yamlValuesData, "test")
	require.NoError(t, err)

	// Should find paths matching the patterns
	expectedPaths := []string{
		"auth.password",
		"auth.secret",
		"secrets.api_key",
		"secrets.api_secret",
		"db.primary.password",
		"db.secondary.password",
	}

	assert.Len(t, result.Secrets, 6)
	for i, secret := range result.Secrets {
		assert.Contains(t, expectedPaths, secret.Path, "Secret %d path should match patterns", i)
	}
}
