{{ define "heatpumpswitch" }}
{{- include "featureset" (list "continuous" "heating" "integrateddevice" "switchdevice") }}
{{- end }}
