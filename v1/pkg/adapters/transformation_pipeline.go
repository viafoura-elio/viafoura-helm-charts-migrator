package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
	"helm-charts-migrator/v1/pkg/yaml"
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

		// Read values
		values, err := tp.file.ReadYAMLToMap(path)
		if err != nil {
			tp.log.Error(err, "Failed to read values file", "path", path)
			return nil // Continue with other files
		}

		// Extract secrets if in envs directory
		if strings.Contains(path, "/envs/") {
			secrets, cleaned := tp.transform.ExtractSecrets(values)

			if len(secrets) > 0 {
				// Save secrets to secrets.dec.yaml
				secretsPath := strings.Replace(path, "values.yaml", "secrets.dec.yaml", 1)

				// Create secrets document with proper structure using yaml package
				if err := tp.saveSecretsDocument(secretsPath, secrets); err != nil {
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
		transformed, err := tp.transform.Transform(values, transformConfig)
		if err != nil {
			tp.log.Error(err, "Failed to transform values", "path", path)
			return nil
		}

		// Save transformed values
		if err := tp.file.WriteYAML(path, transformed); err != nil {
			tp.log.Error(err, "Failed to save transformed values", "path", path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to transform service %s: %w", serviceName, err)
	}

	tp.log.InfoS("Transformed service", "service", serviceName)
	return nil
}

// saveSecretsDocument saves the secrets with proper structure and comments
// This method uses the yaml package to create properly formatted secrets.dec.yaml files
func (tp *TransformationPipeline) saveSecretsDocument(path string, secrets map[string]interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate appropriate comment based on path hierarchy
	headComment := yaml.SecretsHeadComment(path)

	// Create a document to use the NewSecretsDecYaml method
	doc, err := yaml.FromMap(map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	// Create the secrets YAML node with proper structure and comments
	secretsNode := doc.NewSecretsDecYaml(headComment, secrets)

	// Marshal the node to YAML
	yamlData, err := yaml.MarshalNode(&secretsNode)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Write the YAML data to file
	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	return nil
}
