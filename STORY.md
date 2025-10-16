# Helm Chart Migrator

This is a Golang CLI tool to handle helm charts migration

## Assumption

We have on [viafoura-legacy-helm-charts]("/Volumes/Development/clients/viafoura/repos/_viafoura-elio/kubernetes-ops/viafoura/charts") a helm chart repository
with many out-of-standard stuffs with design to kubernetes 1.18.20 (KOPs Kubernetes Cluster "k8s1.xyz" for development and "k8s1.cc" for production)

Main problems:
* values.yaml using PascalCase
* Mixing ConfigMaps and Secrets
* Bad HPA organization
* Bad Service structure
* Use Ingress Nginx
* Old Api Version
* Bad Helm Charts Templates

## Proposal

The Proposal is migrating to a new helm chart structure:
We will be able to handle in a fashion way the rollouts using ArgoCD, ArgoRollouts, Istio

I have created a new [base-chart]("./migration/base-chart") that will work as base-chart to all services.

```yaml
clusters:
  # Production Cluster Configuration
  prod01:
    default: true
    default_namespace: viafoura
    enabled: true
    target: prod01
    source: k8s1.cc
    aws_profile: production-sre
    aws_region: us-east-1
    environments:
      production:
        enabled: true
        namespaces:
          viafoura:
            enabled: true
            name: viafoura

  # Development Cluster Configuration
  dev01:
    default: false
    default_namespace: vf-dev2
    enabled: false
    target: dev01
    source: k8s1.xyz
    awsProfile: test-sre
    awsRegion: us-east-1
    environments:
      development:
        enabled: true
        namespaces:
          vf-dev2:
            enabled: true
            name: vf-dev2
          vf-dev3:
            enabled: false
            name: vf-dev3
          vf-dev4:
            enabled: false
            name: vf-dev4
          vf-dev5:
            enabled: false
            name: vf-dev5
      test:
        enabled: true
        namespaces:
          vf-test2:
            enabled: true
            name: vf-test2
          vf-test3:
            enabled: false
            name: vf-test3
          vf-test4:
            enabled: false
            name: vf-test4

# Global patterns that apply to all services
globals:
  # Auto Inject Key Values Pairs
  autoInject:
    "values.yaml":
      keys:
        - key: 'secrets."root.properties"."9487e74c-2d27-4085-b637-30a82239b0b2"'
          value: misconfigured
          condition: disabled # ifExists, ifNotExists, always, disabled
          description: "Set Default Secret Value for 9487e74c-2d27-4085-b637-30a82239b0b2"
    "envs/dev01/*/values.yaml":
      keys:
        - key: 'configMap."root.properties"."auth.dataSource.user"'
          value: "{environment}-auth"
          condition: disabled # ifExists, ifNotExists, always, disabled
          description: "Set environment-specific auth datasource user"

  # Hierarchical mapping for accurate secrets extraction
  mappings:
    locations:
      scan_mode: filtered
      include: [ "configMap" ]

    normalizer:
      enabled: true
      description: "Global Normalizations Configuration"
      patterns:
        '[Cc]ontainer.[Pp]ort': 'service.targetPort'
        '[Aa]utoscaling.[Tt]arget.[Cc]pu': 'autoscaling.targetCPUUtilizationPercentage'
        '[Aa]utoscaling.[Tt]arget.[Mm]emory': 'autoscaling.targetMemoryUtilizationPercentage'
        '(^|\.)[Rr]eplicas$': 'replicaCount'
        '^[Ee]nv$': 'envVars'
        '^[Dd]eployment.maxSurge$': 'strategy.rollingUpdate.maxSurge'
        '^[Dd]eployment.maxUnavailable$': 'strategy.rollingUpdate.maxUnavailable'
        '^[Dd]eployment.iamRole$': 'serviceAccount.iamRole.name'
    transform:
      enabled: true
      description: "Global Transformations Configuration"
      rules:
        ingress_to_hosts:
          type: "ingress_to_hosts"
          source_path: "[Ii]ngress"
          target_path: "hosts.public.domains"
          description: "Extract valid hosts from ingress configurations and collect them into hosts.public.domains list"
    cleaner:
      enabled: true
      description: "Global Cleaner Configuration - Removes unwanted keys from values files"
      path_patterns:
        - "apps/**/legacy-values.yaml"
        - "apps/**/helm-values.yaml"
        - "apps/**/envs/**/*.yaml"
      key_patterns:
        - "^[Cc]anary$"
        - "^[Pp]odLabels$"
        - "^[Pp]odAnnotations$"
        - "^[Nn]ameOverride$"
        - "^[Ff]ullnameOverride$"
    extract:
      enabled: true
      description: "Global Extractions Configuration"
      patterns:

  # Hierarchical secrets mapping for accurate secrets extraction
  # Global patterns that apply to all services
  secrets:
    # Location configuration for targeted secret scanning
    locations:
      # Base path where secrets are typically located (defaults to "secrets")
      base_path: "secrets"

      # Additional specific paths where secrets might be found
      additional_paths:
        - "auth"
        - "database"
        - "configMap.data"
        - "app.secrets"

      # Path patterns for flexible secret detection
      path_patterns:
        - ".*\\.auth\\..*"
        - ".*\\.credentials\\..*"
        - ".*\\.env\\..*"

      # Scan mode: filtered focuses on secrets section + additional paths
      scan_mode: filtered

    # Common secret patterns across all services
    patterns:
      - ".*\\.password.*"
      - ".*\\.secret.*"
      - ".*jwt\\.secret.*"
      - ".*\\.key$"
      - ".*\\.token.*"
      # Removed overly broad auth pattern - replaced with specific ones below
      - ".*client_secret.*"
      - ".*api_key.*"
      - ".*private_key.*"
      - ".*signing_key.*"
      - ".*encryption_key.*"
      # More specific auth patterns to avoid false positives
      - ".*\\.auth\\.key.*"
      - ".*\\.auth\\.token.*"
      - ".*\\.auth\\.secret.*"

    # Global UUID patterns for client IDs and API keys
    uuids:
      - pattern: ".*client.*uuid.*"
        sensitive: true
        description: "Client UUIDs are typically sensitive identifiers"
      - pattern: "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
        sensitive: true
        description: "Generic UUID pattern - evaluate based on context"

    # Common sensitive values regardless of key name
    values:
      # Base64 encoded secrets (more restrictive to avoid file paths)
      - pattern: "^[A-Za-z0-9+/]{40,}={0,2}$"
        sensitive: true
        description: "Base64 encoded values are likely secrets (40+ chars, no slashes at start)"
      # JWT tokens (start with eyJ)
      - pattern: "^eyJ[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+$"
        sensitive: true
        description: "JWT-like tokens"
      - pattern: "^[A-Fa-f0-9]{32,}$"  # Hex encoded secrets
        sensitive: true
        description: "Long hex strings are likely secrets"

  # Migration Configuration
  migration:
    baseValuesPath: "**/values.yaml"
    envValuesPattern: "**/envs/{cluster}/{namespace}/values.yaml"
    legacyValuesFilename: "legacy-values.yaml"
    helmValuesFilename: "values.yaml"

# Services to Migrate
services:
  heimdall:
    enabled: true
    name: heimdall
    capitalized: Heimdall
    autoInject:
      "values.yaml":
        keys:
          - key: 'secrets."root.properties"."9487e74c-2d27-4085-b637-30a82239b0b2"'
            value: misconfigured
            condition: ifExists # ifExists, ifNotExists, always, disabled
            description: "Set Default Secret Value for 9487e74c-2d27-4085-b637-30a82239b0b2"
      "envs/dev01/*/values.yaml":
        keys:
          - key: 'configMap."root.properties"."auth.dataSource.user"'
            value: "{environment}-auth"
            condition: ifExists # ifExists, ifNotExists, always, disabled
            description: "Set environment-specific auth datasource user"
    mappings: {}
    migration: {}
    secrets:
      # These specific UUIDs are client secrets for heimdall
      keys:
        - "3f4beddd-2061-49b0-ae80-6f1f2ed65b37"
        - "682843b1-d3e0-460e-ab90-6556bc31470f"
        - "936da557-6daa-4444-92cc-161fc290c603"
        - "9487e74c-2d27-4085-b637-30a82239b0b2"
        - "c23203d0-1b8e-4208-92dc-85dc79e6226b"
      patterns:
        - ".*access.refresh.local_client_uuid.*"
        - ".*loginradius.*secret.*"
        - ".*oauth.*secret.*"
        - ".*provider.*secret.*"
        - ".*thirdparty.*apikey.*"
        - ".*thirdparty.parameter.loginradius.*"
      description: "Heimdall authentication service secrets"

  livecomments:
    enabled: true
    name: livecomments
    capitalized: Live Comments
    secrets:
      # Custom key mappings for livecomments
      # Maps configMap keys to different secret keys
      keyMappings:
        "configMap.application.properties": "secrets.application.conf"
        "configMap.database.properties": "secrets.db.conf"
      # Define the merge order for secrets
      # Files are processed in order, later files override earlier ones
      mergeOrder:
        - "apps/{service}/legacy-values.yaml"
        - "apps/{service}/helm-values.yaml"
```

## CLI Commands

### Migrate Command
Orchestrates the complete chart migration process:
```bash
./helm-charts-migrator migrate --config config.yaml --source /path/to/legacy --target apps/

# Options:
# --cluster/-c: Target specific cluster(s), omit to process all enabled clusters
# --services/-s: Migrate specific services only
# --dry-run/-d: Preview without execution
# --cache-dir: Custom cache location (defaults to .cache)
# --cleanup-cache: Force cache refresh
```

### Validate Command
Validates migrated charts:
```bash
./helm-charts-migrator validate --config config.yaml --cluster prod01 --service heimdall

# Options:
# --strict: Fail on warnings
# --validate-secrets: Also validate secret patterns
```

### Template Command
Renders charts locally for testing:
```bash
./helm-charts-migrator template /path/to/chart --release myapp --namespace production

# Options:
# --values/-f: Additional values files
# --set: Override specific values
# --output-dir: Save rendered templates
# --split: Save each resource to separate file
```

### Secrets Command
Manages secret extraction and SOPS encryption:
```bash
# Extract and encrypt secrets for all services
./helm-charts-migrator secrets

# Process specific path
./helm-charts-migrator secrets apps/heimdall

# Extract only (no encryption)
./helm-charts-migrator secrets --extract-only

# Encrypt existing .dec files
./helm-charts-migrator secrets --encrypt-only

# Decrypt existing .enc files  
./helm-charts-migrator secrets --decrypt-only

# Use custom patterns
./helm-charts-migrator secrets --pattern "*.dec.yaml" --pattern "secrets/*.yaml"

# Options:
# --services/-s: Process specific services only
# --aws-profile/-p: AWS profile for KMS operations (default: cicd-sre)
# --sops-config: Path to SOPS config file (default: .sops.yaml)
# --pattern: File patterns to process (can be specified multiple times)
```

## Migration Steps

The main destination path is built as: **"apps/{service}/envs/{cluster}/{environment}/{namespace}"**

Where:
**"apps"**: Is the flag value from **"&targetPath"**
**"{service}"**: Is the current processing service from the "@config.yaml" under services block/section
**"{cluster}"**: Is the **"clusters"** key from the "@config.yaml" under clusters block/section
**"{environment}"**: Is the **"clusters.{cluster}.environments"** environment key from the **"@config.yaml"** under clusters block/section
**"{namespace}"**: Is the **"clusters.{cluster}.environments.{environment}.namespaces"** namespace key from the **"@config.yaml"** under clusters block/section

**DON'T implement any fallback or backward compatibility for legacy/old versions.**

1. I want to copy the base-chart to new the to base-chart to the "v1/cmd/migrate.go" in the "target".
2. You should store the "pathTargetPath" because it will be used many times. Ensure the target path is: "{target}/{service}/".
3. I want to copy "base-chart" by the service name on "@config.yaml".
4. I want to replace "base-chart" by the current "service" name that is currently being processing by "@config.yaml".
5. I want to replace the "Base-Chart" by the current "service" capitalized name from the "capitalized" field in "@config.yaml" (e.g., for heimdall service, use "Heimdall" from services.heimdall.capitalized).
    - **Dashboard Detection**: If `dashboards/*.json` files exist after copying base-chart, automatically set `grafana.dashboards.enabled: true` in `{target}/{service}/values.yaml`
    - **Service Alias**: If service has an `alias` configured in `@config.yaml` (e.g., `services.livecomments.alias: lc`), automatically set `service.alias` in `{target}/{service}/values.yaml`
6. I want to copy from the source path (--source flag) the values.yaml and save with the name defined on "@config.yaml" => "globals.migration.legacyValuesFilename".
    1. Apply the "camelCase" conversion. However, I want to keep all "root.properties", I want to say Java Properties Style and Keys that Starts with 3 or more Uppercase characters
    2. Save it on "{target}/{service}/{legacyValuesFilename}" => "legacyValuesFilename": is available into "globals.migration.legacyValuesFilename"
7. I want to dump from Kubernetes "clusters.{cluster}.source" the "helm-values" for the "service" that is currently being processing.
    1. Apply the "camelCase" conversion. However, I want to keep all "root.properties", I want to say Java Properties Style and Keys that Starts with 3 or more Uppercase characters
    2. Save it on "{target}/{service}/envs/{cluster}/{environment}/{namespace}/values.yaml" => "helmValuesFilename": is available into "globals.migration.helmValuesFilename"
8. Prepare the code to load the globals and override/merge the config names on service layer.
    1. The globals should be available on service level with the service level configs merged.
    2. For now, only log if it has diffs.
9. Copy default cluster helm values to service root and extract manifest data:
    1. When a cluster is marked as `default: true` in config.yaml and has a `default_namespace` configured
    2. Copy values from `.cache/{cluster}/{default_namespace}/{service}/values.yaml`
    3. Apply camelCase conversion (same rules as step 7)
    4. Save to `{target}/{service}/helm-values.yaml`
    5. Extract configuration from manifest.yaml:
        - Parse Kubernetes Deployment manifest from `.cache/{cluster}/{default_namespace}/{service}/manifest.yaml`
        - Extract and convert Datadog annotations from v1 to v2 format:
            - **v1 to v2 Conversion**: Automatically detects old Datadog annotations (ad.datadoghq.com/{service}.check_names, init_configs, instances)
            - **JSON to YAML**: Converts JSON annotation values to structured YAML in helm-values.yaml
            - **Integration Detection**: Identifies integration type (jmx, openmetrics, etc.) from check_names
            - **v2 Structure**: Generates modern Datadog configuration with:
                - `datadog.integration`: Type of integration (jmx, openmetrics, kafka, etc.)
                - `datadog.jmx`: JMX-specific configuration for Java services
                - `datadog.instanceConfig`: Connection configuration (host, port, jmxUrl)
                - `datadog.enabled`: Integration enablement flag
        - Extract from the main container (matching service name):
            - Image configuration (repository, tag, pullPolicy)
            - Probe configurations (liveness, readiness, startup) with enabled flags
            - Service and port configuration following base-chart template structure
            - Container ports (app, jmx, metrics) with proper configuration
            - Resource limits and requests
            - Environment variables
        - Merge extracted data into helm-values.yaml
10. Apply transformations to values files:
    1. The transformation pipeline processes ALL generated values files:
        - `{target}/{service}/legacy-values.yaml` - Legacy source values with camelCase conversion
        - `{target}/{service}/helm-values.yaml` - Default cluster's helm values
        - `{target}/{service}/envs/{cluster}/{environment}/{namespace}/values.yaml` - Environment-specific values
    2. Apply the transformation pipeline in this order:
       a. **Normalizer** (`v1/pkg/normalizers/normalizer.go`): Apply key path transformations based on regex patterns
        - Transforms keys like `Env` → `envVars`, `Deployment.Replicas` → `replicaCount`
        - Restructures deployment settings to match standard Helm chart structure
        - Normalizes deployment configuration: `Deployment.iamRole` → `serviceAccount.iamRole.name`
        - Standardizes rollout strategy: `Deployment.maxSurge` → `strategy.rollingUpdate.maxSurge`
          b. **Transformer** (`v1/pkg/transformers/transformer.go`): Apply complex transformations like ingress to hosts extraction
          c. **Secret Separator** (`v1/pkg/secrets/separator.go`): Extract and separate secrets from configMaps:
        - Detect secrets using global and service-specific patterns from config.yaml
        - Move detected secrets from their current location to a dedicated `secrets` section
        - Preserve existing structure: if `secrets."root.properties"` exists, place secrets there
        - If `secrets: {}` is empty, mirror the structure from configMap (e.g., `configMap."root.properties"` → `secrets."root.properties"`)
        - Remove extracted secrets from their original location (e.g., from configMap)
          d. **Cleaner** (`v1/pkg/cleaner/cleaner.go`): Remove unwanted root-level keys from values files:
        - Identify root-level keys to remove using regex patterns from config.yaml
        - Only removes keys at the root level, preserving nested keys with the same name
        - Apply to specific file paths using glob pattern matching
        - Remove root keys like canary, podLabels, podAnnotations, nameOverride, fullnameOverride
        - Service-specific patterns can override or extend global patterns
        - Example: Removes root-level `canary:` but preserves `rollout.strategy.canary:`
    3. **Legacy Values Merge**: After all transformations are complete:
        - Merge keys from `{target}/{service}/legacy-values.yaml` into `{target}/{service}/values.yaml`
        - Add any keys that exist in legacy-values.yaml but not in values.yaml
        - Update keys that exist in both files with values from legacy-values.yaml
        - Recursively merge nested structures to preserve hierarchy
        - **Preserve all YAML comments** from values.yaml during the merge
        - Track all added and updated keys for reporting
    4. Save transformed and merged values back to their original locations
    5. Generate a transformation report in `{target}/{service}/transformation-report.yaml` containing:
        - List of all processed files (legacy-values.yaml, helm-values.yaml, environment values)
        - Applied normalizations with patterns used for each file
        - Extracted hosts from ingress configurations
        - Detected and separated secrets with their new locations
        - Removed keys with cleaner patterns applied
        - Legacy merge results showing added and updated keys
        - Any warnings or issues encountered during transformation

## Architecture & Design Principles

### Core Principles Applied
- **DRY (Don't Repeat Yourself)**: Single source of truth for configuration structures
- **KISS (Keep It Simple, Stupid)**: Simplified configuration handling and clear separation of concerns
- **SOLID Principles**:
    - **Single Responsibility**: Each package handles one specific aspect of migration
    - **Open/Closed**: Extensible through configuration without modifying core code
    - **Liskov Substitution**: Interfaces used for testability and flexibility
    - **Interface Segregation**: Small, focused interfaces (FileManager, ChartCopier, ValuesExtractor)
    - **Dependency Inversion**: Packages depend on abstractions (config.Config) not concrete implementations

### Path Management System
The `Paths` struct (`v1/pkg/config/paths.go`) provides centralized path management following DRY principles:
- **Builder Pattern**: Chain methods like `ForService()`, `ForCluster()`, `ForEnvironment()` to build context-specific paths
- **Consistent Naming**: All path methods follow clear naming conventions (e.g., `ServiceDir()`, `LegacyValuesPath()`, `ValuesPath()`)
- **No String Concatenation**: All path building is centralized, eliminating scattered `filepath.Join()` calls
- **Easy Refactoring**: Change path structure in one place affects entire codebase
- **Type Safety**: Methods return specific path types, reducing errors from incorrect path usage

Example usage:
```go
paths := config.NewPaths(sourcePath, targetPath, cacheDir)
.ForService("heimdall")
.ForCluster("prod01")
.ForEnvironment("production", "viafoura")

// Get specific paths
valuesPath := paths.ValuesPath()              // apps/heimdall/values.yaml
envValuesPath := paths.EnvironmentValuesPath() // apps/heimdall/envs/prod01/production/viafoura/values.yaml
cachedManifest := paths.CachedManifestPath()   // .cache/prod01/viafoura/heimdall/manifest.yaml
```

### Recent Refactoring (September-October 2025)
- **Unified Configuration System**: All packages now use the centralized `config.Config` from `v1/pkg/config`
- **Eliminated Duplicate Structs**: Removed redundant Config structs from secrets, normalizers, and transformers packages
- **Improved Type Safety**: All configuration is now strongly typed through the config package
- **Better Testability**: Tests use actual config types rather than mocks or duplicates
- **Fixed Normalizer Pattern Precision**: Updated regex patterns to prevent child path matching issues (e.g., `^[Ee]nv$` instead of `[Ee]nv`)
- **Enhanced Deployment Transformations**: Added support for deployment configuration normalization (maxSurge, maxUnavailable, iamRole)
- **Extended Transformation Pipeline**: Now processes both `legacy-values.yaml` and `helm-values.yaml` in addition to environment-specific values
- **Empty File Handling**: Skip processing of empty or comment-only YAML files to prevent EOF errors
- **Cleaner Feature**: Added configurable key removal to eliminate unwanted keys from values files
- **Manifest Extraction**: Automatically extracts configuration from manifest.yaml and merges into helm-values.yaml
- **Datadog v2 Conversion**: Automatically converts v1 Datadog autodiscovery annotations to v2 format:
    - Detects and transforms old annotation format (check_names, init_configs, instances)
    - Converts JSON annotation values to structured YAML configuration
    - Generates modern v2 structure with integration type detection
    - Supports JMX, OpenMetrics, and other Datadog integrations
- **Legacy Values Merge**: Intelligently merges legacy-values.yaml into values.yaml:
    - Adds keys from legacy values that don't exist in values.yaml
    - Updates existing keys with values from legacy values
    - Recursively merges nested structures to preserve hierarchy
    - **Preserves all YAML comments** from values.yaml using yamlutil's comment-aware merge
    - Tracks and reports all added and updated keys
- **Schema Comment Formatting**: Fixed YAML merger to preserve blank lines before `@schema` comment blocks:
    - Ensures compatibility with helm-schema tool requirements
    - Automatically adds blank lines before `@schema` blocks during merge operations
    - Maintains proper spacing for schema generation tools
- **Custom Secret Key Mappings**: Added support for service-level custom key mappings:
    - Configure custom mappings from configMap keys to secret keys (e.g., `configMap.application.properties` → `secrets.application.conf`)
    - Service-specific mappings override global mappings
    - Supports complex path transformations for different file structures
- **Secret Merge Order Control**: Added configurable merge order for secret files:
    - Define the order in which files are processed for secret merging
    - Later files in the order override values from earlier files
    - Supports placeholders like `{service}` for dynamic path resolution
- **SOPS Integration** (October 2025): Added seamless SOPS encryption for secrets management:
    - Integrated SOPS operations into the transformation pipeline (runs per-service after transformations)
    - Automatically reads path_regex patterns from `.sops.yaml` configuration
    - Supports AWS KMS encryption with configurable AWS profiles
    - Secrets command enhanced with path-specific operations and pattern support
    - Automatically extracts secrets to `.dec.yaml` files and encrypts to `.enc.yaml`
    - Hierarchical comment headers for cluster/environment/namespace level secrets
- **Dashboard and Alias Auto-Configuration**: Enhanced base-chart copying:
    - Automatically detects `dashboards/*.json` files and sets `grafana.dashboards.enabled: true`
    - Sets `service.alias` in values.yaml when configured in config.yaml (e.g., `livecomments` → `lc`)
    - Updates occur immediately after base-chart copying for consistency

#### Advanced Secret Merging Strategy Configuration

The new merging strategy feature provides fine-grained control over how secrets are extracted, mapped, and merged at the service level. This is particularly useful when dealing with services that have unique configuration structures or when migrating from legacy configurations.

##### Configuration Structure

Secret merging strategies are configured under the `secrets.merging` section in `config.yaml`. The structure uses target file patterns as keys, allowing different strategies for different files:

```yaml
services:
  livecomments:
    secrets:
      merging:
        # Target file pattern - the file being processed
        "apps/{service}/values.yaml":
          # Custom key mappings for this file
          keyMappings:
            # Map configMap keys to different secret keys
            "configMap.application.properties": "secrets.application.conf"
            "configMap.database.properties": "secrets.db.conf"
          # Define merge order for this file
          mergeOrder:
            - "apps/{service}/legacy-values.yaml"
            - "apps/{service}/helm-values.yaml"
```

##### Key Mappings

Key mappings allow you to transform the destination path when secrets are extracted. This is useful when:
- Converting from old naming conventions to new ones
- Aligning with application-specific configuration requirements
- Separating different types of secrets into logical groups

**Example Use Cases:**

1. **Java Properties to Config Format:**
   ```yaml
   keyMappings:
     "configMap.application.properties": "secrets.application.conf"
   ```
   This maps all secrets found in `configMap.application.properties` to `secrets.application.conf` instead of the default `secrets.application.properties`.

2. **Database Configuration Separation:**
   ```yaml
   keyMappings:
     "configMap.db.properties": "secrets.database.yaml"
     "configMap.redis.properties": "secrets.cache.yaml"
   ```

3. **Environment-Specific Mappings:**
   ```yaml
   merging:
     "apps/{service}/envs/*/values.yaml":
       keyMappings:
         "configMap.env.properties": "secrets.env.conf"
   ```

##### Merge Order

The merge order determines how multiple source files are combined, with later files in the list taking precedence over earlier ones. This is essential for:
- Applying legacy values as a base
- Overlaying environment-specific configurations
- Ensuring proper precedence of secret values

**Example Configurations:**

1. **Standard Migration Pattern:**
   ```yaml
   mergeOrder:
     - "apps/{service}/legacy-values.yaml"  # Base legacy values
     - "apps/{service}/helm-values.yaml"    # Helm default values override
   ```

2. **Environment-Specific Overrides:**
   ```yaml
   mergeOrder:
     - "apps/{service}/base-values.yaml"
     - "apps/{service}/envs/{cluster}/values.yaml"
     - "apps/{service}/envs/{cluster}/{environment}/values.yaml"
   ```

##### Progressive Path Resolution

The system uses progressive path searching to find mappings at different levels of the configuration hierarchy. For example, when processing `configMap.application.properties.database.password`:

1. First checks: `configMap.application.properties.database.password` (full path)
2. Then checks: `configMap.application.properties.database` (parent)
3. Then checks: `configMap.application.properties` (container level)
4. Finally checks: `configMap` (root level)

This allows for both specific and general mapping rules.

##### Migration from Legacy Configuration

If you have existing configurations using the deprecated flat structure:

**Old Format (Deprecated):**
```yaml
services:
  myservice:
    secrets:
      keyMappings:
        "configMap.app.properties": "secrets.app.conf"
      mergeOrder:
        - "legacy.yaml"
        - "values.yaml"
```

**New Format:**
```yaml
services:
  myservice:
    secrets:
      merging:
        "apps/{service}/values.yaml":
          keyMappings:
            "configMap.app.properties": "secrets.app.conf"
          mergeOrder:
            - "legacy.yaml"
            - "values.yaml"
```

##### Complete Example

Here's a comprehensive example showing how to configure merging strategies for a complex service:

```yaml
services:
  livecomments:
    enabled: true
    capitalized: "LiveComments"
    secrets:
      # Service-specific secret patterns
      patterns:
        - ".*\\.password.*"
        - ".*\\.secret.*"
        - ".*\\.apiKey.*"

      # Merging strategies for different file patterns
      merging:
        # Strategy for main values file
        "apps/{service}/values.yaml":
          keyMappings:
            "configMap.application.properties": "secrets.application.conf"
            "configMap.kafka.properties": "secrets.kafka.conf"
          mergeOrder:
            - "apps/{service}/legacy-values.yaml"
            - "apps/{service}/helm-values.yaml"

        # Strategy for environment-specific files
        "apps/{service}/envs/*/values.yaml":
          keyMappings:
            "configMap.env.properties": "secrets.environment.yaml"
          mergeOrder:
            - "apps/{service}/values.yaml"
            - "apps/{service}/envs/{cluster}/base.yaml"
```

##### How It Works During Migration

1. **Target File Detection**: When processing a file, the migrator identifies which merge strategy to use based on the file path pattern.

2. **Key Mapping Application**: During secret extraction, keys are transformed according to the mappings:
    - Original: `configMap.application.properties.jwt.secret`
    - Mapped to: `secrets.application.conf.jwt.secret`

3. **Merge Order Processing**: Files are processed in the specified order, with each subsequent file's values overriding previous ones.

4. **Report Generation**: The transformation report includes details about applied mappings and merge operations.

##### Best Practices

1. **Use Specific Patterns**: More specific file patterns take precedence over general ones.
2. **Document Mappings**: Include comments explaining why specific mappings are needed.
3. **Test Incrementally**: Start with one service and validate before applying broadly.
4. **Preserve Backwards Compatibility**: Keep deprecated structures until all services are migrated.
5. **Use Placeholders**: Leverage `{service}`, `{cluster}`, and `{environment}` placeholders for flexibility.

#### Generic Manifest Resource Extraction

The migrator includes a powerful **abstract manifest extraction system** that can extract data from any Kubernetes resource type using configurable rules. This system works alongside the existing Deployment-specific extraction for comprehensive data gathering.

##### Features

1. **Universal Resource Support**: Extract data from any Kubernetes kind (Service, ConfigMap, Ingress, Secret, PodDisruptionBudget, ServiceAccount, etc.)
2. **Configurable Extraction Rules**: Define extraction patterns in `config.yaml` without modifying code
3. **JSONPath-like Expressions**: Use field paths to extract specific values from resources
4. **Consolidated Output**: Save all extracted data to a single file in the cache for analysis
5. **Type-aware Processing**: Support for arrays, maps, and various data transformations

##### Configuration

Configure extraction rules under `globals.mappings.extract.manifest_resources` in `config.yaml`:

```yaml
extract:
  manifest_resources:
    enabled: true
    description: "Extract configuration from any Kubernetes resource in manifest.yaml"

    # Save consolidated extraction to cache
    consolidated_output:
      enabled: true
      filename: "extracted-manifest-data.yaml"

    # Define extraction rules per Kubernetes kind
    rules:
      - kind: Service
        enabled: true
        description: "Extract Service port configuration"
        extractions:
          - source: "spec.ports[0].port"
            target: "service.port"
            description: "Main service port"
          - source: "spec.ports[0].targetPort"
            target: "service.targetPort"
          - source: "spec.ports[0].protocol"
            target: "service.protocol"
          - source: "spec.type"
            target: "service.type"
          - source: "spec.selector"
            target: "service.selector"
```

##### Extraction Types

- **Simple Field**: Extract a single field value
  ```yaml
  source: "spec.type"
  target: "service.type"
  ```

- **Array Index**: Extract specific array element
  ```yaml
  source: "spec.ports[0].port"
  target: "service.port"
  ```

- **Array Collection**: Collect all matching values
  ```yaml
  source: "spec.rules[*].host"
  target: "hosts.public.domains"
  type: "array_collect"
  ```

- **Array Flattening**: Flatten nested arrays
  ```yaml
  source: "spec.tls[*].hosts"
  target: "hosts.tls.domains"
  type: "array_flatten"
  ```

- **Filtered Extraction**: Extract only matching patterns
  ```yaml
  source: "metadata.annotations"
  target: "ingress.annotations"
  filter: "nginx.*"  # Only nginx annotations
  ```

- **Template Variables**: Use resource data in target paths
  ```yaml
  source: "type"
  target: "secrets.types.{metadata.name}"
  ```

##### How It Works

1. **Resource Parsing**: The system parses all Kubernetes resources in the manifest.yaml
2. **Rule Matching**: For each resource, it finds matching extraction rules by kind
3. **Data Extraction**: Applies source path expressions to extract values
4. **Value Transformation**: Processes values based on type (collect, flatten, filter)
5. **Target Assignment**: Places extracted values at the specified target paths
6. **Consolidation**: Saves all extracted data to cache for review

##### Integration with Migration

The generic extraction system is **fully integrated** with the migration pipeline:

1. Runs automatically during `helm-charts-migrator migrate`
2. Works alongside legacy Deployment extraction
3. Extracted values are merged into helm-values.yaml
4. Consolidated output saved to `.cache/{cluster}/{namespace}/{service}/extracted-manifest-data.yaml`

##### Example Output

After migration, the consolidated extraction file contains:

```yaml
cluster: dev01
namespace: vf-dev2
service_name: livecomments
resources:
  service/livecomments:
  # Full Service resource
  configmap/livecomments:
  # Full ConfigMap resource
values:
  service:
    port: 80
    targetPort: 8080
    protocol: TCP
    type: ClusterIP
  configMap:
    application.properties: |
      # Properties content
  podDisruptionBudget:
    maxUnavailable: 1
```

##### Adding New Resource Types

To extract from a new Kubernetes resource type:

1. Add a rule in `config.yaml` under `manifest_resources.rules`
2. Define the kind and extraction specifications
3. No code changes required - the system automatically handles it

Example for StatefulSet:
```yaml
- kind: StatefulSet
  enabled: true
  description: "Extract StatefulSet configuration"
  extractions:
    - source: "spec.replicas"
      target: "statefulset.replicas"
    - source: "spec.volumeClaimTemplates[0].spec.resources.requests.storage"
      target: "persistence.size"
```

### Key Components

#### Configuration System (`v1/pkg/config/`)
- `config.go`: Main configuration structure and loader
- `clusters.go`: Cluster-specific configuration
- `services.go`: Service configuration and metadata
- `secrets_types.go`: Secret detection patterns and locations
- `mappings.go`: Transformation and normalization rules
- `injection.go`: Auto-injection configuration
- `merger.go`: Configuration merging logic (globals + service-specific)
- `paths.go`: Centralized path management for all file operations

#### Migration Pipeline (`v1/pkg/migration/`)
- `ServiceMigrator`: Orchestrates the complete migration process
- `ReleaseCache`: Caches Helm releases with disk persistence
- `ChartCopier`: Copies and customizes base charts
- `ValuesExtractor`: Extracts and converts values with camelCase preservation
- `ManifestExtractor`: Extracts configuration from manifest.yaml files
    - Parses Kubernetes Deployment manifests
    - **Datadog v1 to v2 Conversion**:
        - Automatically converts old autodiscovery annotations to modern v2 format
        - Detects integration type from check_names (jmx, openmetrics, kafka, etc.)
        - Transforms JSON annotations to structured YAML configuration
        - Generates v2-compliant Datadog configuration following best practices
    - Extracts container configuration (image, probes, ports, resources)
    - Formats service configuration using base-chart template structure
- `TransformationPipeline`: Applies normalizations, transformations, and secret separation to all values files
    - Processes `legacy-values.yaml`, `helm-values.yaml`, and all environment-specific values
    - Generates comprehensive transformation report with applied changes

#### Transformation Components
- **Normalizers** (`v1/pkg/normalizers/`): Key path transformations using regex patterns
- **Transformers** (`v1/pkg/transformers/`): Complex transformations like ingress to hosts extraction
- **Secrets** (`v1/pkg/secrets/`): Secret detection, classification, and separation
- **Cleaner** (`v1/pkg/cleaner/`): Configurable key removal based on patterns

### Technical Notes
- **YAML Processing**: All YAML manipulation uses `v1/pkg/yamlutil/` wrapper with 2-space indentation
- **Error Handling**: Comprehensive error wrapping with context for debugging
- **Logging**: Structured logging with Kubernetes-style verbosity levels
- **Caching**: Intelligent caching with values persistence to disk for reliability
- **Concurrent Processing**: Worker pool pattern for processing multiple services/clusters

## Development Guidelines

### Adding New Features
1. **New Transformations**: Implement in `v1/pkg/transformers/` and register in config
2. **New Secret Patterns**: Add to `config.yaml` under appropriate scope (global or service-specific)
3. **New CLI Commands**: Create in `v1/cmd/` following existing patterns
4. **New Normalizer Patterns**: Add to `globals.mappings.normalizer.patterns` in `config.yaml`
    - Ensure patterns are specific enough to avoid unintended matches
    - Test with sample data before deployment
5. **New Cleaner Patterns**: Add to `globals.mappings.cleaner` in `config.yaml`
    - `path_patterns`: Glob patterns to match files for cleaning
    - `key_patterns`: Regex patterns to match keys for removal
    - Service-specific overrides can be added under `services.<service>.mappings.cleaner`

### Testing
- All packages have comprehensive test coverage
- Table-driven tests for edge cases
- Integration tests for end-to-end validation

### Configuration Best Practices
- Service configurations inherit and can override global settings
- Use specific regex patterns to avoid false positives
    - Use anchors (`^` and `$`) to match exact keys, not substrings
    - Example: `^[Ee]nv$` matches only "env" or "Env", not "environment" or "env.JAVA_OPTS"
- Document complex patterns with descriptions
- Validate patterns compile correctly during startup
- Test normalizer patterns with actual data to ensure proper value preservation
