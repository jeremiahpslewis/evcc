{{ define "features" }}
{{- include "featureset" (list .basefeatures
  (and .average "average")) }}
{{- end }}
