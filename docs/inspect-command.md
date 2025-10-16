# Inspect Command

The `inspect` command provides human-readable reports on the hierarchical configuration structure for services.

## Overview

The inspect command helps you understand:
- How configuration values are inherited and merged across different levels
- Service-specific configuration overrides
- Available clusters, namespaces, and their relationships
- Secrets, mappings, and auto-inject configurations

## Usage

```bash
# Inspect all services (overview)
helm-charts-migrator inspect

# Inspect a specific service
helm-charts-migrator inspect --service heimdall

# Inspect service configuration for a specific cluster
helm-charts-migrator inspect --service heimdall --cluster prod01

# Inspect with verbose output showing all configuration layers
helm-charts-migrator inspect --service heimdall --verbose

# Specify custom config file
helm-charts-migrator inspect --config custom-config.yaml --service auth-service
```

## Flags

- `--service, -s`: Service name to inspect (optional)
- `--cluster, -c`: Cluster name for context (optional)
- `--namespace, -n`: Namespace for context (optional)
- `--verbose`: Show detailed configuration hierarchy (no shorthand due to conflict with klog's -v)
- `--format, -f`: Output format: text (default), json, yaml
- `--config`: Path to configuration file (default: ./config.yaml)

**Note**: The `--verbose` flag does not have a shorthand (`-v`) because it conflicts with klog's verbosity flag. Use the full `--verbose` flag instead.

## Example Outputs

### Overview Mode (All Services)

```
=== Service Configuration Overview ===

Account: production (1 cluster)
  ├─ prod01: enabled (default) - 1 namespace

Account: testdev (1 cluster)
  ├─ dev01: enabled - 7 namespaces

Total: 2 clusters, 8 namespaces

=== Services ===

  ✓ enabled auth-service (alias: auth)
  ✗ disabled auth0-oidc-demo (alias: auth0)
  ✓ enabled heimdall (alias: hmd)
      • Custom secrets configuration
      • Auto-inject rules
  ✓ enabled livecomments (alias: lc)
      • Custom secrets configuration
  ✓ enabled tyrion (alias: tyr)

Total: 5 services (4 enabled, 1 disabled)

=== Global Configuration ===

Pipeline: enabled (7/7 steps active)
Converter: skipJavaProperties=true, skipUppercaseKeys=true, minUppercaseChars=3
Performance: maxConcurrentServices=5, showProgress=true
SOPS: enabled (profile=cicd-sre, workers=5)
Secrets: enabled (14 patterns, 2 UUIDs, 3 values)

💡 Tip: Use --service <name> to inspect specific service configuration
💡 Tip: Use --verbose to see detailed configuration layers
```

### Service-Specific Mode

```
=== Service: heimdall ===

Status: enabled
Name: heimdall
Alias: hmd
Type: service
Repository: https://github.com/viafoura/heimdall

=== Configuration Hierarchy ===

📋 Configuration Sources:
  1. Global defaults (baseline)
  2. Service-specific overrides (3 differences)

🔐 Secrets Configuration:
  Status: enabled
  • Specific keys: 5
  • Patterns: 4
  • UUID patterns: 2
  • Locations:
    - Base path: configMap
    - Store path: secrets
  📝 Heimdall authentication service secrets

💉 Auto-Inject Rules:
  • Pattern: values.yaml
    Rules: 1
  • Pattern: envs/dev01/*/values.yaml
    Rules: 1

💡 Tip: Use --verbose to see detailed configuration values
```

### Verbose Service Mode

```
=== Service: heimdall ===

Status: enabled
Name: heimdall
Alias: hmd
Type: service
Repository: https://github.com/viafoura/heimdall

=== Configuration Hierarchy ===

📋 Configuration Sources:
  1. Global defaults (baseline)
  2. Service-specific overrides (3 differences)

🔍 Service Overrides:
  • Secrets configuration differs from global
  • Auto-inject rules defined
  • Custom key mappings specified

🗺️  Mappings Configuration:
  • Normalizer: 7 patterns
  • Transform: 1 rule
  • Extract: 4 patterns
  • Cleaner: 8 key patterns

🔐 Secrets Configuration:
  Status: enabled
  • Specific keys: 5
    - 3f4beddd-2061-49b0-ae80-6f1f2ed65b37
    - 682843b1-d3e0-460e-ab90-6556bc31470f
    - 936da557-6daa-4444-92cc-161fc290c603
    - 9487e74c-2d27-4085-b637-30a82239b0b2
    - c23203d0-1b8e-4208-92dc-85dc79e6226b
  • Patterns: 4
    - .*access.refresh.local_client_uuid.*
    - .*loginradius.*secret.*
    - .*oauth.*secret.*
    - .*provider.*secret.*
  • UUID patterns: 2
  • Locations:
    - Base path: configMap
    - Store path: secrets
  📝 Heimdall authentication service secrets

💉 Auto-Inject Rules:
  • Pattern: values.yaml
    Rules: 1
    - secrets."root.properties"."9487e74c-2d27-4085-b637-30a82239b0b2": misconfigured (condition: ifExists)
      Set Default Secret Value for 9487e74c-2d27-4085-b637-30a82239b0b2
  • Pattern: envs/dev01/*/values.yaml
    Rules: 1
    - configMap."root.properties"."auth.dataSource.user": {environment}-auth (condition: ifExists)
      Set environment-specific auth datasource user
```

### Service with Cluster Context

```
=== Service: heimdall ===

Status: enabled
Name: heimdall
Alias: hmd
Type: service
Repository: https://github.com/viafoura/heimdall

... (configuration sections) ...

=== Deployment Context: prod01 ===

Source: k8s1.cc
Target: prod01
AWS Profile: production-sre
AWS Region: us-east-1
Default Namespace: viafoura

Enabled Namespaces (1):
  • viafoura (default)
```

## Use Cases

### 1. Understanding Configuration Inheritance

Use inspect to see how service configurations inherit from global settings and what's been overridden:

```bash
helm-charts-migrator inspect --service livecomments --verbose
```

### 2. Verifying Service Setup

Before migrating a service, verify its configuration is correct:

```bash
helm-charts-migrator inspect --service auth-service --cluster prod01
```

### 3. Debugging Configuration Issues

When a service isn't behaving as expected, inspect shows the effective configuration:

```bash
helm-charts-migrator inspect --service heimdall --verbose
```

### 4. Auditing Secrets Configuration

Review which secrets are configured for a service:

```bash
helm-charts-migrator inspect --service tyrion
```

### 5. Getting System Overview

Quickly see all configured services and clusters:

```bash
helm-charts-migrator inspect
```

## Configuration Levels

The inspect command shows how configuration is layered:

1. **Global Defaults** - Baseline configuration in `globals` section
2. **Account Level** - AWS account-specific settings
3. **Cluster Level** - Cluster-specific overrides
4. **Namespace Level** - Namespace-specific customizations
5. **Service Level** - Service-specific overrides

## Integration with Other Commands

The inspect command complements other migrator commands:

- **Before `migrate`**: Use inspect to verify configuration
- **After `init`**: Use inspect to understand the generated config
- **With `validate`**: Use inspect to see what will be validated
- **For troubleshooting**: Use inspect to debug configuration issues

## Tips

- Use `--verbose` to see all configuration details
- Start with overview mode to understand the system
- Use service-specific mode to focus on one service
- Add `--cluster` context to see deployment-specific settings
- Check service-specific overrides to understand customizations