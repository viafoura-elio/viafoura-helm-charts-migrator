# Grafana Dashboards

This directory contains Grafana dashboard JSON files that will be automatically deployed as ConfigMaps when the chart is installed.

## Dashboard Files

- `heimdall.json` - Main application monitoring dashboard

## Usage

1. **Automatic Deployment**: When `grafana.dashboards.enabled: true` and the dashboard file exists, a ConfigMap will be created automatically.

2. **Dashboard Discovery**: The ConfigMap includes labels and annotations for automatic discovery by Grafana:
   ```yaml
   labels:
     grafana_dashboard: "1"
     dashboard_source: "helm-chart"
   annotations:
     grafana.com/dashboard: "true"
     grafana.com/folder: "Java Applications"
   ```

3. **Configuration**: Customize dashboard deployment in `values.yaml`:
   ```yaml
   grafana:
     dashboards:
       enabled: true
       folder: "Java Applications"
       filename: "heimdall.json"
       labels:
         grafana_dashboard: "1"
         dashboard_source: "helm-chart"
       annotations: {}
   ```

## Dashboard Requirements

- Dashboard files must be valid JSON
- Use Prometheus metrics compatible with the application's ServiceMonitor
- Include relevant variables for dynamic filtering
- Follow Grafana dashboard best practices

## Template Variables

The dashboard should include these template variables for dynamic filtering:
- `$namespace` - Kubernetes namespace
- `$pod` - Pod name
- `$instance` - Prometheus instance
- `$job` - Prometheus job name

## Metrics

The dashboard should monitor these key metrics:
- HTTP request rate and latency
- JVM memory and GC metrics
- Database connection pool status
- Custom application metrics
- Error rates and availability

## Development

To create or modify dashboards:
1. Create/edit dashboard in Grafana UI
2. Export dashboard JSON
3. Place in this directory
4. Test with `helm template` to verify ConfigMap creation
5. Deploy and verify dashboard appears in Grafana

## Troubleshooting

- **ConfigMap not created**: Check if dashboard file exists and `grafana.dashboards.enabled: true`
- **Dashboard not visible**: Verify Grafana is configured to discover ConfigMaps with `grafana_dashboard: "1"` label
- **Template errors**: Validate JSON syntax and Helm templating
