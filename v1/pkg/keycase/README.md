# keycase Package

A Go package for converting YAML keys to camelCase with intelligent exclusion rules. This package is designed for normalizing configuration files, especially when migrating from snake_case, kebab-case, or PascalCase formats.

## Features

- **Smart Case Conversion**: Converts snake_case, kebab-case, and PascalCase to camelCase
- **Intelligent Exclusions**:
  - Skips Java/Spring property style keys (containing dots)
  - Skips keys starting with 3+ consecutive uppercase letters (like `HTTPServer`, `AWS_REGION`)
  - Skips UUID-like keys (multiple dash-separated hex segments)
  - Preserves already camelCase keys
  - Custom key preservation list
- **Comment Preservation**: Works with yaml.v3 to maintain comments
- **Statistics Tracking**: Get detailed conversion statistics
- **Customizable**: Configure exclusion rules and transformation logic

## Installation

```go
import "helm-charts-migrator/pkg/keycase"
```

## Usage

### Basic Usage

```go
converter := keycase.NewConverter()
yamlDoc := `
api_version: v1
first-name: John
HomeAddress: "123 Main St"
`

converted, err := converter.ConvertDocument([]byte(yamlDoc))
// Result: apiVersion, firstName, homeAddress
```

### Default Exclusion Rules

The converter automatically skips:

1. **Java/Spring Properties** (keys with dots):
   - `java.lang.String` → `java.lang.String` ✗ Not converted
   - `spring.application.name` → `spring.application.name` ✗ Not converted

2. **Keys Starting with 3+ Consecutive Uppercase Letters**:
   - `HTTPServer` → `HTTPServer` ✗ Not converted
   - `XMLParser` → `XMLParser` ✗ Not converted
   - `AWS_REGION` → `AWS_REGION` ✗ Not converted
   - `DATABASE_URL` → `DATABASE_URL` ✗ Not converted

3. **UUID-like Keys**:
   - `550e8400-e29b-41d4` → `550e8400-e29b-41d4` ✗ Not converted
   - `abc-def-123` → `abc-def-123` ✗ Not converted (hex-like segments)

4. **Already camelCase**:
   - `alreadyCamelCase` → `alreadyCamelCase` ✗ Not converted
   - `simpleWord` → `simpleWord` ✗ Not converted

### Conversion Examples

| Input | Output | Notes |
|-------|--------|-------|
| `snake_case` | `snakeCase` | Standard snake_case |
| `kebab-case` | `kebabCase` | Kebab case conversion |
| `PascalCase` | `pascalCase` | PascalCase to camelCase |
| `mixed_snake-Case` | `mixedSnakeCase` | Mixed formats |
| `first_name` | `firstName` | Common field |
| `home-address` | `homeAddress` | Kebab to camel |
| `HTTPSConnection` | `HTTPSConnection` | Starts with 4+ uppercase (skipped) |
| `HttpConnection` | `httpConnection` | Only 2 uppercase (converted) |
| `version_2` | `version2` | Numbers handled |
| `base_64_encode` | `base64Encode` | Complex conversion |
| `java.property` | `java.property` | Java property (skipped) |
| `AWS_S3_BUCKET` | `AWS_S3_BUCKET` | Starts with 3 uppercase (skipped) |
| `550e8400-e29b` | `550e8400-e29b` | UUID-like (skipped) |

### With Statistics

```go
converter := keycase.NewConverter()
var node yaml.Node
yaml.Unmarshal(yamlDoc, &node)

converted, stats := converter.ConvertWithStats(&node)

fmt.Printf("Total keys: %d\n", stats.TotalKeys)
fmt.Printf("Converted: %d\n", stats.ConvertedKeys)
fmt.Printf("Skipped: %d\n", stats.SkippedKeys)
fmt.Printf("  - Java properties: %d\n", stats.JavaProperties)
fmt.Printf("  - Uppercase keys: %d\n", stats.UppercaseKeys)
fmt.Printf("  - UUID keys: %d\n", stats.UUIDKeys)
fmt.Printf("  - Already camelCase: %d\n", stats.AlreadyCamelCase)
```

### Custom Configuration

```go
converter := keycase.NewConverter()

// Adjust uppercase threshold (default: 3)
converter.MinUppercaseChars = 4  // Now AWS would be converted, but HTTP wouldn't

// Disable Java property skipping
converter.SkipJavaProperties = false

// Preserve specific keys
converter.PreserveSpecialKeys = map[string]bool{
    "api_version": true,  // Keep specific keys unchanged
    "created_at": true,
    "updated_at": true,
}

// Custom transformation
converter.CustomTransform = func(key string) string {
    // Your custom logic here
    return "prefix" + strings.ToUpper(key[:1]) + key[1:]
}
```

### Working with Maps

```go
input := map[string]interface{}{
    "first_name": "John",
    "home-address": map[string]interface{}{
        "street_name": "Main St",
        "zip-code": "12345",
    },
}

result := converter.ConvertMap(input)
// Result: firstName, homeAddress with nested streetName, zipCode
```

## Use Cases

### API Response Normalization

Convert API responses from various naming conventions to consistent camelCase for JavaScript/TypeScript frontends:

```yaml
# Before (from backend)
user_id: 123
first_name: John
last_name: Doe
created-at: 2024-01-01

# After (for frontend)
userId: 123
firstName: John
lastName: Doe
createdAt: 2024-01-01
```

### Configuration Migration

Migrate configuration files from snake_case or kebab-case to camelCase:

```yaml
# Before
database_url: postgres://localhost
max-connections: 100
retry_attempts: 3

# After  
databaseUrl: postgres://localhost
maxConnections: 100
retryAttempts: 3
```

### GraphQL Schema Alignment

Ensure configuration keys match GraphQL field naming conventions (camelCase).

## Performance

The package is optimized for performance with:
- Efficient regex compilation
- Smart string building
- Minimal allocations in hot paths

## Testing

Comprehensive test coverage including:
- Basic conversions (snake_case, kebab-case, PascalCase)
- Edge cases (numbers, UUIDs, special characters)
- Exclusion rules validation

Run tests:
```bash
go test ./pkg/keycase -v
```

## Notes

- The package uses `github.com/elioetibr/yaml` for YAML processing
- Comments in YAML files are preserved when using Node-based conversion
- The converter creates a copy of the input, leaving the original unchanged
- Thread-safe for read operations, not safe for concurrent configuration changes
- Single words are considered already in camelCase and are not modified