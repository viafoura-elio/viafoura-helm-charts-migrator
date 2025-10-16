# Helm Chart Unit Tests

This directory contains comprehensive unit tests for the heimdall Helm chart using the `helm-unittest` plugin.

## Test Structure

- `simple_test.yaml` - Basic rendering and configuration tests (no schema validation)
- `basic_test.yaml` - Core functionality tests with proper schema validation
- `rollout_test.yaml` - Argo Rollouts specific tests
- `istio_test.yaml` - Istio service mesh configuration tests
- `autoscaling_test.yaml` - HPA and autoscaling tests
- `monitoring_test.yaml` - ServiceMonitor and monitoring tests
- `security_test.yaml` - Security contexts and RBAC tests
- `resources_test.yaml` - Resource limits and environment tests

## Running Tests

### Run All Tests
```bash
helm unittest .
```

### Run Specific Test Suite
```bash
helm unittest -f 'tests/simple_test.yaml' .
```

### Run Tests with Custom Values
```bash
helm unittest -v envs/dev01/vf-dev2/values.yaml .
```

### Generate Test Report
```bash
helm unittest -o test-results.xml -t JUnit .
```

## Test Categories

### ✅ Passing Tests (simple_test.yaml)
- Basic Rollout rendering
- Service configuration
- ServiceAccount setup
- HPA conditional rendering
- DestinationRule for canary
- Replica count management

### ⚠️ Schema Validation Issues
Some tests fail due to strict schema validation requiring `additionalProperties: false`. These can be run individually:

```bash
# Run without schema validation
helm unittest --strict=false .
```

## Key Test Scenarios

### Rollout Configuration
- Canary deployment strategy
- Blue-green deployment
- Traffic routing with Istio
- Revision history management

### Service Mesh (Istio)
- Gateway configuration
- VirtualService routing
- DestinationRule subsets
- Certificate management

### Autoscaling
- HPA with CPU metrics
- External metrics integration
- Replica count management
- PodDisruptionBudget

### Monitoring
- ServiceMonitor configuration
- Prometheus integration
- Metrics port exposure
- Dashboard configuration

### Security
- ServiceAccount IAM roles
- Security contexts
- Secret management
- RBAC permissions

## Best Practices

1. **Test Isolation**: Each test should be independent
2. **Value Overrides**: Use `set:` for custom test values
3. **Schema Compliance**: Ensure values match JSON schema
4. **Environment Testing**: Test with actual environment values
5. **Error Scenarios**: Test negative cases and disabled features

## Debugging Failed Tests

1. Check template rendering:
```bash
helm template . --values values.yaml
```

2. Validate against schema:
```bash
helm template . --validate
```

3. View specific template:
```bash
helm template . --show-only templates/rollout.yaml
```

## Continuous Integration

Add to CI pipeline:
```yaml
- name: Run Helm Unit Tests
  run: |
    helm plugin install https://github.com/helm-unittest/helm-unittest.git
    helm unittest . --output-file test-results.xml --output-type JUnit
```
