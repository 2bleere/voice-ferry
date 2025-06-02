{{/*
Expand the name of the chart.
*/}}
{{- define "voice-ferry.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "voice-ferry.fullname" -}}
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
{{- define "voice-ferry.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "voice-ferry.labels" -}}
helm.sh/chart: {{ include "voice-ferry.chart" . }}
{{ include "voice-ferry.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "voice-ferry.selectorLabels" -}}
app.kubernetes.io/name: {{ include "voice-ferry.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "voice-ferry.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "voice-ferry.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the Redis connection string
*/}}
{{- define "voice-ferry.redisConnection" -}}
{{- if .Values.redis.external.enabled }}
{{- range $index, $host := .Values.redis.external.hosts }}
{{- if $index }},{{ end }}{{ $host }}
{{- end }}
{{- else }}
{{- printf "%s-redis:6379" .Release.Name }}
{{- end }}
{{- end }}

{{/*
Create the etcd connection string
*/}}
{{- define "voice-ferry.etcdConnection" -}}
{{- if .Values.etcd.external.enabled }}
{{- range $index, $endpoint := .Values.etcd.external.endpoints }}
{{- if $index }},{{ end }}{{ $endpoint }}
{{- end }}
{{- else }}
{{- printf "%s-etcd:2379" .Release.Name }}
{{- end }}
{{- end }}

{{/*
Create RTPEngine service name
*/}}
{{- define "voice-ferry.rtpengineService" -}}
{{- printf "%s-rtpengine" (include "voice-ferry.fullname" .) }}
{{- end }}

{{/*
Create TLS secret name
*/}}
{{- define "voice-ferry.tlsSecretName" -}}
{{- if .Values.tls.secretName }}
{{- .Values.tls.secretName }}
{{- else }}
{{- printf "%s-tls" (include "voice-ferry.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create ConfigMap name
*/}}
{{- define "voice-ferry.configMapName" -}}
{{- printf "%s-config" (include "voice-ferry.fullname" .) }}
{{- end }}

{{/*
Create image name
*/}}
{{- define "voice-ferry.image" -}}
{{- $registry := .Values.voiceFerry.image.registry }}
{{- $repository := .Values.voiceFerry.image.repository }}
{{- $tag := .Values.voiceFerry.image.tag | default .Chart.AppVersion }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}

{{/*
Create RTPEngine image name
*/}}
{{- define "voice-ferry.rtpengineImage" -}}
{{- $registry := .Values.rtpengine.image.registry }}
{{- $repository := .Values.rtpengine.image.repository }}
{{- $tag := .Values.rtpengine.image.tag }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}
