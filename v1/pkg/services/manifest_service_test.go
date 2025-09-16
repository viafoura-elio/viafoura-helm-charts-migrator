package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"helm-charts-migrator/v1/pkg/config"
)

func TestManifestService_ExtractDeployment(t *testing.T) {
	tests := []struct {
		name        string
		manifest    string
		wantErr     bool
		errorMsg    string
		validate    func(t *testing.T, config *DeploymentConfig)
	}{
		{
			name: "valid deployment manifest",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: test-app
      annotations:
        ad.datadoghq.com/test-app.logs: '[{"service": "test-app", "source": "nodejs"}]'
    spec:
      containers:
      - name: test-app
        image: myregistry/test-app:v1.2.3
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 250m
            memory: 256Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        env:
        - name: NODE_ENV
          value: production
`,
			wantErr: false,
			validate: func(t *testing.T, config *DeploymentConfig) {
				assert.Equal(t, int32(3), config.Replicas)
				assert.Equal(t, "myregistry/test-app", config.Image.Repository)
				assert.Equal(t, "v1.2.3", config.Image.Tag)
				assert.Equal(t, "IfNotPresent", config.Image.PullPolicy)
				assert.Equal(t, "500m", config.Resources.Limits.CPU)
				assert.Equal(t, "512Mi", config.Resources.Limits.Memory)
				assert.NotNil(t, config.Probes.Liveness)
				assert.Equal(t, "/health", config.Probes.Liveness.HTTPGet.Path)
				assert.Len(t, config.Environment, 1)
				assert.Equal(t, "NODE_ENV", config.Environment[0].Name)
				assert.True(t, config.Datadog.Logs.Enabled)
				assert.Equal(t, "test-app", config.Datadog.Logs.Service)
			},
		},
		{
			name: "deployment with datadog v1 annotations",
			manifest: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        tags.datadoghq.com/service: test-service
        tags.datadoghq.com/version: v2.0.0
        tags.datadoghq.com/env: production
        ad.datadoghq.com/test-app.checks: '[{"prometheus_url": "http://%%host%%:9090/metrics"}]'
    spec:
      containers:
      - name: test-app
        image: test-app
`,
			wantErr: false,
			validate: func(t *testing.T, config *DeploymentConfig) {
				assert.True(t, config.Datadog.Enabled)
				assert.Equal(t, "test-service", config.Datadog.Service)
				assert.Equal(t, "v2.0.0", config.Datadog.Version)
				assert.Equal(t, "production", config.Datadog.Environment)
				assert.True(t, config.Datadog.Metrics.Enabled)
			},
		},
		{
			name:     "invalid yaml",
			manifest: "not valid yaml: {[}",
			wantErr:  true,
			errorMsg: "failed to unmarshal deployment",
		},
		{
			name: "not a deployment",
			manifest: `
apiVersion: v1
kind: Service
metadata:
  name: test-service
`,
			wantErr:  true,
			errorMsg: "manifest is not a Deployment",
		},
	}

	cfg := &config.Config{}
	svc := NewManifestService(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ExtractDeployment(tt.manifest)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, result)
			
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestManifestService_ExtractProbes(t *testing.T) {
	tests := []struct {
		name      string
		container *v1.Container
		wantErr   bool
		validate  func(t *testing.T, config *ProbeConfig)
	}{
		{
			name: "container with all probes",
			container: &v1.Container{
				LivenessProbe: &v1.Probe{
					ProbeHandler: v1.ProbeHandler{
						HTTPGet: &v1.HTTPGetAction{
							Path: "/health",
							Port: intstr.FromInt(8080),
						},
					},
					InitialDelaySeconds: 30,
					TimeoutSeconds:      5,
					PeriodSeconds:       10,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				ReadinessProbe: &v1.Probe{
					ProbeHandler: v1.ProbeHandler{
						TCPSocket: &v1.TCPSocketAction{
							Port: intstr.FromInt(8080),
						},
					},
					InitialDelaySeconds: 10,
					PeriodSeconds:       5,
				},
				StartupProbe: &v1.Probe{
					ProbeHandler: v1.ProbeHandler{
						Exec: &v1.ExecAction{
							Command: []string{"/bin/sh", "-c", "test -f /app/ready"},
						},
					},
					InitialDelaySeconds: 0,
					PeriodSeconds:       5,
					FailureThreshold:    30,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *ProbeConfig) {
				// Liveness probe
				require.NotNil(t, config.Liveness)
				assert.NotNil(t, config.Liveness.HTTPGet)
				assert.Equal(t, "/health", config.Liveness.HTTPGet.Path)
				assert.Equal(t, int32(30), config.Liveness.InitialDelaySeconds)
				assert.Equal(t, int32(5), config.Liveness.TimeoutSeconds)
				
				// Readiness probe
				require.NotNil(t, config.Readiness)
				assert.NotNil(t, config.Readiness.TCPSocket)
				assert.Equal(t, int32(10), config.Readiness.InitialDelaySeconds)
				
				// Startup probe
				require.NotNil(t, config.Startup)
				assert.NotNil(t, config.Startup.Exec)
				assert.Equal(t, []string{"/bin/sh", "-c", "test -f /app/ready"}, config.Startup.Exec.Command)
				assert.Equal(t, int32(30), config.Startup.FailureThreshold)
			},
		},
		{
			name:      "nil container",
			container: nil,
			wantErr:   true,
		},
		{
			name:      "container with no probes",
			container: &v1.Container{},
			wantErr:   false,
			validate: func(t *testing.T, config *ProbeConfig) {
				assert.Nil(t, config.Liveness)
				assert.Nil(t, config.Readiness)
				assert.Nil(t, config.Startup)
			},
		},
	}

	cfg := &config.Config{}
	svc := NewManifestService(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ExtractProbes(tt.container)
			
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, result)
			
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestManifestService_ConvertDatadogAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        *DatadogConfig
		wantErr     bool
	}{
		{
			name: "complete datadog annotations",
			annotations: map[string]string{
				"tags.datadoghq.com/service":        "my-service",
				"tags.datadoghq.com/version":        "v1.0.0",
				"tags.datadoghq.com/env":            "production",
				"ad.datadoghq.com/container.logs":   `[{"service": "my-service", "source": "nodejs"}]`,
				"ad.datadoghq.com/container.checks": `[{"prometheus_url": "http://%%host%%:9090/metrics", "port": 9090}]`,
			},
			want: &DatadogConfig{
				Enabled:     true,
				Service:     "my-service",
				Version:     "v1.0.0",
				Environment: "production",
				Logs: DatadogLogs{
					Enabled: true,
					Service: "my-service",
					Source:  "nodejs",
				},
				Metrics: DatadogMetrics{
					Enabled: true,
					Path:    "",
					Port:    0,
				},
			},
			wantErr: false,
		},
		{
			name:        "no datadog annotations",
			annotations: map[string]string{"other": "annotation"},
			want:        nil,
			wantErr:     false,
		},
		{
			name:        "nil annotations",
			annotations: nil,
			want:        nil,
			wantErr:     false,
		},
	}

	cfg := &config.Config{}
	svc := NewManifestService(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.ConvertDatadogAnnotations(tt.annotations)
			
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			
			if tt.want == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.want.Enabled, result.Enabled)
				assert.Equal(t, tt.want.Service, result.Service)
				assert.Equal(t, tt.want.Version, result.Version)
				assert.Equal(t, tt.want.Environment, result.Environment)
				assert.Equal(t, tt.want.Logs.Enabled, result.Logs.Enabled)
				assert.Equal(t, tt.want.Logs.Service, result.Logs.Service)
				assert.Equal(t, tt.want.Metrics.Enabled, result.Metrics.Enabled)
			}
		})
	}
}

func TestManifestService_ExtractManifestValues(t *testing.T) {
	manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: test-app
        image: myregistry/test-app:v1.0.0
        resources:
          limits:
            cpu: 1
            memory: 1Gi
          requests:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
`

	cfg := &config.Config{}
	svc := NewManifestService(cfg)
	
	values, err := svc.ExtractManifestValues(manifest, "test-app")
	require.NoError(t, err)
	require.NotNil(t, values)
	
	// Check image values
	image, ok := values["image"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "myregistry/test-app", image["repository"])
	assert.Equal(t, "v1.0.0", image["tag"])
	
	// Check replica count
	assert.Equal(t, int32(2), values["replicaCount"])
	
	// Check resources
	resources, ok := values["resources"].(map[string]interface{})
	require.True(t, ok)
	limits, ok := resources["limits"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "1", limits["cpu"])
	assert.Equal(t, "1Gi", limits["memory"])
	
	// Check probes
	probes, ok := values["probes"].(map[string]interface{})
	require.True(t, ok)
	liveness, ok := probes["liveness"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, int32(30), liveness["initialDelaySeconds"])
}