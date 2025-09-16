# yaml Package

A comprehensive Go package for working with YAML files while preserving comments, formatting, and structure.</br>
This package is specifically designed for Helm values files and configuration management where comment preservation is critical.

## Features

- **Comment Preservation**: Load, modify, and save YAML files while preserving all comments
- **Deep Merging**: Merge multiple YAML files with configurable merge strategies
- **Path-based Access**: Get and set values using dot notation (e.g., `spec.containers[0].image`)
- **CamelCase Conversion**: Convert YAML keys to camelCase while preserving Java properties and UUIDs
- **Formatting Control**: Configure indentation (2-space default) and other formatting options
- **YAML Node Manipulation**: Direct access to yaml.v3 Node API for advanced use cases
- **Type-safe Operations**: Built on `github.com/elioetibr/yaml` for reliable YAML processing

## Installation

```go
import "helm-charts-migrator/v1/pkg/yaml"
```

## Core Types

### Document

The main type for working with YAML documents:

```go
type Document struct {
    node *yaml.Node
}
```

### Options

Configuration for loading and saving YAML:

```go
type Options struct {
    IndentSize       int  // Number of spaces for indentation (default: 2)
    PreserveComments bool // Preserve comments when loading/saving (default: true)
}
```

### MergeOptions

Configuration for merging YAML documents:

```go
type MergeOptions struct {
    Strategy                MergeStrategy // Merge strategy to use
    PreferSourceComments    bool         // Use comments from source
    KeepDestinationComments bool         // Keep destination comments when source has none
}
```

### MergeStrategy

Available merge strategies:

```go
const (
    MergeOverwrite  // Replace destination values with source values
    MergeDeep       // Recursively merge maps, replace other types (default)
    MergeAppend     // Append source arrays to destination arrays
)
```

### CommentPosition

Position for adding comments:

```go
const (
    CommentAbove  // Place comment on line(s) above the node
    CommentInline // Place comment on the same line as the node
    CommentBelow  // Place comment on line(s) below the node
)
```

## Usage Examples

### Loading and Saving with Comments

```go
// Load YAML file preserving comments
doc, err := yaml.LoadFile("values.yaml", nil)
if err != nil {
    log.Fatal(err)
}

// Modify values
err = doc.SetValue("image.tag", "v2.0.0")
err = doc.SetComment("image.tag", "# Updated version", yaml.CommentAbove)

// Save with 2-space indentation (default)
err = doc.SaveFile("values-updated.yaml", nil)

// Custom options
opts := &yaml.Options{
    IndentSize:       4,
    PreserveComments: true,
}
err = doc.SaveFile("values-custom.yaml", opts)
```

### Merging YAML Files

```go
// Load base and override files
base, _ := yaml.LoadFile("values.yaml", nil)
override, _ := yaml.LoadFile("values-prod.yaml", nil)

// Merge with default options (MergeDeep strategy)
merged, err := yaml.Merge(base, override, nil)

// Custom merge options
opts := &yaml.MergeOptions{
    Strategy:                yaml.MergeDeep,
    PreferSourceComments:    false, // Keep base comments
    KeepDestinationComments: true,  // Preserve existing comments
}
merged, err = yaml.Merge(base, override, opts)

// Merge multiple files
merged, err = yaml.MergeFiles([]string{
    "base.yaml",
    "override1.yaml", 
    "override2.yaml",
}, opts)

// Save merged result
merged.SaveFile("values-merged.yaml", nil)
```

### Path-based Access

```go
// Access nested values using dot notation
value, err := doc.GetValue("spec.replicas")
err = doc.SetValue("spec.replicas", "3")

// Array access
value, err = doc.GetValue("containers[0].image")
err = doc.SetValue("containers[0].ports[0].containerPort", "8080")

// Check if key exists
if doc.HasKey("metadata.labels") {
    // Key exists
}

// Remove a key
err = doc.RemoveKey("metadata.annotations.deprecated")
```

### Working with Comments

```go
// Add comments to specific nodes
doc.SetComment("database.password", "# DO NOT COMMIT", yaml.CommentAbove)
doc.SetComment("version", "# Semantic version", yaml.CommentInline)

// Get comment at specific position
comment, err := doc.GetComment("database.password", yaml.CommentAbove)

// Check if document has comments
hasComments, _ := doc.IsCommented("")  // Check root
hasComments, _ = doc.IsCommented("spec.template")  // Check specific path

// Strip all comments
doc.StripComments()
```

### CamelCase Conversion

```go
// Create a converter
converter := yaml.NewConverter()

// Convert YAML document to camelCase
input := []byte(`
ServiceName: my-service
DatabaseSettings:
  ConnectionString: localhost:5432
  root.properties: config-value  # Java properties preserved
`)

output, err := converter.ConvertYAML(input)
// Result:
// serviceName: my-service
// databaseSettings:
//   connectionString: localhost:5432
//   root.properties: config-value

// Convert a map directly
data := map[string]interface{}{
    "ServiceName": "api",
    "MaxRetries": 3,
}
converted := converter.ConvertMap(data)
```

### Advanced Usage

```go
// Direct access to yaml.Node for advanced manipulation
node := doc.GetNode()

// Find a specific node by path
targetNode, err := doc.FindNode("spec.containers[0]")

// Clone a document
clone := doc.Clone()

// Convert to/from map (loses comments)
data, _ := doc.ToMap()
newDoc, _ := yaml.FromMap(data)

// Write to io.Writer
var buf bytes.Buffer
err = doc.WriteTo(&buf, nil)

// Marshal to bytes
data, err := doc.Marshal(nil)

// Load from bytes
doc, err = yaml.Load(data, nil)
```

### Utility Functions

```go
// Marshal interface{} to YAML with options
data := map[string]interface{}{"key": "value"}
yamlBytes, err := yaml.MarshalWithOptions(data, &yaml.Options{
    IndentSize: 4,
})

// Simple marshal (uses default options)
yamlBytes, err := yaml.Marshal(data)

// Unmarshal YAML to struct (strict mode)
var config Config
err := yaml.UnmarshalStrict(yamlBytes, &config)

// Regular unmarshal
err = yaml.Unmarshal(yamlBytes, &config)
```

## Converter Features

The converter intelligently handles different key patterns:

### Preserved Patterns
- **Java Properties**: Keys like `root.properties`, `server.config`
- **UUIDs**: Standard UUID format (e.g., `a1b2c3d4-e5f6-7890-abcd-ef1234567890`)
- **Uppercase Prefixes**: Keys starting with 3+ uppercase letters (e.g., `AWS_REGION`)
- **Already CamelCase**: Keys already in camelCase are preserved

### Conversion Examples

```yaml
# Input
ServiceName: api
MaxRetries: 3
DATABASE_URL: postgres://localhost
root.properties: config
ClientUUID: 123e4567-e89b-12d3-a456-426614174000

# Output
serviceName: api
maxRetries: 3
DATABASE_URL: postgres://localhost  # Preserved (uppercase prefix)
root.properties: config              # Preserved (Java property)
ClientUUID: 123e4567-e89b-12d3-a456-426614174000  # Preserved (UUID)
```

## Use Cases

This package is ideal for:

- **Helm Chart Management**: Merging base values with environment-specific overrides
- **Configuration Management**: Updating config files while preserving documentation
- **CI/CD Pipelines**: Automated config updates without losing comments
- **Migration Tools**: Transforming YAML structures while maintaining context
- **GitOps Workflows**: Programmatic updates to YAML manifests with comment preservation

## Notes

- The package uses `github.com/elioetibr/yaml` for YAML processing
- Comments are preserved at the node level (head, line, and foot comments)
- Path notation supports nested objects and array indices
- The converter is designed for Helm values files with intelligent pattern preservation
- Default indentation is 2 spaces (Helm standard)
- All operations maintain YAML ordering and structure

## Package Structure

```
v1/pkg/yaml/
├── yamlutil.go       # Core Document type and basic operations
├── helpers.go        # Path-based access and manipulation functions
├── merger.go         # Merge functionality with strategies
├── converter.go      # CamelCase conversion with pattern preservation
└── README.md         # This documentation
```

## Testing

The package includes comprehensive test coverage:
- Unit tests for all major functions
- Table-driven tests for edge cases
- Integration tests for complex scenarios
- Example functions demonstrating usage

Run tests:

```bash
go test ./v1/pkg/yaml/...
```

## Performance Considerations

- Document operations maintain the full YAML structure in memory
- Large files are processed efficiently with streaming where possible
- Comments add minimal overhead to processing
- The converter processes documents in a single pass
- Merge operations are optimized for deep structures

## Thread Safety

The Document type is not thread-safe. For concurrent operations, use synchronization primitives or create separate Document instances for each goroutine.