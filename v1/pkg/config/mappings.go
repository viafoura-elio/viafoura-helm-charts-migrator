package config

// Mappings represents all mapping configurations
type Mappings struct {
	Locations  *MappingLocations `yaml:"locations,omitempty"`
	Normalizer *Normalizer       `yaml:"normalizer,omitempty"`
	Transform  *Transform        `yaml:"transform,omitempty"`
	Extract    *Extract          `yaml:"extract,omitempty"`
	Cleaner    *Cleaner          `yaml:"cleaner,omitempty"`
}

// MappingLocations defines location scanning configuration
type MappingLocations struct {
	ScanMode ScanMode `yaml:"scan_mode,omitempty"`
	Include  []string `yaml:"include,omitempty"`
}

// Normalizer represents key normalization configuration
type Normalizer struct {
	Enabled     bool              `yaml:"enabled"`
	Description string            `yaml:"description"`
	Patterns    map[string]string `yaml:"patterns"` // old_pattern -> new_pattern
}

// Transform represents transformation configuration
type Transform struct {
	Enabled     bool                     `yaml:"enabled"`
	Description string                   `yaml:"description"`
	Rules       map[string]TransformRule `yaml:"rules"`
}

// TransformRule represents a single transformation rule
type TransformRule struct {
	Type           string `yaml:"type"`
	SourcePath     string `yaml:"source_path"`
	TargetPath     string `yaml:"target_path"`
	KeepLegacyKeys bool   `yaml:"keep_legacy_keys"`
	Description    string `yaml:"description"`
}

// Extract represents extraction configuration
type Extract struct {
	Enabled           bool                     `yaml:"enabled"`
	Description       string                   `yaml:"description"`
	Patterns          map[string]string        `yaml:"patterns"`
	ServicePorts      *ServicePortsConfig      `yaml:"service_ports,omitempty"`
	ManifestResources *ManifestResourcesConfig `yaml:"manifest_resources,omitempty"`
}

// ServicePortsConfig represents Service port extraction configuration
type ServicePortsConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Description       string `yaml:"description"`
	PreferServiceSpec bool   `yaml:"prefer_service_spec"`
}

// ManifestResourcesConfig represents abstract manifest extraction configuration
type ManifestResourcesConfig struct {
	Enabled            bool                     `yaml:"enabled"`
	Description        string                   `yaml:"description"`
	ConsolidatedOutput *ConsolidatedOutput      `yaml:"consolidated_output,omitempty"`
	Rules              []ManifestExtractionRule `yaml:"rules,omitempty"`
}

// ConsolidatedOutput configuration for saving extracted data
type ConsolidatedOutput struct {
	Enabled  bool   `yaml:"enabled"`
	Filename string `yaml:"filename"`
}

// ManifestExtractionRule defines extraction rules for a specific Kubernetes kind
type ManifestExtractionRule struct {
	Kind        string           `yaml:"kind"`
	Enabled     bool             `yaml:"enabled"`
	Description string           `yaml:"description"`
	Extractions []ExtractionSpec `yaml:"extractions"`
}

// ExtractionSpec defines a single extraction from source to target
type ExtractionSpec struct {
	Source      string `yaml:"source"` // JSONPath or field path in the resource
	Target      string `yaml:"target"` // Target path in values.yaml
	Description string `yaml:"description,omitempty"`
	Type        string `yaml:"type,omitempty"`   // array_collect, array_flatten, array_append
	Filter      string `yaml:"filter,omitempty"` // Regex filter for values
	Merge       bool   `yaml:"merge,omitempty"`  // Merge with existing values
}

// Cleaner represents cleaning configuration for removing unwanted keys
type Cleaner struct {
	Enabled      bool     `yaml:"enabled"`
	Description  string   `yaml:"description"`
	PathPatterns []string `yaml:"path_patterns"` // File path patterns to apply cleaning
	KeyPatterns  []string `yaml:"key_patterns"`  // Key patterns to remove
}
