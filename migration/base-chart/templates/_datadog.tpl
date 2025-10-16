{{/*
Generate Datadog annotations for pod (using Datadog Agent v2 format - requires Agent v7.36+)
*/}}
{{- define "base-chart.datadogAnnotations" -}}
{{- if .Values.datadog.v2.enabled -}}
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
{{- $config := dict (.Values.datadog.v2.integration | default "openmetrics") (dict "init_config" (.Values.datadog.v2.initConfig | default (dict)) "instances" (list ($instanceConfig | fromYaml))) }}
ad.datadoghq.com/{{ include "base-chart.datadogContainerName" . }}.checks: |
  {{ $config | toJson }}
{{- end }}
{{- end -}}

{{/*
Generate instance configuration based on integration type
*/}}
{{- define "base-chart.datadogInstanceConfig" -}}
{{- if eq (.Values.datadog.v2.integration | default "openmetrics") "openmetrics" -}}
{{- include "base-chart.datadogOpenmetricsConfig" . }}
{{- else if eq .Values.datadog.v2.integration "jmx" -}}
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
openmetrics_endpoint: "http://%%host%%:{{ .Values.service.containerPorts.metrics.containerPort }}{{ .Values.datadog.v2.metricsPath }}"
namespace: "{{ .Values.datadog.v2.namespace }}"
metrics: {{ .Values.datadog.v2.metrics | toYaml | nindent 2 }}
{{- if .Values.datadog.v2.ignoreMetrics }}
ignore_metrics: {{ .Values.datadog.v2.ignoreMetrics | toYaml | nindent 2 }}
{{- end }}
collect_histogram_buckets: {{ .Values.datadog.v2.collectHistogramBuckets }}
send_histograms_buckets: {{ .Values.datadog.v2.sendHistogramsBuckets }}
health_service_check: {{ .Values.datadog.v2.healthServiceCheck }}
timeout: {{ .Values.datadog.v2.timeout }}
{{- end -}}

{{/*
JMX integration configuration
*/}}
{{- define "base-chart.datadogJmxConfig" -}}
host: "%%host%%"
port: {{ .Values.service.containerPorts.jmx.containerPort | default 9016 }}
{{- if .Values.datadog.v2.jmx.jmxUrl }}
jmx_url: "{{ .Values.datadog.v2.jmx.jmxUrl }}"
{{- else }}
jmx_url: "service:jmx:rmi:///jndi/rmi://%%host%%:{{ .Values.service.containerPorts.jmx.containerPort | default 9016 }}/jmxrmi"
{{- end }}
{{- if hasKey .Values.datadog.v2.jmx "isJmx" }}
is_jmx: {{ .Values.datadog.v2.jmx.isJmx }}
{{- else }}
is_jmx: true
{{- end }}
{{- if hasKey .Values.datadog.v2.jmx "collectDefaultMetrics" }}
collect_default_metrics: {{ .Values.datadog.v2.jmx.collectDefaultMetrics }}
{{- else }}
collect_default_metrics: true
{{- end }}
{{- if .Values.datadog.v2.jmx.user }}
user: "{{ .Values.datadog.v2.jmx.user }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.password }}
password: "{{ .Values.datadog.v2.jmx.password }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.processNameRegex }}
process_name_regex: "{{ .Values.datadog.v2.jmx.processNameRegex }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.tools_jar_path }}
tools_jar_path: "{{ .Values.datadog.v2.jmx.tools_jar_path }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.name }}
name: "{{ .Values.datadog.v2.jmx.name }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.java_bin_path }}
java_bin_path: "{{ .Values.datadog.v2.jmx.java_bin_path }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.java_options }}
java_options: "{{ .Values.datadog.v2.jmx.java_options }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.trust_store_path }}
trust_store_path: "{{ .Values.datadog.v2.jmx.trust_store_path }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.trust_store_password }}
trust_store_password: "{{ .Values.datadog.v2.jmx.trust_store_password }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.key_store_path }}
key_store_path: "{{ .Values.datadog.v2.jmx.key_store_path }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.key_store_password }}
key_store_password: "{{ .Values.datadog.v2.jmx.key_store_password }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.rmi_registry_ssl }}
rmi_registry_ssl: {{ .Values.datadog.v2.jmx.rmi_registry_ssl }}
{{- end }}
{{- if .Values.datadog.v2.jmx.rmi_connection_timeout }}
rmi_connection_timeout: {{ .Values.datadog.v2.jmx.rmi_connection_timeout }}
{{- end }}
{{- if .Values.datadog.v2.jmx.rmi_client_timeout }}
rmi_client_timeout: {{ .Values.datadog.v2.jmx.rmi_client_timeout }}
{{- end }}
{{- if .Values.datadog.v2.jmx.collect_default_jvm_metrics }}
collect_default_jvm_metrics: {{ .Values.datadog.v2.jmx.collect_default_jvm_metrics }}
{{- end }}
{{- if .Values.datadog.v2.jmx.new_gc_metrics }}
new_gc_metrics: {{ .Values.datadog.v2.jmx.new_gc_metrics }}
{{- end }}
{{- if .Values.datadog.v2.jmx.service_check_prefix }}
service_check_prefix: "{{ .Values.datadog.v2.jmx.service_check_prefix }}"
{{- end }}
{{- if .Values.datadog.v2.jmx.conf }}
conf: {{ .Values.datadog.v2.jmx.conf | toYaml | nindent 2 }}
{{- end }}
{{- end -}}

{{/*
Custom integration configuration
*/}}
{{- define "base-chart.datadogCustomConfig" -}}
{{- range $key, $value := .Values.datadog.v2.instanceConfig }}
{{ $key }}: {{ $value | toYaml }}
{{- end }}
{{- end -}}

{{/*
Common instance configuration (tags, etc.)
*/}}
{{- define "base-chart.datadogCommonInstanceConfig" -}}
{{- if .Values.datadog.v2.tags }}
{{- $root := . }}
{{- $tagsDict := dict }}
tags:
{{- if kindIs "slice" .Values.datadog.v2.tags }}
  {{- /* Process array format tags */}}
  {{- range .Values.datadog.v2.tags }}
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
{{- else if kindIs "map" .Values.datadog.v2.tags }}
  {{- /* Process map format tags */}}
  {{- range $key, $value := .Values.datadog.v2.tags }}
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
{{- if .Values.datadog.v2.logs.enabled }}
ad.datadoghq.com/{{ include "base-chart.datadogContainerName" . }}.logs: |
  [{"source": "{{ .Values.datadog.v2.logs.source }}", "service": "{{ .Values.datadog.v2.logs.service }}"{{- if .Values.datadog.v2.logs.logProcessingRules }}, "log_processing_rules": {{ .Values.datadog.v2.logs.logProcessingRules | toJson }}{{- end }}}]
{{- end -}}
{{- end -}}

{{/*
Generate logs configuration
*/}}
{{- define "base-chart.datadogLogsConfig" -}}
- source: "{{ .Values.datadog.v2.logs.source }}"
  service: "{{ .Values.datadog.v2.logs.service }}"
  {{- if .Values.datadog.v2.logs.logProcessingRules }}
  log_processing_rules:
{{ .Values.datadog.v2.logs.logProcessingRules | toYaml | indent 4 }}
  {{- end }}
{{- end -}}

{{/*
Generate Datadog labels for resources
*/}}
{{- define "base-chart.datadogLabels" -}}
{{- if .Values.datadog.v2.enabled }}
datadog.monitoring: "true"
datadog.service: {{ include "base-chart.datadogContainerName" . | quote }}
datadog.environment: {{ .Values.datadog.v2.apm.environment | default "production" | quote }}
{{- end }}
{{- end -}}

{{/*
Generate Datadog tags with dynamic version
Supports both array format (["key:value"]) and map format ({key: value})
*/}}
{{- define "base-chart.datadogTags" -}}
{{- $tags := list }}
{{- $root := . }}
{{- if kindIs "slice" .Values.datadog.v2.tags }}
  {{- /* Array format: ["env:production", "team:backend"] */}}
  {{- range .Values.datadog.v2.tags }}
    {{- if hasPrefix "version:" . }}
      {{- $tags = append $tags (printf "version:%s" (include "base-chart.appVersion" $root)) }}
    {{- else }}
      {{- $tags = append $tags . }}
    {{- end }}
  {{- end }}
{{- else if kindIs "map" .Values.datadog.v2.tags }}
  {{- /* Map format: {env: production, team: backend} */}}
  {{- range $key, $value := .Values.datadog.v2.tags }}
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
{{- if .Values.datadog.v2.apm.enabled }}
- name: DD_SERVICE
  value: {{ .Values.datadog.v2.apm.serviceName | quote }}
- name: DD_ENV
  value: {{ .Values.datadog.v2.apm.environment | quote }}
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
