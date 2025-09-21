package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	yamlv3 "github.com/elioetibr/yaml"
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

		// Read values as Document to preserve comments
		doc, err := tp.file.ReadYAML(path)
		if err != nil {
			tp.log.Error(err, "Failed to read values file", "path", path)
			return nil // Continue with other files
		}

		// Get values as map for transformation
		values, err := doc.ToMap()
		if err != nil {
			tp.log.Error(err, "Failed to convert document to map", "path", path)
			return nil
		}

		// Store original values for comparison
		helmValues := values

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

				// Update values for transformation
				if len(secrets) > 0 {
					helmValues = cleaned
				}
			}
		}

		// Apply transformations
		transformConfig := services.TransformConfig{
			ServiceName: serviceName,
		}
		transformed, err := tp.transform.Transform(helmValues, transformConfig)
		if err != nil {
			tp.log.Error(err, "Failed to transform values", "path", path)
			return nil
		}

		// Check if transformations were actually applied
		if !mapsEqual(helmValues, transformed) {
			// Transformations were applied - update the original document in-place
			tp.log.V(2).InfoS("Applying transformations while preserving comments and structure", "path", path)

			// Save the mapped values for debugging (as requested by user) using yaml wrapper for proper formatting
			debugPath := strings.Replace(path, "values.yaml", "mapped-helm-values.yaml", 1)
			if debugData, err := yaml.Marshal(transformed); err != nil {
				tp.log.V(3).InfoS("Failed to marshal debug mapped values", "path", debugPath, "error", err)
			} else if err := os.WriteFile(debugPath, debugData, 0644); err != nil {
				tp.log.V(3).InfoS("Failed to save debug mapped values", "path", debugPath, "error", err)
			}

			// Update the document by iterating through transformed values
			// This preserves the original YAML structure and comments
			if err := tp.updateDocumentValues(doc, transformed); err != nil {
				tp.log.Error(err, "Failed to update document values", "path", path)
				// Fall back to using yaml wrapper to write with proper formatting
				if err := tp.file.WriteYAML(path, transformed); err != nil {
					tp.log.Error(err, "Failed to save transformed values", "path", path)
				}
			} else {
				// Save the updated document with preserved comments and structure using yaml wrapper
				if err := doc.SaveFile(path, yaml.DefaultOptions()); err != nil {
					tp.log.Error(err, "Failed to save document with comments", "path", path)
				} else {
					tp.log.V(2).InfoS("Saved transformed values with comments and structure preserved", "path", path)
				}
			}
		} else {
			// No transformations applied, preserve original file with comments
			tp.log.V(3).InfoS("No transformations applied, preserving original file", "path", path)
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

// updateDocumentValues updates the values in a document while preserving structure and comments
// It iterates through the transformed map and updates only the values that have changed
func (tp *TransformationPipeline) updateDocumentValues(doc *yaml.Document, transformed map[string]interface{}) error {
	// Get the root node of the document
	node := doc.GetNode()
	if node == nil || len(node.Content) == 0 {
		return fmt.Errorf("invalid document structure")
	}

	// The first content node should be the root mapping
	rootNode := node.Content[0]
	if rootNode.Kind != yamlv3.MappingNode {
		return fmt.Errorf("expected mapping node at root")
	}

	// Update values recursively
	return tp.updateNodeValues(rootNode, transformed)
}

// updateNodeValues recursively updates values in a YAML node from a map
func (tp *TransformationPipeline) updateNodeValues(node *yamlv3.Node, values map[string]interface{}) error {
	if node.Kind != yamlv3.MappingNode {
		return nil
	}

	// Process key-value pairs (Content has alternating keys and values)
	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		// Get the key
		key := keyNode.Value

		// Check if this key exists in the transformed values
		if newValue, exists := values[key]; exists {
			// Update the value based on its type
			if err := tp.updateNodeValue(valueNode, newValue); err != nil {
				tp.log.V(3).InfoS("Failed to update node value", "key", key, "error", err)
			}
		}
	}

	return nil
}

// updateNodeValue updates a single node's value while preserving its structure
func (tp *TransformationPipeline) updateNodeValue(node *yamlv3.Node, value interface{}) error {
	if value == nil {
		// Only update the value for null, preserve everything else
		if node.Kind == yamlv3.ScalarNode {
			node.Value = "null"
			node.Tag = "!!null"
		}
		return nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Recursively update nested maps
		if node.Kind == yamlv3.MappingNode {
			return tp.updateNodeValues(node, v)
		}
		// Type mismatch - skip this update to preserve structure
		tp.log.V(3).InfoS("Type mismatch: expected mapping node", "actualKind", node.Kind)
		return nil

	case []interface{}:
		// Update sequence nodes
		if node.Kind != yamlv3.SequenceNode {
			// Type mismatch - skip this update
			tp.log.V(3).InfoS("Type mismatch: expected sequence node", "actualKind", node.Kind)
			return nil
		}
		// For sequences, update each item
		for i, item := range v {
			if i < len(node.Content) {
				if err := tp.updateNodeValue(node.Content[i], item); err != nil {
					tp.log.V(3).InfoS("Failed to update sequence item", "index", i, "error", err)
				}
			}
		}

	case string:
		// Only update scalar nodes
		if node.Kind == yamlv3.ScalarNode {
			// Just update the value, preserve all other properties
			node.Value = v
			// Only set tag if it's empty
			if node.Tag == "" {
				node.Tag = "!!str"
			}
		}

	case bool:
		// Only update scalar nodes
		if node.Kind == yamlv3.ScalarNode {
			if v {
				node.Value = "true"
			} else {
				node.Value = "false"
			}
			// Only set tag if empty
			if node.Tag == "" {
				node.Tag = "!!bool"
			}
		}

	case int, int32, int64:
		// Only update scalar nodes
		if node.Kind == yamlv3.ScalarNode {
			node.Value = fmt.Sprintf("%d", v)
			if node.Tag == "" {
				node.Tag = "!!int"
			}
		}

	case float32, float64:
		// Only update scalar nodes
		if node.Kind == yamlv3.ScalarNode {
			node.Value = fmt.Sprintf("%v", v)
			if node.Tag == "" {
				node.Tag = "!!float"
			}
		}

	default:
		// For any other type, convert to string
		if node.Kind == yamlv3.ScalarNode {
			node.Value = fmt.Sprintf("%v", v)
		}
	}

	return nil
}

// mapsEqual compares two maps for equality
func mapsEqual(a, b map[string]interface{}) bool {
	return reflect.DeepEqual(a, b)
}
