{{/*
Generate hosts for Istio Gateway and VirtualService
*/}}
{{- define "livecomments.hosts" -}}
{{- $hosts := list }}
{{- if and .Values.hosts.public.enabled (gt (len .Values.hosts.public.domains) 0) }}
{{- range .Values.hosts.public.domains }}
  {{- $hosts = append $hosts . }}
{{- end }}
{{- end }}
{{- if and .Values.hosts.private.enabled (gt (len .Values.hosts.private.domains) 0) }}
{{- range .Values.hosts.private.domains }}
  {{- $hosts = append $hosts . }}
{{- end }}
{{- end }}
{{- join "," $hosts }}
{{- end }}

{{/*
Generate hosts list for Istio Gateway and VirtualService
*/}}
{{- define "livecomments.hostsList" -}}
{{- $hosts := list }}
{{- if and .Values.hosts.public.enabled (gt (len .Values.hosts.public.domains) 0) }}
{{- range .Values.hosts.public.domains }}
  {{- $hosts = append $hosts . }}
{{- end }}
{{- end }}
{{- if and .Values.hosts.private.enabled (gt (len .Values.hosts.private.domains) 0) }}
{{- range .Values.hosts.private.domains }}
  {{- $hosts = append $hosts . }}
{{- end }}
{{- end }}
{{- toYaml $hosts }}
{{- end }}

{{/*
Generate hosts list for Private Gateway only
*/}}
{{- define "livecomments.privateHostsList" -}}
{{- $hosts := list }}
{{- if and .Values.hosts.private.enabled (gt (len .Values.hosts.private.domains) 0) }}
{{- range .Values.hosts.private.domains }}
  {{- $hosts = append $hosts . }}
{{- end }}
{{- end }}
{{- toYaml $hosts }}
{{- end }}

{{/*
Generate hosts list for Public Gateway only
*/}}
{{- define "livecomments.publicHostsList" -}}
{{- $hosts := list }}
{{- if and .Values.hosts.public.enabled (gt (len .Values.hosts.public.domains) 0) }}
{{- range .Values.hosts.public.domains }}
  {{- $hosts = append $hosts . }}
{{- end }}
{{- end }}
{{- toYaml $hosts }}
{{- end }}

{{/*
Create Public Gateway name
*/}}
{{- define "livecomments.publicGatewayName" -}}
{{- if .Values.istio.gateway.public.name }}
{{- .Values.istio.gateway.public.name }}
{{- else }}
{{- include "livecomments.fullname" . }}-public-gateway
{{- end }}
{{- end }}

{{/*
Create Private Gateway name
*/}}
{{- define "livecomments.privateGatewayName" -}}
{{- if .Values.istio.gateway.private.name }}
{{- .Values.istio.gateway.private.name }}
{{- else }}
{{- include "livecomments.fullname" . }}-private-gateway
{{- end }}
{{- end }}

{{/*
Create VirtualService name
*/}}
{{- define "livecomments.virtualServiceName" -}}
{{- include "livecomments.fullname" . }}-vs
{{- end }}

{{/*
Create DestinationRule name
*/}}
{{- define "livecomments.destinationRuleName" -}}
{{- include "livecomments.fullname" . }}-dr
{{- end }}

{{/*
Create ServiceMonitor name
*/}}
{{- define "livecomments.serviceMonitorName" -}}
{{- include "livecomments.fullname" . }}-metrics
{{- end }}


{{/*
Gateway Name
*/}}
{{- define "livecomments.gateway" -}}
{{- printf "%s-gateway" (include "livecomments.fullname" .) }}
{{- end -}}

{{/*
VirtualService Name
*/}}
{{- define "livecomments.virtualservice" -}}
{{- printf "%s-vs" (include "livecomments.fullname" .) }}
{{- end -}}

{{/*
DestinationRule Name
*/}}
{{- define "livecomments.destinationrule" -}}
{{ printf "%s-dr" (include "livecomments.fullname" .) }}
{{- end -}}

{{/*
Gateway Labels
*/}}
{{- define "livecomments.gatewayLabels" -}}
{{- include "livecomments.labels" . }}
{{- if .Values.istio.gateway.labels }}
{{- range $key, $val := .Values.istio.gateway.labels }}
{{ $key }}: {{ $val | quote }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
Gateway Annotations
*/}}
{{- define "livecomments.gatewayAnnotations" -}}
{{- if or (hasKey .Values.istio "globals" | and .Values.istio.globals.annotations) .Values.istio.certManager.enabled }}
annotations:
  {{- with .Values.istio.globals.annotations }}
    {{- range $key, $value := . }}
  {{ $key }}: {{ $value | quote }}
    {{- end }}
  {{- end }}
  {{- if .Values.istio.certManager.enabled }}
  cert-manager.io/cluster-issuer: {{ .Values.istio.certManager.issuer | default "letsencrypt-prod" | quote }}
  {{- end }}
{{- end }}
{{- end -}}

{{/*
Common Gateways Selector
*/}}
{{- define "livecomments.commonGatewaySelector" -}}
{{- if .Values.istio.gateway.selector }}
{{- range $key, $value := .Values.istio.gateway.selector }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end -}}
{{- end -}}

{{/*
Private Gateways Selector
*/}}
{{- define "livecomments.privateGatewaySelector" -}}
{{- $hasCommon := .Values.istio.gateway.selector -}}
{{- $hasPrivate := .Values.istio.gateway.private.selector -}}
{{- $first := true -}}
{{- if $hasCommon }}
{{- range $key, $value := .Values.istio.gateway.selector }}
{{- if not $first }}{{ "\n" }}{{ end }}{{ $key }}: {{ $value | quote }}
{{- $first = false -}}
{{- end }}
{{- end }}
{{- if $hasPrivate }}
{{- range $key, $value := .Values.istio.gateway.private.selector }}
{{- if not $first }}{{ "\n" }}{{ end }}{{ $key }}: {{ $value | quote }}
{{- $first = false -}}
{{- end }}
{{- end -}}
{{- end -}}

{{/*
Public Gateways Selector
*/}}
{{- define "livecomments.publicGatewaySelector" -}}
{{- $hasCommon := .Values.istio.gateway.selector -}}
{{- $hasPublic := .Values.istio.gateway.public.selector -}}
{{- $first := true -}}
{{- if $hasCommon }}
{{- range $key, $value := .Values.istio.gateway.selector }}
{{- if not $first }}{{ "\n" }}{{ end }}{{ $key }}: {{ $value | quote }}
{{- $first = false -}}
{{- end }}
{{- end }}
{{- if $hasPublic }}
{{- range $key, $value := .Values.istio.gateway.public.selector }}
{{- if not $first }}{{ "\n" }}{{ end }}{{ $key }}: {{ $value | quote }}
{{- $first = false -}}
{{- end }}
{{- end -}}
{{- end -}}

{{/*
Waypoint Proxy Name for Ambient Mode
*/}}
{{- define "livecomments.waypointName" -}}
{{- printf "%s-waypoint" (include "livecomments.fullname" .) }}
{{- end -}}

{{/*
Ambient Mode Namespace Labels
*/}}
{{- define "livecomments.ambientNamespaceLabels" -}}
{{- if .Values.istio.ambient.enabled }}
{{- range $key, $value := .Values.istio.ambient.namespaceLabels }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
Istio Mode Detection - returns "ambient" or "sidecar"
*/}}
{{- define "livecomments.istioMode" -}}
{{- if and .Values.istio.enabled .Values.istio.ambient.enabled -}}
ambient
{{- else if and .Values.istio.enabled .Values.istio.sidecar.enabled -}}
sidecar
{{- else -}}
ambient
{{- end -}}
{{- end -}}

{{/*
Sidecar Injection Annotations
*/}}
{{- define "livecomments.sidecarAnnotations" -}}
{{- if and .Values.istio.enabled .Values.istio.sidecar.enabled (not .Values.istio.ambient.enabled) }}
{{- if eq .Values.istio.sidecar.injection.mode "enabled" }}
{{ .Values.istio.sidecar.injection.podAnnotation }}: "true"
{{- else if eq .Values.istio.sidecar.injection.mode "disabled" }}
{{ .Values.istio.sidecar.injection.podAnnotation }}: "false"
{{- end }}
{{- if .Values.istio.sidecar.proxy.resources }}
sidecar.istio.io/proxyCPU: {{ .Values.istio.sidecar.proxy.resources.requests.cpu | quote }}
sidecar.istio.io/proxyMemory: {{ .Values.istio.sidecar.proxy.resources.requests.memory | quote }}
sidecar.istio.io/proxyCPULimit: {{ .Values.istio.sidecar.proxy.resources.limits.cpu | quote }}
sidecar.istio.io/proxyMemoryLimit: {{ .Values.istio.sidecar.proxy.resources.limits.memory | quote }}
{{- end }}
{{- if .Values.istio.sidecar.proxy.image }}
sidecar.istio.io/proxyImage: {{ .Values.istio.sidecar.proxy.image | quote }}
{{- end }}
sidecar.istio.io/logLevel: {{ .Values.istio.sidecar.proxy.logLevel | quote }}
{{- range $key, $value := .Values.istio.sidecar.proxy.config }}
sidecar.istio.io/{{ $key }}: {{ $value | quote }}
{{- end }}
{{- else if and .Values.istio.enabled .Values.istio.ambient.enabled }}
{{- /* Ambient mode - explicitly disable sidecar injection */ -}}
{{ .Values.istio.sidecar.injection.podAnnotation | default "sidecar.istio.io/inject" }}: "false"
{{- end -}}
{{- end -}}
