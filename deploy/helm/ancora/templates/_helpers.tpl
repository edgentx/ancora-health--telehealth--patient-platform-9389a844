{{/*
Chart name (respecting nameOverride).
*/}}
{{- define "ancora.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified app name. Truncated at 63 chars for the k8s name limit.
*/}}
{{- define "ancora.fullname" -}}
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

{{/*
Chart label value: "name-version".
*/}}
{{- define "ancora.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
The dedicated namespace all resources deploy into.
*/}}
{{- define "ancora.namespace" -}}
{{- default .Release.Namespace .Values.namespace.name -}}
{{- end -}}

{{/*
Common labels applied to every object.
*/}}
{{- define "ancora.labels" -}}
helm.sh/chart: {{ include "ancora.chart" . }}
{{ include "ancora.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: ancora-platform
{{- end -}}

{{/*
Selector labels (immutable subset used in matchLabels).
*/}}
{{- define "ancora.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ancora.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
ServiceAccount name.
*/}}
{{- define "ancora.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "ancora.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Name of the Secret components consume: an out-of-band existingSecret when set,
otherwise the chart-rendered Secret.
*/}}
{{- define "ancora.secretName" -}}
{{- if .Values.secrets.existingSecret -}}
{{- .Values.secrets.existingSecret -}}
{{- else -}}
{{- printf "%s-secrets" (include "ancora.fullname" .) -}}
{{- end -}}
{{- end -}}

{{/*
ConfigMap name.
*/}}
{{- define "ancora.configMapName" -}}
{{- printf "%s-config" (include "ancora.fullname" .) -}}
{{- end -}}

{{/*
Full image reference for a component's build target.
Usage: include "ancora.image" (dict "root" $ "target" "api")
*/}}
{{- define "ancora.image" -}}
{{- $img := .root.Values.image -}}
{{- $tag := default .root.Chart.AppVersion $img.tag -}}
{{- printf "%s/%s/%s:%s" $img.registry $img.repository .target $tag -}}
{{- end -}}
