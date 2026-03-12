{{/*
Expand the name of the chart, truncated to 63 chars (Kubernetes label limit).
*/}}
{{- define "k8s-gateway-healthcheck.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a fully qualified name: release-chart, truncated to 63 chars.
If fullnameOverride is set, it takes precedence entirely.
*/}}
{{- define "k8s-gateway-healthcheck.fullname" -}}
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
Chart label: "chart-name-chart-version" with + replaced for label validity.
*/}}
{{- define "k8s-gateway-healthcheck.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Selector labels — used by Service selector and Deployment matchLabels.
IMMUTABLE after first deploy: never add dynamic values here.
*/}}
{{- define "k8s-gateway-healthcheck.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-gateway-healthcheck.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common labels — applied to all resources.
Includes selector labels plus additional metadata labels that can safely change.
*/}}
{{- define "k8s-gateway-healthcheck.commonLabels" -}}
helm.sh/chart: {{ include "k8s-gateway-healthcheck.chart" . }}
{{ include "k8s-gateway-healthcheck.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: healthcheck
{{- end }}

{{/*
Full image reference: registry/repository:tag
*/}}
{{- define "k8s-gateway-healthcheck.image" -}}
{{- printf "%s/%s:%s" .Values.image.registry .Values.image.repository .Values.image.tag }}
{{- end }}
