package services

import (
	"fmt"
	"strings"

	"github.com/elioetibr/yaml"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
)

// ManifestService handles extraction and conversion of Kubernetes manifests
type ManifestService interface {
	ExtractDeployment(manifest string) (*DeploymentConfig, error)
	ConvertDatadogAnnotations(annotations map[string]string) (*DatadogConfig, error)
	ExtractProbes(container *v1.Container) (*ProbeConfig, error)
	ExtractManifestValues(manifest string, serviceName string) (map[string]interface{}, error)
}

// DeploymentConfig represents extracted deployment configuration
type DeploymentConfig struct {
	Image       ImageConfig
	Probes      ProbeConfig
	Resources   ResourceConfig
	Datadog     DatadogConfig
	Environment []EnvVar
	Replicas    int32
	Labels      map[string]string
	Annotations map[string]string
}

// ImageConfig represents container image configuration
type ImageConfig struct {
	Repository string
	Tag        string
	PullPolicy string
}

// ProbeConfig represents health check probes
type ProbeConfig struct {
	Liveness  *Probe
	Readiness *Probe
	Startup   *Probe
}

// Probe represents a single health check probe
type Probe struct {
	HTTPGet             *HTTPGetAction
	TCPSocket           *TCPSocketAction
	Exec                *ExecAction
	InitialDelaySeconds int32
	TimeoutSeconds      int32
	PeriodSeconds       int32
	SuccessThreshold    int32
	FailureThreshold    int32
}

// HTTPGetAction describes an action based on HTTP Get requests
type HTTPGetAction struct {
	Path   string
	Port   intstr.IntOrString
	Scheme string
}

// TCPSocketAction describes an action based on opening a socket
type TCPSocketAction struct {
	Port intstr.IntOrString
}

// ExecAction describes a "run in container" action
type ExecAction struct {
	Command []string
}

// ResourceConfig represents resource limits and requests
type ResourceConfig struct {
	Limits   ResourceList
	Requests ResourceList
}

// ResourceList represents CPU and memory resources
type ResourceList struct {
	CPU    string
	Memory string
}

// DatadogConfig represents Datadog monitoring configuration
type DatadogConfig struct {
	Enabled     bool
	Service     string
	Version     string
	Environment string
	Logs        DatadogLogs
	APM         DatadogAPM
	Metrics     DatadogMetrics
}

// DatadogLogs represents Datadog log configuration
type DatadogLogs struct {
	Enabled bool
	Service string
	Source  string
}

// DatadogAPM represents Datadog APM configuration
type DatadogAPM struct {
	Enabled bool
	Service string
}

// DatadogMetrics represents Datadog metrics configuration
type DatadogMetrics struct {
	Enabled bool
	Port    int
	Path    string
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name      string
	Value     string
	ValueFrom *EnvVarSource
}

// EnvVarSource represents a source for the environment variable's value
type EnvVarSource struct {
	SecretKeyRef    *SecretKeySelector
	ConfigMapKeyRef *ConfigMapKeySelector
}

// SecretKeySelector selects a key of a Secret
type SecretKeySelector struct {
	Name string
	Key  string
}

// ConfigMapKeySelector selects a key of a ConfigMap
type ConfigMapKeySelector struct {
	Name string
	Key  string
}

// manifestService implements ManifestService
type manifestService struct {
	config *config.Config
	log    *logger.NamedLogger
}

// NewManifestService creates a new ManifestService
func NewManifestService(cfg *config.Config) ManifestService {
	return &manifestService{
		config: cfg,
		log:    logger.WithName("manifest-service"),
	}
}

// ExtractDeployment extracts deployment configuration from a manifest
func (m *manifestService) ExtractDeployment(manifest string) (*DeploymentConfig, error) {
	var deployment appsv1.Deployment
	if err := yaml.Unmarshal([]byte(manifest), &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	if deployment.Kind != "Deployment" {
		return nil, fmt.Errorf("manifest is not a Deployment, got %s", deployment.Kind)
	}

	config := &DeploymentConfig{
		Replicas:    *deployment.Spec.Replicas,
		Labels:      deployment.Spec.Template.Labels,
		Annotations: deployment.Spec.Template.Annotations,
	}

	// Extract container configuration (assuming single container for simplicity)
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		container := deployment.Spec.Template.Spec.Containers[0]

		// Extract image
		parts := strings.Split(container.Image, ":")
		if len(parts) == 2 {
			config.Image = ImageConfig{
				Repository: parts[0],
				Tag:        parts[1],
				PullPolicy: string(container.ImagePullPolicy),
			}
		} else {
			config.Image = ImageConfig{
				Repository: container.Image,
				Tag:        "latest",
				PullPolicy: string(container.ImagePullPolicy),
			}
		}

		// Extract probes
		probes, err := m.ExtractProbes(&container)
		if err != nil {
			m.log.V(2).InfoS("Failed to extract probes", "error", err)
		} else {
			config.Probes = *probes
		}

		// Extract resources
		config.Resources = m.extractResources(&container)

		// Extract environment variables
		config.Environment = m.extractEnvVars(container.Env)
	}

	// Convert Datadog annotations
	if deployment.Spec.Template.Annotations != nil {
		datadog, err := m.ConvertDatadogAnnotations(deployment.Spec.Template.Annotations)
		if err != nil {
			m.log.V(2).InfoS("Failed to convert Datadog annotations", "error", err)
		} else if datadog != nil {
			config.Datadog = *datadog
		}
	}

	return config, nil
}

// ConvertDatadogAnnotations converts Datadog v1 annotations to v2 format
func (m *manifestService) ConvertDatadogAnnotations(annotations map[string]string) (*DatadogConfig, error) {
	if annotations == nil {
		return nil, nil
	}

	config := &DatadogConfig{}
	hasDatadog := false

	// Check for Datadog annotations
	for key, value := range annotations {
		if strings.HasPrefix(key, "ad.datadoghq.com/") {
			hasDatadog = true

			// Parse annotation key
			parts := strings.Split(key, "/")
			if len(parts) < 2 {
				continue
			}

			annotationType := parts[1]

			switch {
			case strings.HasSuffix(annotationType, ".logs"):
				// Parse logs configuration
				var logsConfig []map[string]interface{}
				if err := yaml.Unmarshal([]byte(value), &logsConfig); err == nil && len(logsConfig) > 0 {
					if service, ok := logsConfig[0]["service"].(string); ok {
						config.Logs.Service = service
						config.Logs.Enabled = true
					}
					if source, ok := logsConfig[0]["source"].(string); ok {
						config.Logs.Source = source
					}
				}

			case strings.HasSuffix(annotationType, ".checks"):
				// Parse metrics configuration
				var checksConfig []map[string]interface{}
				if err := yaml.Unmarshal([]byte(value), &checksConfig); err == nil && len(checksConfig) > 0 {
					config.Metrics.Enabled = true
					if path, ok := checksConfig[0]["path"].(string); ok {
						config.Metrics.Path = path
					}
					if port, ok := checksConfig[0]["port"].(int); ok {
						config.Metrics.Port = port
					}
				}
			}
		}
	}

	// Check for standard Datadog tags
	if tags, exists := annotations["tags.datadoghq.com/service"]; exists {
		config.Service = tags
		hasDatadog = true
	}
	if version, exists := annotations["tags.datadoghq.com/version"]; exists {
		config.Version = version
		hasDatadog = true
	}
	if env, exists := annotations["tags.datadoghq.com/env"]; exists {
		config.Environment = env
		hasDatadog = true
	}

	if !hasDatadog {
		return nil, nil
	}

	config.Enabled = true
	return config, nil
}

// ExtractProbes extracts probe configuration from a container
func (m *manifestService) ExtractProbes(container *v1.Container) (*ProbeConfig, error) {
	if container == nil {
		return nil, fmt.Errorf("container is nil")
	}

	config := &ProbeConfig{}

	// Extract liveness probe
	if container.LivenessProbe != nil {
		config.Liveness = m.convertProbe(container.LivenessProbe)
	}

	// Extract readiness probe
	if container.ReadinessProbe != nil {
		config.Readiness = m.convertProbe(container.ReadinessProbe)
	}

	// Extract startup probe
	if container.StartupProbe != nil {
		config.Startup = m.convertProbe(container.StartupProbe)
	}

	return config, nil
}

// ExtractManifestValues extracts values from a manifest for a specific service
func (m *manifestService) ExtractManifestValues(manifest string, serviceName string) (map[string]interface{}, error) {
	deployment, err := m.ExtractDeployment(manifest)
	if err != nil {
		return nil, err
	}

	values := make(map[string]interface{})

	// Add image configuration
	if deployment.Image.Repository != "" {
		values["image"] = map[string]interface{}{
			"repository": deployment.Image.Repository,
			"tag":        deployment.Image.Tag,
			"pullPolicy": deployment.Image.PullPolicy,
		}
	}

	// Add replica count
	values["replicaCount"] = deployment.Replicas

	// Add resources
	if deployment.Resources.Limits.CPU != "" || deployment.Resources.Limits.Memory != "" ||
		deployment.Resources.Requests.CPU != "" || deployment.Resources.Requests.Memory != "" {
		resources := make(map[string]interface{})

		if deployment.Resources.Limits.CPU != "" || deployment.Resources.Limits.Memory != "" {
			limits := make(map[string]interface{})
			if deployment.Resources.Limits.CPU != "" {
				limits["cpu"] = deployment.Resources.Limits.CPU
			}
			if deployment.Resources.Limits.Memory != "" {
				limits["memory"] = deployment.Resources.Limits.Memory
			}
			resources["limits"] = limits
		}

		if deployment.Resources.Requests.CPU != "" || deployment.Resources.Requests.Memory != "" {
			requests := make(map[string]interface{})
			if deployment.Resources.Requests.CPU != "" {
				requests["cpu"] = deployment.Resources.Requests.CPU
			}
			if deployment.Resources.Requests.Memory != "" {
				requests["memory"] = deployment.Resources.Requests.Memory
			}
			resources["requests"] = requests
		}

		values["resources"] = resources
	}

	// Add probes
	probes := make(map[string]interface{})
	if deployment.Probes.Liveness != nil {
		probes["liveness"] = m.probeToMap(deployment.Probes.Liveness)
	}
	if deployment.Probes.Readiness != nil {
		probes["readiness"] = m.probeToMap(deployment.Probes.Readiness)
	}
	if deployment.Probes.Startup != nil {
		probes["startup"] = m.probeToMap(deployment.Probes.Startup)
	}
	if len(probes) > 0 {
		values["probes"] = probes
	}

	// Add Datadog configuration
	if deployment.Datadog.Enabled {
		values["datadog"] = map[string]interface{}{
			"enabled":     deployment.Datadog.Enabled,
			"service":     deployment.Datadog.Service,
			"version":     deployment.Datadog.Version,
			"environment": deployment.Datadog.Environment,
		}

		if deployment.Datadog.Logs.Enabled {
			values["datadog"].(map[string]interface{})["logs"] = map[string]interface{}{
				"enabled": deployment.Datadog.Logs.Enabled,
				"service": deployment.Datadog.Logs.Service,
				"source":  deployment.Datadog.Logs.Source,
			}
		}

		if deployment.Datadog.Metrics.Enabled {
			values["datadog"].(map[string]interface{})["metrics"] = map[string]interface{}{
				"enabled": deployment.Datadog.Metrics.Enabled,
				"port":    deployment.Datadog.Metrics.Port,
				"path":    deployment.Datadog.Metrics.Path,
			}
		}
	}

	// Add environment variables
	if len(deployment.Environment) > 0 {
		envVars := make([]map[string]interface{}, 0)
		for _, env := range deployment.Environment {
			envVar := map[string]interface{}{
				"name":  env.Name,
				"value": env.Value,
			}
			if env.ValueFrom != nil {
				valueFrom := make(map[string]interface{})
				if env.ValueFrom.SecretKeyRef != nil {
					valueFrom["secretKeyRef"] = map[string]interface{}{
						"name": env.ValueFrom.SecretKeyRef.Name,
						"key":  env.ValueFrom.SecretKeyRef.Key,
					}
				}
				if env.ValueFrom.ConfigMapKeyRef != nil {
					valueFrom["configMapKeyRef"] = map[string]interface{}{
						"name": env.ValueFrom.ConfigMapKeyRef.Name,
						"key":  env.ValueFrom.ConfigMapKeyRef.Key,
					}
				}
				envVar["valueFrom"] = valueFrom
				delete(envVar, "value")
			}
			envVars = append(envVars, envVar)
		}
		values["env"] = envVars
	}

	m.log.InfoS("Extracted manifest values", "service", serviceName, "valuesCount", len(values))
	return values, nil
}

// convertProbe converts a Kubernetes probe to our Probe struct
func (m *manifestService) convertProbe(probe *v1.Probe) *Probe {
	if probe == nil {
		return nil
	}

	p := &Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}

	if probe.HTTPGet != nil {
		p.HTTPGet = &HTTPGetAction{
			Path:   probe.HTTPGet.Path,
			Port:   probe.HTTPGet.Port,
			Scheme: string(probe.HTTPGet.Scheme),
		}
	}

	if probe.TCPSocket != nil {
		p.TCPSocket = &TCPSocketAction{
			Port: probe.TCPSocket.Port,
		}
	}

	if probe.Exec != nil {
		p.Exec = &ExecAction{
			Command: probe.Exec.Command,
		}
	}

	return p
}

// probeToMap converts a Probe to a map for YAML output
func (m *manifestService) probeToMap(probe *Probe) map[string]interface{} {
	if probe == nil {
		return nil
	}

	result := map[string]interface{}{
		"initialDelaySeconds": probe.InitialDelaySeconds,
		"timeoutSeconds":      probe.TimeoutSeconds,
		"periodSeconds":       probe.PeriodSeconds,
		"successThreshold":    probe.SuccessThreshold,
		"failureThreshold":    probe.FailureThreshold,
	}

	if probe.HTTPGet != nil {
		result["httpGet"] = map[string]interface{}{
			"path":   probe.HTTPGet.Path,
			"port":   probe.HTTPGet.Port,
			"scheme": probe.HTTPGet.Scheme,
		}
	}

	if probe.TCPSocket != nil {
		result["tcpSocket"] = map[string]interface{}{
			"port": probe.TCPSocket.Port,
		}
	}

	if probe.Exec != nil {
		result["exec"] = map[string]interface{}{
			"command": probe.Exec.Command,
		}
	}

	return result
}

// extractResources extracts resource configuration from a container
func (m *manifestService) extractResources(container *v1.Container) ResourceConfig {
	config := ResourceConfig{}

	if container.Resources.Limits != nil {
		if cpu := container.Resources.Limits.Cpu(); cpu != nil {
			config.Limits.CPU = cpu.String()
		}
		if memory := container.Resources.Limits.Memory(); memory != nil {
			config.Limits.Memory = memory.String()
		}
	}

	if container.Resources.Requests != nil {
		if cpu := container.Resources.Requests.Cpu(); cpu != nil {
			config.Requests.CPU = cpu.String()
		}
		if memory := container.Resources.Requests.Memory(); memory != nil {
			config.Requests.Memory = memory.String()
		}
	}

	return config
}

// extractEnvVars extracts environment variables
func (m *manifestService) extractEnvVars(envVars []v1.EnvVar) []EnvVar {
	result := make([]EnvVar, 0, len(envVars))

	for _, env := range envVars {
		envVar := EnvVar{
			Name:  env.Name,
			Value: env.Value,
		}

		if env.ValueFrom != nil {
			envVar.ValueFrom = &EnvVarSource{}

			if env.ValueFrom.SecretKeyRef != nil {
				envVar.ValueFrom.SecretKeyRef = &SecretKeySelector{
					Name: env.ValueFrom.SecretKeyRef.Name,
					Key:  env.ValueFrom.SecretKeyRef.Key,
				}
			}

			if env.ValueFrom.ConfigMapKeyRef != nil {
				envVar.ValueFrom.ConfigMapKeyRef = &ConfigMapKeySelector{
					Name: env.ValueFrom.ConfigMapKeyRef.Name,
					Key:  env.ValueFrom.ConfigMapKeyRef.Key,
				}
			}
		}

		result = append(result, envVar)
	}

	return result
}
