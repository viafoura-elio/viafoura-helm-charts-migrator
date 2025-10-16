# Inspect Command - Real Examples

This document shows actual output from the `inspect` command using the current configuration.

## 1. Overview Mode - All Services

```bash
$ go run main.go inspect
```

**Output:**
```
=== Service Configuration Overview ===

Account: production (1 cluster)
  ├─ prod01: enabled - 1 namespace

Account: testdev (1 cluster)
  ├─ dev01: enabled (default) - 7 namespaces

Total: 2 clusters, 8 namespaces

=== Services ===

  ✗ disabled auth-service (alias: auth)
  ✗ disabled auth0-oidc-demo (alias: auth0)
  ✗ disabled comment-import (alias: cimport)
  ✗ disabled console (alias: cons)
  ✗ disabled console-moderation (alias: cm)
  ✗ disabled data-burrito (alias: databur)
  ✗ disabled email (alias: mail)
  ✗ disabled flume (alias: flm)
  ✗ disabled gdpr-mediation (alias: gdprmed)
  ✓ enabled heimdall (alias: hmd)
  ✗ disabled ingestor (alias: ing)
  ✗ disabled legacy-gdpr-connector (alias: leggdprc)
  ✗ disabled livechat (alias: chat)
  ✗ disabled livecomments (alias: lc)
  ✗ disabled livequestions (alias: lq)
  ✗ disabled livereviews (alias: lr)
  ✗ disabled livestories (alias: ls)
  ✗ disabled moderation-orchestrator (alias: modorc)
  ✗ disabled polls (alias: poll)
  ✗ disabled realtime-event-feed (alias: ref)
  ✗ disabled spam-moderation (alias: spam-mod)
  ✗ disabled tyrion (alias: tyr)
  ✗ disabled ucs-moderation (alias: usc-mod)
  ✗ disabled user-import (alias: uimp)
  ✗ disabled user-interaction (alias: user-int)
  ✗ disabled user-notification (alias: usn)
  ✗ disabled webhooks (alias: wh)
  ✗ disabled webhooks-client (alias: whcli)

Total: 28 services (1 enabled, 27 disabled)

=== Global Configuration ===

Pipeline: disabled
Converter: skipJavaProperties=false, skipUppercaseKeys=false, minUppercaseChars=0
Performance: maxConcurrentServices=0, showProgress=false
SOPS: disabled

💡 Tip: Use --service <name> to inspect specific service configuration
💡 Tip: Use --verbose to see detailed configuration layers
```

**Insights:**
- Shows all 28 configured services
- Only 1 service (heimdall) is currently enabled
- 2 accounts configured: production and testdev
- 2 clusters across all accounts (prod01, dev01)
- 8 total namespaces available

## 2. Service-Specific Inspection

```bash
$ go run main.go inspect --service heimdall
```

**Output:**
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

🗺️  Mappings Configuration:

🔐 Secrets Configuration:
  Status: disabled

💉 Auto-Inject Rules:
  • Pattern: values.yaml
    Rules: 1
  • Pattern: envs/dev01/*/values.yaml
    Rules: 1

💡 Tip: Use --verbose to see detailed configuration values
```

**Insights:**
- Service metadata clearly displayed
- Configuration is largely using global defaults
- Has auto-inject rules configured
- Secrets are disabled for this service

## 3. Verbose Mode - Detailed Configuration

```bash
$ go run main.go inspect --service heimdall --verbose
```

**Output:**
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
  2. Service-specific overrides (23 differences)

🔍 Service Overrides:
  • mappings: merged with global mappings
  • secrets.patterns: added .*\.password.* from global
  • secrets.patterns: added .*\.secret.* from global
  • secrets.patterns: added .*jwt\.secret.* from global
  • secrets.patterns: added .*\.key$ from global
  • secrets.patterns: added .*\.token.* from global
  • secrets.patterns: added .*client_secret.* from global
  • secrets.patterns: added .*api_key.* from global
  • secrets.patterns: added .*private_key.* from global
  • secrets.patterns: added .*signing_key.* from global
  • secrets.patterns: added .*encryption_key.* from global
  • secrets.patterns: added .*\.auth\.key.* from global
  • secrets.patterns: added .*\.auth\.token.* from global
  • secrets.patterns: added .*\.auth\.secret.* from global
  • secrets.uuids: added pattern .*client.*uuid.* from global
  • secrets.uuids: added pattern [0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12} from global
  • secrets.values: added pattern ^[A-Za-z0-9+/]{40,}={0,2}$ from global
  • secrets.values: added pattern ^eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$ from global
  • secrets.values: added pattern ^[A-Fa-f0-9]{32,}$ from global
  • migration.baseValuesPath: global=**/values.yaml, service=
  • migration.envValuesPattern: global=**/envs/{cluster}/{environment}/{namespace}/values.yaml, service=
  • migration.legacyValuesFilename: global=legacy-values.yaml, service=
  • migration.helmValuesFilename: global=values.yaml, service=

🗺️  Mappings Configuration:
  • Normalizer: 8 patterns
  • Transform: 1 rule
  • Cleaner: 8 key patterns

🔐 Secrets Configuration:
  Status: disabled

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

**Insights:**
- Shows all 23 configuration differences between service and global
- Reveals exactly which patterns are inherited from global config
- Shows detailed auto-inject rules with keys, values, and conditions
- Lists all mapping configurations (normalizer, transform, cleaner)

## 4. Cluster Context - Deployment Information

```bash
$ go run main.go inspect --service heimdall --cluster prod01
```

**Output:**
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
  2. Service-specific overrides (23 differences)

🗺️  Mappings Configuration:
  • Normalizer: 8 patterns
  • Transform: 1 rule
  • Cleaner: 8 key patterns

🔐 Secrets Configuration:
  Status: disabled

💉 Auto-Inject Rules:
  • Pattern: values.yaml
    Rules: 1
  • Pattern: envs/dev01/*/values.yaml
    Rules: 1

=== Deployment Context: prod01 ===

Source: k8s1.cc
Target: prod01
AWS Profile: production-sre
AWS Region: us-east-1
Default Namespace: viafoura

Enabled Namespaces (1):
  • viafoura (default)

💡 Tip: Use --verbose to see detailed configuration values
```

**Insights:**
- Adds deployment context section at the end
- Shows cluster connection details (source, target)
- Displays AWS configuration (profile, region)
- Lists all enabled namespaces with default marked
- Useful for understanding where service will be deployed

## 5. Combined: Verbose + Cluster Context

```bash
$ go run main.go inspect --service heimdall --cluster prod01 --verbose
```

This combines both verbose output and cluster context, showing:
- All configuration differences in detail
- Complete auto-inject rules with values
- Full deployment context information

## Use Case Examples

### Pre-Migration Checklist

```bash
# 1. Check overall system configuration
go run main.go inspect

# 2. Verify service is enabled
go run main.go inspect --service heimdall

# 3. Check deployment target
go run main.go inspect --service heimdall --cluster prod01

# 4. Review all configuration details
go run main.go inspect --service heimdall --verbose
```

### Debugging Configuration Issues

```bash
# Compare what's inherited vs overridden
go run main.go inspect --service heimdall --verbose | grep "overrides"

# Check secrets configuration
go run main.go inspect --service heimdall --verbose | grep -A 20 "Secrets Configuration"

# Verify cluster settings
go run main.go inspect --service heimdall --cluster dev01
```

### Configuration Auditing

```bash
# List all services and their status
go run main.go inspect | grep "disabled\|enabled"

# Check which services have custom configurations
go run main.go inspect --verbose | grep "Service-specific overrides"

# Review global settings
go run main.go inspect | grep -A 20 "Global Configuration"
```

## Command Reference Quick Guide

| Command | Purpose |
|---------|---------|
| `inspect` | Overview of all services and clusters |
| `inspect -s <service>` | Focus on specific service |
| `inspect -s <service> --verbose` | Detailed service configuration |
| `inspect -s <service> -c <cluster>` | Add deployment context |
| `inspect -s <service> -c <cluster> --verbose` | Complete detailed view |

## Tips

1. **Start broad, then narrow**: Begin with overview, then drill down to specific services
2. **Use verbose for debugging**: When something isn't working, `--verbose` shows inheritance chain
3. **Cluster context for deployment planning**: Add `-c` flag to see where service will deploy
4. **Grep for specific sections**: Pipe output to grep to focus on particular configuration areas
5. **Compare services**: Run inspect on multiple services to understand configuration patterns