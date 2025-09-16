package transformers

import (
	"fmt"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/logger"
)

// TransformerFactory creates and configures transformers
type TransformerFactory struct {
	registry *TransformerRegistry
	config   *config.Config
	log      *logger.NamedLogger
}

// NewTransformerFactory creates a new transformer factory
func NewTransformerFactory(cfg *config.Config) *TransformerFactory {
	factory := &TransformerFactory{
		registry: NewTransformerRegistry(),
		config:   cfg,
		log:      logger.WithName("transformer-factory"),
	}

	// Register built-in transformers
	factory.registerBuiltinTransformers()

	return factory
}

// GetRegistry returns the transformer registry
func (f *TransformerFactory) GetRegistry() *TransformerRegistry {
	return f.registry
}

// registerBuiltinTransformers registers all built-in transformers
func (f *TransformerFactory) registerBuiltinTransformers() {
	// Register CamelCase transformer
	if f.config != nil && f.config.Globals.Converter.MinUppercaseChars > 0 {
		camelCase := NewCamelCaseTransformer(
			f.config.Globals.Converter.SkipJavaProperties,
			f.config.Globals.Converter.SkipUppercaseKeys,
			f.config.Globals.Converter.MinUppercaseChars,
		)
		if err := f.registry.Register(camelCase); err != nil {
			f.log.Error(err, "Failed to register camelCase transformer")
		}
	}

	// Register key normalizer if patterns are configured
	if f.config != nil && f.config.Globals.Mappings != nil {
		patterns := f.extractKeyPatterns(f.config.Globals.Mappings)
		if len(patterns) > 0 {
			normalizer := NewKeyNormalizerTransformer(patterns)
			if err := f.registry.Register(normalizer); err != nil {
				f.log.Error(err, "Failed to register key normalizer")
			}
		}
	}

	// Register value defaults transformer
	defaults := f.extractDefaults()
	if len(defaults) > 0 {
		defaultsTransformer := NewValueDefaultsTransformer(defaults)
		if err := f.registry.Register(defaultsTransformer); err != nil {
			f.log.Error(err, "Failed to register defaults transformer")
		}
	}

	// Register path remap transformer
	mappings := f.extractPathMappings()
	if len(mappings) > 0 {
		remapper := NewPathRemapTransformer(mappings)
		if err := f.registry.Register(remapper); err != nil {
			f.log.Error(err, "Failed to register path remapper")
		}
	}

	f.log.InfoS("Registered built-in transformers", "count", f.registry.Count())
}

// CreateTransformer creates a transformer by name with configuration
func (f *TransformerFactory) CreateTransformer(name string, config map[string]interface{}) (Transformer, error) {
	switch name {
	case "camelCase":
		return f.createCamelCaseTransformer(config)
	case "keyNormalizer":
		return f.createKeyNormalizer(config)
	case "valueDefaults":
		return f.createValueDefaults(config)
	case "pathRemap":
		return f.createPathRemap(config)
	default:
		return nil, fmt.Errorf("unknown transformer type: %s", name)
	}
}

// createCamelCaseTransformer creates a configured camelCase transformer
func (f *TransformerFactory) createCamelCaseTransformer(config map[string]interface{}) (Transformer, error) {
	skipJava := false
	skipUpper := false
	minUpper := 3

	if v, ok := config["skipJavaProperties"].(bool); ok {
		skipJava = v
	}
	if v, ok := config["skipUppercaseKeys"].(bool); ok {
		skipUpper = v
	}
	if v, ok := config["minUppercaseChars"].(int); ok {
		minUpper = v
	}

	return NewCamelCaseTransformer(skipJava, skipUpper, minUpper), nil
}

// createKeyNormalizer creates a configured key normalizer
func (f *TransformerFactory) createKeyNormalizer(config map[string]interface{}) (Transformer, error) {
	patterns := make(map[string]string)

	if patternsConfig, ok := config["patterns"].(map[string]interface{}); ok {
		for old, new := range patternsConfig {
			if newStr, ok := new.(string); ok {
				patterns[old] = newStr
			}
		}
	}

	if len(patterns) == 0 {
		return nil, fmt.Errorf("key normalizer requires patterns configuration")
	}

	return NewKeyNormalizerTransformer(patterns), nil
}

// createValueDefaults creates a configured defaults transformer
func (f *TransformerFactory) createValueDefaults(config map[string]interface{}) (Transformer, error) {
	defaults, ok := config["defaults"].(map[string]interface{})
	if !ok || len(defaults) == 0 {
		return nil, fmt.Errorf("value defaults transformer requires defaults configuration")
	}

	return NewValueDefaultsTransformer(defaults), nil
}

// createPathRemap creates a configured path remap transformer
func (f *TransformerFactory) createPathRemap(config map[string]interface{}) (Transformer, error) {
	mappings := make(map[string]string)

	if mappingsConfig, ok := config["mappings"].(map[string]interface{}); ok {
		for source, target := range mappingsConfig {
			if targetStr, ok := target.(string); ok {
				mappings[source] = targetStr
			}
		}
	}

	if len(mappings) == 0 {
		return nil, fmt.Errorf("path remap transformer requires mappings configuration")
	}

	return NewPathRemapTransformer(mappings), nil
}

// RegisterCustom registers a custom transformer
func (f *TransformerFactory) RegisterCustom(transformer Transformer) error {
	return f.registry.Register(transformer)
}

// LoadFromConfig loads transformers from configuration
func (f *TransformerFactory) LoadFromConfig(transformerConfigs []map[string]interface{}) error {
	for _, tConfig := range transformerConfigs {
		name, ok := tConfig["name"].(string)
		if !ok {
			f.log.Error(nil, "Transformer configuration missing name")
			continue
		}

		transformer, err := f.CreateTransformer(name, tConfig)
		if err != nil {
			f.log.Error(err, "Failed to create transformer", "name", name)
			continue
		}

		if err := f.registry.Register(transformer); err != nil {
			f.log.Error(err, "Failed to register transformer", "name", name)
		}
	}

	return nil
}

// Helper methods to extract configuration

func (f *TransformerFactory) extractKeyPatterns(mappings *config.Mappings) map[string]string {
	patterns := make(map[string]string)

	if mappings == nil || mappings.Normalizer == nil {
		return patterns
	}

	// Extract patterns from normalizer configuration
	if mappings.Normalizer.Patterns != nil {
		for oldPattern, newPattern := range mappings.Normalizer.Patterns {
			if oldPattern != "" && newPattern != "" {
				patterns[oldPattern] = newPattern
			}
		}
	}

	return patterns
}

func (f *TransformerFactory) extractDefaults() map[string]interface{} {
	defaults := make(map[string]interface{})

	// Extract default values from configuration
	// This would come from a defaults section in the config
	// For now, return some common defaults
	defaults["replicaCount"] = 1
	defaults["image"] = map[string]interface{}{
		"pullPolicy": "IfNotPresent",
	}
	defaults["service"] = map[string]interface{}{
		"type": "ClusterIP",
		"port": 80,
	}

	return defaults
}

func (f *TransformerFactory) extractPathMappings() map[string]string {
	mappings := make(map[string]string)

	// Extract path mappings from configuration
	// This would come from a mappings section in the config
	// For example, moving deprecated paths to new locations

	return mappings
}

// ApplyServiceTransformers applies transformers configured for a specific service
func (f *TransformerFactory) ApplyServiceTransformers(serviceName string, data interface{}) (interface{}, error) {
	// Check if service has specific transformer configuration
	if f.config == nil || f.config.Services == nil {
		// Use default transformers
		return f.registry.ApplyAll(data)
	}

	service, exists := f.config.Services[serviceName]
	if !exists || !service.Enabled {
		// Use default transformers
		return f.registry.ApplyAll(data)
	}

	// Apply service-specific transformers
	// This could be extended to support service-specific transformer chains
	result := data
	var err error

	// Apply transformers in priority order
	for _, name := range f.registry.ListByPriority() {
		result, err = f.registry.Apply(name, result)
		if err != nil {
			f.log.V(3).InfoS("Transformer skipped",
				"service", serviceName,
				"transformer", name,
				"error", err)
			// Continue with next transformer
			continue
		}
	}

	f.log.InfoS("Applied service transformers",
		"service", serviceName,
		"transformerCount", f.registry.Count())

	return result, nil
}
