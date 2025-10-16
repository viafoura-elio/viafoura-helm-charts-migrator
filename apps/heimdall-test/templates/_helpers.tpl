{{/*
Expand the name of the chart.
*/}}
{{- define "base-chart.name" -}}
{{- $chartName := "base-chart" }}
{{- if .Chart }}
{{- $chartName = .Chart.Name }}
{{- end }}
{{- $name := $chartName }}
{{- if and .Values .Values.nameOverride }}
{{- $name = .Values.nameOverride }}
{{- end }}
{{- $name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "base-chart.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Application Version
*/}}
{{- define "base-chart.appVersion" -}}
{{- if .Values.image.tag }}
{{- printf "%s" .Values.image.tag }}
{{- else if .Chart.AppVersion }}
{{- printf "%s" .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Application Version Label
*/}}
{{- define "base-chart.appVersionLabel" -}}
app.kubernetes.io/version: {{ include "base-chart.appVersion" . | quote }}
{{- end }}

{{/*
Architecture Label
*/}}
{{- define "base-chart.archLabel" -}}
app.kubernetes.io/arch: {{ .Values.arch | default "amd64" }}
{{- end }}

{{/*
Rollout labels
*/}}
{{- define "base-chart.applicationLabels" -}}
app: {{ include "base-chart.name" . }}
version: {{ include "base-chart.appVersion" . | quote }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "base-chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "base-chart.labels" -}}
helm.sh/chart: {{ include "base-chart.chart" . }}
{{ include "base-chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.defaultAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "base-chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "base-chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "base-chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "base-chart.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the IAM role name for service account
*/}}
{{- define "base-chart.iamRoleName" -}}
{{- if .Values.serviceAccount.iamRole.name }}
{{- .Values.serviceAccount.iamRole.name }}
{{- else }}
{{- printf "%s.%s.pod" (include "base-chart.serviceAccountName" .) .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the IAM role ARN for IRSA
*/}}
{{- define "base-chart.iamRoleArn" -}}
{{- printf "arn:aws:iam::%s:role/%s" .Values.awsAccountId (include "base-chart.iamRoleName" .) }}
{{- end }}

{{/*
Create the Pod Identity Association name
*/}}
{{- define "base-chart.podIdentityAssociationName" -}}
{{- if .Values.serviceAccount.podIdentity.associationName }}
{{- .Values.serviceAccount.podIdentity.associationName }}
{{- else }}
{{- printf "%s-pod-identity" (include "base-chart.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the Pod Identity role ARN
*/}}
{{- define "base-chart.podIdentityRoleArn" -}}
{{- if .Values.serviceAccount.podIdentity.roleArn }}
{{- .Values.serviceAccount.podIdentity.roleArn }}
{{- else }}
{{- printf "arn:aws:iam::%s:role/%s-pod-role" .Values.serviceAccount.podIdentity.accountId (include "base-chart.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create image pull policy
*/}}
{{- define "base-chart.imagePullPolicy" -}}
{{- .Values.image.pullPolicy | default "IfNotPresent" }}
{{- end }}

{{/*
Create full image reference
*/}}
{{- define "base-chart.image" -}}
{{- printf "%s/%s:%s" .Values.image.repository .Values.image.name (include "base-chart.appVersion" .) }}
{{- end }}

{{/*
Create service name
*/}}
{{- define "base-chart.serviceName" -}}
{{- .Values.service.name | default (include "base-chart.fullname" .) }}
{{- end }}

{{/*
Create service alias
*/}}
{{- define "base-chart.serviceAlias" -}}
{{- .Values.service.alias | default (include "base-chart.name" .) }}
{{- end }}

{{/*
Create ConfigMap name
*/}}
{{- define "base-chart.configMapName" -}}
{{- include "base-chart.fullname" . }}-config
{{- end }}

{{/*
Create Secret name
*/}}
{{- define "base-chart.secretName" -}}
{{- include "base-chart.fullname" . }}-secret
{{- end }}

{{/*
Create HPA name
*/}}
{{- define "base-chart.hpaName" -}}
{{- include "base-chart.fullname" . }}-hpa
{{- end }}

{{/*
Create PodDisruptionBudget name
*/}}
{{- define "base-chart.pdbName" -}}
{{- include "base-chart.fullname" . }}-pdb
{{- end }}

{{/*
Create Role name
*/}}
{{- define "base-chart.roleName" -}}
{{- include "base-chart.fullname" . }}-role
{{- end }}

{{/*
Create ClusterRole name
*/}}
{{- define "base-chart.clusterRoleName" -}}
{{- include "base-chart.fullname" . }}-cluster-role
{{- end }}

{{/*
Create RoleBinding name
*/}}
{{- define "base-chart.roleBindingName" -}}
{{- include "base-chart.fullname" . }}-role-binding
{{- end }}

{{/*
Create ClusterRoleBinding name
*/}}
{{- define "base-chart.clusterRoleBindingName" -}}
{{- include "base-chart.fullname" . }}-cluster-role-binding
{{- end }}

{{/*
Create Rollout name
*/}}
{{- define "base-chart.rolloutName" -}}
{{- include "base-chart.fullname" . }}
{{- end }}

{{/*
Generate environment variables from configMapEnvVars
*/}}
{{- define "base-chart.configMapEnvVars" -}}
{{- range $key, $value := .Values.configMapEnvVars }}
- name: {{ $key }}
  valueFrom:
    configMapKeyRef:
      name: {{ include "base-chart.configMapName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate environment variables from secretsEnvVars
*/}}
{{- define "base-chart.secretEnvVars" -}}
{{- range $key, $value := .Values.secretsEnvVars }}
- name: {{ $key }}
  valueFrom:
    secretKeyRef:
      name: {{ include "base-chart.secretName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate direct environment variables
*/}}
{{- define "base-chart.directEnvVars" -}}
{{- range $key, $value := .Values.envVars }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}

{{/*
Generate all environment variables
*/}}
{{- define "base-chart.envVars" -}}
{{- include "base-chart.directEnvVars" . }}
{{- include "base-chart.configMapEnvVars" . }}
{{- include "base-chart.secretEnvVars" . }}
{{- end }}

{{/*
Generate container ports
*/}}
{{- define "base-chart.containerPorts" -}}
- name: {{ .Values.service.portName }}
  containerPort: {{ .Values.service.targetPort | default .Values.service.port }}
  protocol: {{ .Values.service.protocol }}
{{- if .Values.service.containerPorts.jmx.enabled }}
- name: {{ .Values.service.containerPorts.jmx.containerPortName }}
  containerPort: {{ .Values.service.containerPorts.jmx.containerPort }}
  protocol: {{ .Values.service.containerPorts.jmx.containerProtocol }}
{{- end }}
{{- if .Values.service.containerPorts.metrics.enabled }}
- name: {{ .Values.service.containerPorts.metrics.containerPortName }}
  containerPort: {{ .Values.service.containerPorts.metrics.containerPort }}
  protocol: {{ .Values.service.containerPorts.metrics.containerProtocol }}
{{- end }}
{{- end }}

{{/*
Generate service ports
*/}}
{{- define "base-chart.servicePorts" -}}
- name: {{ .Values.service.portName }}
  port: {{ .Values.service.port }}
  targetPort: {{ .Values.service.targetPort }}
  protocol: {{ .Values.service.protocol }}
  {{- if and (eq .Values.service.type "NodePort") .Values.service.nodePort }}
  nodePort: {{ .Values.service.nodePort }}
  {{- end }}
{{- if and .Values.service.containerPorts.jmx.enabled .Values.service.containerPorts.jmx.exposeService }}
- name: {{ .Values.service.containerPorts.jmx.containerPortName }}
  port: {{ .Values.service.containerPorts.jmx.containerPort }}
  targetPort: {{ .Values.service.containerPorts.jmx.containerPort }}
  protocol: {{ .Values.service.containerPorts.jmx.containerProtocol }}
{{- end }}
{{- if and .Values.service.containerPorts.metrics.enabled .Values.service.containerPorts.metrics.exposeService }}
- name: {{ .Values.service.containerPorts.metrics.containerPortName }}
  port: {{ .Values.service.containerPorts.metrics.containerPort }}
  targetPort: {{ .Values.service.containerPorts.metrics.containerPort }}
  protocol: {{ .Values.service.containerPorts.metrics.containerProtocol }}
{{- end }}
{{- end }}

{{/*
Generate pod affinity rules
*/}}
{{- define "base-chart.podAffinity" -}}
{{- if .Values.podAffinity }}
podAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
  {{- range .Values.podAffinity }}
  {{- if .enabled }}
  - weight: {{ .weight }}
    podAffinityTerm:
      labelSelector:
        matchLabels:
          {{- include "base-chart.selectorLabels" $ | nindent 10 }}
          {{- if not .excludeStable }}
          {{- if $.Values.rollout.enabled }}
          rollouts-pod-template-hash: stable
          {{- end }}
          {{- end }}
      topologyKey: {{ .topologyKey }}
  {{- end }}
  {{- end }}
{{- end }}
{{- end }}

{{/*
Generate topology spread constraints
*/}}
{{- define "base-chart.topologySpreadConstraints" -}}
{{- if .Values.topologySpreadConstraints }}
{{- range .Values.topologySpreadConstraints }}
{{- if .enabled }}
- maxSkew: {{ .maxSkew }}
  topologyKey: {{ .topologyKey }}
  whenUnsatisfiable: {{ .whenUnsatisfiable }}
  labelSelector:
    matchLabels:
      {{- include "base-chart.selectorLabels" $ | nindent 6 }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common topology spread constraints for better pod distribution
*/}}
{{- define "common.topologySpreadConstraints" -}}
topologySpreadConstraints:
  # Spread across availability zones
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: {{ include "base-chart.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}

  # Spread across nodes within zones
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: {{ include "base-chart.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
