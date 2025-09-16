{{/*
Expand the name of the chart.
*/}}
{{- define "livechat.name" -}}
{{- $chartName := "livechat" }}
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
{{- define "livechat.fullname" -}}
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
{{- define "livechat.appVersion" -}}
{{- if .Values.image.tag }}
{{- printf "%s" .Values.image.tag }}
{{- else if .Chart.AppVersion }}
{{- printf "%s" .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Application Version Label
*/}}
{{- define "livechat.appVersionLabel" -}}
app.kubernetes.io/version: {{ include "livechat.appVersion" . | quote }}
{{- end }}

{{/*
Architecture Label
*/}}
{{- define "livechat.archLabel" -}}
app.kubernetes.io/arch: {{ .Values.arch | default "amd64" }}
{{- end }}

{{/*
Rollout labels
*/}}
{{- define "livechat.applicationLabels" -}}
app: {{ include "livechat.name" . }}
version: {{ include "livechat.appVersion" . | quote }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "livechat.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "livechat.labels" -}}
helm.sh/chart: {{ include "livechat.chart" . }}
{{ include "livechat.selectorLabels" . }}
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
{{- define "livechat.selectorLabels" -}}
app.kubernetes.io/name: {{ include "livechat.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "livechat.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "livechat.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the IAM role name for service account
*/}}
{{- define "livechat.iamRoleName" -}}
{{- if .Values.serviceAccount.iamRole.name }}
{{- .Values.serviceAccount.iamRole.name }}
{{- else }}
{{- printf "%s.%s.pod" (include "livechat.serviceAccountName" .) .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the IAM role ARN for IRSA
*/}}
{{- define "livechat.iamRoleArn" -}}
{{- printf "arn:aws:iam::%s:role/%s" .Values.serviceAccount.iamRole.accountId (include "livechat.iamRoleName" .) }}
{{- end }}

{{/*
Create the Pod Identity Association name
*/}}
{{- define "livechat.podIdentityAssociationName" -}}
{{- if .Values.serviceAccount.podIdentity.associationName }}
{{- .Values.serviceAccount.podIdentity.associationName }}
{{- else }}
{{- printf "%s-pod-identity" (include "livechat.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the Pod Identity role ARN
*/}}
{{- define "livechat.podIdentityRoleArn" -}}
{{- if .Values.serviceAccount.podIdentity.roleArn }}
{{- .Values.serviceAccount.podIdentity.roleArn }}
{{- else }}
{{- printf "arn:aws:iam::%s:role/%s-pod-role" .Values.serviceAccount.podIdentity.accountId (include "livechat.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create image pull policy
*/}}
{{- define "livechat.imagePullPolicy" -}}
{{- .Values.image.pullPolicy | default "IfNotPresent" }}
{{- end }}

{{/*
Create full image reference
*/}}
{{- define "livechat.image" -}}
{{- printf "%s/%s:%s" .Values.image.repository .Values.image.name (include "livechat.appVersion" .) }}
{{- end }}

{{/*
Create service name
*/}}
{{- define "livechat.serviceName" -}}
{{- .Values.service.name | default (include "livechat.fullname" .) }}
{{- end }}

{{/*
Create service alias
*/}}
{{- define "livechat.serviceAlias" -}}
{{- .Values.service.alias | default (include "livechat.name" .) }}
{{- end }}

{{/*
Create ConfigMap name
*/}}
{{- define "livechat.configMapName" -}}
{{- include "livechat.fullname" . }}-config
{{- end }}

{{/*
Create Secret name
*/}}
{{- define "livechat.secretName" -}}
{{- include "livechat.fullname" . }}-secret
{{- end }}

{{/*
Create HPA name
*/}}
{{- define "livechat.hpaName" -}}
{{- include "livechat.fullname" . }}-hpa
{{- end }}

{{/*
Create PodDisruptionBudget name
*/}}
{{- define "livechat.pdbName" -}}
{{- include "livechat.fullname" . }}-pdb
{{- end }}

{{/*
Create Role name
*/}}
{{- define "livechat.roleName" -}}
{{- include "livechat.fullname" . }}-role
{{- end }}

{{/*
Create ClusterRole name
*/}}
{{- define "livechat.clusterRoleName" -}}
{{- include "livechat.fullname" . }}-cluster-role
{{- end }}

{{/*
Create RoleBinding name
*/}}
{{- define "livechat.roleBindingName" -}}
{{- include "livechat.fullname" . }}-role-binding
{{- end }}

{{/*
Create ClusterRoleBinding name
*/}}
{{- define "livechat.clusterRoleBindingName" -}}
{{- include "livechat.fullname" . }}-cluster-role-binding
{{- end }}

{{/*
Create Rollout name
*/}}
{{- define "livechat.rolloutName" -}}
{{- include "livechat.fullname" . }}
{{- end }}

{{/*
Generate environment variables from configMapEnvVars
*/}}
{{- define "livechat.configMapEnvVars" -}}
{{- range $key, $value := .Values.configMapEnvVars }}
- name: {{ $key }}
  valueFrom:
    configMapKeyRef:
      name: {{ include "livechat.configMapName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate environment variables from secretsEnvVars
*/}}
{{- define "livechat.secretEnvVars" -}}
{{- range $key, $value := .Values.secretsEnvVars }}
- name: {{ $key }}
  valueFrom:
    secretKeyRef:
      name: {{ include "livechat.secretName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate direct environment variables
*/}}
{{- define "livechat.directEnvVars" -}}
{{- range $key, $value := .Values.envVars }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}

{{/*
Generate all environment variables
*/}}
{{- define "livechat.envVars" -}}
{{- include "livechat.directEnvVars" . }}
{{- include "livechat.configMapEnvVars" . }}
{{- include "livechat.secretEnvVars" . }}
{{- end }}

{{/*
Generate container ports
*/}}
{{- define "livechat.containerPorts" -}}
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
{{- define "livechat.servicePorts" -}}
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
{{- define "livechat.podAffinity" -}}
{{- if .Values.podAffinity }}
podAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
  {{- range .Values.podAffinity }}
  {{- if .enabled }}
  - weight: {{ .weight }}
    podAffinityTerm:
      labelSelector:
        matchLabels:
          {{- include "livechat.selectorLabels" $ | nindent 10 }}
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
{{- define "livechat.topologySpreadConstraints" -}}
{{- if .Values.topologySpreadConstraints }}
{{- range .Values.topologySpreadConstraints }}
{{- if .enabled }}
- maxSkew: {{ .maxSkew }}
  topologyKey: {{ .topologyKey }}
  whenUnsatisfiable: {{ .whenUnsatisfiable }}
  labelSelector:
    matchLabels:
      {{- include "livechat.selectorLabels" $ | nindent 6 }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
