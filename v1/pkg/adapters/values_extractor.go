package adapters

import (
	"fmt"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/release"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/keycase"
	"helm-charts-migrator/v1/pkg/logger"
	"helm-charts-migrator/v1/pkg/services"
	"helm-charts-migrator/v1/pkg/yaml"
)

// ValuesExtractor handles extracting values from releases
type ValuesExtractor interface {
	ExtractLegacyHelmValues(serviceRelease *release.Release, outputPath string) error
	ExtractLegacySourceValues(sourcePath, serviceName, outputPath string) error
}

// valuesExtractor implements ValuesExtractor
type valuesExtractor struct {
	config    config.ConverterConfig
	transform services.TransformationService
	log       *logger.NamedLogger
}

// NewValuesExtractor creates a new ValuesExtractor
func NewValuesExtractor(cfg *config.Config, transform services.TransformationService) ValuesExtractor {
	return &valuesExtractor{
		config:    cfg.Globals.Converter,
		transform: transform,
		log:       logger.WithName("values-extractor"),
	}
}

// ExtractLegacyHelmValues extracts and converts values from a Helm release
func (v *valuesExtractor) ExtractLegacyHelmValues(serviceRelease *release.Release, outputPath string) error {
	if serviceRelease == nil || serviceRelease.Config == nil {
		return fmt.Errorf("no values to extract")
	}

	// Convert to camelCase
	values := serviceRelease.Config
	convertedValues := v.convertKeys(values)

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save as YAML using centralized yaml package
	doc, err := yaml.FromMap(convertedValues)
	if err != nil {
		return fmt.Errorf("failed to create YAML document: %w", err)
	}

	if err := doc.SaveFile(outputPath, nil); err != nil {
		return fmt.Errorf("failed to write values: %w", err)
	}

	v.log.InfoS("Extracted legacy Helm values", "path", outputPath)
	return nil
}

// ExtractLegacySourceValues extracts values from source path
func (v *valuesExtractor) ExtractLegacySourceValues(sourcePath, serviceName, outputPath string) error {
	sourceFile := filepath.Join(sourcePath, serviceName, "values.yaml")

	// Check if source file exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		v.log.V(2).InfoS("Source values file not found", "path", sourceFile)
		return nil
	}

	// Read source values using centralized yaml package
	doc, err := yaml.LoadFile(sourceFile, nil)
	if err != nil {
		return fmt.Errorf("failed to read source values: %w", err)
	}

	values, err := doc.ToMap()
	if err != nil {
		return fmt.Errorf("failed to convert YAML to map: %w", err)
	}

	// Convert to camelCase
	convertedValues := v.convertKeys(values)

	// Save to output path using centralized yaml package
	outDoc, err := yaml.FromMap(convertedValues)
	if err != nil {
		return fmt.Errorf("failed to create YAML document: %w", err)
	}

	if err := outDoc.SaveFile(outputPath, nil); err != nil {
		return fmt.Errorf("failed to write values: %w", err)
	}

	v.log.InfoS("Extracted legacy source values", "path", outputPath)
	return nil
}

// convertKeys converts keys to camelCase based on configuration
func (v *valuesExtractor) convertKeys(values map[string]interface{}) map[string]interface{} {
	converter := keycase.NewConverter()
	
	converter.SkipJavaProperties = v.config.SkipJavaProperties
	converter.SkipUppercaseKeys = v.config.SkipUppercaseKeys
	converter.MinUppercaseChars = v.config.MinUppercaseChars
	
	return converter.ConvertMap(values)
}