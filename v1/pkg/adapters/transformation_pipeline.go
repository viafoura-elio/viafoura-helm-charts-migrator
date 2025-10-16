package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
	yaml "github.com/elioetibr/golang-yaml-advanced"
)

// TransformationPipeline handles the transformation process
type TransformationPipeline struct {
	config    *config.Config
	file      services.FileService
	transform services.TransformationService
	log       *logger.NamedLogger
}

// NewTransformationPipeline creates a new TransformationPipeline
func NewTransformationPipeline(cfg *config.Config, file services.FileService, transform services.TransformationService) *TransformationPipeline {
	return &TransformationPipeline{
		config:    cfg,
		file:      file,
		transform: transform,
		log:       logger.WithName("transformation-pipeline"),
	}
}

// TransformService transforms all values files for a service
func (tp *TransformationPipeline) TransformService(serviceName string) error {
	paths := config.NewPaths("", "apps", ".cache").ForService(serviceName)
	serviceDir := paths.ServiceDir()

	// Process all values.yaml files in the service directory
	err := filepath.Walk(serviceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, "values.yaml") {
			return nil
		}

		// Skip certain files
		if strings.Contains(path, "helm-values.yaml") ||
			strings.Contains(path, "legacy-values.yaml") {
			return nil
		}

		// Read values as map for transformation
		baseChartValuesTree, err := tp.file.ReadYAML(path)
		if err != nil {
			tp.log.Error(err, "Failed to read values file", "path", path)
			return nil // Continue with other files
		}

		// Convert to map for processing
		baseChartYamlValuesBytes, err := baseChartValuesTree.ToYAML()
		if err != nil {
			tp.log.Error(err, "Failed to convert NodeTree to YAML", "path", path)
			return nil
		}

		var values map[string]interface{}
		if err := yaml.Unmarshal(baseChartYamlValuesBytes, &values); err != nil {
			tp.log.Error(err, "Failed to unmarshal YAML to map", "path", path)
			return nil
		}

		// Skip transformation for empty files (only comments)
		if len(values) == 0 {
			tp.log.V(3).InfoS("Skipping empty values file (comments only)", "path", path)
			return nil
		}

		// Extract secrets if in envs directory
		if strings.Contains(path, "/envs/") {
			secrets, cleaned := tp.transform.ExtractSecrets(values)

			if len(secrets) > 0 {
				// Save secrets to secrets.dec.yaml
				secretsPath := strings.Replace(path, "values.yaml", "secrets.dec.yaml", 1)

				// Create secrets document with proper structure
				secretsDoc := tp.createSecretsDocument(path, secrets)

				if err := tp.saveSecretsFile(secretsPath, secretsDoc); err != nil {
					tp.log.Error(err, "Failed to save secrets file", "path", secretsPath)
				} else {
					tp.log.V(2).InfoS("Saved secrets file", "path", secretsPath)
				}

				// Update values file with cleaned values
				values = cleaned
			}
		}

		// Apply transformations
		transformConfig := services.TransformConfig{
			ServiceName: serviceName,
		}
		transformedDataMap, err := tp.transform.Transform(values, transformConfig)
		if err != nil {
			tp.log.Error(err, "Failed to transform values", "path", path)
			return nil
		}

		// Convert transformed map back to YAML bytes, then to NodeTree to preserve structure
		transformedYAMLBytes, err := yaml.Marshal(transformedDataMap)
		if err != nil {
			tp.log.Error(err, "Failed to marshal transformed data", "path", path)
			return nil
		}

		// Parse transformed data as NodeTree
		overrideTree, err := yaml.UnmarshalYAML(transformedYAMLBytes)
		if err != nil {
			tp.log.Error(err, "Failed to parse transformed YAML as NodeTree", "path", path)
			return nil
		}

		// Merge the trees (override takes precedence, preserving comments from base)
		mergedTree := yaml.MergeTrees(baseChartValuesTree, overrideTree)

		// Convert merged tree to YAML
		mergedYAML, err := mergedTree.ToYAML()
		if err != nil {
			tp.log.Error(err, "Failed to convert merged tree to YAML", "path", path)
			return nil
		}

		// Write merged result back to the file (preserving comments)
		if err := os.WriteFile(path, mergedYAML, 0644); err != nil {
			tp.log.Error(err, "Failed to write merged YAML file", "path", path)
			return nil
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to transform service %s: %w", serviceName, err)
	}

	tp.log.InfoS("Transformed service", "service", serviceName)
	return nil
}

// createSecretsDocument creates a properly formatted secrets document
func (tp *TransformationPipeline) createSecretsDocument(path string, secrets map[string]interface{}) map[string]interface{} {
	// For now, just return the secrets map
	// TODO: Add proper document structure with comments when needed
	return map[string]interface{}{
		"secrets": secrets,
	}
}

// secretsHeadComment generates hierarchical comment based on path
func (tp *TransformationPipeline) secretsHeadComment(path string) string {
	parts := strings.Split(path, "/")
	depth := 0

	for i, part := range parts {
		if part == "envs" && i+1 < len(parts) {
			depth = len(parts) - i - 2
			break
		}
	}

	switch depth {
	case 0: // cluster level
		return "# Placeholder to cluster secrets level.\n# Using Hierarchical configurations it will be cascaded to lower secrets.\n# It can override any previous secrets.dec.yaml file."
	case 1: // environment level
		return "# Placeholder to environment secrets level.\n# Using Hierarchical configurations it will be cascaded to lower secrets.\n# It can override any previous secrets.dec.yaml file."
	case 2: // namespace level
		return "# Placeholder to namespace secrets level.\n# Using Hierarchical configurations it will be cascaded to lower secrets.\n# It can override any previous secrets.dec.yaml file."
	default:
		return "# Placeholder for secrets.\n# Using Hierarchical configurations it will be cascaded to lower secrets.\n# It can override any previous secrets.dec.yaml file."
	}
}

// saveSecretsFile saves the secrets document to a file
func (tp *TransformationPipeline) saveSecretsFile(path string, doc map[string]interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save as YAML
	return tp.file.WriteYAML(path, doc)
}
