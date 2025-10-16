{{/*
Generate Datadog annotations for pod (using Datadog Agent v2 format - requires Agent v7.36+)
*/}}
{{- define "base-chart.datadogAnnotations" -}}
{{- if .Values.datadog.enabled -}}
{{- include "base-chart.datadogChecks" . }}
{{- include "base-chart.datadogLogs" . }}
{{- end -}}
{{- end -}}

{{/*
Generate Datadog checks annotation
*/}}
{{- define "base-chart.datadogChecks" -}}
{{- $instanceConfig := include "base-chart.datadogInstanceConfig" . }}
{{- if $instanceConfig }}
{{- $config := dict (.Values.datadog.integration | default "openmetrics") (dict "init_config" (.Values.datadog.initConfig | default (dict)) "instances" (list ($instanceConfig | fromYaml))) }}
ad.datadoghq.com/{{ include "base-chart.datadogContainerName" . }}.checks: |
  {{ $config | toJson }}
{{- end }}
{{- end -}}

{{/*
Generate instance configuration based on integration type
*/}}
{{- define "base-chart.datadogInstanceConfig" -}}
{{- if eq (.Values.datadog.integration | default "openmetrics") "openmetrics" -}}
{{- include "base-chart.datadogOpenmetricsConfig" . }}
{{- else if eq .Values.datadog.integration "jmx" -}}
{{- include "base-chart.datadogJmxConfig" . }}
{{- else -}}
{{- include "base-chart.datadogCustomConfig" . }}
{{- end -}}
{{- include "base-chart.datadogCommonInstanceConfig" . }}
{{- end -}}

{{/*
OpenMetrics integration configuration
*/}}
{{- define "base-chart.datadogOpenmetricsConfig" -}}
openmetrics_endpoint: "http://%%host%%:{{ .Values.service.containerPorts.metrics.containerPort }}{{ .Values.datadog.metricsPath }}"
namespace: "{{ .Values.datadog.namespace }}"
metrics: {{ .Values.datadog.metrics | toYaml | nindent 2 }}
{{- if .Values.datadog.ignoreMetrics }}
ignore_metrics: {{ .Values.datadog.ignoreMetrics | toYaml | nindent 2 }}
{{- end }}
collect_histogram_buckets: {{ .Values.datadog.collectHistogramBuckets }}
send_histograms_buckets: {{ .Values.datadog.sendHistogramsBuckets }}
health_service_check: {{ .Values.datadog.healthServiceCheck }}
timeout: {{ .Values.datadog.timeout }}
{{- end -}}

{{/*
JMX integration configuration
*/}}
{{- define "base-chart.datadogJmxConfig" -}}
host: "%%host%%"
port: {{ .Values.service.containerPorts.jmx.containerPort | default 9016 }}
{{- if .Values.datadog.jmx.jmxUrl }}
jmx_url: "{{ .Values.datadog.jmx.jmxUrl }}"
{{- else }}
jmx_url: "service:jmx:rmi:///jndi/rmi://%%host%%:{{ .Values.service.containerPorts.jmx.containerPort | default 9016 }}/jmxrmi"
{{- end }}
{{- if hasKey .Values.datadog.jmx "isJmx" }}
is_jmx: {{ .Values.datadog.jmx.isJmx }}
{{- else }}
is_jmx: true
{{- end }}
{{- if hasKey .Values.datadog.jmx "collectDefaultMetrics" }}
collect_default_metrics: {{ .Values.datadog.jmx.collectDefaultMetrics }}
{{- else }}
collect_default_metrics: true
{{- end }}
{{- if .Values.datadog.jmx.user }}
user: "{{ .Values.datadog.jmx.user }}"
{{- end }}
{{- if .Values.datadog.jmx.password }}
password: "{{ .Values.datadog.jmx.password }}"
{{- end }}
{{- if .Values.datadog.jmx.processNameRegex }}
process_name_regex: "{{ .Values.datadog.jmx.processNameRegex }}"
{{- end }}
{{- if .Values.datadog.jmx.tools_jar_path }}
tools_jar_path: "{{ .Values.datadog.jmx.tools_jar_path }}"
{{- end }}
{{- if .Values.datadog.jmx.name }}
name: "{{ .Values.datadog.jmx.name }}"
{{- end }}
{{- if .Values.datadog.jmx.java_bin_path }}
java_bin_path: "{{ .Values.datadog.jmx.java_bin_path }}"
{{- end }}
{{- if .Values.datadog.jmx.java_options }}
java_options: "{{ .Values.datadog.jmx.java_options }}"
{{- end }}
{{- if .Values.datadog.jmx.trust_store_path }}
trust_store_path: "{{ .Values.datadog.jmx.trust_store_path }}"
{{- end }}
{{- if .Values.datadog.jmx.trust_store_password }}
trust_store_password: "{{ .Values.datadog.jmx.trust_store_password }}"
{{- end }}
{{- if .Values.datadog.jmx.key_store_path }}
key_store_path: "{{ .Values.datadog.jmx.key_store_path }}"
{{- end }}
{{- if .Values.datadog.jmx.key_store_password }}
key_store_password: "{{ .Values.datadog.jmx.key_store_password }}"
{{- end }}
{{- if .Values.datadog.jmx.rmi_registry_ssl }}
rmi_registry_ssl: {{ .Values.datadog.jmx.rmi_registry_ssl }}
{{- end }}
{{- if .Values.datadog.jmx.rmi_connection_timeout }}
rmi_connection_timeout: {{ .Values.datadog.jmx.rmi_connection_timeout }}
{{- end }}
{{- if .Values.datadog.jmx.rmi_client_timeout }}
rmi_client_timeout: {{ .Values.datadog.jmx.rmi_client_timeout }}
{{- end }}
{{- if .Values.datadog.jmx.collect_default_jvm_metrics }}
collect_default_jvm_metrics: {{ .Values.datadog.jmx.collect_default_jvm_metrics }}
{{- end }}
{{- if .Values.datadog.jmx.new_gc_metrics }}
new_gc_metrics: {{ .Values.datadog.jmx.new_gc_metrics }}
{{- end }}
{{- if .Values.datadog.jmx.service_check_prefix }}
service_check_prefix: "{{ .Values.datadog.jmx.service_check_prefix }}"
{{- end }}
{{- if .Values.datadog.jmx.conf }}
conf: {{ .Values.datadog.jmx.conf | toYaml | nindent 2 }}
{{- end }}
{{- end -}}

{{/*
Custom integration configuration
*/}}
{{- define "base-chart.datadogCustomConfig" -}}
{{- range $key, $value := .Values.datadog.instanceConfig }}
{{ $key }}: {{ $value | toYaml }}
{{- end }}
{{- end -}}

{{/*
Common instance configuration (tags, etc.)
*/}}
{{- define "base-chart.datadogCommonInstanceConfig" -}}
{{- if .Values.datadog.tags }}
{{- $root := . }}
{{- $tagsDict := dict }}
tags:
{{- if kindIs "slice" .Values.datadog.tags }}
  {{- /* Process array format tags */}}
  {{- range .Values.datadog.tags }}
    {{- $parts := splitList ":" . }}
    {{- if eq (len $parts) 2 }}
      {{- $key := index $parts 0 }}
      {{- $value := index $parts 1 }}
      {{- if eq $key "version" }}
        {{- $_ := set $tagsDict $key (include "base-chart.appVersion" $root) }}
      {{- else }}
        {{- $_ := set $tagsDict $key $value }}
      {{- end }}
    {{- end }}
  {{- end }}
{{- else if kindIs "map" .Values.datadog.tags }}
  {{- /* Process map format tags */}}
  {{- range $key, $value := .Values.datadog.tags }}
    {{- if eq $key "version" }}
      {{- $_ := set $tagsDict $key (include "base-chart.appVersion" $root) }}
    {{- else }}
      {{- $_ := set $tagsDict $key $value }}
    {{- end }}
  {{- end }}
{{- end }}
  {{- /* Always add version if not present */}}
  {{- if not (hasKey $tagsDict "version") }}
    {{- $_ := set $tagsDict "version" (include "base-chart.appVersion" $root) }}
  {{- end }}
  {{- /* Output unique tags */}}
  {{- range $key, $value := $tagsDict }}
  - {{ printf "%s:%s" $key $value | quote }}
  {{- end }}
{{- end }}
{{- end -}}

{{/*
Generate Datadog logs annotation
*/}}
{{- define "base-chart.datadogLogs" -}}
{{- if .Values.datadog.logs.enabled }}
ad.datadoghq.com/{{ include "base-chart.datadogContainerName" . }}.logs: |
  [{"source": "{{ .Values.datadog.logs.source }}", "service": "{{ .Values.datadog.logs.service }}"{{- if .Values.datadog.logs.logProcessingRules }}, "log_processing_rules": {{ .Values.datadog.logs.logProcessingRules | toJson }}{{- end }}}]
{{- end -}}
{{- end -}}

{{/*
Generate logs configuration
*/}}
{{- define "base-chart.datadogLogsConfig" -}}
- source: "{{ .Values.datadog.logs.source }}"
  service: "{{ .Values.datadog.logs.service }}"
  {{- if .Values.datadog.logs.logProcessingRules }}
  log_processing_rules:
{{ .Values.datadog.logs.logProcessingRules | toYaml | indent 4 }}
  {{- end }}
{{- end -}}

{{/*
Generate Datadog labels for resources
*/}}
{{- define "base-chart.datadogLabels" -}}
{{- if .Values.datadog.enabled }}
datadog.monitoring: "true"
datadog.service: {{ include "base-chart.datadogContainerName" . | quote }}
datadog.environment: {{ .Values.datadog.apm.environment | default "production" | quote }}
{{- end }}
{{- end -}}

{{/*
Generate Datadog tags with dynamic version
Supports both array format (["key:value"]) and map format ({key: value})
*/}}
{{- define "base-chart.datadogTags" -}}
{{- $tags := list }}
{{- $root := . }}
{{- if kindIs "slice" .Values.datadog.tags }}
  {{- /* Array format: ["env:production", "team:backend"] */}}
  {{- range .Values.datadog.tags }}
    {{- if hasPrefix "version:" . }}
      {{- $tags = append $tags (printf "version:%s" (include "base-chart.appVersion" $root)) }}
    {{- else }}
      {{- $tags = append $tags . }}
    {{- end }}
  {{- end }}
{{- else if kindIs "map" .Values.datadog.tags }}
  {{- /* Map format: {env: production, team: backend} */}}
  {{- range $key, $value := .Values.datadog.tags }}
    {{- if eq $key "version" }}
      {{- $tags = append $tags (printf "version:%s" (include "base-chart.appVersion" $root)) }}
    {{- else }}
      {{- $tags = append $tags (printf "%s:%s" $key $value) }}
    {{- end }}
  {{- end }}
{{- end }}
{{- $tags | toJson }}
{{- end -}}

{{/*
Generate Datadog environment variables
*/}}
{{- define "base-chart.datadogEnvVars" -}}
{{- if .Values.datadog.apm.enabled }}
- name: DD_SERVICE
  value: {{ .Values.datadog.apm.serviceName | quote }}
- name: DD_ENV
  value: {{ .Values.datadog.apm.environment | quote }}
- name: DD_VERSION
  value: {{ include "base-chart.appVersion" . | quote }}
- name: DD_TRACE_ENABLED
  value: "true"
- name: DD_LOGS_INJECTION
  value: "true"
- name: DD_PROFILING_ENABLED
  value: "true"
{{- end }}
{{- end -}}

{{/*
Generate Datadog container name - defaults to Chart.Name if containerName is not set
*/}}
{{- define "base-chart.datadogContainerName" -}}
{{- .Values.datadog.containerName | default .Chart.Name }}
{{- end -}}
