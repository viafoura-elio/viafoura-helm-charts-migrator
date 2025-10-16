# Base-Chart Troubleshooting Guide

## Common Issues and Solutions

### Issue: Custom metrics not available to kube-prometheus-stack

#### Root Cause Analysis

The issue was caused by multiple configuration problems:

1. **ServiceMonitor Configuration**: Incorrect namespace selector configuration
2. **HPA Configuration**: Wrong metric type for custom metrics
3. **Prometheus Adapter Configuration**: Missing custom metrics rules

#### Fixes Applied

##### 1. ServiceMonitor Configuration (`templates/servicemonitor.yaml`)
- Fixed namespace selector to support cross-namespace monitoring
- Added proper configuration for monitoring services across namespaces
- Updated values to deploy ServiceMonitor in monitoring namespace

##### 2. HPA Configuration (`templates/hpa.yaml`)
- Changed from `type: Pods` to `type: External` for custom metrics
- Added proper label selectors for metric identification
- Fixed target type to `AverageValue`

##### 3. Prometheus Adapter Configuration Required
The Prometheus Adapter in your kube-prometheus-stack needs custom rules to expose application metrics to the custom metrics API.

#### Verification Steps

1. **Check ServiceMonitor Discovery**:
   ```bash
   kubectl get servicemonitor -n monitoring
   kubectl describe servicemonitor base-chart -n monitoring
   ```

2. **Verify Prometheus Targets**:
   ```bash
   kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090
   # Open http://localhost:9090/targets and look for base-chart
   ```

3. **Check Custom Metrics API**:
   ```bash
   kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1" | jq .
   kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/your-namespace/pods/*/your_custom_metric" | jq .
   ```

4. **Test HPA**:
   ```bash
   kubectl describe hpa base-chart -n your-namespace
   kubectl get hpa base-chart -n your-namespace -o yaml
   ```

## Configuration Issues

### Issue: Configuration files not loading properly

#### Root Cause
The `config-merger` init container may fail to merge ConfigMap and Secret files correctly.

#### Troubleshooting Steps

1. **Check Init Container Logs**:
   ```bash
   kubectl logs -n your-namespace pod-name -c config-merger
   ```

2. **Verify Configuration Files**:
   ```bash
   kubectl exec -n your-namespace pod-name -- ls -la /usr/verticles/conf/
   kubectl exec -n your-namespace pod-name -- cat /usr/verticles/conf/application.properties
   ```

3. **Check ConfigMap and Secret Resources**:
   ```bash
   kubectl get configmap base-chart -n your-namespace -o yaml
   kubectl get secret base-chart -n your-namespace -o yaml
   ```

#### Solution
Ensure your `values.yaml` has properly structured configuration:

```yaml
configMap:
  application.properties:
    server.port: 8080
    kafka.bootstrap.servers: "localhost:9092"

secrets:
  application.properties:
    kafka.username: "secure-user"
    kafka.password: "secure-password"
```

### Issue: Environment variables not available

#### Root Cause
Environment variables from ConfigMaps or Secrets are not being injected correctly.

#### Troubleshooting Steps

1. **Check Pod Environment**:
   ```bash
   kubectl exec -n your-namespace pod-name -- env | grep -E "LOG_LEVEL|DATABASE_PASSWORD"
   ```

2. **Verify EnvVar Resources**:
   ```bash
   kubectl get configmap base-chart-configMapEnvVars -n your-namespace -o yaml
   kubectl get secret base-chart-secretsEnvVars -n your-namespace -o yaml
   ```

#### Solution
Configure environment variables in `values.yaml`:

```yaml
envVars:
  JAVA_OPTIONS: "-Xms100m -Xmx100m"

configMapEnvVars:
  LOG_LEVEL: "INFO"
  DEBUG_MODE: "false"

secretsEnvVars:
  DATABASE_PASSWORD: "postgres-password"
  API_TOKEN: "external-service-token"
```

## Deployment Issues

### Issue: Pod fails to start or crashes

#### Common Causes
1. **Resource Limits**: Insufficient CPU/memory allocation
2. **Security Context**: Permission issues with security constraints
3. **Image Issues**: Wrong image tag or pull policy

#### Troubleshooting Steps

1. **Check Pod Status**:
   ```bash
   kubectl get pods -n your-namespace
   kubectl describe pod base-chart-xxx -n your-namespace
   ```

2. **Check Application Logs**:
   ```bash
   kubectl logs -n your-namespace base-chart-xxx --previous
   kubectl logs -n your-namespace base-chart-xxx -c base-chart
   ```

3. **Verify Resource Usage**:
   ```bash
   kubectl top pods -n your-namespace
   kubectl describe hpa base-chart -n your-namespace
   ```

#### Solution
Adjust resource limits in `values.yaml`:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Issue: Argo Rollouts not progressing

#### Root Cause
Rollout analysis may be failing or traffic routing not working correctly.

#### Troubleshooting Steps

1. **Check Rollout Status**:
   ```bash
   kubectl get rollout base-chart -n your-namespace
   kubectl describe rollout base-chart -n your-namespace
   ```

2. **Check Analysis Results**:
   ```bash
   kubectl get analysisrun -n your-namespace
   kubectl describe analysisrun analysis-run-name -n your-namespace
   ```

3. **Verify Istio Configuration**:
   ```bash
   kubectl get virtualservice base-chart -n your-namespace -o yaml
   kubectl get destinationrule base-chart -n your-namespace -o yaml
   ```

## Monitoring and Observability Issues

### Issue: Metrics not being scraped

#### Root Cause
ServiceMonitor configuration or Prometheus setup issues.

#### Troubleshooting Steps

1. **Check ServiceMonitor Configuration**:
   ```bash
   kubectl get servicemonitor -n monitoring
   kubectl describe servicemonitor base-chart -n monitoring
   ```

2. **Verify Metrics Endpoint**:
   ```bash
   kubectl port-forward -n your-namespace svc/base-chart-metrics 5555:5555
   curl http://localhost:5555/metrics
   ```

3. **Check Prometheus Configuration**:
   ```bash
   kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090
   # Check targets at http://localhost:9090/targets
   ```

### Issue: Datadog integration not working

#### Root Cause
DogStatsD socket mounting or APM configuration issues.

#### Troubleshooting Steps

1. **Check Datadog Agent**:
   ```bash
   kubectl get pods -n datadog
   kubectl logs -n datadog datadog-agent-xxx
   ```

2. **Verify Socket Mount**:
   ```bash
   kubectl exec -n your-namespace pod-name -- ls -la /var/run/datadog/
   ```

3. **Check APM Configuration**:
   ```bash
   kubectl exec -n your-namespace pod-name -- env | grep DD_
   ```

## Security Issues

### Issue: Pod security policy violations

#### Root Cause
Security context configuration conflicts with cluster policies.

#### Solution
Configure appropriate security context in `values.yaml`:

```yaml
securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  seccompProfile:
    type: RuntimeDefault
```

## Network Issues

### Issue: Service connectivity problems

#### Root Cause
Network policies or Istio configuration blocking traffic.

#### Troubleshooting Steps

1. **Check Service Configuration**:
   ```bash
   kubectl get svc base-chart -n your-namespace -o yaml
   kubectl get endpoints base-chart -n your-namespace
   ```

2. **Test Connectivity**:
   ```bash
   kubectl exec -n your-namespace pod-name -- curl http://base-chart:8080/health
   ```

3. **Check Istio Sidecar**:
   ```bash
   kubectl logs -n your-namespace pod-name -c istio-proxy
   ```

## General Debugging Commands

### Pod Inspection
```bash
# Get pod details
kubectl get pods -n your-namespace -o wide

# Check pod events
kubectl get events -n your-namespace --sort-by=.metadata.creationTimestamp

# Describe pod for detailed information
kubectl describe pod base-chart-xxx -n your-namespace

# Check resource usage
kubectl top pods -n your-namespace
```

### Configuration Verification
```bash
# Check all ConfigMaps
kubectl get configmaps -n your-namespace

# Check all Secrets
kubectl get secrets -n your-namespace

# Verify Helm release
helm list -n your-namespace
helm get values base-chart -n your-namespace
```

### Application Logs
```bash
# Current logs
kubectl logs -n your-namespace base-chart-xxx -c base-chart

# Previous container logs
kubectl logs -n your-namespace base-chart-xxx -c base-chart --previous

# Follow logs
kubectl logs -n your-namespace base-chart-xxx -c base-chart -f
```

## Next Steps

1. **Configure Monitoring**: Ensure ServiceMonitor and custom metrics are properly configured
2. **Deploy Changes**: Apply the updated configuration to your cluster
3. **Verify Health**: Check that all components are running correctly
4. **Test Functionality**: Verify that the application works as expected

## Additional Resources

- [Kubernetes Troubleshooting Guide](https://kubernetes.io/docs/tasks/debug-application-cluster/)
- [Istio Troubleshooting](https://istio.io/latest/docs/ops/common-problems/)
- [Prometheus Monitoring](https://prometheus.io/docs/prometheus/latest/troubleshooting/)
- [Argo Rollouts Documentation](https://argoproj.github.io/argo-rollouts/)
