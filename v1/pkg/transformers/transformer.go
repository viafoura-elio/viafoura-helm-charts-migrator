package transformers

import (
	"fmt"
	"regexp"
	"strings"

	"helm-charts-migrator/v1/pkg/config"
	"helm-charts-migrator/v1/pkg/yaml"
)

// ComplexTransformer handles complex transformations like ingress to host extraction
type ComplexTransformer struct {
	config *config.Config
	rules  []compiledRule
}

type compiledRule struct {
	sourceRegex *regexp.Regexp
	rule        config.TransformRule
}

// HostExtraction represents extracted host information
type HostExtraction struct {
	Host    string `yaml:"host"`
	Source  string `yaml:"source"` // ingress field name
	Path    string `yaml:"path"`   // full YAML path
	IsValid bool   `yaml:"is_valid"`
	Reason  string `yaml:"reason,omitempty"`
}

// TransformationResult contains the results of transformation
type TransformationResult struct {
	ExtractedHosts []HostExtraction `yaml:"extracted_hosts"`
	ModifiedPaths  []string         `yaml:"modified_paths"`
	Warnings       []string         `yaml:"warnings"`
}

// New creates a new ComplexTransformer instance
func New(cfg *config.Config) (*ComplexTransformer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	transformer := &ComplexTransformer{
		config: cfg,
		rules:  make([]compiledRule, 0),
	}

	if cfg.Globals.Mappings == nil || cfg.Globals.Mappings.Transform == nil || !cfg.Globals.Mappings.Transform.Enabled {
		return transformer, nil
	}

	// Compile regex patterns for each rule
	for _, rule := range cfg.Globals.Mappings.Transform.Rules {
		regex, err := regexp.Compile(rule.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %v", rule.SourcePath, err)
		}

		transformer.rules = append(transformer.rules, compiledRule{
			sourceRegex: regex,
			rule:        rule,
		})
	}

	return transformer, nil
}

// LoadConfig loads transformer configuration from YAML data
func LoadConfig(yamlData []byte) (*config.Config, error) {
	var cfg config.Config
	if err := yaml.UnmarshalStrict(yamlData, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return &cfg, nil
}

// TransformYAML applies transformations to YAML content
func (t *ComplexTransformer) TransformYAML(yamlData []byte) ([]byte, *TransformationResult, error) {
	result := &TransformationResult{
		ExtractedHosts: make([]HostExtraction, 0),
		ModifiedPaths:  make([]string, 0),
		Warnings:       make([]string, 0),
	}

	if t.config.Globals.Mappings == nil || t.config.Globals.Mappings.Transform == nil || !t.config.Globals.Mappings.Transform.Enabled || len(t.rules) == 0 {
		return yamlData, result, nil
	}

	var data interface{}
	if err := yaml.UnmarshalStrict(yamlData, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Find and extract transformations
	transformations := make(map[string][]HostExtraction)
	t.findTransformations(data, "", transformations, result)

	// Apply transformations
	transformed := t.applyTransformations(data, transformations, result)

	transformedYAML, err := yaml.Marshal(transformed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal transformed YAML: %v", err)
	}

	return transformedYAML, result, nil
}

// findTransformations recursively finds paths that need transformation
func (t *ComplexTransformer) findTransformations(value interface{}, currentPath string, transformations map[string][]HostExtraction, result *TransformationResult) {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			fullPath := buildPath(currentPath, key)

			// Check if this path matches any transformation rule
			matched := false
			for _, rule := range t.rules {
				if rule.sourceRegex.MatchString(fullPath) {
					matched = true
					switch rule.rule.Type {
					case "ingress_to_hosts":
						hosts := t.extractHostsFromIngress(val, fullPath)
						if len(hosts) > 0 {
							transformations[rule.rule.TargetPath] = append(transformations[rule.rule.TargetPath], hosts...)
							result.ExtractedHosts = append(result.ExtractedHosts, hosts...)
							result.ModifiedPaths = append(result.ModifiedPaths, fullPath)
						}
					}
				}
			}

			// Only recurse if this path wasn't transformed
			if !matched {
				t.findTransformations(val, fullPath, transformations, result)
			}
		}
	case []interface{}:
		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", currentPath, i)
			t.findTransformations(item, indexPath, transformations, result)
		}
	}
}

// extractHostsFromIngress extracts valid hosts from ingress configuration
func (t *ComplexTransformer) extractHostsFromIngress(ingressData interface{}, sourcePath string) []HostExtraction {
	var hosts []HostExtraction

	switch ingress := ingressData.(type) {
	case map[string]interface{}:
		// Extract hosts from various ingress fields
		hosts = append(hosts, t.extractFromIngressMap(ingress, sourcePath)...)
	case string:
		// Direct host string
		if host := t.validateHost(ingress); host.IsValid {
			host.Source = "direct_value"
			host.Path = sourcePath
			hosts = append(hosts, host)
		}
	}

	return hosts
}

// extractFromIngressMap extracts hosts from ingress map structure
func (t *ComplexTransformer) extractFromIngressMap(ingress map[string]interface{}, sourcePath string) []HostExtraction {
	var hosts []HostExtraction

	// Common ingress host fields
	hostFields := []string{"host", "infoHost", "orgHost", "domain", "hostname", "hosts"}

	for _, field := range hostFields {
		if value, exists := ingress[field]; exists {
			fieldPath := buildPath(sourcePath, field)

			switch v := value.(type) {
			case string:
				host := t.validateHost(v)
				host.Source = field
				host.Path = fieldPath
				hosts = append(hosts, host)
			case []interface{}:
				for i, item := range v {
					if hostStr, ok := item.(string); ok {
						host := t.validateHost(hostStr)
						host.Source = field
						host.Path = fmt.Sprintf("%s[%d]", fieldPath, i)
						hosts = append(hosts, host)
					}
				}
			}
		}
	}

	return hosts
}

// validateHost validates if a string is a valid hostname
func (t *ComplexTransformer) validateHost(hostStr string) HostExtraction {
	host := HostExtraction{
		Host:    hostStr,
		IsValid: false,
	}

	// Basic validation - not empty
	if strings.TrimSpace(hostStr) == "" {
		host.Reason = "empty host"
		return host
	}

	// Remove protocol if present
	cleanHost := hostStr
	if strings.HasPrefix(hostStr, "http://") {
		cleanHost = strings.TrimPrefix(hostStr, "http://")
	} else if strings.HasPrefix(hostStr, "https://") {
		cleanHost = strings.TrimPrefix(hostStr, "https://")
	}

	// Remove path if present
	if idx := strings.Index(cleanHost, "/"); idx != -1 {
		cleanHost = cleanHost[:idx]
	}

	// Remove port if present
	if idx := strings.LastIndex(cleanHost, ":"); idx != -1 {
		if portPart := cleanHost[idx+1:]; isNumeric(portPart) {
			cleanHost = cleanHost[:idx]
		}
	}

	// Validate hostname format
	if !isValidHostname(cleanHost) {
		host.Reason = "invalid hostname format"
		return host
	}

	// This check is now redundant since isValidHostname checks for dots
	// Keeping for explicit error message

	host.Host = cleanHost
	host.IsValid = true
	return host
}

// applyTransformations applies all found transformations to create new structure
func (t *ComplexTransformer) applyTransformations(data interface{}, transformations map[string][]HostExtraction, result *TransformationResult) interface{} {
	// Start with original data
	transformed := t.copyWithoutTransformed(data, "", result.ModifiedPaths)

	// Apply transformations
	for targetPath, hosts := range transformations {
		validHosts := make([]string, 0)
		for _, host := range hosts {
			if host.IsValid {
				// Avoid duplicates
				if !contains(validHosts, host.Host) {
					validHosts = append(validHosts, host.Host)
				}
			} else {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Skipped invalid host '%s' from %s: %s", host.Host, host.Path, host.Reason))
			}
		}

		if len(validHosts) > 0 {
			t.setNestedValue(transformed, targetPath, validHosts)
		}
	}

	return transformed
}

// copyWithoutTransformed copies the original structure excluding transformed paths
func (t *ComplexTransformer) copyWithoutTransformed(value interface{}, currentPath string, transformedPaths []string) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			fullPath := buildPath(currentPath, key)

			// Skip if this path was transformed
			if contains(transformedPaths, fullPath) {
				continue
			}

			result[key] = t.copyWithoutTransformed(val, fullPath, transformedPaths)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			indexPath := fmt.Sprintf("%s[%d]", currentPath, i)
			result[i] = t.copyWithoutTransformed(item, indexPath, transformedPaths)
		}
		return result
	default:
		return value
	}
}

// setNestedValue creates nested structure and sets the value
func (t *ComplexTransformer) setNestedValue(data interface{}, targetPath string, value interface{}) {
	if root, ok := data.(map[string]interface{}); ok {
		parts := strings.Split(targetPath, ".")
		current := root

		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = value
			} else {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				if next, ok := current[part].(map[string]interface{}); ok {
					current = next
				}
			}
		}
	}
}

// Helper functions
func buildPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func isValidHostname(hostname string) bool {
	// Basic hostname validation using regex
	// Must have at least one dot and be valid hostname format
	if !strings.Contains(hostname, ".") {
		return false
	}
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	return hostnameRegex.MatchString(hostname)
}

func isNumeric(s string) bool {
	numericRegex := regexp.MustCompile(`^\d+$`)
	return numericRegex.MatchString(s)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// IsEnabled returns whether the transformer is enabled
func (t *ComplexTransformer) IsEnabled() bool {
	if t.config.Globals.Mappings == nil || t.config.Globals.Mappings.Transform == nil {
		return false
	}
	return t.config.Globals.Mappings.Transform.Enabled
}

// GetDescription returns the transformer description
func (t *ComplexTransformer) GetDescription() string {
	if t.config.Globals.Mappings == nil || t.config.Globals.Mappings.Transform == nil {
		return ""
	}
	return t.config.Globals.Mappings.Transform.Description
}

// GetRules returns the configured transformation rules
func (t *ComplexTransformer) GetRules() map[string]config.TransformRule {
	if t.config.Globals.Mappings == nil || t.config.Globals.Mappings.Transform == nil {
		return nil
	}
	return t.config.Globals.Mappings.Transform.Rules
}
