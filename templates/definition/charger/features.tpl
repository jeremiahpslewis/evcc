{{ define "features" }}
{{- include "featureset" (list .basefeatures
  (and .heating "heating")
  (and .integrateddevice "integrateddevice")) }}
{{- end }}
