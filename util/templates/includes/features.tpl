{{ define "featureset" }}
{{- $features := featureList . }}
{{- if $features }}
features:
{{- range $features }}
- {{ . }}
{{- end }}
{{- end }}
{{- end }}
