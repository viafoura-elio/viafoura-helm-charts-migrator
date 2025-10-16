# Environment Configuration Hierarchy

This directory follows a hierarchical structure that mirrors AWS account organization and Kubernetes cluster topology.

## Directory Structure

```
envs/
├── README.md                           # This file
├── sops-helper.sh                      # Helper script for SOPS encryption/decryption
├── values.yaml                         # Base values (legacy)
├── secrets.dec.yaml                    # Base decrypted secrets template
├── testdev/                            # Test/Dev AWS Account
│   ├── values.yaml                     # Account-wide defaults
│   ├── secrets.enc.yaml                # Account-wide secrets (encrypted)
│   ├── secrets.dec.yaml                # Account-wide secrets (decrypted template)
│   └── clusters/                       # Clusters in this account
│       ├── dev01/                      # Development cluster 01
│       │   ├── values.yaml             # Cluster-specific values
│       │   ├── secrets.dec.yaml        # Cluster-specific secrets template
│       │   └── namespaces/             # Namespaces in this cluster
│       │       ├── vf-dev2/
│       │       │   ├── values.yaml
│       │       │   ├── secrets.enc.yaml
│       │       │   └── secrets.dec.yaml
│       │       └── vf-dev3/
│       │           └── secrets.dec.yaml
│       ├── dev02/                      # Development cluster 02
│       │   ├── secrets.dec.yaml        # Cluster-specific secrets template
│       │   └── namespaces/
│       │       ├── vf-dev2/
│       │       │   └── secrets.dec.yaml
│       │       └── vf-dev3/
│       │           └── secrets.dec.yaml
│       └── test01/                     # Test cluster 01
│           ├── values.yaml             # Cluster-specific values
│           ├── secrets.dec.yaml        # Cluster-specific secrets template
│           └── namespaces/
│               ├── vf-test2/
│               │   └── secrets.dec.yaml
│               └── vf-test3/
│                   └── secrets.dec.yaml
├── staging/                            # Staging AWS Account
│   ├── secrets.dec.yaml                # Account-wide secrets template
│   └── clusters/
│       └── stg01/
│           └── viafoura/
└── production/                         # Production AWS Account
    ├── values.yaml                     # Account-wide production values
    ├── secrets.dec.yaml                # Account-wide secrets template
    └── clusters/
        └── prod01/
            └── viafoura/
```

## Configuration Inheritance

Values are inherited and merged in the following order (later values override earlier ones):

1. **Base Chart defaults** (`values.yaml` in chart root)
2. **Account level** (`{account}/values.yaml`)
3. **Cluster level** (`{account}/clusters/{cluster}/values.yaml`)
4. **Namespace level** (`{account}/clusters/{cluster}/namespaces/{namespace}/values.yaml`)
5. **Service-specific overrides** (passed during deployment)

## Usage Examples

### Deploying to dev01 cluster in vf-dev2 namespace:
```bash
helm install my-service . \
  -f envs/testdev/values.yaml \
  -f envs/testdev/clusters/dev01/values.yaml \
  -f envs/testdev/clusters/dev01/namespaces/vf-dev2/values.yaml \
  -f envs/testdev/secrets.enc.yaml \
  -f envs/testdev/clusters/dev01/secrets.enc.yaml \
  -f envs/testdev/clusters/dev01/namespaces/vf-dev2/secrets.enc.yaml
```

### Using helmfile (recommended):
```yaml
# helmfile.yaml
environments:
  testdev-dev01-vf-dev2:
    values:
      - envs/testdev/values.yaml
      - envs/testdev/clusters/dev01/values.yaml
      - envs/testdev/clusters/dev01/namespaces/vf-dev2/values.yaml
    secrets:
      - envs/testdev/secrets.enc.yaml
      - envs/testdev/clusters/dev01/secrets.enc.yaml
      - envs/testdev/clusters/dev01/namespaces/vf-dev2/secrets.enc.yaml
```

## Configuration Guidelines

### Account Level (`{account}/`)
- AWS account-wide settings (Account ID, default region)
- IAM roles and policies defaults
- Cost allocation tags
- Default resource quotas
- Shared services endpoints

### Cluster Level (`clusters/{cluster}/`)
- EKS cluster configuration
- Node group settings
- Cluster-wide RBAC
- Ingress controller settings
- Service mesh configuration
- Monitoring endpoints

### Namespace Level (`namespaces/{namespace}/`)
- Namespace-specific resource limits
- Network policies
- Service account configurations
- Environment-specific variables
- Application secrets

## Secrets Management

### Secret Files Structure
Each level in the hierarchy has two secret files:
- `secrets.dec.yaml` - Decrypted secrets template (never commit to git)
- `secrets.enc.yaml` - Encrypted secrets (safe to commit)

### SOPS Encryption

All `secrets.enc.yaml` files should be encrypted using SOPS with AWS KMS:

```bash
# Using the helper script (recommended)
./sops-helper.sh encrypt testdev              # Account level
./sops-helper.sh encrypt testdev dev01        # Cluster level
./sops-helper.sh encrypt testdev dev01 vf-dev2  # Namespace level

# Decrypt for editing
./sops-helper.sh decrypt testdev dev01 vf-dev2

# Edit encrypted file directly
./sops-helper.sh edit testdev dev01 vf-dev2

# Manual SOPS commands
sops -e secrets.dec.yaml > secrets.enc.yaml  # Encrypt
sops -d secrets.enc.yaml > secrets.dec.yaml  # Decrypt
sops secrets.enc.yaml                        # Edit in-place
```

### Important Security Notes
- **Never commit** `secrets.dec.yaml` files (added to .gitignore)
- **Always encrypt** secrets before committing
- **Use different KMS keys** for each AWS account
- **Rotate secrets** regularly

## AWS Account Mapping

| Account Name | AWS Account ID | Purpose | Owner |
|--------------|----------------|---------|-------|
| testdev | 123456789012 | Development & Testing | Engineering |
| staging | 234567890123 | Pre-production | Engineering |
| production | 345678901234 | Production | Operations |

## Migration from Legacy Structure

To migrate from the old structure:
1. Move existing `envs/values.yaml` to appropriate account/cluster/namespace directory
2. Split monolithic configs into hierarchical structure
3. Create `secrets.dec.yaml` templates at each level
4. Encrypt secrets using `./sops-helper.sh encrypt <account> [cluster] [namespace]`
5. Update deployment scripts to use new paths (remove `/accounts` prefix)
6. Test with dry-run before actual deployment

## Helper Scripts

### sops-helper.sh
Manages encryption/decryption of secrets at any hierarchy level:
- `./sops-helper.sh encrypt testdev` - Encrypt account-level secrets
- `./sops-helper.sh decrypt testdev dev01` - Decrypt cluster-level secrets
- `./sops-helper.sh edit testdev dev01 vf-dev2` - Edit namespace-level encrypted secrets

### helm-deploy.sh
Automates helm deployments with proper value file hierarchy:
- Automatically discovers and applies values files in correct order
- Handles SOPS decryption if helm-secrets plugin is installed
- `./helm-deploy.sh testdev dev01 vf-dev2 my-service`