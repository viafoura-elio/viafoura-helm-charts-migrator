{{/*
Expand the name of the chart.
*/}}
{{- define "auth-service.name" -}}
{{- $chartName := "auth-service" }}
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
{{- define "auth-service.fullname" -}}
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
{{- define "auth-service.appVersion" -}}
{{- if .Values.image.tag }}
{{- printf "%s" .Values.image.tag }}
{{- else if .Chart.AppVersion }}
{{- printf "%s" .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Application Version Label
*/}}
{{- define "auth-service.appVersionLabel" -}}
app.kubernetes.io/version: {{ include "auth-service.appVersion" . | quote }}
{{- end }}

{{/*
Architecture Label
*/}}
{{- define "auth-service.archLabel" -}}
app.kubernetes.io/arch: {{ .Values.arch | default "amd64" }}
{{- end }}

{{/*
Rollout labels
*/}}
{{- define "auth-service.applicationLabels" -}}
app: {{ include "auth-service.name" . }}
version: {{ include "auth-service.appVersion" . | quote }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "auth-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "auth-service.labels" -}}
helm.sh/chart: {{ include "auth-service.chart" . }}
{{ include "auth-service.selectorLabels" . }}
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
{{- define "auth-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "auth-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "auth-service.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "auth-service.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the IAM role name for service account
*/}}
{{- define "auth-service.iamRoleName" -}}
{{- if .Values.serviceAccount.iamRole.name }}
{{- .Values.serviceAccount.iamRole.name }}
{{- else }}
{{- printf "%s.%s.pod" (include "auth-service.serviceAccountName" .) .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the IAM role ARN for IRSA
*/}}
{{- define "auth-service.iamRoleArn" -}}
{{- printf "arn:aws:iam::%s:role/%s" .Values.awsAccountId (include "auth-service.iamRoleName" .) }}
{{- end }}

{{/*
Create the Pod Identity Association name
*/}}
{{- define "auth-service.podIdentityAssociationName" -}}
{{- if .Values.serviceAccount.podIdentity.associationName }}
{{- .Values.serviceAccount.podIdentity.associationName }}
{{- else }}
{{- printf "%s-pod-identity" (include "auth-service.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the Pod Identity role ARN
*/}}
{{- define "auth-service.podIdentityRoleArn" -}}
{{- if .Values.serviceAccount.podIdentity.roleArn }}
{{- .Values.serviceAccount.podIdentity.roleArn }}
{{- else }}
{{- printf "arn:aws:iam::%s:role/%s-pod-role" .Values.serviceAccount.podIdentity.accountId (include "auth-service.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create image pull policy
*/}}
{{- define "auth-service.imagePullPolicy" -}}
{{- .Values.image.pullPolicy | default "IfNotPresent" }}
{{- end }}

{{/*
Create full image reference
*/}}
{{- define "auth-service.image" -}}
{{- printf "%s/%s:%s" .Values.image.repository .Values.image.name (include "auth-service.appVersion" .) }}
{{- end }}

{{/*
Create service name
*/}}
{{- define "auth-service.serviceName" -}}
{{- .Values.service.name | default (include "auth-service.fullname" .) }}
{{- end }}

{{/*
Create service alias
*/}}
{{- define "auth-service.serviceAlias" -}}
{{- .Values.service.alias | default (include "auth-service.name" .) }}
{{- end }}

{{/*
Create ConfigMap name
*/}}
{{- define "auth-service.configMapName" -}}
{{- include "auth-service.fullname" . }}-config
{{- end }}

{{/*
Create Secret name
*/}}
{{- define "auth-service.secretName" -}}
{{- include "auth-service.fullname" . }}-secret
{{- end }}

{{/*
Create HPA name
*/}}
{{- define "auth-service.hpaName" -}}
{{- include "auth-service.fullname" . }}-hpa
{{- end }}

{{/*
Create PodDisruptionBudget name
*/}}
{{- define "auth-service.pdbName" -}}
{{- include "auth-service.fullname" . }}-pdb
{{- end }}

{{/*
Create Role name
*/}}
{{- define "auth-service.roleName" -}}
{{- include "auth-service.fullname" . }}-role
{{- end }}

{{/*
Create ClusterRole name
*/}}
{{- define "auth-service.clusterRoleName" -}}
{{- include "auth-service.fullname" . }}-cluster-role
{{- end }}

{{/*
Create RoleBinding name
*/}}
{{- define "auth-service.roleBindingName" -}}
{{- include "auth-service.fullname" . }}-role-binding
{{- end }}

{{/*
Create ClusterRoleBinding name
*/}}
{{- define "auth-service.clusterRoleBindingName" -}}
{{- include "auth-service.fullname" . }}-cluster-role-binding
{{- end }}

{{/*
Create Rollout name
*/}}
{{- define "auth-service.rolloutName" -}}
{{- include "auth-service.fullname" . }}
{{- end }}

{{/*
Generate environment variables from configMapEnvVars
*/}}
{{- define "auth-service.configMapEnvVars" -}}
{{- range $key, $value := .Values.configMapEnvVars }}
- name: {{ $key }}
  valueFrom:
    configMapKeyRef:
      name: {{ include "auth-service.configMapName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate environment variables from secretsEnvVars
*/}}
{{- define "auth-service.secretEnvVars" -}}
{{- range $key, $value := .Values.secretsEnvVars }}
- name: {{ $key }}
  valueFrom:
    secretKeyRef:
      name: {{ include "auth-service.secretName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate direct environment variables
*/}}
{{- define "auth-service.directEnvVars" -}}
{{- range $key, $value := .Values.envVars }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}

{{/*
Generate all environment variables
*/}}
{{- define "auth-service.envVars" -}}
{{- include "auth-service.directEnvVars" . }}
{{- include "auth-service.configMapEnvVars" . }}
{{- include "auth-service.secretEnvVars" . }}
{{- end }}

{{/*
Generate container ports
*/}}
{{- define "auth-service.containerPorts" -}}
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
{{- define "auth-service.servicePorts" -}}
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
{{- define "auth-service.podAffinity" -}}
{{- if .Values.podAffinity }}
podAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
  {{- range .Values.podAffinity }}
  {{- if .enabled }}
  - weight: {{ .weight }}
    podAffinityTerm:
      labelSelector:
        matchLabels:
          {{- include "auth-service.selectorLabels" $ | nindent 10 }}
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
{{- define "auth-service.topologySpreadConstraints" -}}
{{- if .Values.topologySpreadConstraints }}
{{- range .Values.topologySpreadConstraints }}
{{- if .enabled }}
- maxSkew: {{ .maxSkew }}
  topologyKey: {{ .topologyKey }}
  whenUnsatisfiable: {{ .whenUnsatisfiable }}
  labelSelector:
    matchLabels:
      {{- include "auth-service.selectorLabels" $ | nindent 6 }}
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
        app.kubernetes.io/name: {{ include "auth-service.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}

  # Spread across nodes within zones
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: {{ include "auth-service.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
