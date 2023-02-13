{{/*
Expand the name of the chart.
*/}}
{{- define "app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Expand the name of the variant.
*/}}
{{- define "app.variant" -}}
{{- default .Values.variant .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Expand the namespace.
*/}}
{{- define "app.namespace" -}}
{{- default .Values.namespace .Release.Namespace | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "app.fullname" -}}
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
{{- define "app.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "app.labels" -}}
helm.sh/chart: {{ include "app.chart" . }}
{{ include "app.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{/*{{- if .Values.version }}*/}}
{{/*app.kubernetes.io/version: {{ .Values.version | quote }}*/}}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "app.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "app.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/*
env pgHostPortDB
*/}}
{{- define "app.env.pgHostPortDB" -}}
- name: APP_PG_HOST
  valueFrom:
    configMapKeyRef:
      name: {{ include "app.fullname" . }}-config
      key: APP_PG_HOST
- name: APP_PG_PORT
  valueFrom:
    configMapKeyRef:
      name: {{ include "app.fullname" . }}-config
      key: APP_PG_PORT
- name: APP_PG_DBNAME
  valueFrom:
    configMapKeyRef:
      name: {{ include "app.fullname" . }}-config
      key: APP_PG_DBNAME
{{- end }}

{{/*
env pgUserPass
*/}}
{{- define "app.env.pgUserPass" -}}
- name: APP_PG_USER
  valueFrom:
    secretKeyRef:
      name: {{ include "app.fullname" . }}-secret
      key: APP_PG_USER
- name: APP_PG_PASS
  valueFrom:
    secretKeyRef:
      name: {{ include "app.fullname" . }}-secret
      key: APP_PG_PASS
{{- end }}

{{/*
Wait for DB to be ready
*/}}
{{- define "app.pgWait" -}}
['sh', '-c', "until nc -w 2 $(APP_PG_HOST) $(APP_PG_PORT); do echo Waiting for $(APP_PG_HOST):$(APP_PG_PORT) to be ready; sleep 5; done"]
{{- end }}