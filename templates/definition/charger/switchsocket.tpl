{{ define "switchsocket" }}
standbypower: {{ .standbypower }}
{{- include "featureset" (list "switchdevice"
  (and .integrateddevice "integrateddevice")
  (and .heating "heating")) }}
{{- if .icon }}
icon: {{ .icon }}
{{- end }}
{{- end }}
