{{/*
Expand the name of the chart.
*/}}
{{- define "heimdall.name" -}}
{{- $chartName := "heimdall" }}
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
{{- define "heimdall.fullname" -}}
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
{{- define "heimdall.appVersion" -}}
{{- if .Values.image.tag }}
{{- printf "%s" .Values.image.tag }}
{{- else if .Chart.AppVersion }}
{{- printf "%s" .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Application Version Label
*/}}
{{- define "heimdall.appVersionLabel" -}}
app.kubernetes.io/version: {{ include "heimdall.appVersion" . | quote }}
{{- end }}

{{/*
Architecture Label
*/}}
{{- define "heimdall.archLabel" -}}
app.kubernetes.io/arch: {{ .Values.arch | default "amd64" }}
{{- end }}

{{/*
Rollout labels
*/}}
{{- define "heimdall.applicationLabels" -}}
app: {{ include "heimdall.name" . }}
version: {{ include "heimdall.appVersion" . | quote }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "heimdall.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "heimdall.labels" -}}
helm.sh/chart: {{ include "heimdall.chart" . }}
{{ include "heimdall.selectorLabels" . }}
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
{{- define "heimdall.selectorLabels" -}}
app.kubernetes.io/name: {{ include "heimdall.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "heimdall.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "heimdall.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the IAM role name for service account
*/}}
{{- define "heimdall.iamRoleName" -}}
{{- if .Values.serviceAccount.iamRole.name }}
{{- .Values.serviceAccount.iamRole.name }}
{{- else }}
{{- printf "%s.%s.pod" (include "heimdall.serviceAccountName" .) .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the IAM role ARN for IRSA
*/}}
{{- define "heimdall.iamRoleArn" -}}
{{- printf "arn:aws:iam::%s:role/%s" .Values.awsAccountId (include "heimdall.iamRoleName" .) }}
{{- end }}

{{/*
Create the Pod Identity Association name
*/}}
{{- define "heimdall.podIdentityAssociationName" -}}
{{- if .Values.serviceAccount.podIdentity.associationName }}
{{- .Values.serviceAccount.podIdentity.associationName }}
{{- else }}
{{- printf "%s-pod-identity" (include "heimdall.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the Pod Identity role ARN
*/}}
{{- define "heimdall.podIdentityRoleArn" -}}
{{- if .Values.serviceAccount.podIdentity.roleArn }}
{{- .Values.serviceAccount.podIdentity.roleArn }}
{{- else }}
{{- printf "arn:aws:iam::%s:role/%s-pod-role" .Values.serviceAccount.podIdentity.accountId (include "heimdall.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create image pull policy
*/}}
{{- define "heimdall.imagePullPolicy" -}}
{{- .Values.image.pullPolicy | default "IfNotPresent" }}
{{- end }}

{{/*
Create full image reference
*/}}
{{- define "heimdall.image" -}}
{{- printf "%s/%s:%s" .Values.image.repository .Values.image.name (include "heimdall.appVersion" .) }}
{{- end }}

{{/*
Create service name
*/}}
{{- define "heimdall.serviceName" -}}
{{- .Values.service.name | default (include "heimdall.fullname" .) }}
{{- end }}

{{/*
Create service alias
*/}}
{{- define "heimdall.serviceAlias" -}}
{{- .Values.service.alias | default (include "heimdall.name" .) }}
{{- end }}

{{/*
Create ConfigMap name
*/}}
{{- define "heimdall.configMapName" -}}
{{- include "heimdall.fullname" . }}-config
{{- end }}

{{/*
Create Secret name
*/}}
{{- define "heimdall.secretName" -}}
{{- include "heimdall.fullname" . }}-secret
{{- end }}

{{/*
Create HPA name
*/}}
{{- define "heimdall.hpaName" -}}
{{- include "heimdall.fullname" . }}-hpa
{{- end }}

{{/*
Create PodDisruptionBudget name
*/}}
{{- define "heimdall.pdbName" -}}
{{- include "heimdall.fullname" . }}-pdb
{{- end }}

{{/*
Create Role name
*/}}
{{- define "heimdall.roleName" -}}
{{- include "heimdall.fullname" . }}-role
{{- end }}

{{/*
Create ClusterRole name
*/}}
{{- define "heimdall.clusterRoleName" -}}
{{- include "heimdall.fullname" . }}-cluster-role
{{- end }}

{{/*
Create RoleBinding name
*/}}
{{- define "heimdall.roleBindingName" -}}
{{- include "heimdall.fullname" . }}-role-binding
{{- end }}

{{/*
Create ClusterRoleBinding name
*/}}
{{- define "heimdall.clusterRoleBindingName" -}}
{{- include "heimdall.fullname" . }}-cluster-role-binding
{{- end }}

{{/*
Create Rollout name
*/}}
{{- define "heimdall.rolloutName" -}}
{{- include "heimdall.fullname" . }}
{{- end }}

{{/*
Generate environment variables from configMapEnvVars
*/}}
{{- define "heimdall.configMapEnvVars" -}}
{{- range $key, $value := .Values.configMapEnvVars }}
- name: {{ $key }}
  valueFrom:
    configMapKeyRef:
      name: {{ include "heimdall.configMapName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate environment variables from secretsEnvVars
*/}}
{{- define "heimdall.secretEnvVars" -}}
{{- range $key, $value := .Values.secretsEnvVars }}
- name: {{ $key }}
  valueFrom:
    secretKeyRef:
      name: {{ include "heimdall.secretName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate direct environment variables
*/}}
{{- define "heimdall.directEnvVars" -}}
{{- range $key, $value := .Values.envVars }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}

{{/*
Generate all environment variables
*/}}
{{- define "heimdall.envVars" -}}
{{- include "heimdall.directEnvVars" . }}
{{- include "heimdall.configMapEnvVars" . }}
{{- include "heimdall.secretEnvVars" . }}
{{- end }}

{{/*
Generate container ports
*/}}
{{- define "heimdall.containerPorts" -}}
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
{{- define "heimdall.servicePorts" -}}
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
{{- define "heimdall.podAffinity" -}}
{{- if .Values.podAffinity }}
podAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
  {{- range .Values.podAffinity }}
  {{- if .enabled }}
  - weight: {{ .weight }}
    podAffinityTerm:
      labelSelector:
        matchLabels:
          {{- include "heimdall.selectorLabels" $ | nindent 10 }}
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
{{- define "heimdall.topologySpreadConstraints" -}}
{{- if .Values.topologySpreadConstraints }}
{{- range .Values.topologySpreadConstraints }}
{{- if .enabled }}
- maxSkew: {{ .maxSkew }}
  topologyKey: {{ .topologyKey }}
  whenUnsatisfiable: {{ .whenUnsatisfiable }}
  labelSelector:
    matchLabels:
      {{- include "heimdall.selectorLabels" $ | nindent 6 }}
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
        app.kubernetes.io/name: {{ include "heimdall.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}

  # Spread across nodes within zones
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: {{ include "heimdall.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
