package services

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"

	"helm-charts-migrator/v1/pkg/logger"
)

// helmService implements HelmService interface
type helmService struct {
	log *logger.NamedLogger
}

// NewHelmService creates a new HelmService
func NewHelmService() HelmService {
	return &helmService{
		log: logger.WithName("helm-service"),
	}
}

// GetReleaseByName finds a release by service name from a list of releases
func (h *helmService) GetReleaseByName(serviceName string, releases []*release.Release) *release.Release {
	for _, rel := range releases {
		if rel.Name == serviceName {
			return rel
		}
	}
	
	// Try with -v2 suffix for canary deployments
	canaryName := serviceName + "-v2"
	for _, rel := range releases {
		if rel.Name == canaryName {
			h.log.V(2).InfoS("Found canary release", "service", serviceName, "release", canaryName)
			return rel
		}
	}
	
	return nil
}

// ExtractValues extracts values from a Helm release
func (h *helmService) ExtractValues(release *release.Release) (map[string]interface{}, error) {
	if release == nil {
		return nil, fmt.Errorf("release is nil")
	}
	
	if release.Config == nil {
		return make(map[string]interface{}), nil
	}
	
	// The release.Config is already a map[string]interface{}
	values := release.Config
	
	return values, nil
}

// ExtractManifest extracts and processes the manifest from a release
func (h *helmService) ExtractManifest(release *release.Release) (string, error) {
	if release == nil || release.Manifest == "" {
		return "", nil
	}
	
	// Process the manifest to extract relevant resources
	manifest := release.Manifest
	
	// Split manifest into resources
	resources := strings.Split(manifest, "---\n")
	var processedResources []string
	
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		if resource == "" {
			continue
		}
		
		// Check if this is a relevant resource type
		if h.isRelevantResource(resource) {
			processedResources = append(processedResources, resource)
		}
	}
	
	if len(processedResources) == 0 {
		return "", nil
	}
	
	return strings.Join(processedResources, "---\n"), nil
}

// ValidateChart validates a Helm chart
func (h *helmService) ValidateChart(chartPath string) error {
	// Load the chart to validate its structure
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}
	
	// Validate chart metadata
	if chart.Metadata == nil {
		return fmt.Errorf("chart metadata is missing")
	}
	
	if chart.Metadata.Name == "" {
		return fmt.Errorf("chart name is missing")
	}
	
	if chart.Metadata.Version == "" {
		return fmt.Errorf("chart version is missing")
	}
	
	h.log.V(2).InfoS("Chart validated successfully", 
		"path", chartPath, 
		"name", chart.Metadata.Name,
		"version", chart.Metadata.Version)
	
	return nil
}

// isRelevantResource checks if a resource is relevant for extraction
func (h *helmService) isRelevantResource(resource string) bool {
	// Check for common Kubernetes resource types we want to keep
	relevantKinds := []string{
		"kind: Deployment",
		"kind: Service",
		"kind: ConfigMap",
		"kind: Secret",
		"kind: Ingress",
		"kind: StatefulSet",
		"kind: DaemonSet",
		"kind: Job",
		"kind: CronJob",
		"kind: ServiceAccount",
		"kind: Role",
		"kind: RoleBinding",
		"kind: ClusterRole",
		"kind: ClusterRoleBinding",
		"kind: PersistentVolumeClaim",
		"kind: HorizontalPodAutoscaler",
		"kind: NetworkPolicy",
	}
	
	for _, kind := range relevantKinds {
		if strings.Contains(resource, kind) {
			return true
		}
	}
	
	return false
}