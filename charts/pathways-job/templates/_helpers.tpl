{{/*
Expand the name of the chart.
*/}}
{{- define "pathways-job.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "pathways-job.fullname" -}}
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
{{- define "pathways-job.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels of the pathways-job resources.
*/}}
{{- define "pathways-job.labels" -}}
helm.sh/chart: {{ include "pathways-job.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Chart.Version }}
app.kubernetes.io/version: {{ . | quote }}
{{- end }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{ include "pathways-job.selectorLabels" . }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "pathways-job.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pathways-job.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the pathways-job controller.
*/}}
{{- define "pathways-job.controller.name" -}}
{{ include "pathways-job.fullname" . }}
{{- end -}}

{{/*
Common labels of the pathways-job controller.
*/}}
{{- define "pathways-job.controller.labels" -}}
{{ include "pathways-job.labels" . }}
app.kubernetes.io/component: controller
{{- end -}}

{{/*
Create the name of the pathways-job controller deployment.
*/}}
{{- define "pathways-job.controller.deployment.name" -}}
{{ include "pathways-job.controller.name" . }}
{{- end -}}

{{/*
Create the name of the pathways-job manager role
*/}}
{{- define "pathways-job.managerRole.name" -}}
{{ include "pathways-job.fullname" . }}-manager-role
{{- end -}}

{{/*
Create the name of the pathways-job manager rolebinding
*/}}
{{- define "pathways-job.managerRoleBinding.name" -}}
{{ include "pathways-job.fullname" . }}-manager-rolebinding
{{- end -}}

# Pathways Jobs Roles and Bindings
{{/*
Create the name of the pathways-job editor role.
*/}}
{{- define "pathways-job.controller.pathwaysJobEditorRole.name" -}}
{{ include "pathways-job.controller.name" . }}-pathwaysjob-editor-role
{{- end -}}

{{/*
Create the name of the pathways-job viewer role.
*/}}
{{- define "pathways-job.controller.pathwaysJobViewerRole.name" -}}
{{ include "pathways-job.controller.name" . }}-pathwaysjob-viewer-role
{{- end -}}

# Service Account
{{/*
Create the name of the pathways-job controller service account.
*/}}
{{- define "pathways-job.controller.serviceAccount.name" -}}
{{ include "pathways-job.fullname" . }}-controller-manager
{{- end -}}

# Leader Election
{{/*
Create the name of the pathways-job controller leader election role.
*/}}
{{- define "pathways-job.controller.leaderElectionRole.name" -}}
{{ include "pathways-job.controller.name" . }}-leader-election-role
{{- end -}}

{{/*
Create the name of the pathways-job controller leader election role binding.
*/}}
{{- define "pathways-job.controller.leaderElectionRoleBinding.name" -}}
{{ include "pathways-job.controller.name" . }}-leader-election-rolebinding
{{- end -}}

# Metrics Auth and Reader
{{/*
Create the name of the pathways-job controller metrics auth role.
*/}}
{{- define "pathways-job.controller.metricsAuthRole.name" -}}
{{ include "pathways-job.controller.name" . }}-metrics-auth-role
{{- end -}}

{{/*
Create the name of the pathways-job controller metrics auth role binding.
*/}}
{{- define "pathways-job.controller.metricsAuthRoleBinding.name" -}}
{{ include "pathways-job.controller.name" . }}-metrics-auth-rolebinding
{{- end -}}

{{/*
Create the name of the pathways-job controller metrics reader role.
*/}}
{{- define "pathways-job.controller.metricsReaderRole.name" -}}
{{ include "pathways-job.controller.name" . }}-metrics-reader
{{- end -}}

{{/*
Create the name of the pathways-job metrics service.
*/}}
{{- define "pathways-job.metrics.service.name" -}}
{{ include "pathways-job.metrics.name" . }}-metrics-service
{{- end -}}

{{/*
Create the name of the pathways-job controller configmap.
*/}}
{{- define "pathways-job.controller.configMap.name" -}}
{{ include "pathways-job.controller.name" . }}-config
{{- end -}}
