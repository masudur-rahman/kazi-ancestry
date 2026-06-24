{{/* Expand the name of the chart. */}}
{{- define "kazi-ancestry.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Fully qualified app name. */}}
{{- define "kazi-ancestry.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "kazi-ancestry.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Common labels. */}}
{{- define "kazi-ancestry.labels" -}}
helm.sh/chart: {{ include "kazi-ancestry.chart" . }}
{{ include "kazi-ancestry.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Selector labels. */}}
{{- define "kazi-ancestry.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kazi-ancestry.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Service account name. */}}
{{- define "kazi-ancestry.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "kazi-ancestry.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/* Name of the Secret holding credentials (existing or rendered). */}}
{{- define "kazi-ancestry.secretName" -}}
{{- if .Values.existingSecret -}}
{{- .Values.existingSecret -}}
{{- else -}}
{{- include "kazi-ancestry.fullname" . -}}
{{- end -}}
{{- end -}}

{{/* OAuth redirect URL: explicit value, else derived from the gateway hostname. */}}
{{- define "kazi-ancestry.redirectUrl" -}}
{{- if .Values.oauth.redirectUrl -}}
{{- .Values.oauth.redirectUrl -}}
{{- else -}}
{{- printf "https://%s/auth/callback" .Values.gateway.hostname -}}
{{- end -}}
{{- end -}}
