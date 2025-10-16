# Inspect Command - Quick Start Guide

## What is it?

The `inspect` command shows you the hierarchical configuration structure in a human-readable format. Think of it as a "configuration explorer" that helps you understand how your Helm chart migration is set up.

## Quick Commands

```bash
# See everything at a glance
helm-charts-migrator inspect

# Focus on one service
helm-charts-migrator inspect --service heimdall

# See all the details
helm-charts-migrator inspect --service heimdall --verbose

# Include deployment info
helm-charts-migrator inspect --service heimdall --cluster prod01
```

## When to Use

| Situation | Command |
|-----------|---------|
| 🔍 "What services are configured?" | `inspect` |
| 📊 "Is my service set up correctly?" | `inspect -s <service>` |
| 🔬 "Why is this value being used?" | `inspect -s <service> --verbose` |
| 🎯 "Where will this deploy?" | `inspect -s <service> -c <cluster>` |
| ⚙️ "What are the global settings?" | `inspect` (shows at bottom) |

## Reading the Output

### Symbols

- ✓ = enabled
- ✗ = disabled
- 📋 = configuration sources
- 🔍 = detailed overrides
- 🗺️ = mappings
- 🔐 = secrets
- 💉 = auto-inject rules
- 💡 = helpful tips

### Status Markers

- `(default)` = default cluster/namespace
- `(alias: xyz)` = service alias for commands

## Common Workflows

### Before Migration

```bash
# 1. Check system is configured
inspect

# 2. Verify target service exists and is enabled
inspect -s heimdall

# 3. Review deployment target
inspect -s heimdall -c prod01
```

### Troubleshooting

```bash
# Service not behaving as expected?
inspect -s heimdall --verbose

# Want to see where values come from?
inspect -s heimdall --verbose | grep "overrides"

# Check secrets configuration
inspect -s heimdall | grep -A 10 "Secrets"
```

### Configuration Review

```bash
# List all services
inspect | grep -E "enabled|disabled"

# Check specific service details
inspect -s <service> --verbose

# Compare with different cluster
inspect -s <service> -c dev01
```

## Tips & Tricks

1. **Pipe to less for easy scrolling**
   ```bash
   inspect -s heimdall --verbose | less
   ```

2. **Save output for later**
   ```bash
   inspect -s heimdall --verbose > heimdall-config.txt
   ```

3. **Search for specific configuration**
   ```bash
   inspect -s heimdall | grep "secrets"
   ```

4. **Compare services**
   ```bash
   inspect -s heimdall > heimdall.txt
   inspect -s tyrion > tyrion.txt
   diff heimdall.txt tyrion.txt
   ```

## Flags Reference

| Flag | Short | Description |
|------|-------|-------------|
| `--service` | `-s` | Service name |
| `--cluster` | `-c` | Cluster name |
| `--namespace` | `-n` | Namespace |
| `--verbose` | - | Show details |
| `--format` | `-f` | Output format |
| `--config` | - | Config file path |

**Note**: No `-v` shorthand for verbose (conflicts with log verbosity)

## Understanding Output Sections

### Overview Mode
- Account/cluster hierarchy
- Service list with status
- Global configuration summary

### Service Mode
- Service metadata
- Configuration sources
- Mappings configuration
- Secrets configuration
- Auto-inject rules
- Deployment context (with -c flag)

### Verbose Mode
All of the above PLUS:
- Detailed configuration differences
- Specific pattern values
- Complete rule definitions
- Value comparison (global vs service)

## Examples

See [inspect-command-examples.md](./inspect-command-examples.md) for detailed real-world examples with actual output.

## Help

```bash
helm-charts-migrator inspect --help
```

## Related Commands

- `migrate` - Actually run the migration
- `validate` - Validate chart configurations
- `init` - Initialize configuration file