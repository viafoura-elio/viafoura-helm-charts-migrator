{{/*
Expand the name of the chart.
*/}}
{{- define "livecomments.name" -}}
{{- $chartName := "livecomments" }}
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
{{- define "livecomments.fullname" -}}
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
{{- define "livecomments.appVersion" -}}
{{- if .Values.image.tag }}
{{- printf "%s" .Values.image.tag }}
{{- else if .Chart.AppVersion }}
{{- printf "%s" .Chart.AppVersion }}
{{- end }}
{{- end }}

{{/*
Application Version Label
*/}}
{{- define "livecomments.appVersionLabel" -}}
app.kubernetes.io/version: {{ include "livecomments.appVersion" . | quote }}
{{- end }}

{{/*
Architecture Label
*/}}
{{- define "livecomments.archLabel" -}}
app.kubernetes.io/arch: {{ .Values.arch | default "amd64" }}
{{- end }}

{{/*
Rollout labels
*/}}
{{- define "livecomments.applicationLabels" -}}
app: {{ include "livecomments.name" . }}
version: {{ include "livecomments.appVersion" . | quote }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "livecomments.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "livecomments.labels" -}}
helm.sh/chart: {{ include "livecomments.chart" . }}
{{ include "livecomments.selectorLabels" . }}
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
{{- define "livecomments.selectorLabels" -}}
app.kubernetes.io/name: {{ include "livecomments.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "livecomments.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "livecomments.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the IAM role name for service account
*/}}
{{- define "livecomments.iamRoleName" -}}
{{- if .Values.serviceAccount.iamRole.name }}
{{- .Values.serviceAccount.iamRole.name }}
{{- else }}
{{- printf "%s.%s.pod" (include "livecomments.serviceAccountName" .) .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Create the IAM role ARN for IRSA
*/}}
{{- define "livecomments.iamRoleArn" -}}
{{- printf "arn:aws:iam::%s:role/%s" .Values.awsAccountId (include "livecomments.iamRoleName" .) }}
{{- end }}

{{/*
Create the Pod Identity Association name
*/}}
{{- define "livecomments.podIdentityAssociationName" -}}
{{- if .Values.serviceAccount.podIdentity.associationName }}
{{- .Values.serviceAccount.podIdentity.associationName }}
{{- else }}
{{- printf "%s-pod-identity" (include "livecomments.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the Pod Identity role ARN
*/}}
{{- define "livecomments.podIdentityRoleArn" -}}
{{- if .Values.serviceAccount.podIdentity.roleArn }}
{{- .Values.serviceAccount.podIdentity.roleArn }}
{{- else }}
{{- printf "arn:aws:iam::%s:role/%s-pod-role" .Values.serviceAccount.podIdentity.accountId (include "livecomments.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create image pull policy
*/}}
{{- define "livecomments.imagePullPolicy" -}}
{{- .Values.image.pullPolicy | default "IfNotPresent" }}
{{- end }}

{{/*
Create full image reference
*/}}
{{- define "livecomments.image" -}}
{{- printf "%s/%s:%s" .Values.image.repository .Values.image.name (include "livecomments.appVersion" .) }}
{{- end }}

{{/*
Create service name
*/}}
{{- define "livecomments.serviceName" -}}
{{- .Values.service.name | default (include "livecomments.fullname" .) }}
{{- end }}

{{/*
Create service alias
*/}}
{{- define "livecomments.serviceAlias" -}}
{{- .Values.service.alias | default (include "livecomments.name" .) }}
{{- end }}

{{/*
Create ConfigMap name
*/}}
{{- define "livecomments.configMapName" -}}
{{- include "livecomments.fullname" . }}-config
{{- end }}

{{/*
Create Secret name
*/}}
{{- define "livecomments.secretName" -}}
{{- include "livecomments.fullname" . }}-secret
{{- end }}

{{/*
Create HPA name
*/}}
{{- define "livecomments.hpaName" -}}
{{- include "livecomments.fullname" . }}-hpa
{{- end }}

{{/*
Create PodDisruptionBudget name
*/}}
{{- define "livecomments.pdbName" -}}
{{- include "livecomments.fullname" . }}-pdb
{{- end }}

{{/*
Create Role name
*/}}
{{- define "livecomments.roleName" -}}
{{- include "livecomments.fullname" . }}-role
{{- end }}

{{/*
Create ClusterRole name
*/}}
{{- define "livecomments.clusterRoleName" -}}
{{- include "livecomments.fullname" . }}-cluster-role
{{- end }}

{{/*
Create RoleBinding name
*/}}
{{- define "livecomments.roleBindingName" -}}
{{- include "livecomments.fullname" . }}-role-binding
{{- end }}

{{/*
Create ClusterRoleBinding name
*/}}
{{- define "livecomments.clusterRoleBindingName" -}}
{{- include "livecomments.fullname" . }}-cluster-role-binding
{{- end }}

{{/*
Create Rollout name
*/}}
{{- define "livecomments.rolloutName" -}}
{{- include "livecomments.fullname" . }}
{{- end }}

{{/*
Generate environment variables from configMapEnvVars
*/}}
{{- define "livecomments.configMapEnvVars" -}}
{{- range $key, $value := .Values.configMapEnvVars }}
- name: {{ $key }}
  valueFrom:
    configMapKeyRef:
      name: {{ include "livecomments.configMapName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate environment variables from secretsEnvVars
*/}}
{{- define "livecomments.secretEnvVars" -}}
{{- range $key, $value := .Values.secretsEnvVars }}
- name: {{ $key }}
  valueFrom:
    secretKeyRef:
      name: {{ include "livecomments.secretName" $ }}
      key: {{ $key }}
{{- end }}
{{- end }}

{{/*
Generate direct environment variables
*/}}
{{- define "livecomments.directEnvVars" -}}
{{- range $key, $value := .Values.envVars }}
- name: {{ $key }}
  value: {{ $value | quote }}
{{- end }}
{{- end }}

{{/*
Generate all environment variables
*/}}
{{- define "livecomments.envVars" -}}
{{- include "livecomments.directEnvVars" . }}
{{- include "livecomments.configMapEnvVars" . }}
{{- include "livecomments.secretEnvVars" . }}
{{- end }}

{{/*
Generate container ports
*/}}
{{- define "livecomments.containerPorts" -}}
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
{{- define "livecomments.servicePorts" -}}
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
{{- define "livecomments.podAffinity" -}}
{{- if .Values.podAffinity }}
podAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
  {{- range .Values.podAffinity }}
  {{- if .enabled }}
  - weight: {{ .weight }}
    podAffinityTerm:
      labelSelector:
        matchLabels:
          {{- include "livecomments.selectorLabels" $ | nindent 10 }}
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
{{- define "livecomments.topologySpreadConstraints" -}}
{{- if .Values.topologySpreadConstraints }}
{{- range .Values.topologySpreadConstraints }}
{{- if .enabled }}
- maxSkew: {{ .maxSkew }}
  topologyKey: {{ .topologyKey }}
  whenUnsatisfiable: {{ .whenUnsatisfiable }}
  labelSelector:
    matchLabels:
      {{- include "livecomments.selectorLabels" $ | nindent 6 }}
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
        app.kubernetes.io/name: {{ include "livecomments.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}

  # Spread across nodes within zones
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: {{ include "livecomments.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
