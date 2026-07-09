{{- define "go-k8s-deployment.name" -}}
{{ .Chart.Name }}
{{- end -}}

{{- define "go-k8s-deployment.labels" -}}
app.kubernetes.io/name: {{ include "go-k8s-deployment.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
