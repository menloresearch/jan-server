{{/*
Generate chart name
*/}}
{{- define "jan-gateway.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Generate fully qualified app name
*/}}
{{- define "jan-gateway.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name (include "jan-gateway.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Labels
*/}}
{{- define "jan-gateway.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "jan-gateway.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "jan-gateway.selectorLabels" -}}
app.kubernetes.io/name: {{ include "jan-gateway.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "jan-gateway.serviceAccountName" -}}
{{- if .Values.gateway.serviceAccount.create }}
{{- default (include "jan-gateway.fullname" .) .Values.gateway.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.gateway.serviceAccount.name }}
{{- end }}
{{- end }}
