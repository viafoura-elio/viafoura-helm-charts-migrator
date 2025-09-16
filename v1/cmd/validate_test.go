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

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setupFunc func(t *testing.T, tmpDir string)
		wantErr   bool
		validate  func(t *testing.T, output string)
	}{
		{
			name: "validate valid chart",
			args: []string{"--chart", "test-chart"},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create valid chart
				chartDir := filepath.Join(tmpDir, "test-chart")
				err := os.MkdirAll(filepath.Join(chartDir, "templates"), 0755)
				require.NoError(t, err)

				// Chart.yaml
				chartContent := `apiVersion: v2
name: test-chart
description: A test chart
version: 1.0.0`
				err = os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartContent), 0644)
				require.NoError(t, err)

				// values.yaml
				valuesContent := `replicaCount: 1
image:
  repository: nginx
  tag: latest`
				err = os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte(valuesContent), 0644)
				require.NoError(t, err)

				// Valid deployment template
				deploymentContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app: {{ .Release.Name }}
    spec:
      containers:
      - name: app
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"`
				err = os.WriteFile(filepath.Join(chartDir, "templates", "deployment.yaml"), []byte(deploymentContent), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Validating chart")
				// Should pass validation
				assert.NotContains(t, output, "ERROR")
			},
		},
		{
			name: "validate with custom values",
			args: []string{
				"--chart", "test-chart",
				"--values", "custom-values.yaml",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create chart
				chartDir := filepath.Join(tmpDir, "test-chart")
				err := os.MkdirAll(filepath.Join(chartDir, "templates"), 0755)
				require.NoError(t, err)

				chartContent := `apiVersion: v2
name: test-chart
version: 1.0.0`
				err = os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartContent), 0644)
				require.NoError(t, err)

				// Custom values
				customValues := `replicaCount: 3
customValue: test`
				err = os.WriteFile(filepath.Join(tmpDir, "custom-values.yaml"), []byte(customValues), 0644)
				require.NoError(t, err)

				// Template using custom values
				templateContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}
data:
  replicas: "{{ .Values.replicaCount }}"
  custom: "{{ .Values.customValue }}"`
				err = os.WriteFile(filepath.Join(chartDir, "templates", "configmap.yaml"), []byte(templateContent), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "custom-values.yaml")
			},
		},
		{
			name: "validate with strict mode",
			args: []string{
				"--chart", "test-chart",
				"--strict",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create chart with deprecated API
				chartDir := filepath.Join(tmpDir, "test-chart")
				err := os.MkdirAll(filepath.Join(chartDir, "templates"), 0755)
				require.NoError(t, err)

				chartContent := `apiVersion: v2
name: test-chart
version: 1.0.0`
				err = os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartContent), 0644)
				require.NoError(t, err)

				// Template with deprecated API version
				templateContent := `apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: {{ .Release.Name }}
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        backend:
          serviceName: test
          servicePort: 80`
				err = os.WriteFile(filepath.Join(chartDir, "templates", "ingress.yaml"), []byte(templateContent), 0644)
				require.NoError(t, err)
			},
			wantErr: true, // Should fail in strict mode with deprecated API
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "deprecated")
			},
		},
		{
			name: "validate specific service",
			args: []string{
				"--service", "heimdall",
				"--cluster", "dev01",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create config
				configContent := `
clusters:
  dev01:
    enabled: true
    target: dev
services:
  heimdall:
    enabled: true
    name: heimdall`
				err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
				require.NoError(t, err)

				// Create service chart
				serviceDir := filepath.Join(tmpDir, "apps", "heimdall")
				err = os.MkdirAll(filepath.Join(serviceDir, "templates"), 0755)
				require.NoError(t, err)

				chartContent := `apiVersion: v2
name: heimdall
version: 1.0.0`
				err = os.WriteFile(filepath.Join(serviceDir, "Chart.yaml"), []byte(chartContent), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "heimdall")
				assert.Contains(t, output, "dev01")
			},
		},
		{
			name:    "missing chart",
			args:    []string{"--chart", "nonexistent"},
			wantErr: true,
		},
		{
			name: "validate with kubernetes version",
			args: []string{
				"--chart", "test-chart",
				"--kubernetes-version", "1.25.0",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create chart
				chartDir := filepath.Join(tmpDir, "test-chart")
				err := os.MkdirAll(filepath.Join(chartDir, "templates"), 0755)
				require.NoError(t, err)

				chartContent := `apiVersion: v2
name: test-chart
version: 1.0.0
kubeVersion: ">=1.20.0"`
				err = os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartContent), 0644)
				require.NoError(t, err)

				// Template with version-specific features
				templateContent := `apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
spec:
  type: ClusterIP
  ports:
  - port: 80`
				err = os.WriteFile(filepath.Join(chartDir, "templates", "service.yaml"), []byte(templateContent), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "1.25")
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
			cmd.AddCommand(validateCmd)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set args
			args := append([]string{"validate"}, tt.args...)
			cmd.SetArgs(args)

			// Execute
			err = cmd.Execute()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// Note: validate command may not exist yet, so we check for nil
				if validateCmd == nil {
					t.Skip("validate command not implemented")
					return
				}
				assert.NoError(t, err)
			}

			// Validate output
			if tt.validate != nil {
				tt.validate(t, buf.String())
			}
		})
	}
}

func TestValidateCommand_Flags(t *testing.T) {
	// Skip if command doesn't exist
	if validateCmd == nil {
		t.Skip("validate command not implemented")
		return
	}

	// Test that expected flags are registered
	expectedFlags := []string{
		"chart",
		"values",
		"service",
		"cluster",
		"strict",
		"kubernetes-version",
		"output",
	}

	for _, flagName := range expectedFlags {
		flag := validateCmd.Flag(flagName)
		assert.NotNil(t, flag, "Flag %s should be registered", flagName)
	}
}

func TestValidateCommand_MultipleCharts(t *testing.T) {
	if validateCmd == nil {
		t.Skip("validate command not implemented")
		return
	}

	tmpDir := t.TempDir()

	// Create multiple charts
	charts := []string{"chart1", "chart2", "chart3"}
	for _, chartName := range charts {
		chartDir := filepath.Join(tmpDir, chartName)
		err := os.MkdirAll(filepath.Join(chartDir, "templates"), 0755)
		require.NoError(t, err)

		chartContent := `apiVersion: v2
name: ` + chartName + `
version: 1.0.0`
		err = os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartContent), 0644)
		require.NoError(t, err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	// Validate all charts
	for _, chartName := range charts {
		cmd := &cobra.Command{Use: "test"}
		cmd.AddCommand(validateCmd)

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		cmd.SetArgs([]string{"validate", "--chart", chartName})

		err := cmd.Execute()
		assert.NoError(t, err, "Failed to validate %s", chartName)
		assert.Contains(t, buf.String(), chartName)
	}
}
