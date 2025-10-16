{{/*
Shared pod template for both Deployment and Rollout
*/}}
{{- define "base-chart.podTemplate" -}}
metadata:
  annotations:
    {{- if .Values.podAnnotations }}
    {{- toYaml .Values.podAnnotations | nindent 4 }}
    {{- end }}
    {{- if .Values.datadog.enabled }}
    {{- include "base-chart.datadogAnnotations" . | indent 4 }}
    {{- end }}
    {{- if .Values.istio.enabled }}
    {{- $sidecarContent := include "base-chart.sidecarAnnotations" . | trim }}
    {{- if $sidecarContent }}
    {{- $sidecarContent | indent 4 }}
    {{- end }}
    {{- end }}
    {{- if .Values.defaultAnnotations }}
    {{- toYaml .Values.defaultAnnotations | nindent 4 }}
    {{- end }}
    app.kubernetes.io/checksum-values: {{ toYaml .Values | sha256sum }}
  labels:
    {{- include "base-chart.labels" . | nindent 4 }}
    {{- with .Values.podLabels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
spec:
  {{- with .Values.imagePullSecrets }}
  imagePullSecrets:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  serviceAccountName: {{ include "base-chart.serviceAccountName" . }}
  securityContext:
    {{- toYaml .Values.podSecurityContext | nindent 4 }}
  containers:
    - name: {{ .Chart.Name }}
      securityContext:
        {{- if .Values.system.enablePtrace }}
        capabilities:
          add:
            - SYS_PTRACE
          drop:
            - ALL
        readOnlyRootFilesystem: {{ .Values.securityContext.readOnlyRootFilesystem }}
        runAsNonRoot: {{ .Values.securityContext.runAsNonRoot }}
        runAsUser: {{ .Values.securityContext.runAsUser }}
        runAsGroup: {{ .Values.securityContext.runAsGroup }}
        allowPrivilegeEscalation: false
        seccompProfile:
          {{- toYaml .Values.securityContext.seccompProfile | nindent 10 }}
        {{- else }}
        {{- toYaml .Values.securityContext | nindent 8 }}
        {{- end }}
      image: {{ include "base-chart.image" . }}
      imagePullPolicy: {{ include "base-chart.imagePullPolicy" . }}
      ports:
        {{- include "base-chart.containerPorts" . | nindent 8 }}
      {{- if and .Values.livenessProbe.enabled .Values.livenessProbe.config }}
      livenessProbe:
        {{- toYaml .Values.livenessProbe.config | nindent 8 }}
      {{- end }}
      {{- if and .Values.readinessProbe.enabled .Values.readinessProbe.config }}
      readinessProbe:
        {{- toYaml .Values.readinessProbe.config | nindent 8 }}
      {{- end }}
      {{- if and .Values.startupProbe.enabled .Values.startupProbe.config }}
      startupProbe:
        {{- toYaml .Values.startupProbe.config | nindent 8 }}
      {{- end }}
      resources:
        {{- toYaml .Values.resources | nindent 8 }}
      {{- if .Values.envVars }}
      env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
      {{- range $key, $value := .Values.envVars }}
        - name: {{ $key }}
          value: {{ $value | quote }}
      {{- end }}
      {{- end }}
      {{- if or .Values.configMapEnvVars .Values.secretsEnvVars }}
      envFrom:
        {{- with .Values.configMapEnvVars }}
        - configMapRef:
            name: {{ include "base-chart.fullname" . }}-configMapEnvVars
        {{- end }}
        {{- if .Values.secretsEnvVars }}
        - secretRef:
            name: {{ include "base-chart.fullname" . }}-secretsEnvVars
        {{- end }}
      {{- end }}
      volumeMounts:
        - name: dsdsocket
          mountPath: /var/run/datadog
          readOnly: true
        - name: tmp-cache
          mountPath: /tmp
          readOnly: false
        {{- if or .Values.configMap .Values.secrets }}
        - name: {{ include "base-chart.fullname" . }}-config-volume
          mountPath: /usr/verticles/conf
          readOnly: true
        {{- end }}
      {{- with .Values.volumeMounts }}
        {{- toYaml . | nindent 8 }}
      {{- end }}
  {{- if or .Values.nodeSelector .Values.arch }}
  nodeSelector:
    {{- if .Values.arch }}
    kubernetes.io/arch: {{ .Values.arch }}
    {{- end }}
    {{- with .Values.nodeSelector }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- end }}
  {{- with .Values.affinity }}
  affinity:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- if .Values.podAffinity }}
  {{- include "base-chart.podAffinity" . | nindent 2 }}
  {{- end }}
  {{- with .Values.tolerations }}
  tolerations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- if .Values.topologySpreadConstraints }}
  topologySpreadConstraints:
  {{- include "base-chart.topologySpreadConstraints" . | nindent 2 }}
  {{- else if .Values.useCommonTopologySpreadConstraints }}
  {{- include "common.topologySpreadConstraints" . | nindent 2 }}
  {{- end }}
  volumes:
    # Datadog DogStatsD socket - required for metrics collection
    # Security Note: This host path mount is necessary for Datadog agent communication
    # but provides container access to host filesystem. Ensure Datadog agent is properly secured.
    - name: dsdsocket
      hostPath:
        path: /var/run/datadog/
    # Writable /tmp volume for Vert.x cache directory
    # Required when readOnlyRootFilesystem is enabled in security context
    - name: tmp-cache
      emptyDir: {}
    {{- if or .Values.configMap .Values.secrets }}
    # Projected volume combining ConfigMap and Secret sources
    - name: {{ include "base-chart.fullname" . }}-config-volume
      projected:
        defaultMode: 0644
        sources:
        {{- if .Values.configMap }}
        - configMap:
            name: {{ include "base-chart.configMapName" . }}
            optional: true
        {{- end }}
        {{- if .Values.secrets }}
        - secret:
            name: {{ include "base-chart.secretName" . }}
            optional: true
        {{- end }}
    {{- end }}
  {{- with .Values.volumes }}
    {{ toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
