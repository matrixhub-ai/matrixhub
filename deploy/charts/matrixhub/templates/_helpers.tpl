{{- /*
Expand the name of the chart.
*/}}
{{- define "matrixhub.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- /*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "matrixhub.fullname" -}}
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

{{- /*
Create chart name and version as used by the chart label.
*/}}
{{- define "matrixhub.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- /*
Common labels
*/}}
{{- define "matrixhub.labels" -}}
helm.sh/chart: {{ include "matrixhub.chart" . }}
{{ include "matrixhub.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.apiserver.labels }}
{{- range $key, $value := .Values.apiserver.labels }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end }}

{{- /*
Selector labels
*/}}
{{- define "matrixhub.selectorLabels" -}}
app.kubernetes.io/name: {{ include "matrixhub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- /*
MatrixHub API server labels
*/}}
{{- define "matrixhub.apiserver.labels" -}}
{{ include "matrixhub.labels" . }}
{{- end }}

{{- /*
MatrixHub API server pod labels
*/}}
{{- define "matrixhub.apiserver.podLabels" -}}
{{- if .Values.apiserver.podLabels }}
{{- range $key, $value := .Values.apiserver.podLabels }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end }}
{{ include "matrixhub.selectorLabels" . }}
{{- end }}

{{- /*
MatrixHub image
*/}}
{{- define "matrixhub.image" -}}
{{- $registry := .Values.apiserver.image.registry | default .Values.global.imageRegistry | default "" }}
{{- $repository := .Values.apiserver.image.repository }}
{{- $tag := .Values.apiserver.image.tag | default "latest" }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}

{{- /*
MySQL image
*/}}
{{- define "matrixhub.mysql.image" -}}
{{- $registry := .Values.mysql.registry | default .Values.global.imageRegistry | default "" }}
{{- $repository := .Values.mysql.repository }}
{{- $tag := .Values.mysql.tag | default "8.0" }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}

{{- /*
MySQL secret name
*/}}
{{- define "matrixhub.mysql.secretName" -}}
{{- printf "%s-mysql-secret" (include "matrixhub.fullname" .) }}
{{- end }}

{{- /*
MySQL PVC name
*/}}
{{- define "matrixhub.mysql.pvcName" -}}
{{- printf "%s-mysql-pv-claim" (include "matrixhub.fullname" .) }}
{{- end }}

{{- /*
MySQL init ConfigMap name
*/}}
{{- define "matrixhub.mysql.initConfigMapName" -}}
{{- printf "%s-mysql-initdb-config" (include "matrixhub.fullname" .) }}
{{- end }}

{{- /*
Image pull secrets
*/}}
{{- define "matrixhub.imagePullSecrets" -}}
{{- with .Values.global.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end }}

{{- /*
Database DSN
*/}}
{{- define "matrixhub.database.dsn" -}}
{{- if .Values.apiserver.database.dsn }}
{{- .Values.apiserver.database.dsn }}
{{- else if eq .Values.apiserver.database.driver "mysql" }}
{{- printf "matrixhub:%s@tcp(%s-mysql:3306)/matrixhub?charset=utf8mb4&multiStatements=true&parseTime=true" .Values.mysql.rootPassword (include "matrixhub.fullname" .) }}
{{- else if eq .Values.apiserver.database.driver "postgres" }}
{{- printf "user=matrixhub password=%s host=%s-mysql port=5432 dbname=matrixhub sslmode=disable" .Values.mysql.rootPassword (include "matrixhub.fullname" .) }}
{{- end }}
{{- end }}

{{- /*
Validate MatrixHub chart values.
*/}}
{{- define "matrixhub.validateValues" -}}
{{- if eq .Values.apiserver.database.driver "sqlite" }}
{{- fail "SQLite is not supported by the MatrixHub Helm chart; use the single-node Docker Compose deployment instead" }}
{{- end }}
{{- end }}
