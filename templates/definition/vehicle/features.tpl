{{ define "features" }}
{{- include "featureset" (list .basefeatures
  (and .coarsecurrent "coarsecurrent")
  (and .welcomecharge "welcomecharge")
  (and .streaming "streaming")
  (and .climaterdisabled "climaterdisabled")
  (and .autodetectdisabled "autodetectdisabled")
  (and .wakeupdisabled "wakeupdisabled")) }}
{{- end }}
