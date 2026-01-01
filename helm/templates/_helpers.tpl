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
{{- end }}

{{/*
Selector labels
*/}}
{{- define "sql-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "sql-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "sql-exporter.serviceAccountName" -}}
{{- dig "serviceAccount" "name" "default" .Values }}
{{- end }}

{{- define "sql-exporter.volumes" -}}
{{- if or .Values.createConfig .Values.collectorFiles .Values.webConfig.enabled .Values.dynamicConfig.enabled -}}
{{- true | quote -}}
{{- else if .Values.extraVolumes -}}
{{- true | quote -}}
{{- else -}}
{{- false | quote -}}
{{- end -}}
{{- end -}}

{{- define "sql-exporter.basicAuth.secretName" -}}
{{- if .Values.webConfig.basicAuth.initFromSecret.secretName -}}
{{- .Values.webConfig.basicAuth.initFromSecret.secretName -}}
{{- else -}}
{{- printf "%s-%s" (include "sql-exporter.fullname" .) "web-basic-auth" -}}
{{- end -}}
{{- end -}}

{{- define "sql-exporter.dynamicConfig.volumeName" -}}
{{- printf "%s-dynamic-config" (include "sql-exporter.name" .) -}}
{{- end -}}

{{- define "sql-exporter.dynamicConfig.secretVolumeName" -}}
{{- printf "%s-dsn" (include "sql-exporter.name" .) -}}
{{- end -}}

{{- define "sql-exporter.webconfig.yaml" -}}
{{- $conf := "" -}}
{{- if and .Values.webConfig.template (ne .Values.webConfig.template "") -}}
{{- /* User provided custom template */ -}}
{{- if typeIsLike "string" .Values.webConfig.template -}}
{{- $conf = tpl .Values.webConfig.template . | fromYaml -}}
{{- else -}}
{{- $conf = .Values.webConfig.template -}}
{{- end -}}
{{- tpl ($conf | toYaml ) . | fromYaml | toYaml -}}
{{- else -}}
{{- /* Generate default template */ -}}
tls_server_config:
  cert_file: /tls/{{ .Values.webConfig.tls.certFile }}
  key_file: /tls/{{ .Values.webConfig.tls.keyFile }}
  min_version: TLS13
  prefer_server_cipher_suites: true
  cipher_suites:
    - TLS_AES_128_GCM_SHA256
    - TLS_AES_256_GCM_SHA384
{{- if and .Values.webConfig.basicAuth.enabled (not .Values.webConfig.basicAuth.initFromSecret.enabled) }}
basic_auth_users:
{{- range $user, $hash := .Values.webConfig.basicAuth.users }}
  {{ $user }}: {{ $hash | quote }}
{{- end }}
{{- end }}
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

{{/*
Service port - use explicit value if set, otherwise 443 for TLS, 80 for HTTP
*/}}
{{- define "sql-exporter.servicePort" -}}
{{- if .Values.service.port -}}
{{ .Values.service.port }}
{{- else if and .Values.webConfig.enabled .Values.webConfig.tls.secretName -}}
443
{{- else -}}
80
{{- end -}}
{{- end -}}
