package config

// Secrets represents secret patterns and detection configuration
type Secrets struct {
	Locations   *SecretLocations `yaml:"locations,omitempty"`
	Patterns    []string         `yaml:"patterns"`
	UUIDs       []UUIDPattern    `yaml:"uuids,omitempty"`
	Keys        []string         `yaml:"keys,omitempty"`
	Values      []ValuePattern   `yaml:"values,omitempty"`
	Description string           `yaml:"description,omitempty"`

	// Merging contains merge strategy configuration for specific target files
	Merging map[string]*MergeStrategy `yaml:"merging,omitempty"`

	// Deprecated: Use Merging instead
	KeyMappings map[string]string `yaml:"keyMappings,omitempty"`
	// Deprecated: Use Merging instead
	MergeOrder []string `yaml:"mergeOrder,omitempty"`
}

// SecretLocations defines where to look for secrets in the YAML structure
type SecretLocations struct {
	// Base path where secrets are typically found (default: "configMap")
	BasePath string `yaml:"base_path,omitempty"`

	// Store path where extracted secrets should be placed (default: "secrets")
	StorePath string `yaml:"store_path,omitempty"`

	// Additional exact paths to check for secrets (e.g., "auth.password", "configMap.data")
	AdditionalPaths []string `yaml:"additional_paths,omitempty"`

	// Path patterns for flexible targeting (e.g., "secrets.*", "*.password")
	PathPatterns []string `yaml:"path_patterns,omitempty"`

	// Include only these paths (if specified, only these paths are checked)
	Include []string `yaml:"include,omitempty"`

	// Exclude these paths from secret detection
	Exclude []string `yaml:"exclude,omitempty"`

	// Whether to scan all keys (default behavior) or only specified locations
	ScanMode ScanMode `yaml:"scan_mode,omitempty"`
}

// ScanMode defines how the secret scanner should operate
type ScanMode string

const (
	ScanModeAll      ScanMode = "all"      // Scan all keys (default)
	ScanModeTargeted ScanMode = "targeted" // Only scan specified locations
	ScanModeFiltered ScanMode = "filtered" // Scan all but respect include/exclude
)

// UUIDPattern represents UUID-based secret detection rules
type UUIDPattern struct {
	Pattern     string `yaml:"pattern"`
	Sensitive   bool   `yaml:"sensitive"`
	Description string `yaml:"description"`
}

// ValuePattern represents value-based secret detection rules
type ValuePattern struct {
	Pattern     string `yaml:"pattern"`
	Sensitive   bool   `yaml:"sensitive"`
	Description string `yaml:"description"`
}

// MergeStrategy defines the merge strategy for a specific target file pattern
type MergeStrategy struct {
	// KeyMappings maps configMap keys to secret keys
	// e.g., "configMap.application.properties" -> "secrets.application.conf"
	KeyMappings map[string]string `yaml:"keyMappings,omitempty"`

	// MergeOrder defines the order in which files are merged
	// Later files override earlier ones
	// Supports {service}, {cluster}, {namespace} placeholders
	MergeOrder []string `yaml:"mergeOrder,omitempty"`
}
