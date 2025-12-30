{{/*
Expand the name of the chart.
*/}}
{{- define "sql-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "sql-exporter.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "sql-exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create tls secret name based on the chart name
*/}}
{{- define "sql-exporter.tls.name" -}}
{{- if ((.Values.ingress).tls).secretName -}}
{{- .Values.ingress.tls.secretName }}
{{- else -}}
{{- printf "%s-%s" (include "sql-exporter.fullname" .) "tls" }}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "sql-exporter.labels" -}}
helm.sh/chart: {{ include "sql-exporter.chart" . }}
{{ include "sql-exporter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "sql-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sql-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common annotations
*/}}
{{- define "sql-exporter.annotations" -}}
{{- if .Values.commonAnnotations }}
{{ toYaml .Values.commonAnnotations }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "sql-exporter.serviceAccountName" -}}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}

{{- define "sql-exporter.volumes" -}}
{{- if or .Values.createConfig .Values.collectorFiles -}}
{{- true | quote -}}
{{- else if .Values.extraVolumes -}}
{{- true | quote -}}
{{- else -}}
{{- false | quote -}}
{{- end -}}
{{- end -}}

{{- define "sql_exporter.config.yaml" -}}
{{- $conf := "" -}}
{{- if typeIsLike "string" .Values.config -}}
{{- $conf = (tpl .Values.config .) | fromYaml -}}
{{- else -}}
{{- $conf = .Values.config -}}
{{- end -}}
{{- /*
Do the wired "fromYaml | toYaml" to reformat the config.
Reformat '100s' to 100s for example.
*/ -}}
{{- tpl ($conf | toYaml ) . | fromYaml | toYaml -}}
{{- end -}}
