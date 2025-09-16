package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		wantErr    bool
		validateFn func(*testing.T, *Config)
	}{
		{
			name: "valid config with all fields",
			configYAML: `
clusters:
  prod01:
    default: true
    enabled: true
    target: prod-cluster
    source: prod-source
    aws_profile: prod-profile
    aws_region: us-west-2
    namespaces:
      default:
        enabled: true
        name: default
      monitoring:
        enabled: false
        name: monitoring
  staging:
    enabled: false
    target: staging-cluster
services:
  api:
    enabled: true
    name: api-service
  web:
    enabled: false
    name: web-service
cluster: prod01
namespace: default
excludeKeys:
  sensitive: true
  internal: true
`,
			wantErr: false,
			validateFn: func(t *testing.T, cfg *Config) {
				// Validate clusters
				if len(cfg.Clusters) != 2 {
					t.Errorf("expected 2 clusters, got %d", len(cfg.Clusters))
				}

				prod := cfg.Clusters["prod01"]
				if !prod.Default {
					t.Error("prod01 should be default")
				}
				if !prod.Enabled {
					t.Error("prod01 should be enabled")
				}
				if prod.AWSProfile != "prod-profile" {
					t.Errorf("expected aws_profile 'prod-profile', got %s", prod.AWSProfile)
				}

				// Validate services
				if len(cfg.Services) != 2 {
					t.Errorf("expected 2 services, got %d", len(cfg.Services))
				}

				api := cfg.Services["api"]
				if !api.Enabled {
					t.Error("api service should be enabled")
				}

				// Validate global settings
				if cfg.Cluster != "prod01" {
					t.Errorf("expected cluster 'prod01', got %s", cfg.Cluster)
				}
				if cfg.Namespace != "default" {
					t.Errorf("expected namespace 'default', got %s", cfg.Namespace)
				}
			},
		},
		{
			name: "minimal valid config",
			configYAML: `
clusters:
  local:
    enabled: true
services:
  test:
    enabled: true
`,
			wantErr: false,
			validateFn: func(t *testing.T, cfg *Config) {
				if len(cfg.Clusters) != 1 {
					t.Errorf("expected 1 cluster, got %d", len(cfg.Clusters))
				}
				if len(cfg.Services) != 1 {
					t.Errorf("expected 1 service, got %d", len(cfg.Services))
				}
			},
		},
		{
			name:       "invalid YAML",
			configYAML: `invalid: yaml: structure:`,
			wantErr:    true,
		},
		{
			name:       "empty config",
			configYAML: ``,
			wantErr:    false,
			validateFn: func(t *testing.T, cfg *Config) {
				// Empty YAML will result in nil maps, which is expected
				// The application should handle nil maps gracefully
				if cfg == nil {
					t.Error("config should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFn != nil {
				tt.validateFn(t, cfg)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/non/existent/path/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestConfig_GetEnabledClusters(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected []string
	}{
		{
			name: "multiple enabled clusters",
			config: &Config{
				Clusters: map[string]Cluster{
					"prod":    {Enabled: true},
					"staging": {Enabled: false},
					"dev":     {Enabled: true},
				},
			},
			expected: []string{"prod", "dev"},
		},
		{
			name: "no enabled clusters",
			config: &Config{
				Clusters: map[string]Cluster{
					"prod":    {Enabled: false},
					"staging": {Enabled: false},
				},
			},
			expected: []string{},
		},
		{
			name:     "nil clusters map",
			config:   &Config{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEnabledClusters()

			// Convert to map for order-independent comparison
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}

			expectedMap := make(map[string]bool)
			for _, e := range tt.expected {
				expectedMap[e] = true
			}

			if len(resultMap) != len(expectedMap) {
				t.Errorf("expected %d clusters, got %d", len(expectedMap), len(resultMap))
			}

			for key := range expectedMap {
				if !resultMap[key] {
					t.Errorf("expected cluster %s not found", key)
				}
			}
		})
	}
}

func TestConfig_GetEnabledServices(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected []string
	}{
		{
			name: "multiple enabled services",
			config: &Config{
				Services: map[string]Service{
					"api":  {Enabled: true, Name: "api-service"},
					"web":  {Enabled: false, Name: "web-service"},
					"auth": {Enabled: true, Name: "auth-service"},
				},
			},
			expected: []string{"api", "auth"},
		},
		{
			name: "all services disabled",
			config: &Config{
				Services: map[string]Service{
					"api": {Enabled: false},
					"web": {Enabled: false},
				},
			},
			expected: []string{},
		},
		{
			name:     "empty services map",
			config:   &Config{Services: map[string]Service{}},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEnabledServices()

			// Convert to map for order-independent comparison
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}

			expectedMap := make(map[string]bool)
			for _, e := range tt.expected {
				expectedMap[e] = true
			}

			if len(resultMap) != len(expectedMap) {
				t.Errorf("expected %d services, got %d", len(expectedMap), len(resultMap))
			}

			for key := range expectedMap {
				if !resultMap[key] {
					t.Errorf("expected service %s not found", key)
				}
			}
		})
	}
}

func TestConfig_GetEnabledNamespaces(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		clusterName string
		expected    []string
	}{
		{
			name: "cluster with multiple namespaces",
			config: &Config{
				Clusters: map[string]Cluster{
					"prod": {
						Environments: map[string]Environment{
							"production": {
								Enabled: true,
								Namespaces: map[string]Namespace{
									"default":    {Enabled: true, Name: "default"},
									"monitoring": {Enabled: false, Name: "monitoring"},
									"logging":    {Enabled: true, Name: "logging"},
								},
							},
						},
					},
				},
			},
			clusterName: "prod",
			expected:    []string{"default", "logging"},
		},
		{
			name: "cluster with no enabled namespaces",
			config: &Config{
				Clusters: map[string]Cluster{
					"staging": {
						Environments: map[string]Environment{
							"staging": {
								Enabled: true,
								Namespaces: map[string]Namespace{
									"default": {Enabled: false, Name: "default"},
									"test":    {Enabled: false, Name: "test"},
								},
							},
						},
					},
				},
			},
			clusterName: "staging",
			expected:    []string{},
		},
		{
			name: "non-existent cluster",
			config: &Config{
				Clusters: map[string]Cluster{
					"prod": {},
				},
			},
			clusterName: "non-existent",
			expected:    nil,
		},
		{
			name: "cluster with nil namespaces",
			config: &Config{
				Clusters: map[string]Cluster{
					"dev": {
						Enabled: true,
					},
				},
			},
			clusterName: "dev",
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetEnabledNamespaces(tt.clusterName)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			// Convert to map for order-independent comparison
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}

			expectedMap := make(map[string]bool)
			for _, e := range tt.expected {
				expectedMap[e] = true
			}

			if len(resultMap) != len(expectedMap) {
				t.Errorf("expected %d namespaces, got %d", len(expectedMap), len(resultMap))
			}

			for key := range expectedMap {
				if !resultMap[key] {
					t.Errorf("expected namespace %s not found", key)
				}
			}
		})
	}
}

func TestConfig_GetDefaultCluster(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		expectedName   string
		expectedExists bool
	}{
		{
			name: "has default cluster",
			config: &Config{
				Clusters: map[string]Cluster{
					"prod":    {Default: true, Enabled: true, Target: "prod-target"},
					"staging": {Default: false, Enabled: true},
				},
			},
			expectedName:   "prod",
			expectedExists: true,
		},
		{
			name: "no default cluster",
			config: &Config{
				Clusters: map[string]Cluster{
					"prod":    {Default: false},
					"staging": {Default: false},
				},
			},
			expectedName:   "",
			expectedExists: false,
		},
		{
			name: "multiple defaults (first wins)",
			config: &Config{
				Clusters: map[string]Cluster{
					"prod":    {Default: true},
					"staging": {Default: true},
				},
			},
			expectedExists: true, // One of them will be returned
		},
		{
			name:           "empty clusters",
			config:         &Config{Clusters: map[string]Cluster{}},
			expectedName:   "",
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, cluster := tt.config.GetDefaultCluster()

			if tt.expectedExists {
				if cluster == nil {
					t.Error("expected default cluster, got nil")
				}
				if tt.expectedName != "" && name != tt.expectedName {
					t.Errorf("expected cluster name %s, got %s", tt.expectedName, name)
				}
				if cluster != nil && !cluster.Default {
					t.Error("returned cluster should have Default=true")
				}
			} else {
				if cluster != nil {
					t.Errorf("expected no default cluster, got %v", cluster)
				}
				if name != "" {
					t.Errorf("expected empty name, got %s", name)
				}
			}
		})
	}
}

func TestCluster_Properties(t *testing.T) {
	cluster := Cluster{
		Default:    true,
		Enabled:    true,
		Target:     "prod-cluster",
		Source:     "prod-source",
		AWSProfile: "prod-profile",
		AWSRegion:  "us-west-2",
		Environments: map[string]Environment{
			"production": {
				Enabled: true,
				Namespaces: map[string]Namespace{
					"default": {Enabled: true, Name: "default"},
				},
			},
		},
	}

	if !cluster.Default {
		t.Error("Default should be true")
	}
	if !cluster.Enabled {
		t.Error("Enabled should be true")
	}
	if cluster.Target != "prod-cluster" {
		t.Errorf("expected Target 'prod-cluster', got %s", cluster.Target)
	}
	if cluster.AWSProfile != "prod-profile" {
		t.Errorf("expected AWSProfile 'prod-profile', got %s", cluster.AWSProfile)
	}
	if cluster.AWSRegion != "us-west-2" {
		t.Errorf("expected AWSRegion 'us-west-2', got %s", cluster.AWSRegion)
	}
}

func TestService_Properties(t *testing.T) {
	service := Service{
		Enabled: true,
		Name:    "api-service",
	}

	if !service.Enabled {
		t.Error("Enabled should be true")
	}
	if service.Name != "api-service" {
		t.Errorf("expected Name 'api-service', got %s", service.Name)
	}
}

func TestNamespace_Properties(t *testing.T) {
	namespace := Namespace{
		Enabled: true,
		Name:    "production",
	}

	if !namespace.Enabled {
		t.Error("Enabled should be true")
	}
	if namespace.Name != "production" {
		t.Errorf("expected Name 'production', got %s", namespace.Name)
	}
}
