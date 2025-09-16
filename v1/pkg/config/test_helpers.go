package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConfigBuilder helps build test configurations
type TestConfigBuilder struct {
	config *Config
}

// NewTestConfigBuilder creates a new test config builder
func NewTestConfigBuilder() *TestConfigBuilder {
	return &TestConfigBuilder{
		config: &Config{
			Clusters: make(map[string]Cluster),
			Services: make(map[string]Service),
			Globals: Globals{
				Converter: ConverterConfig{
					MinUppercaseChars:  3,
					SkipJavaProperties: false,
					SkipUppercaseKeys:  false,
				},
				Performance: PerformanceConfig{
					MaxConcurrentServices: 5,
					ShowProgress:         true,
				},
				SOPS: SOPSConfig{
					Enabled:         false,
					ParallelWorkers: 5,
					Timeout:         30,
				},
			},
		},
	}
}

// WithCluster adds a cluster to the test config
func (b *TestConfigBuilder) WithCluster(name string, cluster Cluster) *TestConfigBuilder {
	b.config.Clusters[name] = cluster
	return b
}

// WithService adds a service to the test config
func (b *TestConfigBuilder) WithService(name string, service Service) *TestConfigBuilder {
	b.config.Services[name] = service
	return b
}

// WithGlobals sets the global configuration
func (b *TestConfigBuilder) WithGlobals(globals Globals) *TestConfigBuilder {
	b.config.Globals = globals
	return b
}

// Build returns the built configuration
func (b *TestConfigBuilder) Build() *Config {
	return b.config
}

// CreateTestHierarchy creates a test hierarchical configuration
func CreateTestHierarchy() *HierarchicalConfig {
	hc := NewHierarchicalConfig()
	
	// Set defaults
	hc.SetDefaults(map[string]interface{}{
		"globals": map[string]interface{}{
			"converter": map[string]interface{}{
				"minUppercaseChars":  3,
				"skipJavaProperties": false,
			},
			"performance": map[string]interface{}{
				"maxConcurrentServices": 5,
				"showProgress":         true,
			},
		},
		"clusters": map[string]interface{}{},
		"services": map[string]interface{}{},
	})
	
	// Set globals (overrides defaults)
	hc.SetGlobals(map[string]interface{}{
		"globals": map[string]interface{}{
			"converter": map[string]interface{}{
				"minUppercaseChars":  5,
				"skipJavaProperties": true,
			},
			"performance": map[string]interface{}{
				"maxConcurrentServices": 10,
			},
			"sops": map[string]interface{}{
				"enabled":    true,
				"awsProfile": "test-profile",
			},
		},
	})
	
	// Set cluster configs
	hc.SetClusterConfig("prod01", map[string]interface{}{
		"cluster": "prod01",
		"source":  "kops-prod",
		"target":  "eks-prod",
		"default": true,
		"globals": map[string]interface{}{
			"performance": map[string]interface{}{
				"maxConcurrentServices": 20,
			},
		},
	})
	
	hc.SetClusterConfig("dev01", map[string]interface{}{
		"cluster": "dev01",
		"source":  "kops-dev",
		"target":  "eks-dev",
		"default": false,
	})
	
	// Set environment configs
	hc.SetEnvironmentConfig("prod01", "production", map[string]interface{}{
		"enabled": true,
		"namespaces": map[string]interface{}{
			"default": map[string]interface{}{
				"enabled": true,
			},
			"monitoring": map[string]interface{}{
				"enabled": true,
			},
		},
	})
	
	hc.SetEnvironmentConfig("dev01", "development", map[string]interface{}{
		"enabled": true,
		"namespaces": map[string]interface{}{
			"default": map[string]interface{}{
				"enabled": true,
			},
		},
	})
	
	// Set service configs
	hc.SetServiceConfig("heimdall", map[string]interface{}{
		"services": map[string]interface{}{
			"heimdall": map[string]interface{}{
				"enabled":     true,
				"capitalized": "Heimdall",
				"converter": map[string]interface{}{
					"minUppercaseChars": 10,
				},
			},
		},
	})
	
	hc.SetServiceConfig("auth", map[string]interface{}{
		"services": map[string]interface{}{
			"auth": map[string]interface{}{
				"enabled":     true,
				"capitalized": "Auth",
			},
		},
	})
	
	return hc
}

// CreateTestConfigDirectory creates a temporary directory with test config files
func CreateTestConfigDirectory(t *testing.T) string {
	tmpDir := t.TempDir()
	
	// Create defaults.yaml
	defaultsContent := `
defaults:
  converter:
    minUppercaseChars: 3
    skipJavaProperties: false
  performance:
    maxConcurrentServices: 5
    showProgress: true
`
	err := os.WriteFile(filepath.Join(tmpDir, "defaults.yaml"), []byte(defaultsContent), 0644)
	require.NoError(t, err)
	
	// Create globals.yaml
	globalsContent := `
globals:
  converter:
    minUppercaseChars: 5
    skipJavaProperties: true
  performance:
    maxConcurrentServices: 10
  sops:
    enabled: true
    awsProfile: test-profile
`
	err = os.WriteFile(filepath.Join(tmpDir, "globals.yaml"), []byte(globalsContent), 0644)
	require.NoError(t, err)
	
	// Create clusters directory
	clustersDir := filepath.Join(tmpDir, "clusters")
	err = os.MkdirAll(clustersDir, 0755)
	require.NoError(t, err)
	
	// Create prod01.yaml
	prod01Content := `
cluster: prod01
source: kops-prod
target: eks-prod
default: true
performance:
  maxConcurrentServices: 20
`
	err = os.WriteFile(filepath.Join(clustersDir, "prod01.yaml"), []byte(prod01Content), 0644)
	require.NoError(t, err)
	
	// Create dev01.yaml
	dev01Content := `
cluster: dev01
source: kops-dev
target: eks-dev
default: false
`
	err = os.WriteFile(filepath.Join(clustersDir, "dev01.yaml"), []byte(dev01Content), 0644)
	require.NoError(t, err)
	
	// Create services directory
	servicesDir := filepath.Join(tmpDir, "services")
	err = os.MkdirAll(servicesDir, 0755)
	require.NoError(t, err)
	
	// Create heimdall.yaml
	heimdallContent := `
enabled: true
capitalized: Heimdall
converter:
  minUppercaseChars: 10
`
	err = os.WriteFile(filepath.Join(servicesDir, "heimdall.yaml"), []byte(heimdallContent), 0644)
	require.NoError(t, err)
	
	// Create auth.yaml
	authContent := `
enabled: true
capitalized: Auth
`
	err = os.WriteFile(filepath.Join(servicesDir, "auth.yaml"), []byte(authContent), 0644)
	require.NoError(t, err)
	
	return tmpDir
}

// TestCluster creates a test cluster configuration
func TestCluster(name string, source string, target string, isDefault bool) Cluster {
	return Cluster{
		Source:           source,
		Target:           target,
		Default:          isDefault,
		Enabled:          true,
		DefaultNamespace: "default",
		AWSProfile:       "test-profile",
		AWSRegion:        "us-west-2",
		Environments: map[string]Environment{
			"test": {
				Enabled: true,
				Namespaces: map[string]Namespace{
					"default": {
						Enabled: true,
						Name:    "default",
					},
				},
			},
		},
	}
}

// TestService creates a test service configuration
func TestService(name string, capitalized string, enabled bool) Service {
	return Service{
		Enabled:     enabled,
		Capitalized: capitalized,
		AutoInject:  make(map[string]AutoInjectFile),
		Secrets:     &Secrets{},
	}
}

// TestEnvironment creates a test environment configuration
func TestEnvironment(enabled bool, namespaces ...string) Environment {
	env := Environment{
		Enabled:    enabled,
		Namespaces: make(map[string]Namespace),
	}
	
	for _, ns := range namespaces {
		env.Namespaces[ns] = Namespace{
			Enabled: true,
			Name:    ns,
		}
	}
	
	return env
}

// AssertConfigEqual asserts two configs are equal with detailed error messages
func AssertConfigEqual(t *testing.T, expected, actual *Config) {
	t.Helper()
	
	// Check basic fields
	require.Equal(t, expected.Cluster, actual.Cluster, "Cluster mismatch")
	require.Equal(t, expected.Namespace, actual.Namespace, "Namespace mismatch")
	
	// Check globals
	require.Equal(t, expected.Globals.Converter, actual.Globals.Converter, "Converter config mismatch")
	require.Equal(t, expected.Globals.Performance, actual.Globals.Performance, "Performance config mismatch")
	require.Equal(t, expected.Globals.SOPS, actual.Globals.SOPS, "SOPS config mismatch")
	
	// Check clusters
	require.Equal(t, len(expected.Clusters), len(actual.Clusters), "Cluster count mismatch")
	for name, expectedCluster := range expected.Clusters {
		actualCluster, exists := actual.Clusters[name]
		require.True(t, exists, "Cluster %s not found", name)
		require.Equal(t, expectedCluster, actualCluster, "Cluster %s mismatch", name)
	}
	
	// Check services
	require.Equal(t, len(expected.Services), len(actual.Services), "Service count mismatch")
	for name, expectedService := range expected.Services {
		actualService, exists := actual.Services[name]
		require.True(t, exists, "Service %s not found", name)
		require.Equal(t, expectedService, actualService, "Service %s mismatch", name)
	}
}

// MockConfigForTesting creates a complete mock configuration for testing
func MockConfigForTesting() *Config {
	return &Config{
		Cluster:   "test-cluster",
		Namespace: "test-namespace",
		Globals: Globals{
			Converter: ConverterConfig{
				MinUppercaseChars:  5,
				SkipJavaProperties: true,
				SkipUppercaseKeys:  false,
			},
			Performance: PerformanceConfig{
				MaxConcurrentServices: 10,
				ShowProgress:         true,
			},
			SOPS: SOPSConfig{
				Enabled:         true,
				AwsProfile:      "test-profile",
				ParallelWorkers: 5,
				ConfigFile:      ".sops.yaml",
				PathRegex:       ".*\\.enc\\.yaml$",
				SkipUnchanged:   true,
				Timeout:         30,
			},
			AutoInject: map[string]AutoInjectFile{
				"configmap": {
					Keys: []AutoInjectKey{
						{
							Key:   "app.config",
							Value: "test-value",
						},
					},
				},
			},
			Mappings: &Mappings{},
			Secrets: &Secrets{
				Locations: &SecretLocations{
					BasePath:        "secrets",
					AdditionalPaths: []string{"auth", "database"},
					PathPatterns:    []string{".*secret.*", ".*password.*"},
				},
				Patterns: []string{".*_KEY$", ".*_SECRET$"},
				Keys:     []string{"apiKey", "secretKey"},
			},
			Migration: Migration{
				BaseValuesPath:       "base-values.yaml",
				EnvValuesPattern:     "**/values.yaml",
				LegacyValuesFilename: "legacy-values.yaml",
				HelmValuesFilename:   "helm-values.yaml",
			},
		},
		Clusters: map[string]Cluster{
			"prod01": TestCluster("prod01", "kops-prod", "eks-prod", true),
			"dev01":  TestCluster("dev01", "kops-dev", "eks-dev", false),
		},
		Services: map[string]Service{
			"heimdall": TestService("heimdall", "Heimdall", true),
			"auth":     TestService("auth", "Auth", true),
			"disabled": TestService("disabled", "Disabled", false),
		},
	}
}