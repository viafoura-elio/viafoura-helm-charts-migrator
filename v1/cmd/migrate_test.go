package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setupFunc func(t *testing.T, tmpDir string)
		wantErr   bool
		validate  func(t *testing.T, tmpDir string, output string)
	}{
		{
			name: "dry run with minimal config",
			args: []string{"--dry-run", "--config", "config.yaml", "--target", "apps"},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create minimal config
				configContent := `
clusters:
  test:
    enabled: true
    default: true
    target: test-cluster
    source: test-context
    namespaces:
      default:
        enabled: true
        name: default
services:
  testservice:
    enabled: true
    name: testservice
    capitalized: TestService
`
				err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
				require.NoError(t, err)

				// Create base chart
				baseDir := filepath.Join(tmpDir, "migration", "base-chart")
				err = os.MkdirAll(baseDir, 0755)
				require.NoError(t, err)

				chartYaml := `apiVersion: v2
name: base-chart
description: Base chart for Base-Chart
version: 0.1.0`
				err = os.WriteFile(filepath.Join(baseDir, "Chart.yaml"), []byte(chartYaml), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string, output string) {
				// Check that dry run message appears
				assert.Contains(t, output, "DRY RUN")
			},
		},
		{
			name: "with service filter",
			args: []string{
				"--dry-run",
				"--config", "config.yaml",
				"--services", "service1,service2",
				"--target", "apps",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				configContent := `
clusters:
  test:
    enabled: true
    default: true
    target: test
    source: test
services:
  service1:
    enabled: true
    name: service1
  service2:
    enabled: true
    name: service2
  service3:
    enabled: true
    name: service3
`
				err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string, output string) {
				// Verify service filtering worked
				assert.Contains(t, output, "service1")
				assert.Contains(t, output, "service2")
			},
		},
		{
			name: "with cluster filter",
			args: []string{
				"--dry-run",
				"--config", "config.yaml",
				"--cluster", "dev01",
				"--target", "apps",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				configContent := `
clusters:
  dev01:
    enabled: true
    target: dev
    source: dev-context
  prod01:
    enabled: true
    target: prod
    source: prod-context
services:
  testservice:
    enabled: true
    name: testservice
`
				err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string, output string) {
				// Verify cluster filtering
				assert.Contains(t, output, "dev01")
			},
		},
		{
			name:    "missing config file",
			args:    []string{"--config", "nonexistent.yaml"},
			wantErr: true,
		},
		{
			name: "with cache options",
			args: []string{
				"--dry-run",
				"--config", "config.yaml",
				"--cache-dir", ".test-cache",
				"--cleanup-cache",
				"--target", "apps",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create config
				configContent := `
clusters:
  test:
    enabled: true
    default: true
    target: test
    source: test
services:
  test:
    enabled: true
    name: test
`
				err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
				require.NoError(t, err)

				// Create cache dir with some content
				cacheDir := filepath.Join(tmpDir, ".test-cache")
				err = os.MkdirAll(cacheDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(cacheDir, "test.txt"), []byte("cached"), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string, output string) {
				// Cache should be cleaned
				cacheDir := filepath.Join(tmpDir, ".test-cache")
				_, err := os.Stat(filepath.Join(cacheDir, "test.txt"))
				// File should not exist after cleanup
				assert.True(t, os.IsNotExist(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			oldWd, _ := os.Getwd()
			defer os.Chdir(oldWd)
			err := os.Chdir(tmpDir)
			require.NoError(t, err)

			// Setup test environment
			if tt.setupFunc != nil {
				tt.setupFunc(t, tmpDir)
			}

			// Create command
			cmd := &cobra.Command{Use: "test"}
			cmd.AddCommand(migrateCmd)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set args
			args := append([]string{"migrate"}, tt.args...)
			cmd.SetArgs(args)

			// Execute
			err = cmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Validate
			if tt.validate != nil {
				tt.validate(t, tmpDir, buf.String())
			}
		})
	}
}

func TestMigrateCommand_Flags(t *testing.T) {
	// Test that all expected flags are registered
	expectedFlags := []string{
		"base",
		"dry-run",
		"source",
		"target",
		"cache-dir",
		"cleanup-cache",
		"no-refresh-cache",
		"aws-profile",
		"no-sops",
		"cluster",
		"namespaces",
		"services",
	}

	for _, flagName := range expectedFlags {
		flag := migrateCmd.Flag(flagName)
		assert.NotNil(t, flag, "Flag %s should be registered", flagName)
	}

	// Test default values
	assert.Equal(t, "migration/base-chart", migrateCmd.Flag("base").DefValue)
	assert.Equal(t, "false", migrateCmd.Flag("dry-run").DefValue)
	assert.Equal(t, "apps/", migrateCmd.Flag("target").DefValue)
	assert.Equal(t, ".cache", migrateCmd.Flag("cache-dir").DefValue)
	assert.Equal(t, "cicd-sre", migrateCmd.Flag("aws-profile").DefValue)
}

func TestMigrateCommand_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test (set INTEGRATION_TEST=true to run)")
	}

	tmpDir := t.TempDir()

	// Create a complete test environment
	setupCompleteTestEnvironment(t, tmpDir)

	// Change to tmp dir
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	// Run migration
	cmd := &cobra.Command{Use: "test"}
	cmd.AddCommand(migrateCmd)

	cmd.SetArgs([]string{
		"migrate",
		"--dry-run",
		"--config", "config.yaml",
		"--target", "apps",
		"--services", "heimdall,livecomments",
		"--cluster", "dev01",
	})

	err = cmd.Execute()
	assert.NoError(t, err)

	// Verify structure was created (in dry-run, structure is logged but not created)
	// In a real test, we'd check the actual files created
}

func setupCompleteTestEnvironment(t *testing.T, tmpDir string) {
	// Create comprehensive config
	configContent := `
clusters:
  dev01:
    enabled: true
    default: true
    target: dev
    source: dev-context
    aws_profile: test-sre
    aws_region: us-east-1
    namespaces:
      default:
        enabled: true
        name: default
      staging:
        enabled: true
        name: staging
  prod01:
    enabled: true
    target: prod
    source: prod-context
    aws_profile: production-sre
    aws_region: us-east-1
    namespaces:
      default:
        enabled: true
        name: default

services:
  heimdall:
    enabled: true
    name: heimdall
    capitalized: Heimdall
  livecomments:
    enabled: true
    name: livecomments
    capitalized: LiveComments
  auth-service:
    enabled: false
    name: auth-service
    capitalized: AuthService

globals:
  migration:
    baseValuesPath: "**/values.yaml"
    envValuesPattern: "**/envs/{cluster}/{namespace}/values.yaml"
    legacyValuesFilename: "legacy-values.yaml"
    helmValuesFilename: "helm-values.yaml"
  converter:
    skipJavaProperties: true
    skipUppercaseKeys: true
    minUppercaseChars: 3
`
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Create base chart
	baseDir := filepath.Join(tmpDir, "migration", "base-chart")
	err = os.MkdirAll(filepath.Join(baseDir, "templates"), 0755)
	require.NoError(t, err)

	// Chart.yaml
	chartContent := `apiVersion: v2
name: base-chart
description: A Helm chart for Base-Chart
type: application
version: 0.1.0
appVersion: "1.0"`
	err = os.WriteFile(filepath.Join(baseDir, "Chart.yaml"), []byte(chartContent), 0644)
	require.NoError(t, err)

	// values.yaml
	valuesContent := `replicaCount: 1
image:
  repository: base-chart
  pullPolicy: IfNotPresent
  tag: latest
service:
  type: ClusterIP
  port: 80`
	err = os.WriteFile(filepath.Join(baseDir, "values.yaml"), []byte(valuesContent), 0644)
	require.NoError(t, err)

	// deployment.yaml template
	deploymentContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
spec:
  replicas: {{ .Values.replicaCount }}
  template:
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}`
	err = os.WriteFile(filepath.Join(baseDir, "templates", "deployment.yaml"), []byte(deploymentContent), 0644)
	require.NoError(t, err)
}
