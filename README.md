# Helm Charts Migrator

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/Version-1.0.0-brightgreen.svg)](https://github.com/yourusername/helm-charts-migrator/releases)

A powerful CLI tool for migrating Helm charts between Kubernetes clusters with advanced transformation capabilities, secret management, and parallel processing support.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [CLI Commands](#cli-commands)
  - [init](#init-command)
  - [migrate](#migrate-command)
  - [validate](#validate-command)
  - [secrets](#secrets-command)
  - [template](#template-command)
  - [version](#version-command)
- [Advanced Usage](#advanced-usage)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## Features

### Core Capabilities
- 🚀 **Parallel Processing** - Migrate multiple services concurrently with configurable worker pools
- 🔄 **Intelligent Transformation** - Automatic key normalization, camelCase conversion, and value transformations
- 🔐 **Secret Management** - SOPS integration for secure secret encryption/decryption
- 📊 **Hierarchical Configuration** - Override chain: defaults → globals → cluster → environment → namespace → service
- 🏗️ **Template Rendering** - Local chart rendering with multiple values files support
- ✅ **Comprehensive Validation** - Helm template validation, Kubernetes manifest checking, deprecated API detection
- 💾 **Smart Caching** - Intelligent caching with disk persistence for faster migrations
- 📝 **Detailed Reporting** - Transformation reports, extraction summaries, and migration logs

### Key Benefits
- **Production-Ready** - Battle-tested with 30+ microservices in production environments
- **Extensible** - Plugin architecture for custom transformers and validators
- **Observable** - Comprehensive metrics and structured logging
- **Resilient** - Retry logic with exponential backoff for transient failures

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/helm-charts-migrator.git
cd helm-charts-migrator

# Build the binary
go build -o helm-charts-migrator .

# Optional: Install globally
sudo mv helm-charts-migrator /usr/local/bin/

# Or install with go install
go install github.com/yourusername/helm-charts-migrator@latest
```

### Build with Version Information

```bash
# Build with version info
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "\
  -X main.Version=$VERSION \
  -X main.GitCommit=$COMMIT \
  -X main.BuildDate=$DATE" \
  -o helm-charts-migrator .
```

### Prerequisites

- **Go** 1.21 or higher
- **kubectl** configured with access to source and target clusters
- **Helm** 3.x installed
- **AWS CLI** configured (if using SOPS with AWS KMS)
- **SOPS** installed for secret encryption/decryption

## Quick Start

### 1. Initialize Configuration

```bash
# Create a default configuration file
helm-charts-migrator init

# Output:
# ✅ Configuration file created successfully!
# 
# 📝 Next steps:
#   1. Edit the configuration file to match your environment:
#      $ vi config.yaml
#   2. Update cluster contexts (dev01, prod01) to match your Kubernetes clusters
#   3. Configure services you want to migrate
#   4. Run migration:
#      $ helm-charts-migrator migrate

# Create config in custom location
helm-charts-migrator init --output /etc/migrator/config.yaml

# Force overwrite existing config
helm-charts-migrator init --force
```

### 2. Configure Your Environment

Edit the generated `config.yaml`:

```yaml
# Example: Configure your clusters
clusters:
  dev01:
    source: "old-dev-cluster"      # kubectl context for source cluster
    target: "new-dev-cluster"      # kubectl context for target cluster
    aws_profile: "dev-profile"     # AWS profile for this environment
    enabled: true
    
  prod01:
    source: "old-prod-cluster"
    target: "new-prod-cluster"
    aws_profile: "prod-profile"
    enabled: true
    default: true                  # Default cluster for operations

# Example: Configure services to migrate
services:
  api-gateway:
    enabled: true
    capitalized: "ApiGateway"      # Used for chart naming
    
  auth-service:
    enabled: true
    capitalized: "AuthService"
    secrets:                       # Service-specific secret patterns
      patterns:
        - ".*oauth.*"
        - ".*jwt.*"
```

### 3. Run Migration

```bash
# Migrate all enabled services across all clusters
helm-charts-migrator migrate

# Migrate specific service
helm-charts-migrator migrate --services api-gateway

# Migrate to specific cluster only
helm-charts-migrator migrate --cluster prod01

# Dry run to preview changes
helm-charts-migrator migrate --dry-run

# Migration with verbose output for debugging
helm-charts-migrator migrate --services heimdall -v 2

# Force cache refresh and migrate
helm-charts-migrator migrate --cleanup-cache
```

## Output Structure

After migration, the tool creates the following directory structure:

```
apps/
├── [service-name]/
│   ├── Chart.yaml                 # Helm chart metadata
│   ├── values.yaml                # Base values
│   ├── helm-values.yaml           # Values from default cluster
│   ├── templates/                 # Chart templates
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ...
│   └── envs/
│       ├── dev01/
│       │   ├── default/
│       │   │   ├── values.yaml        # Environment values
│       │   │   ├── secrets.enc.yaml   # SOPS encrypted secrets
│       │   │   ├── legacy-values.yaml # Original values (camelCase)
│       │   │   └── manifest.yaml      # Kubernetes manifest
│       │   └── [namespace]/
│       │       └── ...
│       └── prod01/
│           └── ...
```

## Configuration

### Configuration Structure

The configuration file supports hierarchical overrides with the following precedence (highest to lowest):

1. **Service-specific** configuration
2. **Namespace** configuration  
3. **Environment** configuration
4. **Cluster** configuration
5. **Global** configuration
6. **Default** configuration

### Key Configuration Sections

#### Global Settings

```yaml
globals:
  # CamelCase converter settings
  converter:
    skipJavaProperties: true      # Preserve "root.properties" style keys
    skipUppercaseKeys: true       # Skip keys like AWS_REGION
    minUppercaseChars: 3          # Min consecutive uppercase to skip
    
  # Performance tuning
  performance:
    maxConcurrentServices: 10     # Parallel service processing
    showProgress: true             # Display progress indicators
    
  # SOPS encryption
  sops:
    enabled: true
    awsProfile: "production-sre"
    parallelWorkers: 5
    timeout: 30
    
  # Secret detection patterns
  secrets:
    locations:
      base_path: "secrets"
      additional_paths: ["auth", "database"]
      scan_mode: "targeted"       # all, targeted, or filtered
    patterns:
      - ".*[Pp]assword.*"
      - ".*[Ss]ecret.*"
      - ".*[Tt]oken.*"
```

#### Transformation Rules

```yaml
globals:
  mappings:
    # Key normalization
    normalizer:
      enabled: true
      patterns:
        '(^|\.)[Rr]eplicas$': 'replicaCount'
        '[Aa]utoscaling.[Mm]ax': 'autoscaling.maxReplicas'
        
    # Ingress to host extraction
    transform:
      enabled: true
      rules:
        ingress:
          source_regex: '^ingress$'
          target_key: 'hosts'
          extract_hosts: true
```

#### Auto-Injection

```yaml
globals:
  autoInject:
    "values.yaml":
      keys:
        - key: 'global.imageRegistry'
          value: "registry.example.com"
          condition: "ifNotExists"    # always, ifExists, ifNotExists, disabled
          description: "Set default registry"
```

### Environment Variables

All configuration values can be overridden using environment variables with the `MIGRATOR_` prefix:

```bash
# Override cluster
export MIGRATOR_CLUSTER=prod01

# Override AWS profile
export MIGRATOR_AWS_PROFILE=production

# Run migration with overrides
helm-charts-migrator migrate
```

## CLI Commands

### init Command

Initialize a new configuration file with sensible defaults.

```bash
# Basic usage
helm-charts-migrator init

# Custom output location
helm-charts-migrator init --output /etc/migrator/config.yaml

# Force overwrite existing
helm-charts-migrator init --force

# Examples with output
$ helm-charts-migrator init
I0915 10:30:00.123456   12345 logger.go:67] "[init] Configuration file created successfully" path="./config.yaml"
✅ Configuration file created successfully!

📝 Next steps:
  1. Edit the configuration file to match your environment:
     $ vi ./config.yaml
  2. Update cluster contexts (dev01, prod01) to match your Kubernetes clusters
  3. Configure services you want to migrate
  4. Run migration:
     $ helm-charts-migrator migrate
```

### migrate Command

The primary command for migrating Helm charts between clusters.

#### Basic Usage

```bash
# Migrate all enabled services to all enabled clusters
helm-charts-migrator migrate

# Dry run - preview without making changes
helm-charts-migrator migrate --dry-run

# Migrate specific services
helm-charts-migrator migrate --services api-gateway,auth-service

# Migrate to specific cluster
helm-charts-migrator migrate --cluster prod01

# Combine filters
helm-charts-migrator migrate --cluster dev01 --services api-gateway --dry-run
```

#### Advanced Options

```bash
# Skip SOPS encryption
helm-charts-migrator migrate --no-sops

# Custom cache directory
helm-charts-migrator migrate --cache-dir /tmp/migration-cache

# Force cache refresh
helm-charts-migrator migrate --cleanup-cache

# Specify AWS profile for SOPS
helm-charts-migrator migrate --aws-profile production-sre

# Custom source and target paths
helm-charts-migrator migrate \
  --source /path/to/legacy/charts \
  --target /path/to/new/charts \
  --base /path/to/base-chart
```

#### Migration Process

The migration command performs these steps for each service:

1. **Connect to Source Cluster** - Switches kubectl context to source cluster
2. **Fetch Helm Releases** - Retrieves all releases for the service
3. **Cache Release Values** - Saves values to disk in cache directory
4. **Copy Base Chart** - Copies template with "base-chart" → service name replacement
5. **Extract & Transform Values** - Applies camelCase conversion and normalization
6. **Copy Default Values** - Copies helm-values.yaml from default cluster
7. **Extract Secrets** - Identifies sensitive values using patterns
8. **Encrypt with SOPS** - Encrypts secrets using AWS KMS
9. **Generate Reports** - Creates transformation summary

#### Example Output

```bash
$ helm-charts-migrator migrate --services api-gateway --cluster prod01

I0915 10:35:00.123456   12345 migrator.go:89] "Starting migration" cluster="prod01" service="api-gateway"
I0915 10:35:01.234567   12345 kubernetes.go:45] "Connected to cluster" context="old-prod-cluster"
I0915 10:35:02.345678   12345 helm.go:67] "Found Helm releases" count=3
I0915 10:35:03.456789   12345 cache.go:34] "Caching release values" release="api-gateway-prod"
I0915 10:35:04.567890   12345 transformer.go:78] "Applying transformations" 
  ├─ Key normalization: 45 keys converted
  ├─ CamelCase conversion: 23 keys updated
  └─ Host extraction: 3 hosts extracted from ingress
I0915 10:35:05.678901   12345 secrets.go:89] "Extracting secrets" found=12
I0915 10:35:06.789012   12345 sops.go:45] "Encrypting secrets" file="secrets.enc.yaml"
I0915 10:35:07.890123   12345 migrator.go:234] "Migration completed" duration="7.8s"

✅ Migration Summary:
  Service: api-gateway
  Cluster: prod01
  Values extracted: 156 keys
  Secrets found: 12
  Transformations applied: 71
  Files created:
    - apps/api-gateway/Chart.yaml
    - apps/api-gateway/values.yaml
    - apps/api-gateway/envs/prod01/values.yaml
    - apps/api-gateway/envs/prod01/secrets.enc.yaml
    - apps/api-gateway/envs/prod01/legacy-values.yaml
```

### validate Command

Validate migrated Helm charts for correctness and compatibility.

```bash
# Validate all migrated services
helm-charts-migrator validate

# Validate specific service
helm-charts-migrator validate --service api-gateway

# Validate for specific cluster
helm-charts-migrator validate --cluster prod01

# Strict validation (fail on warnings)
helm-charts-migrator validate --strict

# Validate with specific Kubernetes version
helm-charts-migrator validate --kube-version 1.28

# Validate secrets decryption
helm-charts-migrator validate --validate-secrets
```

#### Validation Checks

1. **Helm Template Rendering** - Ensures charts render without errors
2. **Kubernetes Manifest Validation** - Validates against Kubernetes schemas
3. **Deprecated API Detection** - Identifies usage of deprecated APIs
4. **Secret Validation** - Verifies SOPS encryption/decryption
5. **Resource Quotas** - Checks resource limits and requests
6. **Security Policies** - Validates PodSecurityPolicies and NetworkPolicies

#### Example Output

```bash
$ helm-charts-migrator validate --service api-gateway --cluster prod01

🔍 Validating: api-gateway (prod01)
  ✅ Helm template rendering: PASSED
  ✅ Kubernetes manifest validation: PASSED
  ⚠️  Deprecated APIs: 1 warning
      - Ingress uses networking.k8s.io/v1beta1 (use networking.k8s.io/v1)
  ✅ Secret validation: PASSED (12 secrets validated)
  ✅ Resource quotas: PASSED
  ✅ Security policies: PASSED

📊 Validation Summary:
  Total services validated: 1
  Passed: 1
  Warnings: 1
  Failed: 0
  
⚠️  Run with --strict to treat warnings as errors
```

### secrets Command

Manage secrets extraction, encryption, and decryption using SOPS.

#### Subcommands

```bash
# Extract secrets from values
helm-charts-migrator secrets extract --input values.yaml --output secrets.yaml

# Encrypt secrets file
helm-charts-migrator secrets encrypt secrets.yaml

# Decrypt secrets file
helm-charts-migrator secrets decrypt secrets.enc.yaml

# Re-encrypt with different key
helm-charts-migrator secrets rotate --old-profile old-kms --new-profile new-kms

# Validate encrypted files
helm-charts-migrator secrets validate secrets.enc.yaml
```

#### Extract Secrets

```bash
# Extract with custom patterns
helm-charts-migrator secrets extract \
  --input values.yaml \
  --output secrets.yaml \
  --patterns ".*password.*,.*token.*,.*key.*"

# Extract with confidence threshold
helm-charts-migrator secrets extract \
  --input values.yaml \
  --min-confidence 0.8

# Service-specific extraction
helm-charts-migrator secrets extract \
  --service api-gateway \
  --cluster prod01
```

#### Encrypt/Decrypt

```bash
# Encrypt with AWS KMS
helm-charts-migrator secrets encrypt secrets.yaml \
  --aws-profile production \
  --kms-key arn:aws:kms:us-east-1:123456789:key/abc-123

# Encrypt all secrets in directory
helm-charts-migrator secrets encrypt \
  --path ./apps/*/envs/*/secrets.yaml \
  --parallel 10

# Decrypt for viewing
helm-charts-migrator secrets decrypt secrets.enc.yaml \
  --aws-profile production

# Decrypt to stdout (for piping)
helm-charts-migrator secrets decrypt secrets.enc.yaml --output -
```

#### Example Output

```bash
$ helm-charts-migrator secrets extract --input values.yaml

🔍 Analyzing values.yaml for secrets...

📊 Secret Detection Results:
  Total keys analyzed: 245
  Secrets detected: 18
  Confidence levels:
    High (0.9-1.0): 12
    Medium (0.7-0.9): 4
    Low (0.5-0.7): 2

🔐 Extracted Secrets:
  database.password: ****** (confidence: 0.95)
  auth.clientSecret: ****** (confidence: 0.92)
  jwt.signingKey: ****** (confidence: 0.90)
  redis.authToken: ****** (confidence: 0.88)
  ...

✅ Secrets written to: secrets.yaml
⚠️  Remember to encrypt this file before committing!

# Encrypt the extracted secrets
$ helm-charts-migrator secrets encrypt secrets.yaml
✅ Encrypted: secrets.enc.yaml
🔐 Encryption key: arn:aws:kms:us-east-1:123456789:key/abc-123
```

### template Command

Render chart templates locally for testing and debugging.

```bash
# Basic template rendering
helm-charts-migrator template ./charts/api-gateway

# With custom values
helm-charts-migrator template ./charts/api-gateway \
  --values custom-values.yaml \
  --set image.tag=v2.0.0

# Render specific template
helm-charts-migrator template ./charts/api-gateway \
  --show-only templates/deployment.yaml

# Output to directory
helm-charts-migrator template ./charts/api-gateway \
  --output-dir ./rendered \
  --split  # Separate file per resource

# With release name and namespace
helm-charts-migrator template ./charts/api-gateway \
  --release api-gateway-prod \
  --namespace production
```

#### Advanced Template Options

```bash
# Multiple values files (last wins)
helm-charts-migrator template ./charts/api-gateway \
  --values base-values.yaml \
  --values prod-values.yaml \
  --values secrets.dec.yaml

# Validate against Kubernetes version
helm-charts-migrator template ./charts/api-gateway \
  --kube-version 1.28 \
  --validate

# Debug template rendering
helm-charts-migrator template ./charts/api-gateway \
  --debug \
  --dry-run
```

#### Example Output

```bash
$ helm-charts-migrator template ./charts/api-gateway \
    --values prod-values.yaml \
    --split \
    --output-dir ./rendered

📦 Rendering templates for: api-gateway
  Using values from: prod-values.yaml
  Target directory: ./rendered

✅ Templates rendered successfully:
  - deployment.yaml (1024 lines)
  - service.yaml (45 lines)
  - ingress.yaml (67 lines)
  - configmap.yaml (234 lines)
  - serviceaccount.yaml (12 lines)
  - horizontalpodautoscaler.yaml (34 lines)

📊 Rendering Summary:
  Total templates: 6
  Total resources: 8
  Largest file: deployment.yaml
  Total size: 42KB
```

### version Command

Display version information and build details.

```bash
# Show version
helm-charts-migrator version

# Output:
# Helm Charts Migrator
# Version: 1.0.0
# Git Commit: abc123def456
# Built: 2024-09-15T10:30:00Z
# Go Version: go1.21.5
# OS/Arch: darwin/amd64

# JSON output for scripting
helm-charts-migrator version --json
# {"version":"1.0.0","commit":"abc123def456","built":"2024-09-15T10:30:00Z","go":"go1.21.5","os":"darwin","arch":"amd64"}

# Short version only
helm-charts-migrator version --short
# v1.0.0
```

## Advanced Usage

### Parallel Migration

Migrate multiple services concurrently for faster processing:

```yaml
# config.yaml
globals:
  performance:
    maxConcurrentServices: 20  # Increase worker pool size
```

```bash
# Migrate with parallel processing
helm-charts-migrator migrate --services svc1,svc2,svc3,svc4,svc5

# Monitor parallel execution
helm-charts-migrator migrate --services all -v 2
```

### Custom Transformers

Add custom transformation rules:

```yaml
# config.yaml
globals:
  mappings:
    transform:
      enabled: true
      rules:
        custom-rule:
          source_regex: '^old\.path\.(.+)'
          target_key: 'new.path.$1'
          transformer: 'myCustomTransformer'
```

### Service-Specific Overrides

Configure per-service behavior:

```yaml
# config.yaml
services:
  api-gateway:
    enabled: true
    capitalized: "ApiGateway"
    # Override global settings for this service
    converter:
      minUppercaseChars: 5
    secrets:
      patterns:
        - ".*oauth.*"
        - ".*jwt.*"
    autoInject:
      "values.yaml":
        keys:
          - key: "image.repository"
            value: "registry.example.com/api-gateway"
            condition: "always"
```

### Multi-Cluster Migration

Migrate across multiple clusters with different configurations:

```bash
# Migrate to all clusters
helm-charts-migrator migrate --services api-gateway

# Selective cluster migration
for cluster in dev01 staging01 prod01; do
  helm-charts-migrator migrate \
    --cluster $cluster \
    --services api-gateway \
    --aws-profile $cluster-profile
done
```

### CI/CD Integration

#### GitLab CI Example

```yaml
# .gitlab-ci.yml
migrate-charts:
  stage: migrate
  image: golang:1.21
  script:
    - helm-charts-migrator init --force
    - helm-charts-migrator migrate --dry-run
    - helm-charts-migrator validate --strict
    - helm-charts-migrator migrate
  artifacts:
    paths:
      - apps/
    expire_in: 1 week
```

#### GitHub Actions Example

```yaml
# .github/workflows/migrate.yml
name: Migrate Helm Charts
on:
  push:
    branches: [main]

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Build migrator
        run: go build -o helm-charts-migrator .
        
      - name: Initialize config
        run: ./helm-charts-migrator init --force
        
      - name: Run migration
        run: |
          ./helm-charts-migrator migrate \
            --cluster ${{ matrix.cluster }} \
            --aws-profile ${{ secrets.AWS_PROFILE }}
            
      - name: Validate charts
        run: ./helm-charts-migrator validate --strict
```

### Debugging and Troubleshooting

#### Verbose Logging

```bash
# Increase verbosity levels
helm-charts-migrator migrate -v 1  # Info level
helm-charts-migrator migrate -v 2  # Debug level
helm-charts-migrator migrate -v 3  # Trace level
helm-charts-migrator migrate -v 4  # Deep trace

# File-specific debugging
helm-charts-migrator migrate --vmodule=migrator=3,transformer=4

# Log to file
helm-charts-migrator migrate \
  --logtostderr=false \
  --log_dir=/tmp/migrator-logs
```

#### Cache Management

```bash
# View cache contents
ls -la .cache/

# Clear cache for specific cluster
rm -rf .cache/prod01/

# Force cache refresh
helm-charts-migrator migrate --cleanup-cache

# Use different cache directory
helm-charts-migrator migrate --cache-dir /tmp/migrator-cache
```

#### Dry Run Analysis

```bash
# Preview all changes without applying
helm-charts-migrator migrate --dry-run -v 2 | tee migration-plan.log

# Analyze what would be migrated
helm-charts-migrator migrate --dry-run --services all | \
  grep "Would migrate" | wc -l
```

## Architecture

### Component Overview

```
┌─────────────────────┐
│   CLI Interface     │
├─────────────────────┤
│  Command Handlers   │
├─────────────────────┤
│  Migration Factory  │
├─────────────────────┤
│     Services        │
├─────────────────────┤
│ Kubernetes │  Helm  │
│   Client   │  SDK   │
└─────────────────────┘
```

### Core Services

- **ServiceMigrator** - Orchestrates the complete migration process
- **ReleaseCache** - Caches Helm releases with disk persistence
- **ChartCopier** - Copies base-chart with template replacement
- **ValuesExtractor** - Extracts and transforms configuration values
- **FileManager** - Handles YAML file operations
- **ClusterManager** - Manages multi-cluster operations
- **SecretDetector** - Identifies sensitive values using patterns
- **SOPSService** - Encrypts/decrypts secrets with AWS KMS

### Parallel Processing

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Worker 1    │     │  Worker 2    │     │  Worker N    │
├──────────────┤     ├──────────────┤     ├──────────────┤
│  Service A   │     │  Service B   │     │  Service C   │
└──────────────┘     └──────────────┘     └──────────────┘
       ↓                    ↓                    ↓
┌────────────────────────────────────────────────────────┐
│                    Task Queue                          │
└────────────────────────────────────────────────────────┘
```

## Troubleshooting

### Common Issues

#### 1. Config File Not Found

```bash
Error: config file not found at ./config.yaml

Solution:
# Initialize configuration
helm-charts-migrator init

# Or specify config path
helm-charts-migrator migrate --config /path/to/config.yaml
```

#### 2. Cluster Connection Failed

```bash
Error: failed to connect to cluster: context "old-prod-cluster" not found

Solution:
# Check available contexts
kubectl config get-contexts

# Update config.yaml with correct context names
# Or set context explicitly
kubectl config use-context correct-context-name
```

#### 3. SOPS Encryption Failed

```bash
Error: failed to encrypt secrets: no KMS key configured

Solution:
# Configure AWS profile
export AWS_PROFILE=production-sre

# Or specify in command
helm-charts-migrator secrets encrypt file.yaml --aws-profile production-sre

# Or add to config.yaml
globals:
  sops:
    awsProfile: "production-sre"
```

#### 4. Cache Issues

```bash
Error: cached values not found for release

Solution:
# Clear and rebuild cache
helm-charts-migrator migrate --cleanup-cache

# Or remove cache directory
rm -rf .cache/
```

#### 5. Memory Issues with Large Migrations

```bash
Error: out of memory

Solution:
# Reduce parallel workers
globals:
  performance:
    maxConcurrentServices: 3  # Reduce from default

# Or migrate services individually
helm-charts-migrator migrate --services service1
helm-charts-migrator migrate --services service2
```

### Debug Commands

```bash
# Test cluster connectivity
kubectl cluster-info --context=your-context

# Verify Helm releases
helm list -A --kube-context=your-context

# Test SOPS
sops --version
aws kms describe-key --key-id your-key-id

# Check file permissions
ls -la config.yaml
ls -la .cache/

# Test configuration parsing
helm-charts-migrator validate --dry-run
```

### Performance Tuning

#### For Large Migrations (100+ services)

```yaml
# config.yaml
globals:
  performance:
    maxConcurrentServices: 20  # Increase workers
  cache:
    ttl: 3600                 # Cache for 1 hour
    maxSize: "10GB"           # Increase cache size
  sops:
    parallelWorkers: 10       # Parallel encryption
```

#### For Limited Resources

```yaml
# config.yaml
globals:
  performance:
    maxConcurrentServices: 2  # Reduce workers
    showProgress: false       # Disable progress bars
  cache:
    compression: true         # Enable compression
```

## Best Practices

### 1. Configuration Management

- Keep environment-specific configs in separate files
- Use version control for configuration files
- Encrypt sensitive configuration values
- Document custom transformation rules

### 2. Migration Strategy

- Always run dry-run first
- Migrate and validate in lower environments before production
- Keep backups of original values
- Use incremental migrations for large deployments

### 3. Security

- Never commit decrypted secrets
- Use different KMS keys per environment
- Rotate encryption keys regularly
- Audit secret access logs

### 4. Monitoring

- Enable verbose logging for production migrations
- Keep migration logs for audit trails
- Set up alerts for failed migrations
- Track migration metrics

## Real-World Examples

### Example: Migrating a Microservices Platform

```bash
# 1. Initialize and configure for production
helm-charts-migrator init
vi config.yaml  # Configure clusters and services

# 2. Test with a single service first
helm-charts-migrator migrate --services heimdall --cluster dev01 --dry-run

# 3. Migrate development environment
helm-charts-migrator migrate --cluster dev01

# 4. Validate migrated charts
helm-charts-migrator validate --cluster dev01 --strict

# 5. If validation passes, proceed to production
helm-charts-migrator migrate --cluster prod01 --aws-profile production-sre
```

### Example: Service-Specific Configuration

```yaml
# config.yaml - Real service configuration
services:
  heimdall:
    enabled: true
    capitalized: "Heimdall"
    secrets:
      patterns:
        - ".*oauth.*"
        - ".*client[Ss]ecret.*"
      locations:
        base_path: "auth"
        scan_mode: "targeted"
    
  livecomments:
    enabled: true
    capitalized: "LiveComments"
    converter:
      skipJavaProperties: true  # Preserve Java-style config keys
    autoInject:
      "values.yaml":
        keys:
          - key: "datadog.enabled"
            value: true
            condition: "ifNotExists"
```

### Example: Handling Legacy Chart Structures

```bash
# For charts with non-standard structures
helm-charts-migrator migrate \
  --services legacy-service \
  --source /legacy/charts/path \
  --base /custom/base-chart \
  --cache-dir /tmp/legacy-cache
```

### Example: Batch Migration Script

```bash
#!/bin/bash
# migrate-all.sh - Migrate services in batches

SERVICES=(
  "heimdall"
  "livecomments"
  "auth-service"
  "api-gateway"
)

for service in "${SERVICES[@]}"; do
  echo "Migrating $service..."
  
  # Migrate with retries
  for attempt in 1 2 3; do
    if helm-charts-migrator migrate --services "$service"; then
      echo "✅ $service migrated successfully"
      break
    else
      echo "⚠️ Attempt $attempt failed for $service"
      sleep 5
    fi
  done
  
  # Validate immediately
  helm-charts-migrator validate --service "$service"
done
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone repository
git clone https://github.com/yourusername/helm-charts-migrator.git
cd helm-charts-migrator

# Install dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build .

# Run locally
./helm-charts-migrator --help
```

### Running Tests

```bash
# Unit tests
go test ./v1/pkg/...

# Integration tests
go test ./v1/tests/integration/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. ./v1/pkg/workers/
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [https://docs.example.com/helm-charts-migrator](https://docs.example.com/helm-charts-migrator)
- **Issues**: [GitHub Issues](https://github.com/yourusername/helm-charts-migrator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/helm-charts-migrator/discussions)
- **Slack**: [#helm-charts-migrator](https://example.slack.com/channels/helm-charts-migrator)

## Acknowledgments

- Helm community for the excellent Helm SDK
- SOPS project for secure secret management
- Kubernetes SIG for client libraries
- All contributors and users of this project

---

**Version**: 1.0.0 | **Last Updated**: September 2024 | **Status**: Production Ready