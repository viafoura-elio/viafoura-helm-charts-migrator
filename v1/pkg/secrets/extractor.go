package secrets

import (
	"fmt"
	"regexp"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/yaml"
)

// SecretExtractor handles secret detection and classification
type SecretExtractor struct {
	config               *config.Config // Main project config
	globalPatterns       []*regexp.Regexp
	globalUUIDPatterns   []compiledUUIDPattern
	globalValuePatterns  []compiledValuePattern
	globalLocations      *compiledLocations
	servicePatterns      map[string][]compiledServicePattern
	serviceUUIDPatterns  map[string][]compiledUUIDPattern
	serviceValuePatterns map[string][]compiledValuePattern
	serviceLocations     map[string]*compiledLocations
}

// compiledLocations holds compiled location patterns for efficient matching
type compiledLocations struct {
	scanMode        config.ScanMode
	basePath        string           // Base path to prioritize (e.g., "configMap")
	storePath       string           // Store path where secrets are placed (e.g., "secrets")
	additionalPaths map[string]bool  // Additional exact paths for O(1) lookup
	pathPatterns    []*regexp.Regexp // Compiled regex patterns
	include         map[string]bool  // Include paths for O(1) lookup
	includePatterns []*regexp.Regexp // Include patterns
	exclude         map[string]bool  // Exclude paths for O(1) lookup
	excludePatterns []*regexp.Regexp // Exclude patterns
}

type compiledUUIDPattern struct {
	regex       *regexp.Regexp
	sensitive   bool
	description string
}

type compiledValuePattern struct {
	regex       *regexp.Regexp
	sensitive   bool
	description string
}

type compiledServicePattern struct {
	regex       *regexp.Regexp
	pattern     string
	description string
}

// SecretMatch represents a detected secret
type SecretMatch struct {
	Path            string          `yaml:"path"`
	Key             string          `yaml:"key"`
	Value           string          `yaml:"value"`
	MaskedValue     string          `yaml:"masked_value"`
	Classification  Classification  `yaml:"classification"`
	MatchedBy       []MatchReason   `yaml:"matched_by"`
	Confidence      ConfidenceLevel `yaml:"confidence"`
	ServiceSpecific bool            `yaml:"service_specific"`
	Warnings        []string        `yaml:"warnings,omitempty"`
}

// MatchReason explains why a value was classified as a secret
type MatchReason struct {
	Type        MatchType       `yaml:"type"`
	Pattern     string          `yaml:"pattern"`
	Description string          `yaml:"description"`
	Confidence  ConfidenceLevel `yaml:"confidence"`
}

// Classification represents the type of secret detected
type Classification string

const (
	ClassificationPassword Classification = "password"
	ClassificationAPIKey   Classification = "api_key"
	ClassificationToken    Classification = "token"
	ClassificationSecret   Classification = "secret"
	ClassificationJWT      Classification = "jwt"
	ClassificationUUID     Classification = "uuid"
	ClassificationBase64   Classification = "base64"
	ClassificationHex      Classification = "hex"
	ClassificationUnknown  Classification = "unknown"
)

// MatchType represents how the secret was detected
type MatchType string

const (
	MatchTypeKeyPattern     MatchType = "key_pattern"
	MatchTypeValuePattern   MatchType = "value_pattern"
	MatchTypeUUIDPattern    MatchType = "uuid_pattern"
	MatchTypeServicePattern MatchType = "service_pattern"
	MatchTypeExactKey       MatchType = "exact_key"
)

// ConfidenceLevel represents confidence in secret detection
type ConfidenceLevel string

const (
	ConfidenceHigh   ConfidenceLevel = "high"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceLow    ConfidenceLevel = "low"
)

// ExtractionResult contains the results of secret extraction
type ExtractionResult struct {
	Secrets     []SecretMatch `yaml:"secrets"`
	Summary     Summary       `yaml:"summary"`
	Warnings    []string      `yaml:"warnings"`
	ServiceName string        `yaml:"service_name,omitempty"`
	SourceFile  string        `yaml:"source_file,omitempty"`
}

// Summary provides statistics about extracted secrets
type Summary struct {
	TotalSecrets     int                     `yaml:"total_secrets"`
	ByClassification map[Classification]int  `yaml:"by_classification"`
	ByConfidence     map[ConfidenceLevel]int `yaml:"by_confidence"`
	ServiceSpecific  int                     `yaml:"service_specific"`
	GloballyDetected int                     `yaml:"globally_detected"`
}

// NewFromMainConfig creates a SecretExtractor from the main project config
func NewFromMainConfig(mainConfig *config.Config) (*SecretExtractor, error) {
	if mainConfig == nil {
		return nil, fmt.Errorf("main config cannot be nil")
	}

	// Parse the secrets configuration from the main config
	secretsConfig, err := parseSecretsConfig(mainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secrets config: %w", err)
	}

	return New(secretsConfig)
}

// parseSecretsConfig extracts the secrets configuration from the main config
// This function is now simplified since we use config.Config directly
func parseSecretsConfig(mainConfig *config.Config) (*config.Config, error) {
	// Just return the main config directly since we now use it as-is
	return mainConfig, nil
}

// New creates a new SecretExtractor from the main project config
func New(cfg *config.Config) (*SecretExtractor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	extractor := &SecretExtractor{
		config:               cfg,
		globalPatterns:       make([]*regexp.Regexp, 0),
		globalUUIDPatterns:   make([]compiledUUIDPattern, 0),
		globalValuePatterns:  make([]compiledValuePattern, 0),
		servicePatterns:      make(map[string][]compiledServicePattern),
		serviceUUIDPatterns:  make(map[string][]compiledUUIDPattern),
		serviceValuePatterns: make(map[string][]compiledValuePattern),
		serviceLocations:     make(map[string]*compiledLocations),
	}

	// Compile global key patterns
	if cfg.Globals.Secrets != nil {
		for _, pattern := range cfg.Globals.Secrets.Patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid global pattern '%s': %v", pattern, err)
			}
			extractor.globalPatterns = append(extractor.globalPatterns, regex)
		}

		// Compile global UUID patterns
		for _, uuidPattern := range cfg.Globals.Secrets.UUIDs {
			regex, err := regexp.Compile(uuidPattern.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid UUID pattern '%s': %v", uuidPattern.Pattern, err)
			}
			extractor.globalUUIDPatterns = append(extractor.globalUUIDPatterns, compiledUUIDPattern{
				regex:       regex,
				sensitive:   uuidPattern.Sensitive,
				description: uuidPattern.Description,
			})
		}

		// Compile global value patterns
		for _, valuePattern := range cfg.Globals.Secrets.Values {
			regex, err := regexp.Compile(valuePattern.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid value pattern '%s': %v", valuePattern.Pattern, err)
			}
			extractor.globalValuePatterns = append(extractor.globalValuePatterns, compiledValuePattern{
				regex:       regex,
				sensitive:   valuePattern.Sensitive,
				description: valuePattern.Description,
			})
		}
	}

	// Compile service-specific patterns
	for serviceName, serviceConfig := range cfg.Services {
		if !serviceConfig.Enabled || serviceConfig.Secrets == nil {
			continue
		}

		// Compile service key patterns
		servicePatterns := make([]compiledServicePattern, 0)
		for _, pattern := range serviceConfig.Secrets.Patterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid service pattern '%s' for service '%s': %v", pattern, serviceName, err)
			}
			servicePatterns = append(servicePatterns, compiledServicePattern{
				regex:       regex,
				pattern:     pattern,
				description: serviceConfig.Secrets.Description,
			})
		}
		extractor.servicePatterns[serviceName] = servicePatterns

		// Compile service-specific UUID patterns (extend global patterns)
		serviceUUIDPatterns := make([]compiledUUIDPattern, 0)
		for _, uuidPattern := range serviceConfig.Secrets.UUIDs {
			regex, err := regexp.Compile(uuidPattern.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid service UUID pattern '%s' for service '%s': %v", uuidPattern.Pattern, serviceName, err)
			}
			serviceUUIDPatterns = append(serviceUUIDPatterns, compiledUUIDPattern{
				regex:       regex,
				sensitive:   uuidPattern.Sensitive,
				description: fmt.Sprintf("%s (service-specific)", uuidPattern.Description),
			})
		}
		extractor.serviceUUIDPatterns[serviceName] = serviceUUIDPatterns

		// Compile service-specific value patterns (extend global patterns)
		serviceValuePatterns := make([]compiledValuePattern, 0)
		for _, valuePattern := range serviceConfig.Secrets.Values {
			regex, err := regexp.Compile(valuePattern.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid service value pattern '%s' for service '%s': %v", valuePattern.Pattern, serviceName, err)
			}
			serviceValuePatterns = append(serviceValuePatterns, compiledValuePattern{
				regex:       regex,
				sensitive:   valuePattern.Sensitive,
				description: fmt.Sprintf("%s (service-specific)", valuePattern.Description),
			})
		}
		extractor.serviceValuePatterns[serviceName] = serviceValuePatterns

		// Compile service-specific locations
		if serviceConfig.Secrets.Locations != nil {
			locations, err := extractor.compileLocations(serviceConfig.Secrets.Locations)
			if err != nil {
				return nil, fmt.Errorf("invalid service locations for service '%s': %v", serviceName, err)
			}
			extractor.serviceLocations[serviceName] = locations
		}
	}

	// Compile global locations
	if cfg.Globals.Secrets != nil && cfg.Globals.Secrets.Locations != nil {
		locations, err := extractor.compileLocations(cfg.Globals.Secrets.Locations)
		if err != nil {
			return nil, fmt.Errorf("invalid global locations: %v", err)
		}
		extractor.globalLocations = locations
	}

	return extractor, nil
}

// LoadConfig loads secrets configuration from YAML data
// LoadConfig loads a config from YAML data
func LoadConfig(yamlData []byte) (*config.Config, error) {
	var lc config.Config
	if err := yaml.UnmarshalStrict(yamlData, &lc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return &lc, nil
}

// compileLocations compiles location patterns for efficient matching
func (e *SecretExtractor) compileLocations(locations *config.SecretLocations) (*compiledLocations, error) {
	compiled := &compiledLocations{
		scanMode:        locations.ScanMode,
		basePath:        locations.BasePath,
		storePath:       locations.StorePath,
		additionalPaths: make(map[string]bool),
		pathPatterns:    make([]*regexp.Regexp, 0),
		include:         make(map[string]bool),
		includePatterns: make([]*regexp.Regexp, 0),
		exclude:         make(map[string]bool),
		excludePatterns: make([]*regexp.Regexp, 0),
	}

	// Set default scan mode and base path
	if compiled.scanMode == "" {
		compiled.scanMode = config.ScanModeFiltered // Default to filtered mode focusing on secrets
	}

	// Set default base path to "configMap"
	compiled.basePath = locations.BasePath
	if compiled.basePath == "" {
		compiled.basePath = "configMap"
	}

	// Set default store path to "secrets"
	compiled.storePath = locations.StorePath
	if compiled.storePath == "" {
		compiled.storePath = "secrets"
	}

	// Compile additional exact paths
	for _, path := range locations.AdditionalPaths {
		compiled.additionalPaths[path] = true
	}

	// Compile path patterns
	for _, pattern := range locations.PathPatterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid path pattern '%s': %v", pattern, err)
		}
		compiled.pathPatterns = append(compiled.pathPatterns, regex)
	}

	// Compile include paths and patterns
	for _, path := range locations.Include {
		if isPattern(path) {
			regex, err := regexp.Compile(path)
			if err != nil {
				return nil, fmt.Errorf("invalid include pattern '%s': %v", path, err)
			}
			compiled.includePatterns = append(compiled.includePatterns, regex)
		} else {
			compiled.include[path] = true
		}
	}

	// Compile exclude paths and patterns
	for _, path := range locations.Exclude {
		if isPattern(path) {
			regex, err := regexp.Compile(path)
			if err != nil {
				return nil, fmt.Errorf("invalid exclude pattern '%s': %v", path, err)
			}
			compiled.excludePatterns = append(compiled.excludePatterns, regex)
		} else {
			compiled.exclude[path] = true
		}
	}

	return compiled, nil
}

// isPattern checks if a string contains regex metacharacters
func isPattern(s string) bool {
	return strings.ContainsAny(s, ".*+?[]{}()^$|\\")
}

// shouldScanPath determines if a path should be scanned for secrets
func (e *SecretExtractor) shouldScanPath(path, serviceName string) bool {
	// Check service-specific locations first
	if serviceLocations := e.serviceLocations[serviceName]; serviceLocations != nil {
		return e.checkLocations(path, serviceLocations)
	}

	// Fall back to global locations
	if e.globalLocations != nil {
		return e.checkLocations(path, e.globalLocations)
	}

	// Default behavior: scan all paths
	return true
}

// checkLocations checks if a path should be scanned based on compiled locations
func (e *SecretExtractor) checkLocations(path string, locations *compiledLocations) bool {
	// Never scan the store path (where secrets are placed)
	if locations.storePath != "" && (strings.HasPrefix(path, locations.storePath+".") || path == locations.storePath) {
		return false
	}

	switch locations.scanMode {
	case config.ScanModeTargeted:
		// Only scan base path, additional paths, and patterns
		if locations.basePath != "" && (strings.HasPrefix(path, locations.basePath+".") || path == locations.basePath) {
			return true
		}
		return e.pathMatches(path, locations.additionalPaths, locations.pathPatterns)

	case config.ScanModeFiltered:
		// Default: scan base path + additional paths, respect include/exclude
		isInBasePath := locations.basePath != "" && (strings.HasPrefix(path, locations.basePath+".") || path == locations.basePath)
		isAdditionalPath := e.pathMatches(path, locations.additionalPaths, locations.pathPatterns)

		var shouldScan bool
		// If no specific includes are set, default to base path + additional paths
		if len(locations.include) == 0 && len(locations.includePatterns) == 0 {
			shouldScan = isInBasePath || isAdditionalPath
		} else {
			// If includes are specified, path must match include
			shouldScan = e.pathMatches(path, locations.include, locations.includePatterns)
		}

		// Apply excludes
		if shouldScan && e.pathMatches(path, locations.exclude, locations.excludePatterns) {
			return false
		}

		return shouldScan

	default: // config.ScanModeAll
		// Scan all paths (ignores location specifications)
		return true
	}
}

// pathMatches checks if a path matches any of the exact paths or patterns
func (e *SecretExtractor) pathMatches(path string, exactPaths map[string]bool, patterns []*regexp.Regexp) bool {
	// Check exact paths
	if exactPaths[path] {
		return true
	}

	// Check patterns
	for _, pattern := range patterns {
		if pattern.MatchString(path) {
			return true
		}
	}

	return false
}

// ExtractSecrets analyzes YAML content and extracts secrets
func (e *SecretExtractor) ExtractSecrets(yamlData []byte, serviceName string) (*ExtractionResult, error) {
	result := &ExtractionResult{
		Secrets:     make([]SecretMatch, 0),
		Warnings:    make([]string, 0),
		ServiceName: serviceName,
		Summary: Summary{
			ByClassification: make(map[Classification]int),
			ByConfidence:     make(map[ConfidenceLevel]int),
		},
	}

	var data interface{}
	if err := yaml.UnmarshalStrict(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Extract secrets recursively
	e.analyzeValue(data, "", result, serviceName)

	// Generate summary
	e.generateSummary(result)

	return result, nil
}

// analyzeValue recursively analyzes YAML values for secrets
func (e *SecretExtractor) analyzeValue(value interface{}, currentPath string, result *ExtractionResult, serviceName string) {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			fullPath := buildPath(currentPath, key)
			e.analyzeKeyValue(key, val, fullPath, result, serviceName)
			e.analyzeValue(val, fullPath, result, serviceName)
		}
	case []interface{}:
		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", currentPath, i)
			e.analyzeValue(item, indexPath, result, serviceName)
		}
	}
}

// analyzeKeyValue analyzes a specific key-value pair for secrets
func (e *SecretExtractor) analyzeKeyValue(key string, value interface{}, path string, result *ExtractionResult, serviceName string) {
	// Check if this path should be scanned for secrets
	if !e.shouldScanPath(path, serviceName) {
		return
	}

	// Only analyze string values for secrets
	stringValue, ok := value.(string)
	if !ok || strings.TrimSpace(stringValue) == "" {
		return
	}

	secretMatch := SecretMatch{
		Path:        path,
		Key:         key,
		Value:       stringValue,
		MaskedValue: e.maskValue(stringValue),
		MatchedBy:   make([]MatchReason, 0),
		Warnings:    make([]string, 0),
	}

	// Check service-specific exact key matches first
	if e.checkServiceExactKeys(key, serviceName, &secretMatch) {
		secretMatch.ServiceSpecific = true
		secretMatch.Confidence = ConfidenceHigh
	}

	// Check service-specific patterns
	if e.checkServicePatterns(path, serviceName, &secretMatch) {
		secretMatch.ServiceSpecific = true
		if secretMatch.Confidence == "" {
			secretMatch.Confidence = ConfidenceHigh
		}
	}

	// Check global key patterns
	e.checkGlobalPatterns(path, &secretMatch)

	// Check UUID patterns (both key and value)
	e.checkUUIDPatterns(key, stringValue, serviceName, &secretMatch)

	// Check value patterns
	e.checkValuePatterns(stringValue, serviceName, &secretMatch)

	// If we found any matches, add to results
	if len(secretMatch.MatchedBy) > 0 {
		e.classifySecret(&secretMatch)
		// Only set default confidence if none was set by classifySecret
		if secretMatch.Confidence == "" {
			secretMatch.Confidence = ConfidenceMedium
		}
		result.Secrets = append(result.Secrets, secretMatch)
	}
}

// checkServiceExactKeys checks if key matches exact service-specific keys
func (e *SecretExtractor) checkServiceExactKeys(key, serviceName string, match *SecretMatch) bool {
	serviceConfig, exists := e.config.Services[serviceName]
	if !exists || !serviceConfig.Enabled || serviceConfig.Secrets == nil {
		return false
	}

	for _, exactKey := range serviceConfig.Secrets.Keys {
		if key == exactKey {
			match.MatchedBy = append(match.MatchedBy, MatchReason{
				Type:        MatchTypeExactKey,
				Pattern:     exactKey,
				Description: fmt.Sprintf("Exact key match for service %s", serviceName),
				Confidence:  ConfidenceHigh,
			})
			return true
		}
	}
	return false
}

// checkServicePatterns checks service-specific patterns
func (e *SecretExtractor) checkServicePatterns(path, serviceName string, match *SecretMatch) bool {
	patterns, exists := e.servicePatterns[serviceName]
	if !exists {
		return false
	}

	found := false
	for _, pattern := range patterns {
		if pattern.regex.MatchString(path) {
			match.MatchedBy = append(match.MatchedBy, MatchReason{
				Type:        MatchTypeServicePattern,
				Pattern:     pattern.pattern,
				Description: pattern.description,
				Confidence:  ConfidenceHigh,
			})
			found = true
		}
	}
	return found
}

// checkGlobalPatterns checks global key patterns
func (e *SecretExtractor) checkGlobalPatterns(path string, match *SecretMatch) {
	for i, pattern := range e.globalPatterns {
		if pattern.MatchString(path) {
			match.MatchedBy = append(match.MatchedBy, MatchReason{
				Type:        MatchTypeKeyPattern,
				Pattern:     e.config.Globals.Secrets.Patterns[i],
				Description: "Global key pattern match",
				Confidence:  ConfidenceMedium,
			})
		}
	}
}

// checkUUIDPatterns checks UUID patterns in both key and value (global + service-specific)
func (e *SecretExtractor) checkUUIDPatterns(key, value, serviceName string, match *SecretMatch) {
	// Check service-specific UUID patterns first
	if servicePatterns, exists := e.serviceUUIDPatterns[serviceName]; exists {
		for _, pattern := range servicePatterns {
			if pattern.regex.MatchString(key) || pattern.regex.MatchString(value) {
				confidence := ConfidenceMedium
				if pattern.sensitive {
					confidence = ConfidenceHigh
				}

				match.MatchedBy = append(match.MatchedBy, MatchReason{
					Type:        MatchTypeUUIDPattern,
					Pattern:     pattern.regex.String(),
					Description: pattern.description,
					Confidence:  confidence,
				})
				match.ServiceSpecific = true
			}
		}
	}

	// Then check global UUID patterns
	for _, pattern := range e.globalUUIDPatterns {
		if pattern.regex.MatchString(key) || pattern.regex.MatchString(value) {
			confidence := ConfidenceLow
			if pattern.sensitive {
				confidence = ConfidenceMedium
			}

			match.MatchedBy = append(match.MatchedBy, MatchReason{
				Type:        MatchTypeUUIDPattern,
				Pattern:     pattern.regex.String(),
				Description: pattern.description,
				Confidence:  confidence,
			})
		}
	}
}

// checkValuePatterns checks value content patterns (global + service-specific)
func (e *SecretExtractor) checkValuePatterns(value, serviceName string, match *SecretMatch) {
	// Check service-specific value patterns first
	if servicePatterns, exists := e.serviceValuePatterns[serviceName]; exists {
		for _, pattern := range servicePatterns {
			if pattern.regex.MatchString(value) {
				confidence := ConfidenceMedium
				if pattern.sensitive {
					confidence = ConfidenceHigh
				}

				match.MatchedBy = append(match.MatchedBy, MatchReason{
					Type:        MatchTypeValuePattern,
					Pattern:     pattern.regex.String(),
					Description: pattern.description,
					Confidence:  confidence,
				})
				match.ServiceSpecific = true
			}
		}
	}

	// Then check global value patterns
	for _, pattern := range e.globalValuePatterns {
		if pattern.regex.MatchString(value) {
			confidence := ConfidenceLow
			if pattern.sensitive {
				confidence = ConfidenceMedium
			}

			match.MatchedBy = append(match.MatchedBy, MatchReason{
				Type:        MatchTypeValuePattern,
				Pattern:     pattern.regex.String(),
				Description: pattern.description,
				Confidence:  confidence,
			})
		}
	}
}

// classifySecret determines the classification based on patterns matched
func (e *SecretExtractor) classifySecret(match *SecretMatch) {
	key := strings.ToLower(match.Key)

	// First check for value-based patterns from the matched reasons
	// Check in order of specificity (most specific first)
	for _, reason := range match.MatchedBy {
		if reason.Type == MatchTypeValuePattern {
			if strings.Contains(reason.Description, "JWT") || strings.Contains(reason.Pattern, "eyJ") {
				match.Classification = ClassificationJWT
				break
			}
		}
	}

	// Check hex patterns before base64 since hex is more specific
	if match.Classification == "" {
		for _, reason := range match.MatchedBy {
			if reason.Type == MatchTypeValuePattern {
				if strings.Contains(reason.Description, "Hex") || strings.Contains(reason.Pattern, "[A-Fa-f0-9]") {
					match.Classification = ClassificationHex
					break
				}
			}
		}
	}

	// Then check base64 patterns
	if match.Classification == "" {
		for _, reason := range match.MatchedBy {
			if reason.Type == MatchTypeValuePattern {
				if strings.Contains(reason.Description, "Base64") || strings.Contains(reason.Pattern, "[A-Za-z0-9+/]{40,}") {
					match.Classification = ClassificationBase64
					break
				}
			}
		}
	}

	// Then check key-based classification (only if no value-based classification was set)
	if match.Classification == "" {
		switch {
		case strings.Contains(key, "password") || strings.Contains(key, "passwd"):
			match.Classification = ClassificationPassword
		case strings.Contains(key, "api") && (strings.Contains(key, "key") || strings.Contains(key, "token")):
			match.Classification = ClassificationAPIKey
		case strings.Contains(key, "jwt"):
			match.Classification = ClassificationJWT
		case strings.Contains(key, "token"):
			match.Classification = ClassificationToken
		case strings.Contains(key, "secret"):
			match.Classification = ClassificationSecret
		case e.isUUIDValue(match.Value):
			match.Classification = ClassificationUUID
		case e.isHexValue(match.Value):
			match.Classification = ClassificationHex
		case e.isBase64Value(match.Value):
			match.Classification = ClassificationBase64
		default:
			match.Classification = ClassificationUnknown
		}
	}

	// Determine the highest confidence from all match reasons
	highestConfidence := ConfidenceLow
	for _, reason := range match.MatchedBy {
		switch reason.Confidence {
		case ConfidenceHigh:
			highestConfidence = ConfidenceHigh
		case ConfidenceMedium:
			if highestConfidence == ConfidenceLow {
				highestConfidence = ConfidenceMedium
			}
		}
	}

	// Apply the highest confidence, with service-specific bonus
	if match.ServiceSpecific {
		if highestConfidence == ConfidenceHigh || match.Classification != ClassificationUnknown {
			match.Confidence = ConfidenceHigh
		} else if highestConfidence == ConfidenceMedium {
			match.Confidence = ConfidenceHigh // Promote service-specific medium to high
		} else {
			match.Confidence = ConfidenceMedium // Service-specific minimum
		}
	} else {
		match.Confidence = highestConfidence
		// Ensure we have at least low confidence for non-service-specific
		if match.Confidence == "" {
			match.Confidence = ConfidenceLow
		}
	}
}

// Helper functions
func (e *SecretExtractor) maskValue(value string) string {
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func (e *SecretExtractor) isUUIDValue(value string) bool {
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidPattern.MatchString(value)
}

func (e *SecretExtractor) isBase64Value(value string) bool {
	base64Pattern := regexp.MustCompile(`^[A-Za-z0-9+/]{40,}={0,2}$`)
	return base64Pattern.MatchString(value)
}

func (e *SecretExtractor) isHexValue(value string) bool {
	hexPattern := regexp.MustCompile(`^[A-Fa-f0-9]{32,}$`)
	return hexPattern.MatchString(value)
}

func (e *SecretExtractor) generateSummary(result *ExtractionResult) {
	result.Summary.TotalSecrets = len(result.Secrets)

	for _, secret := range result.Secrets {
		result.Summary.ByClassification[secret.Classification]++
		result.Summary.ByConfidence[secret.Confidence]++

		if secret.ServiceSpecific {
			result.Summary.ServiceSpecific++
		} else {
			result.Summary.GloballyDetected++
		}
	}
}

func buildPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// GetServiceNames returns all enabled service names
func (e *SecretExtractor) GetServiceNames() []string {
	services := make([]string, 0)
	for name, svcConfig := range e.config.Services {
		if svcConfig.Enabled {
			services = append(services, name)
		}
	}
	return services
}

// HasServiceConfig returns true if the service has specific secret configuration
func (e *SecretExtractor) HasServiceConfig(serviceName string) bool {
	serviceConfig, exists := e.config.Services[serviceName]
	return exists && serviceConfig.Enabled && serviceConfig.Secrets != nil
}
