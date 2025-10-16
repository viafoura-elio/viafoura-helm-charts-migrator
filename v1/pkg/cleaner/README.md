# cleaner Package

The cleaner package provides functionality to remove unwanted keys from YAML files based on configurable regex patterns. It's designed to clean up Helm values files by removing deprecated or unnecessary configuration keys at the root level while preserving nested keys with the same name.

## Features

- **Root-level Key Removal**: Removes keys only at the root level of YAML documents
- **Regex Pattern Matching**: Uses regular expressions to identify keys to remove
- **File Path Filtering**: Process only specific files using glob patterns
- **Service-specific Configuration**: Override global patterns for specific services
- **Detailed Reporting**: Tracks all removed keys for audit purposes
- **Depth-aware Processing**: Preserves nested keys even if they match removal patterns

## Installation

```go
import "helm-charts-migrator/v1/pkg/cleaner"
```

## Core Types

### Cleaner
The main type for cleaning YAML files:
```go
type Cleaner struct {
    config      *config.Config
    enabled     bool
    keyPatterns []*regexp.Regexp
    pathPatterns []string
}
```

### CleanResult
Result of cleaning operation:
```go
type CleanResult struct {
    FilePath     string   // Path to the cleaned file
    RemovedKeys  []string // List of removed key paths
    KeyCount     int      // Number of keys removed
}
```

## Usage

### Basic Usage

```go
// Create cleaner from configuration
cfg := &config.Config{
    Globals: config.Globals{
        Mappings: map[string]interface{}{
            "cleaner": map[string]interface{}{
                "enabled": true,
                "key_patterns": []string{
                    "^[Cc]anary$",
                    "^[Pp]odLabels$",
                    "^[Nn]ameOverride$",
                },
                "path_patterns": []string{
                    "apps/**/legacy-values.yaml",
                    "apps/**/helm-values.yaml",
                },
            },
        },
    },
}

cleaner, err := cleaner.New(cfg)
if err != nil {
    log.Fatal(err)
}

// Clean a YAML file
yamlData := []byte(`
canary: true
podLabels:
  app: test
nameOverride: custom
deployment:
  canary: false  # This nested key is preserved
  replicas: 3
`)

cleanedData, result, err := cleaner.CleanYAML(yamlData, "apps/service/values.yaml")
if err != nil {
    log.Fatal(err)
}

// Check results
fmt.Printf("Removed %d keys: %v\n", result.KeyCount, result.RemovedKeys)
// Output: Removed 3 keys: [canary, podLabels, nameOverride]
```

### Configuration Examples

```yaml
# Global cleaner configuration
globals:
  mappings:
    cleaner:
      enabled: true
      description: "Remove unwanted root-level keys from values files"
      path_patterns:
        - "apps/**/legacy-values.yaml"
        - "apps/**/helm-values.yaml"
        - "apps/**/envs/**/*.yaml"
      key_patterns:
        - "^[Cc]anary$"                # Remove canary configuration
        - "^[Pp]odLabels$"             # Remove podLabels
        - "^[Pp]odAnnotations$"        # Remove podAnnotations
        - "^[Nn]ameOverride$"          # Remove nameOverride
        - "^[Ff]ullnameOverride$"      # Remove fullnameOverride

# Service-specific override
services:
  my-service:
    mappings:
      cleaner:
        enabled: true
        key_patterns:
          - "^[Dd]eprecated.*"         # Additional pattern for this service
          - "^[Tt]emp.*"               # Remove temporary keys
```

## Methods

### Constructor
```go
func New(cfg *config.Config) (*Cleaner, error)
```
Creates a new Cleaner instance from configuration.

### CleanYAML
```go
func (c *Cleaner) CleanYAML(yamlData []byte, filePath string) ([]byte, *CleanResult, error)
```
Cleans unwanted keys from YAML data. Returns cleaned YAML and a result containing removed keys.

### IsEnabled
```go
func (c *Cleaner) IsEnabled() bool
```
Returns whether the cleaner is enabled in configuration.

### GetDescription
```go
func (c *Cleaner) GetDescription() string
```
Returns the description from configuration.

### GetKeyPatterns
```go
func (c *Cleaner) GetKeyPatterns() []string
```
Returns the list of key patterns being used.

### GetPathPatterns
```go
func (c *Cleaner) GetPathPatterns() []string
```
Returns the list of path patterns for file filtering.

## Key Features Explained

### Root-level Only Removal
The cleaner only removes keys at the root level (depth 0) of the YAML document:
```yaml
# Before cleaning
canary: true              # Removed (root level)
deployment:
  canary: false           # Preserved (nested)
  strategy:
    canary:               # Preserved (nested)
      enabled: true
```

### Pattern Matching
Uses Go regular expressions for flexible key matching:
- `^[Cc]anary$` - Matches "canary" or "Canary" exactly
- `^[Dd]eprecated.*` - Matches any key starting with "deprecated" or "Deprecated"
- `.*Override$` - Matches any key ending with "Override"

### File Path Filtering
Only processes files matching glob patterns:
- `apps/**/values.yaml` - All values.yaml files under apps
- `**/*-values.yaml` - All files ending with -values.yaml
- `apps/*/legacy-values.yaml` - Specific service legacy values files

## Integration with Transformation Pipeline

The cleaner is integrated into the transformation pipeline and runs as the final step:
1. Normalizer - Key path transformations
2. Transformer - Complex transformations
3. Secret Separator - Extract and move secrets
4. **Cleaner** - Remove unwanted root keys

## Use Cases

- **Migration Cleanup**: Remove deprecated configuration during migration
- **Standardization**: Ensure consistent structure across services
- **Security**: Remove sensitive or unnecessary metadata
- **Optimization**: Clean up unused configuration keys
- **Compliance**: Remove non-compliant configuration patterns

## Performance Considerations

- Regex patterns are compiled once during initialization
- Single-pass processing for efficiency
- Minimal memory overhead with streaming YAML processing
- Depth tracking adds negligible overhead

## Error Handling

The cleaner handles various error scenarios:
- Invalid regex patterns in configuration
- Malformed YAML input
- File path matching errors
- Returns original data on processing errors

## Testing

Run tests:
```bash
go test ./v1/pkg/cleaner/...
```

## Notes

- Preserves YAML formatting and comments where possible
- Order of patterns doesn't matter as all are evaluated
- Empty pattern lists disable cleaning
- Service-specific patterns extend (not replace) global patterns
- Thread-safe for concurrent use on different files